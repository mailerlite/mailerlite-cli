use anyhow::Result;
use clap::{Args, Subcommand};
use serde_json::{json, Value};

use crate::api;
use crate::cli::Ctx;
use crate::output;
use crate::prompt;
use crate::util::{jfloat, jint, jstr};

#[derive(Args)]
#[command(long_about = "List, view, and update carts within a shop.")]
pub struct Cmd {
    /// shop ID (required)
    #[arg(long, global = true, default_value = "")]
    shop: String,
    #[command(subcommand)]
    command: Sub,
}

#[derive(Subcommand)]
enum Sub {
    /// List carts
    List {
        /// maximum number of carts to return (0 = all)
        #[arg(long, default_value_t = 25)]
        limit: u64,
    },
    /// Get cart details
    #[command(arg_required_else_help = true)]
    Get {
        /// cart ID
        cart_id: String,
    },
    /// Update a cart
    #[command(arg_required_else_help = true)]
    Update {
        /// cart ID
        cart_id: String,
        /// customer ID
        #[arg(long)]
        customer: Option<String>,
        /// cart currency
        #[arg(long)]
        currency: Option<String>,
    },
    /// Get total cart count
    Count,
}

pub fn run(ctx: &Ctx, cmd: Cmd) -> Result<()> {
    let shop_id = prompt::require_arg(&cmd.shop, "shop", "Shop ID")?;
    match cmd.command {
        Sub::List { limit } => list(ctx, &shop_id, limit),
        Sub::Get { cart_id } => get(ctx, &shop_id, &cart_id),
        Sub::Update {
            cart_id,
            customer,
            currency,
        } => update(ctx, &shop_id, &cart_id, customer, currency),
        Sub::Count => count(ctx, &shop_id),
    }
}

fn list(ctx: &Ctx, shop_id: &str, limit: u64) -> Result<()> {
    let client = ctx.client()?;
    let carts = api::fetch_all_paged(
        &client,
        &format!("/ecommerce/shops/{shop_id}/carts"),
        &[],
        limit,
    )?;

    if ctx.json {
        return output::json(&carts);
    }

    let rows: Vec<Vec<String>> = carts
        .iter()
        .map(|c| {
            vec![
                jstr(c, "id"),
                jstr(c, "customer_id"),
                jstr(c, "currency"),
                format!("{:.2}", jfloat(c, "total")),
                jstr(c, "created_at"),
            ]
        })
        .collect();

    output::table(&["ID", "CUSTOMER", "CURRENCY", "TOTAL", "CREATED"], &rows);
    Ok(())
}

fn get(ctx: &Ctx, shop_id: &str, cart_id: &str) -> Result<()> {
    let client = ctx.client()?;
    let body = client.get(&format!("/ecommerce/shops/{shop_id}/carts/{cart_id}"))?;

    if ctx.json {
        return output::json(body.get("data").unwrap_or(&Value::Null));
    }

    let c = body.get("data").cloned().unwrap_or(Value::Null);
    let rows = vec![
        vec!["ID".to_string(), jstr(&c, "id")],
        vec!["Customer ID".to_string(), jstr(&c, "customer_id")],
        vec!["Currency".to_string(), jstr(&c, "currency")],
        vec!["Total".to_string(), format!("{:.2}", jfloat(&c, "total"))],
        vec!["Created".to_string(), jstr(&c, "created_at")],
        vec!["Updated".to_string(), jstr(&c, "updated_at")],
    ];

    output::table(&["FIELD", "VALUE"], &rows);
    Ok(())
}

fn update(
    ctx: &Ctx,
    shop_id: &str,
    cart_id: &str,
    customer: Option<String>,
    currency: Option<String>,
) -> Result<()> {
    let client = ctx.client()?;

    let mut body = json!({});
    if let Some(customer) = customer {
        body["customer_id"] = json!(customer);
    }
    if let Some(currency) = currency {
        body["currency"] = json!(currency);
    }
    if body.as_object().is_some_and(|o| o.is_empty()) {
        anyhow::bail!("no flags provided; use --help to see available options");
    }

    let result = client.put(
        &format!("/ecommerce/shops/{shop_id}/carts/{cart_id}"),
        &body,
    )?;

    if ctx.json {
        return output::json(result.get("data").unwrap_or(&Value::Null));
    }

    let data = result.get("data").cloned().unwrap_or(Value::Null);
    output::success(&format!("Cart updated: {}", jstr(&data, "id")));
    Ok(())
}

fn count(ctx: &Ctx, shop_id: &str) -> Result<()> {
    let client = ctx.client()?;
    let body = client.get_with(
        &format!("/ecommerce/shops/{shop_id}/carts"),
        &[("limit", "0".to_string())],
    )?;

    if ctx.json {
        return output::json(&json!({ "total": jint(&body, "total") }));
    }

    println!("Total carts: {}", jint(&body, "total"));
    Ok(())
}
