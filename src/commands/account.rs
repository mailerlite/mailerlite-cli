use anyhow::{anyhow, bail, Result};
use clap::{Args, Subcommand};
use serde_json::Value;

use crate::cli::Ctx;
use crate::config;
use crate::output;
use crate::prompt;
use crate::util::jstr;

#[derive(Args)]
#[command(
    long_about = "List and switch between MailerLite accounts (requires OAuth authentication)."
)]
pub struct Cmd {
    #[command(subcommand)]
    command: Sub,
}

#[derive(Subcommand)]
enum Sub {
    /// List accounts you have access to
    List,
    /// Switch active account
    #[command(
        long_about = "Switch to a different MailerLite account. If no account ID is provided, shows an interactive picker."
    )]
    Switch {
        /// account ID
        account_id: Option<String>,
    },
}

pub fn run(ctx: &Ctx, cmd: Cmd) -> Result<()> {
    match cmd.command {
        Sub::List => list(ctx),
        Sub::Switch { account_id } => switch(ctx, account_id),
    }
}

fn fetch_accounts(ctx: &Ctx) -> Result<Vec<Value>> {
    let client = ctx.client()?;
    let body = client
        .get("/accounts")
        .map_err(|e| anyhow!("failed to fetch accounts: {e}"))?;
    Ok(body
        .get("data")
        .and_then(Value::as_array)
        .cloned()
        .unwrap_or_default())
}

fn list(ctx: &Ctx) -> Result<()> {
    let accounts = fetch_accounts(ctx)?;

    if ctx.json {
        return output::json(&accounts);
    }

    let active_id = config::get_account_id(ctx.profile.as_deref()).unwrap_or_default();
    let rows: Vec<Vec<String>> = accounts
        .iter()
        .map(|a| {
            let mut name = jstr(a, "name");
            if !active_id.is_empty() && jstr(a, "id") == active_id {
                name.push_str(" (active)");
            }
            vec![jstr(a, "id"), name, jstr(a, "status")]
        })
        .collect();

    output::table(&["ID", "Name", "Status"], &rows);
    Ok(())
}

fn switch(ctx: &Ctx, account_arg: Option<String>) -> Result<()> {
    let accounts = fetch_accounts(ctx)?;

    let account_id = if let Some(id) = account_arg {
        if !accounts.iter().any(|a| jstr(a, "id") == id) {
            bail!("account {id:?} not found; use 'account list' to see available accounts");
        }
        id
    } else if accounts.is_empty() {
        bail!("no accounts found");
    } else if accounts.len() == 1 {
        let id = jstr(&accounts[0], "id");
        output::success(&format!(
            "Only one account available: {} ({})",
            jstr(&accounts[0], "name"),
            id
        ));
        id
    } else if !prompt::is_interactive() {
        bail!("account ID argument is required in non-interactive mode");
    } else {
        let active_id = config::get_account_id(ctx.profile.as_deref()).unwrap_or_default();
        let labels = accounts
            .iter()
            .map(|a| {
                let mut label = format!("{} ({})", jstr(a, "name"), jstr(a, "id"));
                if !active_id.is_empty() && jstr(a, "id") == active_id {
                    label.push_str(" [active]");
                }
                label
            })
            .collect();
        let values = accounts.iter().map(|a| jstr(a, "id")).collect();
        prompt::select_labeled("Select account", labels, values)?
    };

    let mut cfg = config::load()?;
    let prof_name = match ctx.profile.clone().filter(|s| !s.is_empty()) {
        Some(n) => n,
        None => config::active_profile(&cfg)?.0,
    };

    let Some(prof) = cfg.profiles.get_mut(&prof_name) else {
        bail!("profile {prof_name:?} not found");
    };
    prof.account_id = Some(account_id.clone());
    config::save(&cfg)?;

    output::success(&format!("Switched to account: {account_id}"));
    Ok(())
}
