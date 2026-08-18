use ratatui::style::{Color, Modifier, Style};
use ratatui::text::{Line, Span};

use super::pad_right;
use crate::tui::theme;

pub struct Column {
    pub title: &'static str,
    pub width: usize,
}

#[derive(Clone)]
pub struct Cell {
    pub text: String,
    pub fg: Option<Color>,
}

impl Cell {
    pub fn plain(text: impl Into<String>) -> Self {
        Self {
            text: text.into(),
            fg: None,
        }
    }

    pub fn colored(text: impl Into<String>, fg: Color) -> Self {
        Self {
            text: text.into(),
            fg: Some(fg),
        }
    }
}

pub struct Table {
    columns: Vec<Column>,
    rows: Vec<Vec<Cell>>,
    cursor: usize,
    offset: usize,
    width: usize,
    height: usize,
    focused: bool,
    loading: bool,
    empty_msg: &'static str,
}

impl Table {
    pub fn new(columns: Vec<Column>, empty_msg: &'static str) -> Self {
        Self {
            columns,
            rows: Vec::new(),
            cursor: 0,
            offset: 0,
            width: 0,
            height: 0,
            focused: false,
            loading: false,
            empty_msg,
        }
    }

    pub fn set_rows(&mut self, rows: Vec<Vec<Cell>>) {
        self.rows = rows;
        self.cursor = 0;
        self.offset = 0;
    }

    pub fn set_size(&mut self, width: usize, height: usize) {
        self.width = width;
        self.height = height;
    }

    pub fn set_focused(&mut self, focused: bool) {
        self.focused = focused;
    }

    pub fn set_loading(&mut self, loading: bool) {
        self.loading = loading;
    }

    pub fn cursor(&self) -> usize {
        self.cursor
    }

    pub fn move_up(&mut self) {
        if self.cursor > 0 {
            self.cursor -= 1;
            self.update_offset();
        }
    }

    pub fn move_down(&mut self) {
        if self.cursor + 1 < self.rows.len() {
            self.cursor += 1;
            self.update_offset();
        }
    }

    pub fn goto_top(&mut self) {
        self.cursor = 0;
        self.offset = 0;
    }

    pub fn goto_bottom(&mut self) {
        if !self.rows.is_empty() {
            self.cursor = self.rows.len() - 1;
        }
        self.update_offset();
    }

    fn update_offset(&mut self) {
        let visible = self.visible_row_count();
        if visible == 0 {
            return;
        }
        if self.cursor >= self.offset + visible {
            self.offset = self.cursor - visible + 1;
        }
        if self.cursor < self.offset {
            self.offset = self.cursor;
        }
    }

    fn visible_row_count(&self) -> usize {
        self.height.saturating_sub(3)
    }

    fn content_width(&self) -> usize {
        let widths: usize = self.columns.iter().map(|c| c.width).sum();
        widths + 2 * self.columns.len().saturating_sub(1)
    }

    fn fit(&self, text: &str, width: usize) -> String {
        let len = text.chars().count();
        if len > width {
            if width > 3 {
                let truncated: String = text.chars().take(width - 3).collect();
                return format!("{truncated}...");
            }
            return text.chars().take(width).collect();
        }
        pad_right(text, width)
    }

    pub fn lines(&self) -> Vec<Line<'static>> {
        let empty_style = Style::new().fg(theme::MUTED).add_modifier(Modifier::ITALIC);

        if self.loading {
            return vec![Line::styled("Loading...", empty_style)];
        }
        if self.rows.is_empty() {
            return vec![Line::styled(self.empty_msg, empty_style)];
        }

        let mut lines = Vec::new();

        let header: Vec<String> = self
            .columns
            .iter()
            .map(|c| self.fit(c.title, c.width))
            .collect();
        lines.push(Line::styled(
            header.join("  "),
            Style::new().fg(theme::PRIMARY).add_modifier(Modifier::BOLD),
        ));
        lines.push(Line::styled(
            "─".repeat(self.content_width()),
            Style::new().fg(theme::MUTED),
        ));

        let visible = self.visible_row_count().max(1);
        let start = self.offset;
        let end = (start + visible).min(self.rows.len());

        for i in start..end {
            let row = &self.rows[i];
            if i == self.cursor {
                let bg = if self.focused {
                    theme::ACCENT
                } else {
                    theme::BG_SELECTED
                };
                let cells: Vec<String> = self
                    .columns
                    .iter()
                    .enumerate()
                    .map(|(idx, col)| {
                        let text = row.get(idx).map(|c| c.text.as_str()).unwrap_or("");
                        self.fit(text, col.width)
                    })
                    .collect();
                let content = pad_right(&cells.join("  "), self.width.saturating_sub(4));
                lines.push(Line::styled(content, Style::new().bg(bg).fg(theme::TEXT)));
            } else {
                let mut spans: Vec<Span<'static>> = Vec::new();
                for (idx, col) in self.columns.iter().enumerate() {
                    if idx > 0 {
                        spans.push(Span::raw("  "));
                    }
                    let (text, fg) = row
                        .get(idx)
                        .map(|c| (c.text.as_str(), c.fg))
                        .unwrap_or(("", None));
                    let fitted = self.fit(text, col.width);
                    match fg {
                        Some(color) => spans.push(Span::styled(fitted, Style::new().fg(color))),
                        None => spans.push(Span::raw(fitted)),
                    }
                }
                lines.push(Line::from(spans));
            }
        }

        lines
    }
}
