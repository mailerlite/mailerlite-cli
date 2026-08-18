use anyhow::Result;
use clap::{Args, Subcommand};
use serde_json::{json, Value};

use crate::api;
use crate::cli::Ctx;
use crate::output;
use crate::prompt;
use crate::util::{jfloat, jint, jstr};

#[derive(Args)]
#[command(long_about = "List, create, update, and delete categories within a shop.")]
pub struct Cmd {
    /// shop ID (required)
    #[arg(long, global = true, default_value = "")]
    shop: String,
    #[command(subcommand)]
    command: Sub,
}

#[derive(Subcommand)]
enum Sub {
    /// List categories
    List {
        /// maximum number of categories to return (0 = all)
        #[arg(long, default_value_t = 25)]
        limit: u64,
    },
    /// Get category details
    #[command(arg_required_else_help = true)]
    Get {
        /// category ID
        category_id: String,
    },
    /// Create a new category
    Create {
        /// category name (required)
        #[arg(long, default_value = "")]
        name: String,
    },
    /// Update a category
    #[command(arg_required_else_help = true)]
    Update {
        /// category ID
        category_id: String,
        /// category name
        #[arg(long)]
        name: Option<String>,
    },
    /// Delete a category
    #[command(arg_required_else_help = true)]
    Delete {
        /// category ID
        category_id: String,
    },
    /// Get total category count
    Count,
    /// List products in a category
    #[command(arg_required_else_help = true)]
    Products {
        /// category ID
        category_id: String,
    },
    /// Assign a product to a category
    #[command(arg_required_else_help = true)]
    AssignProduct {
        /// category ID
        category_id: String,
        /// product ID (required)
        #[arg(long, default_value = "")]
        product: String,
    },
    /// Remove a product from a category
    #[command(arg_required_else_help = true)]
    UnassignProduct {
        /// category ID
        category_id: String,
        /// product ID (required)
        #[arg(long, default_value = "")]
        product: String,
    },
}

pub fn run(ctx: &Ctx, cmd: Cmd) -> Result<()> {
    let shop_id = prompt::require_arg(&cmd.shop, "shop", "Shop ID")?;
    match cmd.command {
        Sub::List { limit } => list(ctx, &shop_id, limit),
        Sub::Get { category_id } => get(ctx, &shop_id, &category_id),
        Sub::Create { name } => create(ctx, &shop_id, &name),
        Sub::Update { category_id, name } => update(ctx, &shop_id, &category_id, name),
        Sub::Delete { category_id } => delete(ctx, &shop_id, &category_id),
        Sub::Count => count(ctx, &shop_id),
        Sub::Products { category_id } => products(ctx, &shop_id, &category_id),
        Sub::AssignProduct {
            category_id,
            product,
        } => assign_product(ctx, &shop_id, &category_id, &product),
        Sub::UnassignProduct {
            category_id,
            product,
        } => unassign_product(ctx, &shop_id, &category_id, &product),
    }
}

fn list(ctx: &Ctx, shop_id: &str, limit: u64) -> Result<()> {
    let client = ctx.client()?;
    let categories = api::fetch_all_paged(
        &client,
        &format!("/ecommerce/shops/{shop_id}/categories"),
        &[],
        limit,
    )?;

    if ctx.json {
        return output::json(&categories);
    }

    let rows: Vec<Vec<String>> = categories
        .iter()
        .map(|c| vec![jstr(c, "id"), jstr(c, "name"), jstr(c, "created_at")])
        .collect();

    output::table(&["ID", "NAME", "CREATED"], &rows);
    Ok(())
}

fn get(ctx: &Ctx, shop_id: &str, category_id: &str) -> Result<()> {
    let client = ctx.client()?;
    let body = client.get(&format!(
        "/ecommerce/shops/{shop_id}/categories/{category_id}"
    ))?;

    if ctx.json {
        return output::json(body.get("data").unwrap_or(&Value::Null));
    }

    let c = body.get("data").cloned().unwrap_or(Value::Null);
    let rows = vec![
        vec!["ID".to_string(), jstr(&c, "id")],
        vec!["Name".to_string(), jstr(&c, "name")],
        vec!["Created".to_string(), jstr(&c, "created_at")],
        vec!["Updated".to_string(), jstr(&c, "updated_at")],
    ];

    output::table(&["FIELD", "VALUE"], &rows);
    Ok(())
}

fn create(ctx: &Ctx, shop_id: &str, name: &str) -> Result<()> {
    let client = ctx.client()?;
    let name = prompt::require_arg(name, "name", "Category name")?;

    let body = json!({ "name": name });
    let result = client.post(&format!("/ecommerce/shops/{shop_id}/categories"), &body)?;

    if ctx.json {
        return output::json(result.get("data").unwrap_or(&Value::Null));
    }

    let data = result.get("data").cloned().unwrap_or(Value::Null);
    output::success(&format!(
        "Category created: {} (ID: {})",
        jstr(&data, "name"),
        jstr(&data, "id")
    ));
    Ok(())
}

fn update(ctx: &Ctx, shop_id: &str, category_id: &str, name: Option<String>) -> Result<()> {
    let client = ctx.client()?;

    let mut body = json!({});
    if let Some(name) = name {
        body["name"] = json!(name);
    }
    if body.as_object().is_some_and(|o| o.is_empty()) {
        anyhow::bail!("no flags provided; use --help to see available options");
    }

    let result = client.put(
        &format!("/ecommerce/shops/{shop_id}/categories/{category_id}"),
        &body,
    )?;

    if ctx.json {
        return output::json(result.get("data").unwrap_or(&Value::Null));
    }

    let data = result.get("data").cloned().unwrap_or(Value::Null);
    output::success(&format!(
        "Category updated: {} (ID: {})",
        jstr(&data, "name"),
        jstr(&data, "id")
    ));
    Ok(())
}

fn delete(ctx: &Ctx, shop_id: &str, category_id: &str) -> Result<()> {
    let client = ctx.client()?;
    client.delete(&format!(
        "/ecommerce/shops/{shop_id}/categories/{category_id}"
    ))?;
    output::success(&format!("Category {category_id} deleted."));
    Ok(())
}

fn count(ctx: &Ctx, shop_id: &str) -> Result<()> {
    let client = ctx.client()?;
    let body = client.get_with(
        &format!("/ecommerce/shops/{shop_id}/categories"),
        &[("limit", "0".to_string())],
    )?;

    if ctx.json {
        return output::json(&json!({ "total": jint(&body, "total") }));
    }

    println!("Total categories: {}", jint(&body, "total"));
    Ok(())
}

fn products(ctx: &Ctx, shop_id: &str, category_id: &str) -> Result<()> {
    let client = ctx.client()?;
    let body = client.get(&format!(
        "/ecommerce/shops/{shop_id}/categories/{category_id}/products"
    ))?;

    if ctx.json {
        return output::json(body.get("data").unwrap_or(&Value::Null));
    }

    let rows: Vec<Vec<String>> = body
        .get("data")
        .and_then(Value::as_array)
        .map(|items| {
            items
                .iter()
                .map(|p| {
                    vec![
                        jstr(p, "id"),
                        jstr(p, "name"),
                        format!("{:.2}", jfloat(p, "price")),
                        jstr(p, "created_at"),
                    ]
                })
                .collect()
        })
        .unwrap_or_default();

    output::table(&["ID", "NAME", "PRICE", "CREATED"], &rows);
    Ok(())
}

fn assign_product(ctx: &Ctx, shop_id: &str, category_id: &str, product: &str) -> Result<()> {
    let client = ctx.client()?;
    let product_id = prompt::require_arg(product, "product", "Product ID")?;

    let body = json!({ "product_id": product_id });
    client.post(
        &format!("/ecommerce/shops/{shop_id}/categories/{category_id}/products"),
        &body,
    )?;

    output::success(&format!(
        "Product {product_id} assigned to category {category_id}."
    ));
    Ok(())
}

fn unassign_product(ctx: &Ctx, shop_id: &str, category_id: &str, product: &str) -> Result<()> {
    let client = ctx.client()?;
    let product_id = prompt::require_arg(product, "product", "Product ID")?;

    client.delete(&format!(
        "/ecommerce/shops/{shop_id}/categories/{category_id}/products/{product_id}"
    ))?;

    output::success(&format!(
        "Product {product_id} removed from category {category_id}."
    ));
    Ok(())
}
