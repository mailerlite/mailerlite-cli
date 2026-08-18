use anyhow::Result;
use serde_json::Value;

use super::jpath_int;
use crate::api::{self, Client};
use crate::tui::components::detail::DetailRow;
use crate::tui::components::table::{Cell, Column};
use crate::util::{jpath, jstr};

pub const EMPTY: &str = "No campaigns found.";

pub fn columns() -> Vec<Column> {
    vec![
        Column {
            title: "NAME",
            width: 28,
        },
        Column {
            title: "TYPE",
            width: 10,
        },
        Column {
            title: "STATUS",
            width: 12,
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
            title: "CLICKS",
            width: 8,
        },
    ]
}

pub fn fetch(client: &Client) -> Result<Vec<Value>> {
    api::fetch_all_paged(client, "/campaigns", &[], 100)
}

pub fn row(c: &Value) -> Vec<Cell> {
    vec![
        Cell::plain(jstr(c, "name")),
        Cell::plain(jstr(c, "type_for_humans")),
        Cell::plain(jstr(c, "status")),
        Cell::plain(jpath_int(c, "stats.sent").to_string()),
        Cell::plain(jpath_int(c, "stats.opens_count").to_string()),
        Cell::plain(jpath_int(c, "stats.clicks_count").to_string()),
    ]
}

pub fn detail(c: &Value) -> (String, Vec<DetailRow>) {
    let mut rows = vec![
        DetailRow::new("ID", jstr(c, "id")),
        DetailRow::new("Name", jstr(c, "name")),
        DetailRow::new("Type", jstr(c, "type_for_humans")),
        DetailRow::new("Status", jstr(c, "status")),
        DetailRow::new("Sent", jpath_int(c, "stats.sent").to_string()),
        DetailRow::new("Opens", jpath_int(c, "stats.opens_count").to_string()),
        DetailRow::new("Clicks", jpath_int(c, "stats.clicks_count").to_string()),
        DetailRow::new("Open Rate", jpath(c, "stats.open_rate.string")),
        DetailRow::new("Click Rate", jpath(c, "stats.click_rate.string")),
        DetailRow::new("Created", jstr(c, "created_at")),
    ];

    let scheduled = jstr(c, "scheduled_for");
    if !scheduled.is_empty() {
        rows.push(DetailRow::new("Scheduled For", scheduled));
    }

    (format!("Campaign: {}", jstr(c, "name")), rows)
}
