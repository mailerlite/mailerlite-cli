use anyhow::Result;
use clap::{Args, Subcommand};
use serde_json::{json, Value};

use crate::api;
use crate::cli::Ctx;
use crate::output;
use crate::prompt;
use crate::util::{jfloat, jint, jstr};

#[derive(Args)]
#[command(long_about = "List, create, update, and delete products within a shop.")]
pub struct Cmd {
    /// shop ID (required)
    #[arg(long, global = true, default_value = "")]
    shop: String,
    #[command(subcommand)]
    command: Sub,
}

#[derive(Subcommand)]
enum Sub {
    /// List products
    List {
        /// maximum number of products to return (0 = all)
        #[arg(long, default_value_t = 25)]
        limit: u64,
    },
    /// Get product details
    #[command(arg_required_else_help = true)]
    Get {
        /// product ID
        product_id: String,
    },
    /// Create a new product
    Create {
        /// product name (required)
        #[arg(long, default_value = "")]
        name: String,
        /// product price (required)
        #[arg(long, default_value_t = 0.0)]
        price: f64,
        /// product URL
        #[arg(long, default_value = "")]
        url: String,
        /// product image URL
        #[arg(long, default_value = "")]
        image_url: String,
        /// product description
        #[arg(long, default_value = "")]
        description: String,
        /// product quantity
        #[arg(long)]
        quantity: Option<i64>,
    },
    /// Update a product
    #[command(arg_required_else_help = true)]
    Update {
        /// product ID
        product_id: String,
        /// product name
        #[arg(long)]
        name: Option<String>,
        /// product price
        #[arg(long)]
        price: Option<f64>,
        /// product URL
        #[arg(long)]
        url: Option<String>,
        /// product image URL
        #[arg(long)]
        image_url: Option<String>,
        /// product description
        #[arg(long)]
        description: Option<String>,
        /// product quantity
        #[arg(long)]
        quantity: Option<i64>,
    },
    /// Delete a product
    #[command(arg_required_else_help = true)]
    Delete {
        /// product ID
        product_id: String,
    },
    /// Get total product count
    Count,
}

pub fn run(ctx: &Ctx, cmd: Cmd) -> Result<()> {
    let shop_id = prompt::require_arg(&cmd.shop, "shop", "Shop ID")?;
    match cmd.command {
        Sub::List { limit } => list(ctx, &shop_id, limit),
        Sub::Get { product_id } => get(ctx, &shop_id, &product_id),
        Sub::Create {
            name,
            price,
            url,
            image_url,
            description,
            quantity,
        } => create(
            ctx,
            &shop_id,
            &name,
            price,
            &url,
            &image_url,
            &description,
            quantity,
        ),
        Sub::Update {
            product_id,
            name,
            price,
            url,
            image_url,
            description,
            quantity,
        } => update(
            ctx,
            &shop_id,
            &product_id,
            name,
            price,
            url,
            image_url,
            description,
            quantity,
        ),
        Sub::Delete { product_id } => delete(ctx, &shop_id, &product_id),
        Sub::Count => count(ctx, &shop_id),
    }
}

fn list(ctx: &Ctx, shop_id: &str, limit: u64) -> Result<()> {
    let client = ctx.client()?;
    let products = api::fetch_all_paged(
        &client,
        &format!("/ecommerce/shops/{shop_id}/products"),
        &[],
        limit,
    )?;

    if ctx.json {
        return output::json(&products);
    }

    let rows: Vec<Vec<String>> = products
        .iter()
        .map(|p| {
            vec![
                jstr(p, "id"),
                jstr(p, "name"),
                format!("{:.2}", jfloat(p, "price")),
                jint(p, "quantity").to_string(),
                jstr(p, "created_at"),
            ]
        })
        .collect();

    output::table(&["ID", "NAME", "PRICE", "QUANTITY", "CREATED"], &rows);
    Ok(())
}

fn get(ctx: &Ctx, shop_id: &str, product_id: &str) -> Result<()> {
    let client = ctx.client()?;
    let body = client.get(&format!("/ecommerce/shops/{shop_id}/products/{product_id}"))?;

    if ctx.json {
        return output::json(body.get("data").unwrap_or(&Value::Null));
    }

    let p = body.get("data").cloned().unwrap_or(Value::Null);
    let rows = vec![
        vec!["ID".to_string(), jstr(&p, "id")],
        vec!["Name".to_string(), jstr(&p, "name")],
        vec!["Price".to_string(), format!("{:.2}", jfloat(&p, "price"))],
        vec!["URL".to_string(), jstr(&p, "url")],
        vec!["Image URL".to_string(), jstr(&p, "image_url")],
        vec![
            "Description".to_string(),
            output::truncate(&jstr(&p, "description"), 60),
        ],
        vec!["Quantity".to_string(), jint(&p, "quantity").to_string()],
        vec!["Created".to_string(), jstr(&p, "created_at")],
        vec!["Updated".to_string(), jstr(&p, "updated_at")],
    ];

    output::table(&["FIELD", "VALUE"], &rows);
    Ok(())
}

#[allow(clippy::too_many_arguments)]
fn create(
    ctx: &Ctx,
    shop_id: &str,
    name: &str,
    price: f64,
    url: &str,
    image_url: &str,
    description: &str,
    quantity: Option<i64>,
) -> Result<()> {
    let client = ctx.client()?;
    let name = prompt::require_arg(name, "name", "Product name")?;

    let mut body = json!({ "name": name, "price": price });
    if !url.is_empty() {
        body["url"] = json!(url);
    }
    if !image_url.is_empty() {
        body["image_url"] = json!(image_url);
    }
    if !description.is_empty() {
        body["description"] = json!(description);
    }
    if let Some(quantity) = quantity {
        body["quantity"] = json!(quantity);
    }

    let result = client.post(&format!("/ecommerce/shops/{shop_id}/products"), &body)?;

    if ctx.json {
        return output::json(result.get("data").unwrap_or(&Value::Null));
    }

    let data = result.get("data").cloned().unwrap_or(Value::Null);
    output::success(&format!(
        "Product created: {} (ID: {})",
        jstr(&data, "name"),
        jstr(&data, "id")
    ));
    Ok(())
}

#[allow(clippy::too_many_arguments)]
fn update(
    ctx: &Ctx,
    shop_id: &str,
    product_id: &str,
    name: Option<String>,
    price: Option<f64>,
    url: Option<String>,
    image_url: Option<String>,
    description: Option<String>,
    quantity: Option<i64>,
) -> Result<()> {
    let client = ctx.client()?;

    let mut body = json!({});
    if let Some(name) = name {
        body["name"] = json!(name);
    }
    if let Some(price) = price {
        body["price"] = json!(price);
    }
    if let Some(url) = url {
        body["url"] = json!(url);
    }
    if let Some(image_url) = image_url {
        body["image_url"] = json!(image_url);
    }
    if let Some(description) = description {
        body["description"] = json!(description);
    }
    if let Some(quantity) = quantity {
        body["quantity"] = json!(quantity);
    }
    if body.as_object().is_some_and(|o| o.is_empty()) {
        anyhow::bail!("no flags provided; use --help to see available options");
    }

    let result = client.put(
        &format!("/ecommerce/shops/{shop_id}/products/{product_id}"),
        &body,
    )?;

    if ctx.json {
        return output::json(result.get("data").unwrap_or(&Value::Null));
    }

    let data = result.get("data").cloned().unwrap_or(Value::Null);
    output::success(&format!(
        "Product updated: {} (ID: {})",
        jstr(&data, "name"),
        jstr(&data, "id")
    ));
    Ok(())
}

fn delete(ctx: &Ctx, shop_id: &str, product_id: &str) -> Result<()> {
    let client = ctx.client()?;
    client.delete(&format!("/ecommerce/shops/{shop_id}/products/{product_id}"))?;
    output::success(&format!("Product {product_id} deleted."));
    Ok(())
}

fn count(ctx: &Ctx, shop_id: &str) -> Result<()> {
    let client = ctx.client()?;
    let body = client.get_with(
        &format!("/ecommerce/shops/{shop_id}/products"),
        &[("limit", "0".to_string())],
    )?;

    if ctx.json {
        return output::json(&json!({ "total": jint(&body, "total") }));
    }

    println!("Total products: {}", jint(&body, "total"));
    Ok(())
}
