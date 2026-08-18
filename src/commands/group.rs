use anyhow::Result;
use clap::{Args, Subcommand};
use serde_json::json;

use crate::api;
use crate::cli::Ctx;
use crate::output;
use crate::prompt;
use crate::util::{jint, jpath, jstr};

#[derive(Args)]
#[command(
    long_about = "List, create, update, and delete groups. Manage group subscriber assignments."
)]
pub struct Cmd {
    #[command(subcommand)]
    command: Sub,
}

#[derive(Subcommand)]
enum Sub {
    /// List groups
    List {
        /// maximum number of groups to return (0 = all)
        #[arg(long, default_value_t = 25)]
        limit: u64,
        /// sort field (e.g. name, created_at)
        #[arg(long, default_value = "")]
        sort: String,
    },
    /// Create a group
    Create {
        /// group name (required)
        #[arg(long, default_value = "")]
        name: String,
    },
    /// Update a group
    #[command(arg_required_else_help = true)]
    Update {
        /// group ID
        id: String,
        /// group name (required)
        #[arg(long, default_value = "")]
        name: String,
    },
    /// Delete a group
    #[command(arg_required_else_help = true)]
    Delete {
        /// group ID
        id: String,
    },
    /// List subscribers in a group
    #[command(arg_required_else_help = true)]
    Subscribers {
        /// group ID
        group_id: String,
        /// maximum number of subscribers to return (0 = all)
        #[arg(long, default_value_t = 25)]
        limit: u64,
    },
    /// Assign a subscriber to a group
    #[command(arg_required_else_help = true)]
    Assign {
        /// group ID
        group_id: String,
        /// subscriber ID
        subscriber_id: String,
    },
    /// Unassign a subscriber from a group
    #[command(arg_required_else_help = true)]
    Unassign {
        /// group ID
        group_id: String,
        /// subscriber ID
        subscriber_id: String,
    },
}

pub fn run(ctx: &Ctx, cmd: Cmd) -> Result<()> {
    match cmd.command {
        Sub::List { limit, sort } => list(ctx, limit, &sort),
        Sub::Create { name } => create(ctx, &name),
        Sub::Update { id, name } => update(ctx, &id, &name),
        Sub::Delete { id } => delete(ctx, &id),
        Sub::Subscribers { group_id, limit } => subscribers(ctx, &group_id, limit),
        Sub::Assign {
            group_id,
            subscriber_id,
        } => assign(ctx, &group_id, &subscriber_id),
        Sub::Unassign {
            group_id,
            subscriber_id,
        } => unassign(ctx, &group_id, &subscriber_id),
    }
}

fn list(ctx: &Ctx, limit: u64, sort: &str) -> Result<()> {
    let client = ctx.client()?;

    let mut query: Vec<(&str, String)> = Vec::new();
    if !sort.is_empty() {
        query.push(("sort", sort.to_string()));
    }

    let groups = api::fetch_all_paged(&client, "/groups", &query, limit)?;

    if ctx.json {
        return output::json(&groups);
    }

    let rows: Vec<Vec<String>> = groups
        .iter()
        .map(|g| {
            vec![
                jstr(g, "id"),
                output::truncate(&jstr(g, "name"), 40),
                jint(g, "active_count").to_string(),
                jint(g, "sent_count").to_string(),
                jint(g, "opens_count").to_string(),
                jpath(g, "click_rate.string"),
                jstr(g, "created_at"),
            ]
        })
        .collect();

    output::table(
        &[
            "ID",
            "NAME",
            "ACTIVE",
            "SENT",
            "OPENS",
            "CLICK RATE",
            "CREATED AT",
        ],
        &rows,
    );
    Ok(())
}

fn create(ctx: &Ctx, name: &str) -> Result<()> {
    let client = ctx.client()?;
    let name = prompt::require_arg(name, "name", "Group name")?;

    let result = client.post("/groups", &json!({ "name": name }))?;

    if ctx.json {
        return output::json(&result);
    }

    let id = result
        .get("data")
        .map(|d| jstr(d, "id"))
        .unwrap_or_default();
    output::success(&format!("Group created successfully. ID: {id}"));
    Ok(())
}

fn update(ctx: &Ctx, id: &str, name: &str) -> Result<()> {
    let client = ctx.client()?;
    let name = prompt::require_arg(name, "name", "Group name")?;

    let result = client.put(&format!("/groups/{id}"), &json!({ "name": name }))?;

    if ctx.json {
        return output::json(&result);
    }

    output::success(&format!("Group {id} updated successfully."));
    Ok(())
}

fn delete(ctx: &Ctx, id: &str) -> Result<()> {
    let client = ctx.client()?;

    if !ctx.confirm(&format!("Delete group {id}?"))? {
        return Ok(());
    }

    client.delete(&format!("/groups/{id}"))?;
    output::success(&format!("Group {id} deleted successfully."));
    Ok(())
}

fn subscribers(ctx: &Ctx, group_id: &str, limit: u64) -> Result<()> {
    let client = ctx.client()?;

    let subscribers = api::fetch_all_paged(
        &client,
        &format!("/groups/{group_id}/subscribers"),
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

fn assign(ctx: &Ctx, group_id: &str, subscriber_id: &str) -> Result<()> {
    let client = ctx.client()?;

    client.post(
        &format!("/subscribers/{subscriber_id}/groups/{group_id}"),
        &json!({}),
    )?;

    output::success(&format!(
        "Subscriber {subscriber_id} assigned to group {group_id} successfully."
    ));
    Ok(())
}

fn unassign(ctx: &Ctx, group_id: &str, subscriber_id: &str) -> Result<()> {
    let client = ctx.client()?;

    client.delete(&format!("/subscribers/{subscriber_id}/groups/{group_id}"))?;

    output::success(&format!(
        "Subscriber {subscriber_id} unassigned from group {group_id} successfully."
    ));
    Ok(())
}
