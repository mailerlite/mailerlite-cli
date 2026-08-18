use anyhow::{bail, Result};
use clap::{Args, Subcommand};
use serde_json::json;

use crate::cli::Ctx;
use crate::config;
use crate::output;
use crate::prompt;

#[derive(Args)]
pub struct Cmd {
    #[command(subcommand)]
    command: Sub,
}

#[derive(Subcommand)]
enum Sub {
    /// Add a new profile
    #[command(arg_required_else_help = true)]
    Add {
        /// profile name
        name: String,
        /// API token for this profile
        #[arg(long, default_value = "")]
        token: String,
    },
    /// List all profiles
    List,
    /// Switch active profile
    #[command(arg_required_else_help = true)]
    Switch {
        /// profile name
        name: String,
    },
    /// Remove a profile
    #[command(arg_required_else_help = true)]
    Remove {
        /// profile name
        name: String,
    },
}

pub fn run(ctx: &Ctx, cmd: Cmd) -> Result<()> {
    match cmd.command {
        Sub::Add { name, token } => add(ctx, &name, token),
        Sub::List => list(ctx),
        Sub::Switch { name } => switch(&name),
        Sub::Remove { name } => remove(ctx, &name),
    }
}

fn add(ctx: &Ctx, name: &str, mut token: String) -> Result<()> {
    if token.is_empty() && prompt::is_interactive() {
        token = prompt::input("API Token", "")?;
    }
    if token.is_empty() {
        bail!("--token is required");
    }

    let mut cfg = config::load()?;

    if cfg.profiles.contains_key(name)
        && !ctx.confirm(&format!("Profile {name:?} already exists. Overwrite?"))?
    {
        return Ok(());
    }

    cfg.profiles.insert(
        name.to_string(),
        config::Profile {
            api_token: Some(token),
            ..Default::default()
        },
    );
    if cfg.active_profile.is_empty() {
        cfg.active_profile = name.to_string();
    }

    config::save(&cfg)?;

    output::success(&format!("Profile {name:?} added."));
    Ok(())
}

fn list(ctx: &Ctx) -> Result<()> {
    let cfg = config::load()?;

    if ctx.json {
        let profiles: Vec<_> = cfg
            .profiles
            .iter()
            .map(|(name, p)| {
                json!({
                    "name": name,
                    "active": *name == cfg.active_profile,
                    "has_token": p.api_token.as_deref().is_some_and(|t| !t.is_empty()),
                })
            })
            .collect();
        return output::json(&profiles);
    }

    if cfg.profiles.is_empty() {
        println!("No profiles configured. Run 'mailerlite profile add <name>' to create one.");
        return Ok(());
    }

    let rows: Vec<Vec<String>> = cfg
        .profiles
        .keys()
        .map(|name| {
            let active = if *name == cfg.active_profile { "*" } else { "" };
            vec![active.to_string(), name.clone(), "token".to_string()]
        })
        .collect();

    output::table(&["", "NAME", "METHOD"], &rows);
    Ok(())
}

fn switch(name: &str) -> Result<()> {
    let mut cfg = config::load()?;

    if !cfg.profiles.contains_key(name) {
        bail!("profile {name:?} not found");
    }

    cfg.active_profile = name.to_string();
    config::save(&cfg)?;

    output::success(&format!("Switched to profile: {name}"));
    Ok(())
}

fn remove(ctx: &Ctx, name: &str) -> Result<()> {
    let mut cfg = config::load()?;

    if !cfg.profiles.contains_key(name) {
        bail!("profile {name:?} not found");
    }

    if !ctx.confirm(&format!("Remove profile {name:?}?"))? {
        return Ok(());
    }

    cfg.profiles.remove(name);
    if cfg.active_profile == name {
        cfg.active_profile = cfg.profiles.keys().next().cloned().unwrap_or_default();
    }

    config::save(&cfg)?;

    output::success(&format!("Profile {name:?} removed."));
    Ok(())
}
