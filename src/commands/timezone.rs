use anyhow::Result;
use clap::{Args, Subcommand};
use serde_json::Value;

use crate::cli::Ctx;
use crate::output;
use crate::util::{jint, jstr};

#[derive(Args)]
#[command(long_about = "List available timezones.")]
pub struct Cmd {
    #[command(subcommand)]
    command: Sub,
}

#[derive(Subcommand)]
enum Sub {
    /// List timezones
    List,
}

pub fn run(ctx: &Ctx, cmd: Cmd) -> Result<()> {
    match cmd.command {
        Sub::List => list(ctx),
    }
}

fn list(ctx: &Ctx) -> Result<()> {
    let client = ctx.client()?;
    let body = client.get("/timezones")?;

    let data = body.get("data").cloned().unwrap_or(Value::Null);

    if ctx.json {
        return output::json(&data);
    }

    let rows: Vec<Vec<String>> = data
        .as_array()
        .map(|tzs| {
            tzs.iter()
                .map(|tz| {
                    vec![
                        jstr(tz, "id"),
                        jstr(tz, "name"),
                        jint(tz, "offset").to_string(),
                    ]
                })
                .collect()
        })
        .unwrap_or_default();

    output::table(&["ID", "NAME", "OFFSET"], &rows);
    Ok(())
}
