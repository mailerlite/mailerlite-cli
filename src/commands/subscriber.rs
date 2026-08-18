use anyhow::Result;
use clap::{Args, Subcommand};
use serde_json::{json, Value};

use crate::api;
use crate::cli::Ctx;
use crate::output;
use crate::prompt;
use crate::util::{jfloat, jint, jstr, parse_kv_fields};

#[derive(Args)]
pub struct Cmd {
    #[command(subcommand)]
    command: Sub,
}

#[derive(Subcommand)]
enum Sub {
    /// List subscribers
    List {
        /// maximum number of subscribers to return (0 = all)
        #[arg(long, default_value_t = 25)]
        limit: u64,
        /// filter by status (active, unsubscribed, unconfirmed, bounced, junk)
        #[arg(long, default_value = "")]
        status: String,
        /// filter by email address
        #[arg(long, default_value = "")]
        email: String,
    },
    /// Get total subscriber count
    Count,
    /// Get subscriber details
    #[command(arg_required_else_help = true)]
    Get {
        /// subscriber ID or email
        id_or_email: String,
    },
    /// Create or update a subscriber
    Upsert {
        /// subscriber email (required)
        #[arg(long, default_value = "")]
        email: String,
        /// subscriber status
        #[arg(long)]
        status: Option<String>,
        /// group IDs to assign
        #[arg(long, value_delimiter = ',')]
        groups: Option<Vec<String>>,
        /// custom fields as key=value pairs
        #[arg(long, value_delimiter = ',')]
        fields: Option<Vec<String>>,
    },
    /// Update a subscriber
    #[command(arg_required_else_help = true)]
    Update {
        /// subscriber ID
        id: String,
        /// subscriber email
        #[arg(long)]
        email: Option<String>,
        /// subscriber status
        #[arg(long)]
        status: Option<String>,
        /// custom fields as key=value pairs
        #[arg(long, value_delimiter = ',')]
        fields: Option<Vec<String>>,
    },
    /// Delete a subscriber
    #[command(arg_required_else_help = true)]
    Delete {
        /// subscriber ID
        id: String,
    },
    /// Forget a subscriber (GDPR)
    #[command(
        arg_required_else_help = true,
        long_about = "Permanently forget a subscriber and all their data. This action cannot be undone."
    )]
    Forget {
        /// subscriber ID
        id: String,
    },
}

pub fn run(ctx: &Ctx, cmd: Cmd) -> Result<()> {
    match cmd.command {
        Sub::List {
            limit,
            status,
            email,
        } => list(ctx, limit, &status, &email),
        Sub::Count => count(ctx),
        Sub::Get { id_or_email } => get(ctx, &id_or_email),
        Sub::Upsert {
            email,
            status,
            groups,
            fields,
        } => upsert(ctx, &email, status, groups, fields),
        Sub::Update {
            id,
            email,
            status,
            fields,
        } => update(ctx, &id, email, status, fields),
        Sub::Delete { id } => delete(ctx, &id),
        Sub::Forget { id } => forget(ctx, &id),
    }
}

fn list(ctx: &Ctx, limit: u64, status: &str, email: &str) -> Result<()> {
    let client = ctx.client()?;

    let mut query: Vec<(&str, String)> = Vec::new();
    if !status.is_empty() {
        query.push(("filter[status]", status.to_string()));
    }
    if !email.is_empty() {
        query.push(("filter[email]", email.to_string()));
    }

    let subscribers = api::fetch_all_cursor(&client, "/subscribers", &query, limit)?;

    if ctx.json {
        return output::json(&subscribers);
    }

    let rows: Vec<Vec<String>> = subscribers
        .iter()
        .map(|s| {
            vec![
                output::truncate(&jstr(s, "email"), 40),
                jstr(s, "status"),
                jstr(s, "source"),
                jint(s, "opens_count").to_string(),
                jint(s, "clicks_count").to_string(),
                jstr(s, "subscribed_at"),
            ]
        })
        .collect();

    output::table(
        &[
            "EMAIL",
            "STATUS",
            "SOURCE",
            "OPENS",
            "CLICKS",
            "SUBSCRIBED AT",
        ],
        &rows,
    );
    Ok(())
}

fn count(ctx: &Ctx) -> Result<()> {
    let client = ctx.client()?;
    let body = client.get_with("/subscribers", &[("limit", "0".to_string())])?;

    if ctx.json {
        return output::json(&body);
    }

    println!("Total subscribers: {}", jint(&body, "total"));
    Ok(())
}

fn get(ctx: &Ctx, id_or_email: &str) -> Result<()> {
    let client = ctx.client()?;
    let body = client.get(&format!("/subscribers/{id_or_email}"))?;

    if ctx.json {
        return output::json(&body);
    }

    let s = body.get("data").cloned().unwrap_or(Value::Null);
    println!("ID:            {}", jstr(&s, "id"));
    println!("Email:         {}", jstr(&s, "email"));
    println!("Status:        {}", jstr(&s, "status"));
    println!("Source:        {}", jstr(&s, "source"));
    println!("Opens:         {}", jint(&s, "opens_count"));
    println!("Clicks:        {}", jint(&s, "clicks_count"));
    println!("Open Rate:     {:.2}%", jfloat(&s, "open_rate"));
    println!("Click Rate:    {:.2}%", jfloat(&s, "click_rate"));
    println!("Subscribed At: {}", jstr(&s, "subscribed_at"));
    println!("Created At:    {}", jstr(&s, "created_at"));
    println!("Updated At:    {}", jstr(&s, "updated_at"));

    if let Some(fields) = s.get("fields").and_then(Value::as_object) {
        let set: Vec<(&String, &Value)> = fields.iter().filter(|(_, v)| !v.is_null()).collect();
        if !set.is_empty() {
            println!();
            println!("Fields:");
            for (k, v) in set {
                match v {
                    Value::String(s) => println!("  {k}: {s}"),
                    other => println!("  {k}: {other}"),
                }
            }
        }
    }

    if let Some(groups) = s.get("groups").and_then(Value::as_array) {
        if !groups.is_empty() {
            println!();
            println!("Groups:");
            for g in groups {
                println!("  - {} ({})", jstr(g, "name"), jstr(g, "id"));
            }
        }
    }

    Ok(())
}

fn upsert(
    ctx: &Ctx,
    email: &str,
    status: Option<String>,
    groups: Option<Vec<String>>,
    fields: Option<Vec<String>>,
) -> Result<()> {
    let client = ctx.client()?;
    let email = prompt::require_arg(email, "email", "Subscriber email")?;

    let mut body = json!({ "email": email });
    if let Some(status) = status {
        body["status"] = json!(status);
    }
    if let Some(groups) = groups {
        body["groups"] = json!(groups);
    }
    if let Some(fields) = fields {
        body["fields"] = Value::Object(parse_kv_fields(&fields)?);
    }

    let result = client.post("/subscribers", &body)?;

    if ctx.json {
        return output::json(&result);
    }

    let id = result
        .get("data")
        .map(|d| jstr(d, "id"))
        .unwrap_or_default();
    output::success(&format!("Subscriber upserted successfully. ID: {id}"));
    Ok(())
}

fn update(
    ctx: &Ctx,
    id: &str,
    email: Option<String>,
    status: Option<String>,
    fields: Option<Vec<String>>,
) -> Result<()> {
    let client = ctx.client()?;

    let mut body = json!({});
    if let Some(email) = email {
        body["email"] = json!(email);
    }
    if let Some(status) = status {
        body["status"] = json!(status);
    }
    if let Some(fields) = fields {
        body["fields"] = Value::Object(parse_kv_fields(&fields)?);
    }

    let result = client.put(&format!("/subscribers/{id}"), &body)?;

    if ctx.json {
        return output::json(&result);
    }

    output::success(&format!("Subscriber {id} updated successfully."));
    Ok(())
}

fn delete(ctx: &Ctx, id: &str) -> Result<()> {
    let client = ctx.client()?;

    if !ctx.confirm(&format!("Delete subscriber {id}?"))? {
        return Ok(());
    }

    client.delete(&format!("/subscribers/{id}"))?;
    output::success(&format!("Subscriber {id} deleted successfully."));
    Ok(())
}

fn forget(ctx: &Ctx, id: &str) -> Result<()> {
    let client = ctx.client()?;

    if !ctx.confirm(&format!(
        "Permanently forget subscriber {id}? This cannot be undone."
    ))? {
        return Ok(());
    }

    client.post(&format!("/subscribers/{id}/forget"), &json!({}))?;
    output::success(&format!("Subscriber {id} forgotten successfully."));
    Ok(())
}
