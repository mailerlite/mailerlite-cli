use anyhow::Result;
use clap::{Args, Subcommand};
use serde_json::{json, Value};

use crate::api;
use crate::cli::Ctx;
use crate::output;
use crate::prompt;
use crate::util::{jint, jstr};

#[derive(Args)]
#[command(long_about = "List, create, update, and delete customers within a shop.")]
pub struct Cmd {
    /// shop ID (required)
    #[arg(long, global = true, default_value = "")]
    shop: String,
    #[command(subcommand)]
    command: Sub,
}

#[derive(Subcommand)]
enum Sub {
    /// List customers
    List {
        /// maximum number of customers to return (0 = all)
        #[arg(long, default_value_t = 25)]
        limit: u64,
    },
    /// Get customer details
    #[command(arg_required_else_help = true)]
    Get {
        /// customer ID
        customer_id: String,
    },
    /// Create a new customer
    Create {
        /// customer email (required)
        #[arg(long, default_value = "")]
        email: String,
        /// customer first name
        #[arg(long, default_value = "")]
        first_name: String,
        /// customer last name
        #[arg(long, default_value = "")]
        last_name: String,
    },
    /// Update a customer
    #[command(arg_required_else_help = true)]
    Update {
        /// customer ID
        customer_id: String,
        /// customer email
        #[arg(long)]
        email: Option<String>,
        /// customer first name
        #[arg(long)]
        first_name: Option<String>,
        /// customer last name
        #[arg(long)]
        last_name: Option<String>,
    },
    /// Delete a customer
    #[command(arg_required_else_help = true)]
    Delete {
        /// customer ID
        customer_id: String,
    },
    /// Get total customer count
    Count,
}

pub fn run(ctx: &Ctx, cmd: Cmd) -> Result<()> {
    let shop_id = prompt::require_arg(&cmd.shop, "shop", "Shop ID")?;
    match cmd.command {
        Sub::List { limit } => list(ctx, &shop_id, limit),
        Sub::Get { customer_id } => get(ctx, &shop_id, &customer_id),
        Sub::Create {
            email,
            first_name,
            last_name,
        } => create(ctx, &shop_id, &email, &first_name, &last_name),
        Sub::Update {
            customer_id,
            email,
            first_name,
            last_name,
        } => update(ctx, &shop_id, &customer_id, email, first_name, last_name),
        Sub::Delete { customer_id } => delete(ctx, &shop_id, &customer_id),
        Sub::Count => count(ctx, &shop_id),
    }
}

fn list(ctx: &Ctx, shop_id: &str, limit: u64) -> Result<()> {
    let client = ctx.client()?;
    let customers = api::fetch_all_paged(
        &client,
        &format!("/ecommerce/shops/{shop_id}/customers"),
        &[],
        limit,
    )?;

    if ctx.json {
        return output::json(&customers);
    }

    let rows: Vec<Vec<String>> = customers
        .iter()
        .map(|c| {
            vec![
                jstr(c, "id"),
                jstr(c, "email"),
                jstr(c, "first_name"),
                jstr(c, "last_name"),
                jstr(c, "created_at"),
            ]
        })
        .collect();

    output::table(
        &["ID", "EMAIL", "FIRST NAME", "LAST NAME", "CREATED"],
        &rows,
    );
    Ok(())
}

fn get(ctx: &Ctx, shop_id: &str, customer_id: &str) -> Result<()> {
    let client = ctx.client()?;
    let body = client.get(&format!(
        "/ecommerce/shops/{shop_id}/customers/{customer_id}"
    ))?;

    if ctx.json {
        return output::json(body.get("data").unwrap_or(&Value::Null));
    }

    let c = body.get("data").cloned().unwrap_or(Value::Null);
    let rows = vec![
        vec!["ID".to_string(), jstr(&c, "id")],
        vec!["Email".to_string(), jstr(&c, "email")],
        vec!["First Name".to_string(), jstr(&c, "first_name")],
        vec!["Last Name".to_string(), jstr(&c, "last_name")],
        vec!["Created".to_string(), jstr(&c, "created_at")],
        vec!["Updated".to_string(), jstr(&c, "updated_at")],
    ];

    output::table(&["FIELD", "VALUE"], &rows);
    Ok(())
}

fn create(ctx: &Ctx, shop_id: &str, email: &str, first_name: &str, last_name: &str) -> Result<()> {
    let client = ctx.client()?;
    let email = prompt::require_arg(email, "email", "Customer email")?;

    let mut body = json!({ "email": email });
    if !first_name.is_empty() {
        body["first_name"] = json!(first_name);
    }
    if !last_name.is_empty() {
        body["last_name"] = json!(last_name);
    }

    let result = client.post(&format!("/ecommerce/shops/{shop_id}/customers"), &body)?;

    if ctx.json {
        return output::json(result.get("data").unwrap_or(&Value::Null));
    }

    let data = result.get("data").cloned().unwrap_or(Value::Null);
    output::success(&format!(
        "Customer created: {} (ID: {})",
        jstr(&data, "email"),
        jstr(&data, "id")
    ));
    Ok(())
}

fn update(
    ctx: &Ctx,
    shop_id: &str,
    customer_id: &str,
    email: Option<String>,
    first_name: Option<String>,
    last_name: Option<String>,
) -> Result<()> {
    let client = ctx.client()?;

    let mut body = json!({});
    if let Some(email) = email {
        body["email"] = json!(email);
    }
    if let Some(first_name) = first_name {
        body["first_name"] = json!(first_name);
    }
    if let Some(last_name) = last_name {
        body["last_name"] = json!(last_name);
    }
    if body.as_object().is_some_and(|o| o.is_empty()) {
        anyhow::bail!("no flags provided; use --help to see available options");
    }

    let result = client.put(
        &format!("/ecommerce/shops/{shop_id}/customers/{customer_id}"),
        &body,
    )?;

    if ctx.json {
        return output::json(result.get("data").unwrap_or(&Value::Null));
    }

    let data = result.get("data").cloned().unwrap_or(Value::Null);
    output::success(&format!(
        "Customer updated: {} (ID: {})",
        jstr(&data, "email"),
        jstr(&data, "id")
    ));
    Ok(())
}

fn delete(ctx: &Ctx, shop_id: &str, customer_id: &str) -> Result<()> {
    let client = ctx.client()?;
    client.delete(&format!(
        "/ecommerce/shops/{shop_id}/customers/{customer_id}"
    ))?;
    output::success(&format!("Customer {customer_id} deleted."));
    Ok(())
}

fn count(ctx: &Ctx, shop_id: &str) -> Result<()> {
    let client = ctx.client()?;
    let body = client.get_with(
        &format!("/ecommerce/shops/{shop_id}/customers"),
        &[("limit", "0".to_string())],
    )?;

    if ctx.json {
        return output::json(&json!({ "total": jint(&body, "total") }));
    }

    println!("Total customers: {}", jint(&body, "total"));
    Ok(())
}
