use anyhow::Result;
use clap::{Args, Subcommand};
use serde_json::{json, Value};

use crate::api;
use crate::cli::Ctx;
use crate::output;
use crate::prompt;
use crate::util::{jint, jstr};

#[derive(Args)]
#[command(long_about = "List, view, update, and delete forms.")]
pub struct Cmd {
    #[command(subcommand)]
    command: Sub,
}

#[derive(Subcommand)]
enum Sub {
    /// List forms
    List {
        /// maximum number of forms to return (0 = all)
        #[arg(long, default_value_t = 25)]
        limit: u64,
        /// form type (popup, embedded, promotion)
        #[arg(long = "type", default_value = "popup")]
        form_type: String,
        /// sort field
        #[arg(long, default_value = "")]
        sort: String,
    },
    /// Get form details
    #[command(arg_required_else_help = true)]
    Get {
        /// form ID
        form_id: String,
    },
    /// Update a form
    #[command(arg_required_else_help = true)]
    Update {
        /// form ID
        form_id: String,
        /// form name (required)
        #[arg(long, default_value = "")]
        name: String,
    },
    /// Delete a form
    #[command(arg_required_else_help = true)]
    Delete {
        /// form ID
        form_id: String,
    },
    /// List form subscribers
    #[command(arg_required_else_help = true)]
    Subscribers {
        /// form ID
        form_id: String,
        /// maximum number of subscribers to return (0 = all)
        #[arg(long, default_value_t = 25)]
        limit: u64,
    },
}

pub fn run(ctx: &Ctx, cmd: Cmd) -> Result<()> {
    match cmd.command {
        Sub::List {
            limit,
            form_type,
            sort,
        } => list(ctx, limit, &form_type, &sort),
        Sub::Get { form_id } => get(ctx, &form_id),
        Sub::Update { form_id, name } => update(ctx, &form_id, &name),
        Sub::Delete { form_id } => delete(ctx, &form_id),
        Sub::Subscribers { form_id, limit } => subscribers(ctx, &form_id, limit),
    }
}

fn yes_no(v: &Value, key: &str) -> String {
    if v.get(key).and_then(Value::as_bool).unwrap_or(false) {
        "Yes".to_string()
    } else {
        "No".to_string()
    }
}

fn list(ctx: &Ctx, limit: u64, form_type: &str, sort: &str) -> Result<()> {
    let client = ctx.client()?;

    let mut query: Vec<(&str, String)> = Vec::new();
    if !sort.is_empty() {
        query.push(("sort", sort.to_string()));
    }

    let forms = api::fetch_all_paged(&client, &format!("/forms/{form_type}"), &query, limit)?;

    if ctx.json {
        return output::json(&forms);
    }

    let rows: Vec<Vec<String>> = forms
        .iter()
        .map(|f| {
            vec![
                jstr(f, "id"),
                output::truncate(&jstr(f, "name"), 40),
                jstr(f, "type"),
                yes_no(f, "active"),
                jint(f, "conversions_count").to_string(),
                jint(f, "opens_count").to_string(),
            ]
        })
        .collect();

    output::table(
        &["ID", "NAME", "TYPE", "ACTIVE", "CONVERSIONS", "OPENS"],
        &rows,
    );
    Ok(())
}

fn get(ctx: &Ctx, form_id: &str) -> Result<()> {
    let client = ctx.client()?;
    let body = client.get(&format!("/forms/{form_id}"))?;

    if ctx.json {
        return output::json(&body);
    }

    let d = body.get("data").cloned().unwrap_or(Value::Null);

    println!("ID:           {}", jstr(&d, "id"));
    println!("Name:         {}", jstr(&d, "name"));
    println!("Type:         {}", jstr(&d, "type"));
    println!("Active:       {}", yes_no(&d, "active"));
    println!("Conversions:  {}", jint(&d, "conversions_count"));
    println!("Opens:        {}", jint(&d, "opens_count"));
    println!("Created At:   {}", jstr(&d, "created_at"));

    Ok(())
}

fn update(ctx: &Ctx, form_id: &str, name: &str) -> Result<()> {
    let client = ctx.client()?;
    let name = prompt::require_arg(name, "name", "Form name")?;

    let result = client.put(&format!("/forms/{form_id}"), &json!({ "name": name }))?;

    if ctx.json {
        return output::json(&result);
    }

    output::success(&format!("Form {form_id} updated successfully."));
    Ok(())
}

fn delete(ctx: &Ctx, form_id: &str) -> Result<()> {
    let client = ctx.client()?;

    if !ctx.confirm(&format!("Are you sure you want to delete form {form_id}?"))? {
        return Ok(());
    }

    client.delete(&format!("/forms/{form_id}"))?;
    output::success(&format!("Form {form_id} deleted successfully."));
    Ok(())
}

fn subscribers(ctx: &Ctx, form_id: &str, limit: u64) -> Result<()> {
    let client = ctx.client()?;

    let subscribers = api::fetch_all_paged(
        &client,
        &format!("/forms/{form_id}/subscribers"),
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
                jstr(s, "created_at"),
            ]
        })
        .collect();

    output::table(&["ID", "EMAIL", "STATUS", "CREATED AT"], &rows);
    Ok(())
}
