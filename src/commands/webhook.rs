use anyhow::Result;
use clap::{Args, Subcommand};
use serde_json::{json, Value};

use crate::api;
use crate::cli::Ctx;
use crate::output;
use crate::prompt;
use crate::util::jstr;

const VALID_EVENTS: &str = "subscriber.created, subscriber.updated, subscriber.unsubscribed, subscriber.added_to_group, subscriber.removed_from_group, subscriber.bounced, subscriber.automation_triggered, subscriber.automation_completed, campaign.sent, campaign.draft_created";

#[derive(Args)]
#[command(long_about = "List, view, create, update, and delete webhooks.")]
pub struct Cmd {
    #[command(subcommand)]
    command: Sub,
}

#[derive(Subcommand)]
enum Sub {
    /// List webhooks
    List {
        /// maximum number of webhooks to return (0 = all)
        #[arg(long, default_value_t = 25)]
        limit: u64,
        /// sort order
        #[arg(long, default_value = "")]
        sort: String,
    },
    /// Get webhook details
    #[command(arg_required_else_help = true)]
    Get {
        /// webhook ID
        webhook_id: String,
    },
    /// Create a webhook
    #[command(long_about = format!("Create a new webhook.\n\nValid events: {VALID_EVENTS}"))]
    Create {
        /// webhook name (required)
        #[arg(long, default_value = "")]
        name: String,
        /// webhook URL (required)
        #[arg(long, default_value = "")]
        url: String,
        /// webhook events (required)
        #[arg(long, value_delimiter = ',')]
        events: Option<Vec<String>>,
        /// whether the webhook is enabled
        #[arg(long, num_args = 0..=1, default_missing_value = "true")]
        enabled: Option<bool>,
    },
    /// Update a webhook
    #[command(
        arg_required_else_help = true,
        long_about = format!("Update an existing webhook.\n\nValid events: {VALID_EVENTS}")
    )]
    Update {
        /// webhook ID
        webhook_id: String,
        /// webhook name
        #[arg(long)]
        name: Option<String>,
        /// webhook URL
        #[arg(long)]
        url: Option<String>,
        /// webhook events
        #[arg(long, value_delimiter = ',')]
        events: Option<Vec<String>>,
        /// whether the webhook is enabled
        #[arg(long, num_args = 0..=1, default_missing_value = "true")]
        enabled: Option<bool>,
    },
    /// Delete a webhook
    #[command(arg_required_else_help = true)]
    Delete {
        /// webhook ID
        webhook_id: String,
    },
}

pub fn run(ctx: &Ctx, cmd: Cmd) -> Result<()> {
    match cmd.command {
        Sub::List { limit, sort } => list(ctx, limit, &sort),
        Sub::Get { webhook_id } => get(ctx, &webhook_id),
        Sub::Create {
            name,
            url,
            events,
            enabled: _,
        } => create(ctx, &name, &url, events),
        Sub::Update {
            webhook_id,
            name,
            url,
            events,
            enabled,
        } => update(ctx, &webhook_id, name, url, events, enabled),
        Sub::Delete { webhook_id } => delete(ctx, &webhook_id),
    }
}

fn list(ctx: &Ctx, limit: u64, sort: &str) -> Result<()> {
    let client = ctx.client()?;

    let mut query: Vec<(&str, String)> = Vec::new();
    if !sort.is_empty() {
        query.push(("sort", sort.to_string()));
    }

    let webhooks = api::fetch_all_paged(&client, "/webhooks", &query, limit)?;

    if ctx.json {
        return output::json(&webhooks);
    }

    let rows: Vec<Vec<String>> = webhooks
        .iter()
        .map(|w| {
            let enabled = if w.get("enabled").and_then(Value::as_bool).unwrap_or(false) {
                "Yes"
            } else {
                "No"
            };
            vec![
                jstr(w, "id"),
                output::truncate(&jstr(w, "name"), 40),
                output::truncate(&jstr(w, "url"), 50),
                enabled.to_string(),
                jstr(w, "created_at"),
            ]
        })
        .collect();

    output::table(&["ID", "NAME", "URL", "ENABLED", "CREATED AT"], &rows);
    Ok(())
}

fn get(ctx: &Ctx, webhook_id: &str) -> Result<()> {
    let client = ctx.client()?;
    let body = client.get(&format!("/webhooks/{webhook_id}"))?;

    if ctx.json {
        return output::json(&body);
    }

    let d = body.get("data").cloned().unwrap_or(Value::Null);

    let enabled = if d.get("enabled").and_then(Value::as_bool).unwrap_or(false) {
        "Yes"
    } else {
        "No"
    };

    println!("ID:           {}", jstr(&d, "id"));
    println!("Name:         {}", jstr(&d, "name"));
    println!("URL:          {}", jstr(&d, "url"));
    println!("Enabled:      {enabled}");
    println!("Created At:   {}", jstr(&d, "created_at"));
    println!("Updated At:   {}", jstr(&d, "updated_at"));

    println!();
    println!("Events:");
    if let Some(events) = d.get("events").and_then(Value::as_array) {
        for e in events {
            match e {
                Value::String(s) => println!("  - {s}"),
                other => println!("  - {other}"),
            }
        }
    }

    Ok(())
}

fn create(ctx: &Ctx, name: &str, url: &str, events: Option<Vec<String>>) -> Result<()> {
    let client = ctx.client()?;
    let name = prompt::require_arg(name, "name", "Webhook name")?;
    let url = prompt::require_arg(url, "url", "Webhook URL")?;
    let events =
        prompt::require_slice_arg(&events.unwrap_or_default(), "events", "Webhook events")?;

    let body = json!({ "name": name, "url": url, "events": events });
    let result = client.post("/webhooks", &body)?;

    if ctx.json {
        return output::json(&result);
    }

    let id = result
        .get("data")
        .map(|d| jstr(d, "id"))
        .unwrap_or_default();
    output::success(&format!("Webhook created successfully. ID: {id}"));
    Ok(())
}

fn update(
    ctx: &Ctx,
    webhook_id: &str,
    name: Option<String>,
    url: Option<String>,
    events: Option<Vec<String>>,
    enabled: Option<bool>,
) -> Result<()> {
    let client = ctx.client()?;

    let mut body = json!({});
    if let Some(name) = name {
        body["name"] = json!(name);
    }
    if let Some(url) = url {
        body["url"] = json!(url);
    }
    if let Some(events) = events {
        body["events"] = json!(events);
    }
    if let Some(enabled) = enabled {
        body["enabled"] = json!(if enabled { "true" } else { "false" });
    }

    let result = client.put(&format!("/webhooks/{webhook_id}"), &body)?;

    if ctx.json {
        return output::json(&result);
    }

    output::success(&format!("Webhook {webhook_id} updated successfully."));
    Ok(())
}

fn delete(ctx: &Ctx, webhook_id: &str) -> Result<()> {
    let client = ctx.client()?;

    if !ctx.confirm(&format!("Delete webhook {webhook_id}?"))? {
        return Ok(());
    }

    client.delete(&format!("/webhooks/{webhook_id}"))?;
    output::success(&format!("Webhook {webhook_id} deleted successfully."));
    Ok(())
}
