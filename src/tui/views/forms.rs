use anyhow::Result;
use serde_json::Value;

use crate::api::{self, Client};
use crate::tui::components::detail::DetailRow;
use crate::tui::components::table::{Cell, Column};
use crate::tui::theme;
use crate::util::{jint, jpath, jstr};

pub const EMPTY: &str = "No forms found.";

#[derive(Clone, Copy, PartialEq, Eq)]
pub enum FormTab {
    Popup,
    Embedded,
    Promotion,
}

impl FormTab {
    pub const ALL: [FormTab; 3] = [FormTab::Popup, FormTab::Embedded, FormTab::Promotion];

    pub fn label(self) -> &'static str {
        match self {
            FormTab::Popup => "Popup",
            FormTab::Embedded => "Embedded",
            FormTab::Promotion => "Promotion",
        }
    }

    pub fn api_value(self) -> &'static str {
        match self {
            FormTab::Popup => "popup",
            FormTab::Embedded => "embedded",
            FormTab::Promotion => "promotion",
        }
    }

    pub fn next(self) -> FormTab {
        match self {
            FormTab::Popup => FormTab::Embedded,
            FormTab::Embedded => FormTab::Promotion,
            FormTab::Promotion => FormTab::Popup,
        }
    }

    pub fn prev(self) -> FormTab {
        match self {
            FormTab::Popup => FormTab::Promotion,
            FormTab::Embedded => FormTab::Popup,
            FormTab::Promotion => FormTab::Embedded,
        }
    }
}

pub fn columns() -> Vec<Column> {
    vec![
        Column {
            title: "NAME",
            width: 28,
        },
        Column {
            title: "TYPE",
            width: 12,
        },
        Column {
            title: "ACTIVE",
            width: 8,
        },
        Column {
            title: "CONVERSIONS",
            width: 12,
        },
        Column {
            title: "OPENS",
            width: 8,
        },
    ]
}

pub fn fetch(client: &Client, tab: FormTab) -> Result<Vec<Value>> {
    api::fetch_all_paged(client, &format!("/forms/{}", tab.api_value()), &[], 100)
}

fn active(f: &Value) -> bool {
    f.get("active").and_then(Value::as_bool).unwrap_or(false)
}

pub fn row(f: &Value) -> Vec<Cell> {
    let active_cell = if active(f) {
        Cell::colored("yes", theme::SUCCESS)
    } else {
        Cell::colored("no", theme::ERROR)
    };

    vec![
        Cell::plain(jstr(f, "name")),
        Cell::plain(jstr(f, "type")),
        active_cell,
        Cell::plain(jint(f, "conversions_count").to_string()),
        Cell::plain(jint(f, "opens_count").to_string()),
    ]
}

pub fn detail(f: &Value) -> (String, Vec<DetailRow>) {
    (
        format!("Form: {}", jstr(f, "name")),
        vec![
            DetailRow::new("ID", jstr(f, "id")),
            DetailRow::new("Name", jstr(f, "name")),
            DetailRow::new("Type", jstr(f, "type")),
            DetailRow::new("Active", if active(f) { "Yes" } else { "No" }),
            DetailRow::new("Conversions", jint(f, "conversions_count").to_string()),
            DetailRow::new("Conversion Rate", jpath(f, "conversions_rate.string")),
            DetailRow::new("Opens", jint(f, "opens_count").to_string()),
            DetailRow::new("Created", jstr(f, "created_at")),
        ],
    )
}
