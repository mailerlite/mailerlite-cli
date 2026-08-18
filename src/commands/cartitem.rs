use anyhow::Result;
use clap::{Args, Subcommand};
use serde_json::{json, Value};

use crate::api;
use crate::cli::Ctx;
use crate::output;
use crate::prompt;
use crate::util::{jfloat, jint, jstr};

#[derive(Args)]
#[command(long_about = "List, create, update, and delete items within a cart.")]
pub struct Cmd {
    /// shop ID (required)
    #[arg(long, global = true, default_value = "")]
    shop: String,
    /// cart ID (required)
    #[arg(long, global = true, default_value = "")]
    cart: String,
    #[command(subcommand)]
    command: Sub,
}

#[derive(Subcommand)]
enum Sub {
    /// List cart items
    List {
        /// maximum number of items to return (0 = all)
        #[arg(long, default_value_t = 25)]
        limit: u64,
    },
    /// Get cart item details
    #[command(arg_required_else_help = true)]
    Get {
        /// item ID
        item_id: String,
    },
    /// Add an item to a cart
    Create {
        /// product ID (required)
        #[arg(long, default_value = "")]
        product: String,
        /// item quantity
        #[arg(long, default_value_t = 1)]
        quantity: i64,
        /// item price
        #[arg(long, default_value_t = 0.0)]
        price: f64,
    },
    /// Update a cart item
    #[command(arg_required_else_help = true)]
    Update {
        /// item ID
        item_id: String,
        /// item quantity
        #[arg(long)]
        quantity: Option<i64>,
        /// item price
        #[arg(long)]
        price: Option<f64>,
    },
    /// Delete a cart item
    #[command(arg_required_else_help = true)]
    Delete {
        /// item ID
        item_id: String,
    },
    /// Get total cart item count
    Count,
}

fn base_path(shop_id: &str, cart_id: &str) -> String {
    format!("/ecommerce/shops/{shop_id}/carts/{cart_id}/items")
}

pub fn run(ctx: &Ctx, cmd: Cmd) -> Result<()> {
    let shop_id = prompt::require_arg(&cmd.shop, "shop", "Shop ID")?;
    let cart_id = prompt::require_arg(&cmd.cart, "cart", "Cart ID")?;
    match cmd.command {
        Sub::List { limit } => list(ctx, &shop_id, &cart_id, limit),
        Sub::Get { item_id } => get(ctx, &shop_id, &cart_id, &item_id),
        Sub::Create {
            product,
            quantity,
            price,
        } => create(ctx, &shop_id, &cart_id, &product, quantity, price),
        Sub::Update {
            item_id,
            quantity,
            price,
        } => update(ctx, &shop_id, &cart_id, &item_id, quantity, price),
        Sub::Delete { item_id } => delete(ctx, &shop_id, &cart_id, &item_id),
        Sub::Count => count(ctx, &shop_id, &cart_id),
    }
}

fn list(ctx: &Ctx, shop_id: &str, cart_id: &str, limit: u64) -> Result<()> {
    let client = ctx.client()?;
    let items = api::fetch_all_paged(&client, &base_path(shop_id, cart_id), &[], limit)?;

    if ctx.json {
        return output::json(&items);
    }

    let rows: Vec<Vec<String>> = items
        .iter()
        .map(|i| {
            vec![
                jstr(i, "id"),
                jstr(i, "product_id"),
                jint(i, "quantity").to_string(),
                format!("{:.2}", jfloat(i, "price")),
                jstr(i, "created_at"),
            ]
        })
        .collect();

    output::table(&["ID", "PRODUCT", "QUANTITY", "PRICE", "CREATED"], &rows);
    Ok(())
}

fn get(ctx: &Ctx, shop_id: &str, cart_id: &str, item_id: &str) -> Result<()> {
    let client = ctx.client()?;
    let body = client.get(&format!("{}/{item_id}", base_path(shop_id, cart_id)))?;

    if ctx.json {
        return output::json(body.get("data").unwrap_or(&Value::Null));
    }

    let i = body.get("data").cloned().unwrap_or(Value::Null);
    let rows = vec![
        vec!["ID".to_string(), jstr(&i, "id")],
        vec!["Product ID".to_string(), jstr(&i, "product_id")],
        vec!["Quantity".to_string(), jint(&i, "quantity").to_string()],
        vec!["Price".to_string(), format!("{:.2}", jfloat(&i, "price"))],
        vec!["Created".to_string(), jstr(&i, "created_at")],
        vec!["Updated".to_string(), jstr(&i, "updated_at")],
    ];

    output::table(&["FIELD", "VALUE"], &rows);
    Ok(())
}

fn create(
    ctx: &Ctx,
    shop_id: &str,
    cart_id: &str,
    product: &str,
    quantity: i64,
    price: f64,
) -> Result<()> {
    let client = ctx.client()?;
    let product_id = prompt::require_arg(product, "product", "Product ID")?;

    let body = json!({
        "product_id": product_id,
        "quantity": quantity,
        "price": price,
    });

    let result = client.post(&base_path(shop_id, cart_id), &body)?;

    if ctx.json {
        return output::json(result.get("data").unwrap_or(&Value::Null));
    }

    let data = result.get("data").cloned().unwrap_or(Value::Null);
    output::success(&format!(
        "Cart item created: {} (ID: {})",
        jstr(&data, "product_id"),
        jstr(&data, "id")
    ));
    Ok(())
}

fn update(
    ctx: &Ctx,
    shop_id: &str,
    cart_id: &str,
    item_id: &str,
    quantity: Option<i64>,
    price: Option<f64>,
) -> Result<()> {
    let client = ctx.client()?;

    let mut body = json!({});
    if let Some(quantity) = quantity {
        body["quantity"] = json!(quantity);
    }
    if let Some(price) = price {
        body["price"] = json!(price);
    }
    if body.as_object().is_some_and(|o| o.is_empty()) {
        anyhow::bail!("no flags provided; use --help to see available options");
    }

    let result = client.put(&format!("{}/{item_id}", base_path(shop_id, cart_id)), &body)?;

    if ctx.json {
        return output::json(result.get("data").unwrap_or(&Value::Null));
    }

    let data = result.get("data").cloned().unwrap_or(Value::Null);
    output::success(&format!("Cart item updated: {}", jstr(&data, "id")));
    Ok(())
}

fn delete(ctx: &Ctx, shop_id: &str, cart_id: &str, item_id: &str) -> Result<()> {
    let client = ctx.client()?;
    client.delete(&format!("{}/{item_id}", base_path(shop_id, cart_id)))?;
    output::success(&format!("Cart item {item_id} deleted."));
    Ok(())
}

fn count(ctx: &Ctx, shop_id: &str, cart_id: &str) -> Result<()> {
    let client = ctx.client()?;
    let body = client.get_with(&base_path(shop_id, cart_id), &[("limit", "0".to_string())])?;

    if ctx.json {
        return output::json(&json!({ "total": jint(&body, "total") }));
    }

    println!("Total cart items: {}", jint(&body, "total"));
    Ok(())
}
