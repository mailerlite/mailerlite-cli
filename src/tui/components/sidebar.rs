use ratatui::layout::Rect;
use ratatui::style::{Modifier, Style};
use ratatui::text::Line;
use ratatui::widgets::{Block, Borders, Padding, Paragraph};
use ratatui::Frame;

use super::pad_right;
use crate::tui::theme;
use crate::tui::views::ViewType;

pub const WIDTH: u16 = 22;

pub struct Sidebar {
    active: ViewType,
    focused: bool,
}

impl Sidebar {
    pub fn new() -> Self {
        Self {
            active: ViewType::Subscribers,
            focused: false,
        }
    }

    pub fn set_active(&mut self, v: ViewType) {
        self.active = v;
    }

    pub fn active(&self) -> ViewType {
        self.active
    }

    pub fn set_focused(&mut self, focused: bool) {
        self.focused = focused;
    }

    pub fn next(&mut self) {
        let idx = (self.active.index() + 1) % ViewType::ALL.len();
        self.active = ViewType::ALL[idx];
    }

    pub fn prev(&mut self) {
        let idx = if self.active.index() == 0 {
            ViewType::ALL.len() - 1
        } else {
            self.active.index() - 1
        };
        self.active = ViewType::ALL[idx];
    }

    pub fn render(&self, frame: &mut Frame, area: Rect) {
        let block = Block::new()
            .borders(Borders::RIGHT)
            .border_style(Style::new().fg(theme::MUTED))
            .padding(Padding::new(1, 1, 1, 1));

        let mut lines: Vec<Line> = Vec::new();
        for v in ViewType::ALL {
            let is_active = v == self.active;
            let prefix = if is_active { "▸ " } else { "  " };
            let text = pad_right(&format!("{}{} {}", prefix, v.icon(), v.label()), 16);
            let style = if is_active {
                if self.focused {
                    Style::new()
                        .fg(theme::TEXT)
                        .bg(theme::ACCENT)
                        .add_modifier(Modifier::BOLD)
                } else {
                    Style::new()
                        .fg(theme::PRIMARY)
                        .bg(theme::BG_SELECTED)
                        .add_modifier(Modifier::BOLD)
                }
            } else {
                Style::new()
            };
            lines.push(Line::styled(format!(" {text} "), style));
        }

        frame.render_widget(Paragraph::new(lines).block(block), area);
    }
}
