use anyhow::Result;
use clap::{Args, Subcommand};
use serde_json::json;

use crate::api;
use crate::cli::Ctx;
use crate::output;
use crate::prompt;
use crate::util::{jint, jpath, jstr};

#[derive(Args)]
#[command(long_about = "List, update, delete segments and view segment subscribers.")]
pub struct Cmd {
    #[command(subcommand)]
    command: Sub,
}

#[derive(Subcommand)]
enum Sub {
    /// List segments
    List {
        /// maximum number of segments to return (0 = all)
        #[arg(long, default_value_t = 25)]
        limit: u64,
    },
    /// Update a segment
    #[command(arg_required_else_help = true)]
    Update {
        /// segment ID
        segment_id: String,
        /// new segment name (required)
        #[arg(long, default_value = "")]
        name: String,
    },
    /// Delete a segment
    #[command(arg_required_else_help = true)]
    Delete {
        /// segment ID
        segment_id: String,
    },
    /// List subscribers in a segment
    #[command(arg_required_else_help = true)]
    Subscribers {
        /// segment ID
        segment_id: String,
        /// maximum number of subscribers to return (0 = all)
        #[arg(long, default_value_t = 25)]
        limit: u64,
    },
}

pub fn run(ctx: &Ctx, cmd: Cmd) -> Result<()> {
    match cmd.command {
        Sub::List { limit } => list(ctx, limit),
        Sub::Update { segment_id, name } => update(ctx, &segment_id, &name),
        Sub::Delete { segment_id } => delete(ctx, &segment_id),
        Sub::Subscribers { segment_id, limit } => subscribers(ctx, &segment_id, limit),
    }
}

fn list(ctx: &Ctx, limit: u64) -> Result<()> {
    let client = ctx.client()?;

    let segments = api::fetch_all_paged(&client, "/segments", &[], limit)?;

    if ctx.json {
        return output::json(&segments);
    }

    let rows: Vec<Vec<String>> = segments
        .iter()
        .map(|s| {
            vec![
                jstr(s, "id"),
                jstr(s, "name"),
                jint(s, "total").to_string(),
                jpath(s, "open_rate.string"),
                jpath(s, "click_rate.string"),
                jstr(s, "created_at"),
            ]
        })
        .collect();

    output::table(
        &[
            "ID",
            "NAME",
            "TOTAL",
            "OPEN RATE",
            "CLICK RATE",
            "CREATED AT",
        ],
        &rows,
    );
    Ok(())
}

fn update(ctx: &Ctx, segment_id: &str, name: &str) -> Result<()> {
    let client = ctx.client()?;
    let name = prompt::require_arg(name, "name", "New segment name")?;

    let result = client.put(&format!("/segments/{segment_id}"), &json!({ "name": name }))?;

    if ctx.json {
        return output::json(&result);
    }

    output::success(&format!("Segment {segment_id} updated successfully."));
    Ok(())
}

fn delete(ctx: &Ctx, segment_id: &str) -> Result<()> {
    let client = ctx.client()?;

    if !ctx.confirm(&format!("Delete segment {segment_id}?"))? {
        return Ok(());
    }

    client.delete(&format!("/segments/{segment_id}"))?;
    output::success(&format!("Segment {segment_id} deleted successfully."));
    Ok(())
}

fn subscribers(ctx: &Ctx, segment_id: &str, limit: u64) -> Result<()> {
    let client = ctx.client()?;

    let subscribers = api::fetch_all_after(
        &client,
        &format!("/segments/{segment_id}/subscribers"),
        &[],
        limit,
    )?;

    if ctx.json {
        return output::json(&subscribers);
    }

    let rows: Vec<Vec<String>> = subscribers
        .iter()
        .map(|s| {
            vec![
                jstr(s, "id"),
                jstr(s, "email"),
                jstr(s, "status"),
                jstr(s, "subscribed_at"),
                jstr(s, "created_at"),
            ]
        })
        .collect();

    output::table(
        &["ID", "EMAIL", "STATUS", "SUBSCRIBED AT", "CREATED AT"],
        &rows,
    );
    Ok(())
}
