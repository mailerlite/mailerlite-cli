use anyhow::Result;
use clap::{Args, Subcommand};
use serde_json::json;

use crate::api;
use crate::cli::Ctx;
use crate::output;
use crate::prompt;
use crate::util::jstr;

#[derive(Args)]
#[command(long_about = "List, create, update, and delete subscriber fields.")]
pub struct Cmd {
    #[command(subcommand)]
    command: Sub,
}

#[derive(Subcommand)]
enum Sub {
    /// List fields
    List {
        /// maximum number of fields to return (0 = all)
        #[arg(long, default_value_t = 25)]
        limit: u64,
        /// sort order (e.g. name)
        #[arg(long, default_value = "")]
        sort: String,
    },
    /// Create a field
    Create {
        /// field name (required)
        #[arg(long, default_value = "")]
        name: String,
        /// field type: text, number, or date (required)
        #[arg(long = "type", default_value = "")]
        field_type: String,
    },
    /// Update a field
    #[command(arg_required_else_help = true)]
    Update {
        /// field ID
        field_id: String,
        /// new field name (required)
        #[arg(long, default_value = "")]
        name: String,
    },
    /// Delete a field
    #[command(arg_required_else_help = true)]
    Delete {
        /// field ID
        field_id: String,
    },
}

pub fn run(ctx: &Ctx, cmd: Cmd) -> Result<()> {
    match cmd.command {
        Sub::List { limit, sort } => list(ctx, limit, &sort),
        Sub::Create { name, field_type } => create(ctx, &name, &field_type),
        Sub::Update { field_id, name } => update(ctx, &field_id, &name),
        Sub::Delete { field_id } => delete(ctx, &field_id),
    }
}

fn list(ctx: &Ctx, limit: u64, sort: &str) -> Result<()> {
    let client = ctx.client()?;

    let mut query: Vec<(&str, String)> = Vec::new();
    if !sort.is_empty() {
        query.push(("sort", sort.to_string()));
    }

    let fields = api::fetch_all_paged(&client, "/fields", &query, limit)?;

    if ctx.json {
        return output::json(&fields);
    }

    let rows: Vec<Vec<String>> = fields
        .iter()
        .map(|f| {
            vec![
                jstr(f, "id"),
                jstr(f, "name"),
                jstr(f, "key"),
                jstr(f, "type"),
            ]
        })
        .collect();

    output::table(&["ID", "NAME", "KEY", "TYPE"], &rows);
    Ok(())
}

fn create(ctx: &Ctx, name: &str, field_type: &str) -> Result<()> {
    let client = ctx.client()?;
    let name = prompt::require_arg(name, "name", "Field name")?;
    let field_type = prompt::require_arg(field_type, "type", "Field type (text, number, date)")?;

    let result = client.post("/fields", &json!({ "name": name, "type": field_type }))?;

    if ctx.json {
        return output::json(&result);
    }

    let id = result
        .get("data")
        .map(|d| jstr(d, "id"))
        .unwrap_or_default();
    output::success(&format!("Field created successfully. ID: {id}"));
    Ok(())
}

fn update(ctx: &Ctx, field_id: &str, name: &str) -> Result<()> {
    let client = ctx.client()?;
    let name = prompt::require_arg(name, "name", "New field name")?;

    let result = client.put(&format!("/fields/{field_id}"), &json!({ "name": name }))?;

    if ctx.json {
        return output::json(&result);
    }

    output::success(&format!("Field {field_id} updated successfully."));
    Ok(())
}

fn delete(ctx: &Ctx, field_id: &str) -> Result<()> {
    let client = ctx.client()?;

    if !ctx.confirm(&format!("Delete field {field_id}?"))? {
        return Ok(());
    }

    client.delete(&format!("/fields/{field_id}"))?;
    output::success(&format!("Field {field_id} deleted successfully."));
    Ok(())
}
