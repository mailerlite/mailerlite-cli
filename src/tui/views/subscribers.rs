use anyhow::Result;
use serde_json::Value;

use super::fmt_date;
use crate::api::{self, Client};
use crate::tui::components::detail::DetailRow;
use crate::tui::components::table::{Cell, Column};
use crate::tui::theme;
use crate::util::{jfloat, jint, jstr};

pub const EMPTY: &str = "No subscribers found.";

pub fn columns() -> Vec<Column> {
    vec![
        Column {
            title: "EMAIL",
            width: 30,
        },
        Column {
            title: "STATUS",
            width: 10,
        },
        Column {
            title: "SOURCE",
            width: 12,
        },
        Column {
            title: "OPENS",
            width: 8,
        },
        Column {
            title: "CLICKS",
            width: 8,
        },
        Column {
            title: "SUBSCRIBED",
            width: 12,
        },
    ]
}

pub fn fetch(client: &Client) -> Result<Vec<Value>> {
    api::fetch_all_cursor(client, "/subscribers", &[], 100)
}

fn status_badge(status: &str) -> Cell {
    match status {
        "active" => Cell::colored("active", theme::SUCCESS),
        "unsubscribed" => Cell::colored("unsub", theme::ERROR),
        "unconfirmed" => Cell::colored("unconf", theme::MUTED),
        "bounced" => Cell::colored("bounced", theme::ERROR),
        "junk" => Cell::colored("junk", theme::ERROR),
        _ => Cell::plain(status.to_string()),
    }
}

pub fn row(s: &Value) -> Vec<Cell> {
    vec![
        Cell::plain(jstr(s, "email")),
        status_badge(&jstr(s, "status")),
        Cell::plain(jstr(s, "source")),
        Cell::plain(jint(s, "opens_count").to_string()),
        Cell::plain(jint(s, "clicks_count").to_string()),
        Cell::plain(fmt_date(&jstr(s, "subscribed_at"))),
    ]
}

pub fn detail(s: &Value) -> (String, Vec<DetailRow>) {
    (
        format!("Subscriber: {}", jstr(s, "email")),
        vec![
            DetailRow::new("ID", jstr(s, "id")),
            DetailRow::new("Email", jstr(s, "email")),
            DetailRow::new("Status", jstr(s, "status")),
            DetailRow::new("Source", jstr(s, "source")),
            DetailRow::new("Opens", jint(s, "opens_count").to_string()),
            DetailRow::new("Clicks", jint(s, "clicks_count").to_string()),
            DetailRow::new("Open Rate", format!("{:.1}%", jfloat(s, "open_rate"))),
            DetailRow::new("Click Rate", format!("{:.1}%", jfloat(s, "click_rate"))),
            DetailRow::new("Subscribed", jstr(s, "subscribed_at")),
        ],
    )
}
