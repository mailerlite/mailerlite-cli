use anyhow::Result;
use serde_json::Value;

use super::jpath_int;
use crate::api::{self, Client};
use crate::tui::components::detail::DetailRow;
use crate::tui::components::table::{Cell, Column};
use crate::tui::theme;
use crate::util::{jint, jstr};

pub const EMPTY: &str = "No automations found.";

pub fn columns() -> Vec<Column> {
    vec![
        Column {
            title: "NAME",
            width: 30,
        },
        Column {
            title: "ENABLED",
            width: 9,
        },
        Column {
            title: "EMAILS",
            width: 8,
        },
        Column {
            title: "COMPLETED",
            width: 10,
        },
        Column {
            title: "IN QUEUE",
            width: 10,
        },
    ]
}

pub fn fetch(client: &Client) -> Result<Vec<Value>> {
    api::fetch_all_paged(client, "/automations", &[], 100)
}

fn enabled(v: &Value) -> bool {
    v.get("enabled").and_then(Value::as_bool).unwrap_or(false)
}

fn enabled_badge(on: bool) -> Cell {
    if on {
        Cell::colored("yes", theme::SUCCESS)
    } else {
        Cell::colored("no", theme::MUTED)
    }
}

pub fn row(a: &Value) -> Vec<Cell> {
    vec![
        Cell::plain(jstr(a, "name")),
        enabled_badge(enabled(a)),
        Cell::plain(jint(a, "emails_count").to_string()),
        Cell::plain(jpath_int(a, "stats.completed_subscribers_count").to_string()),
        Cell::plain(jpath_int(a, "stats.subscribers_in_queue_count").to_string()),
    ]
}

pub fn detail(a: &Value) -> (String, Vec<DetailRow>) {
    (
        format!("Automation: {}", jstr(a, "name")),
        vec![
            DetailRow::new("ID", jstr(a, "id")),
            DetailRow::new("Name", jstr(a, "name")),
            DetailRow::new("Enabled", if enabled(a) { "Yes" } else { "No" }),
            DetailRow::new("Emails", jint(a, "emails_count").to_string()),
            DetailRow::new(
                "Completed",
                jpath_int(a, "stats.completed_subscribers_count").to_string(),
            ),
            DetailRow::new(
                "In Queue",
                jpath_int(a, "stats.subscribers_in_queue_count").to_string(),
            ),
            DetailRow::new("Sent", jpath_int(a, "stats.sent").to_string()),
            DetailRow::new("Opens", jpath_int(a, "stats.opens_count").to_string()),
            DetailRow::new("Clicks", jpath_int(a, "stats.clicks_count").to_string()),
            DetailRow::new("Created", jstr(a, "created_at")),
        ],
    )
}
