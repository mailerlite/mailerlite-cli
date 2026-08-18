use ratatui::style::{Modifier, Style};
use ratatui::text::{Line, Span};

use super::pad_right;
use crate::tui::theme;

pub struct DetailRow {
    pub label: String,
    pub value: String,
}

impl DetailRow {
    pub fn new(label: impl Into<String>, value: impl Into<String>) -> Self {
        Self {
            label: label.into(),
            value: value.into(),
        }
    }
}

pub struct DetailPanel {
    title: String,
    rows: Vec<DetailRow>,
}

impl DetailPanel {
    pub fn new() -> Self {
        Self {
            title: String::new(),
            rows: Vec::new(),
        }
    }

    pub fn set_title(&mut self, title: impl Into<String>) {
        self.title = title.into();
    }

    pub fn set_rows(&mut self, rows: Vec<DetailRow>) {
        self.rows = rows;
    }

    pub fn lines(&self) -> Vec<Line<'static>> {
        let label_style = Style::new().fg(theme::MUTED);
        let value_style = Style::new().fg(theme::TEXT);

        let mut lines: Vec<Line<'static>> = vec![
            Line::styled(
                self.title.clone(),
                Style::new().fg(theme::PRIMARY).add_modifier(Modifier::BOLD),
            ),
            Line::raw(""),
        ];

        for row in &self.rows {
            if row.label.is_empty() && row.value.is_empty() {
                lines.push(Line::raw(""));
                continue;
            }
            let label = if row.label.is_empty() {
                pad_right("", 20)
            } else {
                pad_right(&format!("{}:", row.label), 20)
            };
            lines.push(Line::from(vec![
                Span::styled(label, label_style),
                Span::raw(" "),
                Span::styled(row.value.clone(), value_style),
            ]));
        }

        lines.push(Line::raw(""));
        lines.push(Line::styled(
            "Press Esc or Backspace to go back",
            Style::new().fg(theme::MUTED).add_modifier(Modifier::ITALIC),
        ));

        lines
    }
}
