use anyhow::{anyhow, Result};
use clap::{Args, Subcommand};
use serde_json::Value;

use crate::cli::Ctx;
use crate::output;
use crate::prompt;

#[derive(Args)]
#[command(long_about = "Import categories, products, or orders in bulk from a JSON file.")]
pub struct Cmd {
    #[command(subcommand)]
    command: Sub,
}

#[derive(Subcommand)]
enum Sub {
    /// Bulk import categories
    Categories {
        /// shop ID (required)
        #[arg(long, default_value = "")]
        shop: String,
        /// path to JSON file (required)
        #[arg(long, default_value = "")]
        file: String,
    },
    /// Bulk import products
    Products {
        /// shop ID (required)
        #[arg(long, default_value = "")]
        shop: String,
        /// path to JSON file (required)
        #[arg(long, default_value = "")]
        file: String,
    },
    /// Bulk import orders
    Orders {
        /// shop ID (required)
        #[arg(long, default_value = "")]
        shop: String,
        /// path to JSON file (required)
        #[arg(long, default_value = "")]
        file: String,
    },
}

pub fn run(ctx: &Ctx, cmd: Cmd) -> Result<()> {
    match cmd.command {
        Sub::Categories { shop, file } => do_import(ctx, &shop, &file, "categories"),
        Sub::Products { shop, file } => do_import(ctx, &shop, &file, "products"),
        Sub::Orders { shop, file } => do_import(ctx, &shop, &file, "orders"),
    }
}

fn read_json_file(path: &str) -> Result<Value> {
    let data = std::fs::read(path).map_err(|e| anyhow!("failed to read file {path}: {e}"))?;
    serde_json::from_slice(&data).map_err(|_| anyhow!("file {path} does not contain valid JSON"))
}

fn do_import(ctx: &Ctx, shop: &str, file: &str, resource: &str) -> Result<()> {
    let shop_id = prompt::require_arg(shop, "shop", "Shop ID")?;
    let file_path = prompt::require_arg(file, "file", "Path to JSON file")?;

    let data = read_json_file(&file_path)?;

    let client = ctx.client()?;
    let result = client.post(
        &format!("/ecommerce/shops/{shop_id}/{resource}/import"),
        &data,
    )?;

    if ctx.json {
        return output::json(&result);
    }

    output::success(&format!("Import of {resource} completed successfully."));
    Ok(())
}
