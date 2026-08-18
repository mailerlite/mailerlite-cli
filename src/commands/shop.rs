use anyhow::Result;
use clap::{Args, Subcommand};
use serde_json::{json, Value};

use crate::api;
use crate::cli::Ctx;
use crate::output;
use crate::prompt;
use crate::util::{jint, jstr};

#[derive(Args)]
#[command(long_about = "List, create, update, and delete e-commerce shops.")]
pub struct Cmd {
    #[command(subcommand)]
    command: Sub,
}

#[derive(Subcommand)]
enum Sub {
    /// List shops
    List {
        /// maximum number of shops to return (0 = all)
        #[arg(long, default_value_t = 25)]
        limit: u64,
    },
    /// Get shop details
    #[command(arg_required_else_help = true)]
    Get {
        /// shop ID
        shop_id: String,
    },
    /// Create a new shop
    Create {
        /// shop name (required)
        #[arg(long, default_value = "")]
        name: String,
        /// shop URL (required)
        #[arg(long, default_value = "")]
        url: String,
    },
    /// Update a shop
    #[command(arg_required_else_help = true)]
    Update {
        /// shop ID
        shop_id: String,
        /// shop name
        #[arg(long)]
        name: Option<String>,
        /// shop URL
        #[arg(long)]
        url: Option<String>,
    },
    /// Delete a shop
    #[command(arg_required_else_help = true)]
    Delete {
        /// shop ID
        shop_id: String,
    },
    /// Get total shop count
    Count,
}

pub fn run(ctx: &Ctx, cmd: Cmd) -> Result<()> {
    match cmd.command {
        Sub::List { limit } => list(ctx, limit),
        Sub::Get { shop_id } => get(ctx, &shop_id),
        Sub::Create { name, url } => create(ctx, &name, &url),
        Sub::Update { shop_id, name, url } => update(ctx, &shop_id, name, url),
        Sub::Delete { shop_id } => delete(ctx, &shop_id),
        Sub::Count => count(ctx),
    }
}

fn list(ctx: &Ctx, limit: u64) -> Result<()> {
    let client = ctx.client()?;
    let shops = api::fetch_all_paged(&client, "/ecommerce/shops", &[], limit)?;

    if ctx.json {
        return output::json(&shops);
    }

    let rows: Vec<Vec<String>> = shops
        .iter()
        .map(|s| {
            vec![
                jstr(s, "id"),
                jstr(s, "name"),
                jstr(s, "url"),
                jstr(s, "created_at"),
            ]
        })
        .collect();

    output::table(&["ID", "NAME", "URL", "CREATED"], &rows);
    Ok(())
}

fn get(ctx: &Ctx, shop_id: &str) -> Result<()> {
    let client = ctx.client()?;
    let body = client.get(&format!("/ecommerce/shops/{shop_id}"))?;

    if ctx.json {
        return output::json(body.get("data").unwrap_or(&Value::Null));
    }

    let s = body.get("data").cloned().unwrap_or(Value::Null);
    let rows = vec![
        vec!["ID".to_string(), jstr(&s, "id")],
        vec!["Name".to_string(), jstr(&s, "name")],
        vec!["URL".to_string(), jstr(&s, "url")],
        vec!["Created".to_string(), jstr(&s, "created_at")],
        vec!["Updated".to_string(), jstr(&s, "updated_at")],
    ];

    output::table(&["FIELD", "VALUE"], &rows);
    Ok(())
}

fn create(ctx: &Ctx, name: &str, url: &str) -> Result<()> {
    let client = ctx.client()?;
    let name = prompt::require_arg(name, "name", "Shop name")?;
    let url = prompt::require_arg(url, "url", "Shop URL")?;

    let body = json!({ "name": name, "url": url });
    let result = client.post("/ecommerce/shops", &body)?;

    if ctx.json {
        return output::json(result.get("data").unwrap_or(&Value::Null));
    }

    let data = result.get("data").cloned().unwrap_or(Value::Null);
    output::success(&format!(
        "Shop created: {} (ID: {})",
        jstr(&data, "name"),
        jstr(&data, "id")
    ));
    Ok(())
}

fn update(ctx: &Ctx, shop_id: &str, name: Option<String>, url: Option<String>) -> Result<()> {
    let client = ctx.client()?;

    let mut body = json!({});
    if let Some(name) = name {
        body["name"] = json!(name);
    }
    if let Some(url) = url {
        body["url"] = json!(url);
    }
    if body.as_object().is_some_and(|o| o.is_empty()) {
        anyhow::bail!("no flags provided; use --help to see available options");
    }

    let result = client.put(&format!("/ecommerce/shops/{shop_id}"), &body)?;

    if ctx.json {
        return output::json(result.get("data").unwrap_or(&Value::Null));
    }

    let data = result.get("data").cloned().unwrap_or(Value::Null);
    output::success(&format!(
        "Shop updated: {} (ID: {})",
        jstr(&data, "name"),
        jstr(&data, "id")
    ));
    Ok(())
}

fn delete(ctx: &Ctx, shop_id: &str) -> Result<()> {
    let client = ctx.client()?;
    client.delete(&format!("/ecommerce/shops/{shop_id}"))?;
    output::success(&format!("Shop {shop_id} deleted."));
    Ok(())
}

fn count(ctx: &Ctx) -> Result<()> {
    let client = ctx.client()?;
    let body = client.get_with("/ecommerce/shops", &[("limit", "0".to_string())])?;

    if ctx.json {
        return output::json(&json!({ "total": jint(&body, "total") }));
    }

    println!("Total shops: {}", jint(&body, "total"));
    Ok(())
}
