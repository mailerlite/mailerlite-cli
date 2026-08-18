use ratatui::layout::Rect;
use ratatui::style::Style;
use ratatui::text::{Line, Span};
use ratatui::widgets::{Block, Borders, Paragraph};
use ratatui::Frame;

use crate::tui::theme;

pub struct StatusBar {
    left: String,
    profile: String,
    loading: bool,
    load_text: String,
}

impl StatusBar {
    pub fn new() -> Self {
        Self {
            left: String::new(),
            profile: String::new(),
            loading: false,
            load_text: String::new(),
        }
    }

    pub fn set_left(&mut self, text: impl Into<String>) {
        self.left = text.into();
    }

    pub fn set_profile(&mut self, name: impl Into<String>) {
        self.profile = name.into();
    }

    pub fn set_loading(&mut self, loading: bool, text: impl Into<String>) {
        self.loading = loading;
        self.load_text = text.into();
    }

    pub fn render(&self, frame: &mut Frame, area: Rect) {
        let block = Block::new()
            .borders(Borders::TOP)
            .border_style(Style::new().fg(theme::MUTED));
        let inner = block.inner(area);
        frame.render_widget(block, area);
        if inner.width == 0 || inner.height == 0 {
            return;
        }

        let width = inner.width as usize;
        let muted = Style::new().fg(theme::MUTED);
        let key_style = Style::new().fg(theme::KEY);
        let value_style = Style::new().fg(theme::TEXT_SUB);

        let center = if self.loading {
            self.load_text.clone()
        } else {
            String::new()
        };

        let mut hints: Vec<(&str, String)> = vec![
            ("↑↓", "navigate".to_string()),
            ("enter", "select".to_string()),
            ("?", "help".to_string()),
        ];
        if !self.profile.is_empty() {
            hints.push(("profile:", self.profile.clone()));
        }

        let mut right_spans: Vec<Span> = Vec::new();
        let mut right_len = 0usize;
        for (i, (key, desc)) in hints.iter().enumerate() {
            if i > 0 {
                right_spans.push(Span::raw("  "));
                right_len += 2;
            }
            right_spans.push(Span::styled(key.to_string(), key_style));
            right_spans.push(Span::raw(" "));
            right_spans.push(Span::styled(desc.clone(), value_style));
            right_len += key.chars().count() + 1 + desc.chars().count();
        }

        let left_len = self.left.chars().count();
        let center_len = center.chars().count();
        let content_width = width.saturating_sub(4);

        let mut spans: Vec<Span> = vec![Span::styled(self.left.clone(), muted)];

        if left_len + center_len + right_len >= content_width {
            let gap = content_width.saturating_sub(left_len + right_len).max(1);
            spans.push(Span::raw(" ".repeat(gap)));
            spans.extend(right_spans);
        } else {
            let remaining = content_width - left_len - center_len - right_len;
            let left_gap = remaining / 2;
            let right_gap = remaining - left_gap;
            spans.push(Span::raw(" ".repeat(left_gap)));
            spans.push(Span::styled(center, muted));
            spans.push(Span::raw(" ".repeat(right_gap)));
            spans.extend(right_spans);
        }

        frame.render_widget(Paragraph::new(Line::from(spans)), inner);
    }
}
