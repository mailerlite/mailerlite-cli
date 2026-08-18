use anyhow::Result;
use clap::{Args, Subcommand};
use serde_json::{json, Value};

use crate::api;
use crate::cli::Ctx;
use crate::output;
use crate::prompt;
use crate::util::{jint, jstr};

#[derive(Args)]
pub struct Cmd {
    #[command(subcommand)]
    command: Sub,
}

#[derive(Subcommand)]
enum Sub {
    /// List campaigns
    List {
        /// maximum number of campaigns to return (0 = all)
        #[arg(long, default_value_t = 25)]
        limit: u64,
        /// filter by status (sent, draft, ready)
        #[arg(long, default_value = "")]
        status: String,
        /// filter by type (regular, ab, resend)
        #[arg(long, default_value = "")]
        r#type: String,
    },
    /// Get campaign details
    #[command(arg_required_else_help = true)]
    Get {
        /// campaign ID
        campaign_id: String,
    },
    /// Create a campaign
    Create {
        /// campaign name (required)
        #[arg(long, default_value = "")]
        name: String,
        /// campaign type (regular, ab, resend)
        #[arg(long, default_value = "regular")]
        r#type: String,
        /// email subject (required)
        #[arg(long, default_value = "")]
        subject: String,
        /// sender email address (required)
        #[arg(long, default_value = "")]
        from: String,
        /// sender name (required)
        #[arg(long, default_value = "")]
        from_name: String,
        /// email HTML content
        #[arg(long, default_value = "")]
        content: String,
        /// group IDs
        #[arg(long, value_delimiter = ',')]
        groups: Option<Vec<String>>,
        /// segment IDs
        #[arg(long, value_delimiter = ',')]
        segments: Option<Vec<String>>,
    },
    /// Update a campaign
    #[command(arg_required_else_help = true)]
    Update {
        /// campaign ID
        campaign_id: String,
        /// campaign name
        #[arg(long)]
        name: Option<String>,
        /// campaign type (regular, ab, resend)
        #[arg(long)]
        r#type: Option<String>,
        /// email subject
        #[arg(long)]
        subject: Option<String>,
        /// sender email address
        #[arg(long)]
        from: Option<String>,
        /// sender name
        #[arg(long)]
        from_name: Option<String>,
        /// email HTML content
        #[arg(long)]
        content: Option<String>,
        /// group IDs
        #[arg(long, value_delimiter = ',')]
        groups: Option<Vec<String>>,
        /// segment IDs
        #[arg(long, value_delimiter = ',')]
        segments: Option<Vec<String>>,
    },
    /// Schedule a campaign
    #[command(arg_required_else_help = true)]
    Schedule {
        /// campaign ID
        campaign_id: String,
        /// delivery type (instant, scheduled)
        #[arg(long, default_value = "instant")]
        delivery: String,
        /// schedule date (YYYY-MM-DD)
        #[arg(long, default_value = "")]
        date: String,
        /// schedule hours (00-23)
        #[arg(long, default_value = "")]
        hours: String,
        /// schedule minutes (00-59)
        #[arg(long, default_value = "")]
        minutes: String,
        /// timezone ID
        #[arg(long, default_value_t = 0)]
        timezone_id: i64,
    },
    /// Cancel a campaign
    #[command(arg_required_else_help = true)]
    Cancel {
        /// campaign ID
        campaign_id: String,
    },
    /// List campaign subscriber activity
    #[command(arg_required_else_help = true)]
    Subscribers {
        /// campaign ID
        campaign_id: String,
        /// maximum number of subscribers to return (0 = all)
        #[arg(long, default_value_t = 25)]
        limit: u64,
    },
    /// List campaign languages
    Languages,
    /// Delete a campaign
    #[command(arg_required_else_help = true)]
    Delete {
        /// campaign ID
        campaign_id: String,
    },
}

pub fn run(ctx: &Ctx, cmd: Cmd) -> Result<()> {
    match cmd.command {
        Sub::List {
            limit,
            status,
            r#type,
        } => list(ctx, limit, &status, &r#type),
        Sub::Get { campaign_id } => get(ctx, &campaign_id),
        Sub::Create {
            name,
            r#type,
            subject,
            from,
            from_name,
            content,
            groups,
            segments,
        } => create(
            ctx, &name, &r#type, &subject, &from, &from_name, &content, groups, segments,
        ),
        Sub::Update {
            campaign_id,
            name,
            r#type,
            subject,
            from,
            from_name,
            content,
            groups,
            segments,
        } => update(
            ctx,
            &campaign_id,
            name,
            r#type,
            subject,
            from,
            from_name,
            content,
            groups,
            segments,
        ),
        Sub::Schedule {
            campaign_id,
            delivery,
            date,
            hours,
            minutes,
            timezone_id,
        } => schedule(
            ctx,
            &campaign_id,
            &delivery,
            &date,
            &hours,
            &minutes,
            timezone_id,
        ),
        Sub::Cancel { campaign_id } => cancel(ctx, &campaign_id),
        Sub::Subscribers { campaign_id, limit } => subscribers(ctx, &campaign_id, limit),
        Sub::Languages => languages(ctx),
        Sub::Delete { campaign_id } => delete(ctx, &campaign_id),
    }
}

fn list(ctx: &Ctx, limit: u64, status: &str, campaign_type: &str) -> Result<()> {
    let client = ctx.client()?;

    let mut query: Vec<(&str, String)> = Vec::new();
    if !status.is_empty() {
        query.push(("filter[status]", status.to_string()));
    }
    if !campaign_type.is_empty() {
        query.push(("filter[type]", campaign_type.to_string()));
    }

    let campaigns = api::fetch_all_paged(&client, "/campaigns", &query, limit)?;

    if ctx.json {
        return output::json(&campaigns);
    }

    let rows: Vec<Vec<String>> = campaigns
        .iter()
        .map(|c| {
            let stats = c.get("stats").cloned().unwrap_or(Value::Null);
            vec![
                jstr(c, "id"),
                output::truncate(&jstr(c, "name"), 40),
                jstr(c, "type"),
                jstr(c, "status"),
                jint(&stats, "sent").to_string(),
                jint(&stats, "opens_count").to_string(),
                jint(&stats, "clicks_count").to_string(),
            ]
        })
        .collect();

    output::table(
        &["ID", "NAME", "TYPE", "STATUS", "SENT", "OPENS", "CLICKS"],
        &rows,
    );
    Ok(())
}

fn get(ctx: &Ctx, id: &str) -> Result<()> {
    let client = ctx.client()?;
    let body = client.get(&format!("/campaigns/{id}"))?;

    if ctx.json {
        return output::json(&body);
    }

    let d = body.get("data").cloned().unwrap_or(Value::Null);

    println!("ID:           {}", jstr(&d, "id"));
    println!("Name:         {}", jstr(&d, "name"));
    println!("Type:         {}", jstr(&d, "type"));
    println!("Status:       {}", jstr(&d, "status"));
    println!("Created At:   {}", jstr(&d, "created_at"));
    println!("Updated At:   {}", jstr(&d, "updated_at"));
    let scheduled_for = jstr(&d, "scheduled_for");
    if !scheduled_for.is_empty() {
        println!("Scheduled:    {scheduled_for}");
    }

    let stats = d.get("stats").cloned().unwrap_or(Value::Null);
    println!();
    println!("Stats:");
    println!("  Sent:       {}", jint(&stats, "sent"));
    println!("  Opens:      {}", jint(&stats, "opens_count"));
    println!("  Clicks:     {}", jint(&stats, "clicks_count"));

    if let Some(emails) = d.get("emails").and_then(Value::as_array) {
        if !emails.is_empty() {
            println!();
            println!("Emails:");
            for e in emails {
                println!(
                    "  - {} (from: {} <{}>)",
                    jstr(e, "subject"),
                    jstr(e, "from_name"),
                    jstr(e, "from")
                );
            }
        }
    }

    Ok(())
}

#[allow(clippy::too_many_arguments)]
fn create(
    ctx: &Ctx,
    name: &str,
    campaign_type: &str,
    subject: &str,
    from: &str,
    from_name: &str,
    content: &str,
    groups: Option<Vec<String>>,
    segments: Option<Vec<String>>,
) -> Result<()> {
    let client = ctx.client()?;

    let name = prompt::require_arg(name, "name", "Campaign name")?;
    let subject = prompt::require_arg(subject, "subject", "Email subject")?;
    let from = prompt::require_arg(from, "from", "Sender email address")?;
    let from_name = prompt::require_arg(from_name, "from-name", "Sender name")?;

    let mut body = json!({
        "name": name,
        "type": campaign_type,
        "emails": [{
            "subject": subject,
            "from_name": from_name,
            "from": from,
            "content": content,
        }],
    });
    if let Some(groups) = groups {
        if !groups.is_empty() {
            body["groups"] = json!(groups);
        }
    }
    if let Some(segments) = segments {
        if !segments.is_empty() {
            body["segments"] = json!(segments);
        }
    }

    let result = client.post("/campaigns", &body)?;

    if ctx.json {
        return output::json(&result);
    }

    let id = result
        .get("data")
        .map(|d| jstr(d, "id"))
        .unwrap_or_default();
    output::success(&format!("Campaign created successfully. ID: {id}"));
    Ok(())
}

#[allow(clippy::too_many_arguments)]
fn update(
    ctx: &Ctx,
    id: &str,
    name: Option<String>,
    campaign_type: Option<String>,
    subject: Option<String>,
    from: Option<String>,
    from_name: Option<String>,
    content: Option<String>,
    groups: Option<Vec<String>>,
    segments: Option<Vec<String>>,
) -> Result<()> {
    let client = ctx.client()?;

    // First get the existing campaign to preserve unchanged fields.
    let existing = client.get(&format!("/campaigns/{id}"))?;
    let d = existing.get("data").cloned().unwrap_or(Value::Null);

    let name = name.unwrap_or_else(|| jstr(&d, "name"));
    let campaign_type = campaign_type.unwrap_or_else(|| jstr(&d, "type"));

    let existing_email = d
        .get("emails")
        .and_then(Value::as_array)
        .and_then(|e| e.first())
        .cloned()
        .unwrap_or(Value::Null);

    let subject = subject.unwrap_or_else(|| jstr(&existing_email, "subject"));
    let from = from.unwrap_or_else(|| jstr(&existing_email, "from"));
    let from_name = from_name.unwrap_or_else(|| jstr(&existing_email, "from_name"));
    let content = content.unwrap_or_default();

    let mut body = json!({
        "name": name,
        "type": campaign_type,
        "emails": [{
            "subject": subject,
            "from_name": from_name,
            "from": from,
            "content": content,
        }],
    });
    if let Some(groups) = groups {
        if !groups.is_empty() {
            body["groups"] = json!(groups);
        }
    }
    if let Some(segments) = segments {
        if !segments.is_empty() {
            body["segments"] = json!(segments);
        }
    }

    let result = client.put(&format!("/campaigns/{id}"), &body)?;

    if ctx.json {
        return output::json(&result);
    }

    output::success(&format!("Campaign {id} updated successfully."));
    Ok(())
}

fn schedule(
    ctx: &Ctx,
    id: &str,
    delivery: &str,
    date: &str,
    hours: &str,
    minutes: &str,
    timezone_id: i64,
) -> Result<()> {
    let client = ctx.client()?;

    let mut body = json!({ "delivery": delivery });
    if delivery.eq_ignore_ascii_case("scheduled") {
        let mut schedule = json!({
            "date": date,
            "hours": hours,
            "minutes": minutes,
        });
        if timezone_id != 0 {
            schedule["timezone_id"] = json!(timezone_id);
        }
        body["schedule"] = schedule;
    }

    let result = client.post(&format!("/campaigns/{id}/schedule"), &body)?;

    if ctx.json {
        return output::json(&result);
    }

    output::success(&format!("Campaign {id} scheduled successfully."));
    Ok(())
}

fn cancel(ctx: &Ctx, id: &str) -> Result<()> {
    let client = ctx.client()?;
    let result = client.post(&format!("/campaigns/{id}/cancel"), &json!({}))?;

    if ctx.json {
        return output::json(&result);
    }

    output::success(&format!("Campaign {id} cancelled successfully."));
    Ok(())
}

fn subscribers(ctx: &Ctx, id: &str, limit: u64) -> Result<()> {
    let client = ctx.client()?;
    let subscribers = fetch_all_activity(&client, id, limit)?;

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
                jint(s, "opens_count").to_string(),
                jint(s, "clicks_count").to_string(),
            ]
        })
        .collect();

    output::table(&["ID", "EMAIL", "OPENS", "CLICKS"], &rows);
    Ok(())
}

// Subscriber activity is a POST endpoint paginated via the request body.
fn fetch_all_activity(client: &api::Client, id: &str, limit: u64) -> Result<Vec<Value>> {
    let per = if limit > 0 && limit < 25 { limit } else { 25 };
    let path = format!("/campaigns/{id}/reports/subscriber-activity");
    let mut all = Vec::new();
    let mut page: u64 = 1;
    loop {
        let body = client.post(&path, &json!({ "page": page, "limit": per }))?;
        let items = body
            .get("data")
            .and_then(Value::as_array)
            .cloned()
            .unwrap_or_default();
        let got = items.len() as u64;
        for item in items {
            all.push(item);
            if limit > 0 && all.len() as u64 >= limit {
                return Ok(all);
            }
        }
        let last_page = body
            .get("meta")
            .and_then(|m| m.get("last_page"))
            .and_then(Value::as_u64);
        let done = match last_page {
            Some(last) => page >= last,
            None => got < per,
        };
        if done || got == 0 {
            return Ok(all);
        }
        page += 1;
    }
}

fn languages(ctx: &Ctx) -> Result<()> {
    let client = ctx.client()?;
    let body = client.get("/campaigns/languages")?;

    let data = body.get("data").cloned().unwrap_or(Value::Null);

    if ctx.json {
        return output::json(&data);
    }

    let rows: Vec<Vec<String>> = data
        .as_array()
        .map(|langs| {
            langs
                .iter()
                .map(|l| vec![jstr(l, "id"), jstr(l, "name"), jstr(l, "shortcode")])
                .collect()
        })
        .unwrap_or_default();

    output::table(&["ID", "NAME", "SHORTCODE"], &rows);
    Ok(())
}

fn delete(ctx: &Ctx, id: &str) -> Result<()> {
    let client = ctx.client()?;

    if !ctx.confirm(&format!("Are you sure you want to delete campaign {id}?"))? {
        return Ok(());
    }

    client.delete(&format!("/campaigns/{id}"))?;
    output::success(&format!("Campaign {id} deleted successfully."));
    Ok(())
}
