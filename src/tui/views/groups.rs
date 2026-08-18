use anyhow::Result;
use serde_json::Value;

use super::fmt_date;
use crate::api::{self, Client};
use crate::tui::components::detail::DetailRow;
use crate::tui::components::table::{Cell, Column};
use crate::util::{jint, jpath, jstr};

pub const EMPTY: &str = "No groups found.";

pub fn columns() -> Vec<Column> {
    vec![
        Column {
            title: "NAME",
            width: 28,
        },
        Column {
            title: "ACTIVE",
            width: 8,
        },
        Column {
            title: "SENT",
            width: 8,
        },
        Column {
            title: "OPENS",
            width: 8,
        },
        Column {
            title: "CLICK RATE",
            width: 12,
        },
        Column {
            title: "CREATED",
            width: 12,
        },
    ]
}

pub fn fetch(client: &Client) -> Result<Vec<Value>> {
    api::fetch_all_paged(client, "/groups", &[], 100)
}

pub fn row(g: &Value) -> Vec<Cell> {
    vec![
        Cell::plain(jstr(g, "name")),
        Cell::plain(jint(g, "active_count").to_string()),
        Cell::plain(jint(g, "sent_count").to_string()),
        Cell::plain(jint(g, "opens_count").to_string()),
        Cell::plain(jpath(g, "click_rate.string")),
        Cell::plain(fmt_date(&jstr(g, "created_at"))),
    ]
}

pub fn detail(g: &Value) -> (String, Vec<DetailRow>) {
    (
        format!("Group: {}", jstr(g, "name")),
        vec![
            DetailRow::new("ID", jstr(g, "id")),
            DetailRow::new("Name", jstr(g, "name")),
            DetailRow::new("Active", jint(g, "active_count").to_string()),
            DetailRow::new("Sent", jint(g, "sent_count").to_string()),
            DetailRow::new("Opens", jint(g, "opens_count").to_string()),
            DetailRow::new("Open Rate", jpath(g, "open_rate.string")),
            DetailRow::new("Clicks", jint(g, "clicks_count").to_string()),
            DetailRow::new("Click Rate", jpath(g, "click_rate.string")),
            DetailRow::new("Unsubscribed", jint(g, "unsubscribed_count").to_string()),
            DetailRow::new("Bounced", jint(g, "bounced_count").to_string()),
            DetailRow::new("Created", jstr(g, "created_at")),
        ],
    )
}
