use anyhow::{anyhow, Result};
use clap::{Args, Subcommand};
use serde::{Deserialize, Serialize};
use serde_json::{json, Value};

use crate::api;
use crate::cli::Ctx;
use crate::output;
use crate::prompt;
use crate::util::{jfloat, jint, jstr};

#[derive(Args)]
#[command(long_about = "List, create, update, and delete orders within a shop.")]
pub struct Cmd {
    /// shop ID (required)
    #[arg(long, global = true, default_value = "")]
    shop: String,
    #[command(subcommand)]
    command: Sub,
}

#[derive(Subcommand)]
enum Sub {
    /// List orders
    List {
        /// maximum number of orders to return (0 = all)
        #[arg(long, default_value_t = 25)]
        limit: u64,
    },
    /// Get order details
    #[command(arg_required_else_help = true)]
    Get {
        /// order ID
        order_id: String,
    },
    /// Create a new order
    Create {
        /// customer ID (required)
        #[arg(long, default_value = "")]
        customer: String,
        /// order status (required)
        #[arg(long, default_value = "")]
        status: String,
        /// order total (required)
        #[arg(long, default_value_t = 0.0)]
        total: f64,
        /// order currency
        #[arg(long, default_value = "USD")]
        currency: String,
        /// order items as JSON array
        #[arg(long, default_value = "")]
        items: String,
    },
    /// Update an order
    #[command(arg_required_else_help = true)]
    Update {
        /// order ID
        order_id: String,
        /// customer ID
        #[arg(long)]
        customer: Option<String>,
        /// order status
        #[arg(long)]
        status: Option<String>,
        /// order total
        #[arg(long)]
        total: Option<f64>,
        /// order currency
        #[arg(long)]
        currency: Option<String>,
        /// order items as JSON array
        #[arg(long)]
        items: Option<String>,
    },
    /// Delete an order
    #[command(arg_required_else_help = true)]
    Delete {
        /// order ID
        order_id: String,
    },
    /// Get total order count
    Count,
}

#[derive(Deserialize, Serialize)]
struct OrderItem {
    #[serde(default)]
    product_id: String,
    #[serde(default)]
    quantity: i64,
    #[serde(default)]
    price: f64,
}

fn parse_items(items: &str) -> Result<Value> {
    let parsed: Vec<OrderItem> =
        serde_json::from_str(items).map_err(|e| anyhow!("invalid --items JSON: {e}"))?;
    Ok(serde_json::to_value(parsed)?)
}

pub fn run(ctx: &Ctx, cmd: Cmd) -> Result<()> {
    let shop_id = prompt::require_arg(&cmd.shop, "shop", "Shop ID")?;
    match cmd.command {
        Sub::List { limit } => list(ctx, &shop_id, limit),
        Sub::Get { order_id } => get(ctx, &shop_id, &order_id),
        Sub::Create {
            customer,
            status,
            total,
            currency,
            items,
        } => create(ctx, &shop_id, &customer, &status, total, &currency, &items),
        Sub::Update {
            order_id,
            customer,
            status,
            total,
            currency,
            items,
        } => update(
            ctx, &shop_id, &order_id, customer, status, total, currency, items,
        ),
        Sub::Delete { order_id } => delete(ctx, &shop_id, &order_id),
        Sub::Count => count(ctx, &shop_id),
    }
}

fn list(ctx: &Ctx, shop_id: &str, limit: u64) -> Result<()> {
    let client = ctx.client()?;
    let orders = api::fetch_all_paged(
        &client,
        &format!("/ecommerce/shops/{shop_id}/orders"),
        &[],
        limit,
    )?;

    if ctx.json {
        return output::json(&orders);
    }

    let rows: Vec<Vec<String>> = orders
        .iter()
        .map(|o| {
            vec![
                jstr(o, "id"),
                jstr(o, "customer_id"),
                jstr(o, "status"),
                format!("{:.2}", jfloat(o, "total")),
                jstr(o, "currency"),
                jstr(o, "created_at"),
            ]
        })
        .collect();

    output::table(
        &["ID", "CUSTOMER", "STATUS", "TOTAL", "CURRENCY", "CREATED"],
        &rows,
    );
    Ok(())
}

fn get(ctx: &Ctx, shop_id: &str, order_id: &str) -> Result<()> {
    let client = ctx.client()?;
    let body = client.get(&format!("/ecommerce/shops/{shop_id}/orders/{order_id}"))?;

    if ctx.json {
        return output::json(body.get("data").unwrap_or(&Value::Null));
    }

    let o = body.get("data").cloned().unwrap_or(Value::Null);
    let items_len = o.get("items").and_then(Value::as_array).map_or(0, Vec::len);
    let rows = vec![
        vec!["ID".to_string(), jstr(&o, "id")],
        vec!["Customer ID".to_string(), jstr(&o, "customer_id")],
        vec!["Status".to_string(), jstr(&o, "status")],
        vec!["Total".to_string(), format!("{:.2}", jfloat(&o, "total"))],
        vec!["Currency".to_string(), jstr(&o, "currency")],
        vec!["Items".to_string(), items_len.to_string()],
        vec!["Created".to_string(), jstr(&o, "created_at")],
        vec!["Updated".to_string(), jstr(&o, "updated_at")],
    ];

    output::table(&["FIELD", "VALUE"], &rows);
    Ok(())
}

fn create(
    ctx: &Ctx,
    shop_id: &str,
    customer: &str,
    status: &str,
    total: f64,
    currency: &str,
    items: &str,
) -> Result<()> {
    let client = ctx.client()?;
    let customer_id = prompt::require_arg(customer, "customer", "Customer ID")?;
    let status = prompt::require_arg(status, "status", "Order status")?;

    let mut body = json!({
        "customer_id": customer_id,
        "status": status,
        "total": total,
        "currency": currency,
    });
    if !items.is_empty() {
        body["items"] = parse_items(items)?;
    }

    let result = client.post(&format!("/ecommerce/shops/{shop_id}/orders"), &body)?;

    if ctx.json {
        return output::json(result.get("data").unwrap_or(&Value::Null));
    }

    let data = result.get("data").cloned().unwrap_or(Value::Null);
    output::success(&format!(
        "Order created: {} (ID: {})",
        jstr(&data, "status"),
        jstr(&data, "id")
    ));
    Ok(())
}

#[allow(clippy::too_many_arguments)]
fn update(
    ctx: &Ctx,
    shop_id: &str,
    order_id: &str,
    customer: Option<String>,
    status: Option<String>,
    total: Option<f64>,
    currency: Option<String>,
    items: Option<String>,
) -> Result<()> {
    let client = ctx.client()?;

    let mut body = json!({});
    if let Some(customer) = customer {
        body["customer_id"] = json!(customer);
    }
    if let Some(status) = status {
        body["status"] = json!(status);
    }
    if let Some(total) = total {
        body["total"] = json!(total);
    }
    if let Some(currency) = currency {
        body["currency"] = json!(currency);
    }
    if let Some(items) = items {
        body["items"] = parse_items(&items)?;
    }
    if body.as_object().is_some_and(|o| o.is_empty()) {
        anyhow::bail!("no flags provided; use --help to see available options");
    }

    let result = client.put(
        &format!("/ecommerce/shops/{shop_id}/orders/{order_id}"),
        &body,
    )?;

    if ctx.json {
        return output::json(result.get("data").unwrap_or(&Value::Null));
    }

    let data = result.get("data").cloned().unwrap_or(Value::Null);
    output::success(&format!(
        "Order updated: {} (ID: {})",
        jstr(&data, "status"),
        jstr(&data, "id")
    ));
    Ok(())
}

fn delete(ctx: &Ctx, shop_id: &str, order_id: &str) -> Result<()> {
    let client = ctx.client()?;
    client.delete(&format!("/ecommerce/shops/{shop_id}/orders/{order_id}"))?;
    output::success(&format!("Order {order_id} deleted."));
    Ok(())
}

fn count(ctx: &Ctx, shop_id: &str) -> Result<()> {
    let client = ctx.client()?;
    let body = client.get_with(
        &format!("/ecommerce/shops/{shop_id}/orders"),
        &[("limit", "0".to_string())],
    )?;

    if ctx.json {
        return output::json(&json!({ "total": jint(&body, "total") }));
    }

    println!("Total orders: {}", jint(&body, "total"));
    Ok(())
}
