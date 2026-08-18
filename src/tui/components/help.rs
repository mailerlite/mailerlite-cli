use ratatui::layout::Rect;
use ratatui::style::{Modifier, Style};
use ratatui::text::{Line, Span};
use ratatui::widgets::{Block, BorderType, Clear, Padding, Paragraph};
use ratatui::Frame;

use super::pad_right;
use crate::tui::keys;
use crate::tui::theme;

pub fn render(frame: &mut Frame, area: Rect) {
    let key_style = Style::new().fg(theme::KEY);
    let desc_style = Style::new().fg(theme::TEXT_SUB);

    let mut lines: Vec<Line<'static>> = vec![
        Line::styled(
            "Keyboard Shortcuts",
            Style::new().fg(theme::PRIMARY).add_modifier(Modifier::BOLD),
        ),
        Line::raw(""),
    ];

    for (i, section) in keys::HELP_SECTIONS.iter().enumerate() {
        if i > 0 {
            lines.push(Line::raw(""));
        }
        lines.push(Line::styled(
            section.title,
            Style::new().add_modifier(Modifier::BOLD),
        ));
        for binding in section.bindings {
            lines.push(Line::from(vec![
                Span::styled(pad_right(binding.key, 14), key_style),
                Span::styled(binding.desc, desc_style),
            ]));
        }
    }

    lines.push(Line::raw(""));
    lines.push(Line::styled(
        "Press ? or Esc to close",
        Style::new().fg(theme::MUTED),
    ));

    let overlay_width = 44u16.min(area.width);
    let overlay_height = (lines.len() as u16 + 4).min(area.height);
    let x = area.x + area.width.saturating_sub(overlay_width) / 2;
    let y = area.y + area.height.saturating_sub(overlay_height) / 2;
    let overlay = Rect::new(x, y, overlay_width, overlay_height);

    let block = Block::bordered()
        .border_type(BorderType::Rounded)
        .border_style(Style::new().fg(theme::PRIMARY))
        .padding(Padding::new(2, 2, 1, 1))
        .style(Style::new().bg(theme::BG_OVERLAY));

    frame.render_widget(Clear, overlay);
    frame.render_widget(Paragraph::new(lines).block(block), overlay);
}
