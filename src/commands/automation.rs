use anyhow::Result;
use clap::{Args, Subcommand};
use serde_json::Value;

use crate::api;
use crate::cli::Ctx;
use crate::output;
use crate::util::{jint, jstr};

#[derive(Args)]
pub struct Cmd {
    #[command(subcommand)]
    command: Sub,
}

#[derive(Subcommand)]
enum Sub {
    /// List automations
    List {
        /// maximum number of automations to return (0 = all)
        #[arg(long, default_value_t = 25)]
        limit: u64,
        /// filter by enabled status (true, false)
        #[arg(long, default_value = "")]
        enabled: String,
    },
    /// Get automation details
    #[command(arg_required_else_help = true)]
    Get {
        /// automation ID
        automation_id: String,
    },
    /// List automation subscriber activity
    #[command(arg_required_else_help = true)]
    Subscribers {
        /// automation ID
        automation_id: String,
        /// maximum number of subscribers to return (0 = all)
        #[arg(long, default_value_t = 25)]
        limit: u64,
    },
}

pub fn run(ctx: &Ctx, cmd: Cmd) -> Result<()> {
    match cmd.command {
        Sub::List { limit, enabled } => list(ctx, limit, &enabled),
        Sub::Get { automation_id } => get(ctx, &automation_id),
        Sub::Subscribers {
            automation_id,
            limit,
        } => subscribers(ctx, &automation_id, limit),
    }
}

fn yes_no(v: &Value, key: &str) -> &'static str {
    if v.get(key).and_then(Value::as_bool).unwrap_or(false) {
        "Yes"
    } else {
        "No"
    }
}

fn list(ctx: &Ctx, limit: u64, enabled: &str) -> Result<()> {
    let client = ctx.client()?;

    let mut query: Vec<(&str, String)> = Vec::new();
    if !enabled.is_empty() {
        query.push(("filter[enabled]", enabled.to_string()));
    }

    let automations = api::fetch_all_paged(&client, "/automations", &query, limit)?;

    if ctx.json {
        return output::json(&automations);
    }

    let rows: Vec<Vec<String>> = automations
        .iter()
        .map(|a| {
            let stats = a.get("stats").cloned().unwrap_or(Value::Null);
            vec![
                jstr(a, "id"),
                output::truncate(&jstr(a, "name"), 40),
                yes_no(a, "enabled").to_string(),
                jint(a, "emails_count").to_string(),
                jint(&stats, "completed_subscribers_count").to_string(),
                jint(&stats, "subscribers_in_queue_count").to_string(),
            ]
        })
        .collect();

    output::table(
        &["ID", "NAME", "ENABLED", "EMAILS", "COMPLETED", "IN QUEUE"],
        &rows,
    );
    Ok(())
}

fn get(ctx: &Ctx, id: &str) -> Result<()> {
    let client = ctx.client()?;
    let body = client.get(&format!("/automations/{id}"))?;

    if ctx.json {
        return output::json(&body);
    }

    let d = body.get("data").cloned().unwrap_or(Value::Null);

    println!("ID:           {}", jstr(&d, "id"));
    println!("Name:         {}", jstr(&d, "name"));
    println!("Enabled:      {}", yes_no(&d, "enabled"));
    println!("Emails:       {}", jint(&d, "emails_count"));
    println!("Created At:   {}", jstr(&d, "created_at"));

    let stats = d.get("stats").cloned().unwrap_or(Value::Null);
    println!();
    println!("Stats:");
    println!(
        "  Completed:  {}",
        jint(&stats, "completed_subscribers_count")
    );
    println!(
        "  In Queue:   {}",
        jint(&stats, "subscribers_in_queue_count")
    );
    println!("  Sent:       {}", jint(&stats, "sent"));
    println!("  Opens:      {}", jint(&stats, "opens_count"));
    println!("  Clicks:     {}", jint(&stats, "clicks_count"));

    if let Some(steps) = d.get("steps").and_then(Value::as_array) {
        if !steps.is_empty() {
            println!();
            println!("Steps:");
            for s in steps {
                println!("  - {} ({})", jstr(s, "description"), jstr(s, "type"));
            }
        }
    }

    Ok(())
}

fn subscribers(ctx: &Ctx, id: &str, limit: u64) -> Result<()> {
    let client = ctx.client()?;

    let subscribers =
        api::fetch_all_paged(&client, &format!("/automations/{id}/activity"), &[], limit)?;

    if ctx.json {
        return output::json(&subscribers);
    }

    let rows: Vec<Vec<String>> = subscribers
        .iter()
        .map(|s| {
            let subscriber = s.get("subscriber").cloned().unwrap_or(Value::Null);
            vec![
                jstr(s, "id"),
                jstr(&subscriber, "email"),
                jstr(s, "status"),
                jstr(s, "date"),
            ]
        })
        .collect();

    output::table(&["ID", "EMAIL", "STATUS", "DATE"], &rows);
    Ok(())
}
