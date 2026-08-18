pub mod automations;
pub mod campaigns;
pub mod forms;
pub mod groups;
pub mod subscribers;

use chrono::NaiveDateTime;
use crossterm::event::KeyCode;
use ratatui::layout::{Constraint, Layout, Rect};
use ratatui::style::{Modifier, Style};
use ratatui::text::{Line, Span};
use ratatui::widgets::Paragraph;
use ratatui::Frame;
use serde_json::Value;

use self::forms::FormTab;
use super::components::detail::DetailPanel;
use super::components::table::Table;
use super::theme;

#[derive(Clone, Copy, PartialEq, Eq)]
pub enum ViewType {
    Subscribers,
    Campaigns,
    Automations,
    Groups,
    Forms,
}

impl ViewType {
    pub const ALL: [ViewType; 5] = [
        ViewType::Subscribers,
        ViewType::Campaigns,
        ViewType::Automations,
        ViewType::Groups,
        ViewType::Forms,
    ];

    pub fn index(self) -> usize {
        match self {
            ViewType::Subscribers => 0,
            ViewType::Campaigns => 1,
            ViewType::Automations => 2,
            ViewType::Groups => 3,
            ViewType::Forms => 4,
        }
    }

    pub fn label(self) -> &'static str {
        match self {
            ViewType::Subscribers => "Subscribers",
            ViewType::Campaigns => "Campaigns",
            ViewType::Automations => "Automations",
            ViewType::Groups => "Groups",
            ViewType::Forms => "Forms",
        }
    }

    pub fn icon(self) -> &'static str {
        match self {
            ViewType::Subscribers => "◉",
            ViewType::Campaigns => "◈",
            ViewType::Automations => "◆",
            ViewType::Groups => "◇",
            ViewType::Forms => "◌",
        }
    }
}

/// Parses "YYYY-MM-DD HH:MM:SS" and renders the date part; "" on failure.
pub fn fmt_date(s: &str) -> String {
    match NaiveDateTime::parse_from_str(s, "%Y-%m-%d %H:%M:%S") {
        Ok(t) => t.format("%Y-%m-%d").to_string(),
        Err(_) => String::new(),
    }
}

/// Extracts a nested integer via a dotted path (0 when absent).
pub fn jpath_int(v: &Value, path: &str) -> i64 {
    let mut cur = v;
    for part in path.split('.') {
        match cur.get(part) {
            Some(next) => cur = next,
            None => return 0,
        }
    }
    cur.as_i64().unwrap_or(0)
}

pub enum KeyResult {
    None,
    Fetch,
}

pub struct View {
    pub kind: ViewType,
    pub table: Table,
    pub detail: DetailPanel,
    pub items: Vec<Value>,
    pub loading: bool,
    pub showing_detail: bool,
    pub tab: FormTab,
}

impl View {
    pub fn new(kind: ViewType) -> Self {
        let (columns, empty_msg) = match kind {
            ViewType::Subscribers => (subscribers::columns(), subscribers::EMPTY),
            ViewType::Campaigns => (campaigns::columns(), campaigns::EMPTY),
            ViewType::Automations => (automations::columns(), automations::EMPTY),
            ViewType::Groups => (groups::columns(), groups::EMPTY),
            ViewType::Forms => (forms::columns(), forms::EMPTY),
        };
        let mut table = Table::new(columns, empty_msg);
        table.set_loading(true);
        Self {
            kind,
            table,
            detail: DetailPanel::new(),
            items: Vec::new(),
            loading: true,
            showing_detail: false,
            tab: FormTab::Popup,
        }
    }

    pub fn set_loaded(&mut self, items: Vec<Value>) {
        let rows = items
            .iter()
            .map(|v| match self.kind {
                ViewType::Subscribers => subscribers::row(v),
                ViewType::Campaigns => campaigns::row(v),
                ViewType::Automations => automations::row(v),
                ViewType::Groups => groups::row(v),
                ViewType::Forms => forms::row(v),
            })
            .collect();
        self.items = items;
        self.table.set_rows(rows);
        self.table.set_loading(false);
        self.loading = false;
    }

    pub fn handle_key(&mut self, code: KeyCode) -> KeyResult {
        if self.showing_detail {
            if matches!(code, KeyCode::Esc | KeyCode::Backspace) {
                self.showing_detail = false;
            }
            return KeyResult::None;
        }

        match code {
            KeyCode::Char('j') | KeyCode::Down => self.table.move_down(),
            KeyCode::Char('k') | KeyCode::Up => self.table.move_up(),
            KeyCode::Char('g') => self.table.goto_top(),
            KeyCode::Char('G') => self.table.goto_bottom(),
            KeyCode::Enter => self.show_detail(),
            KeyCode::Char('r') => {
                self.loading = true;
                self.table.set_loading(true);
                return KeyResult::Fetch;
            }
            KeyCode::Char('h') | KeyCode::Left if self.kind == ViewType::Forms => {
                self.tab = self.tab.prev();
                self.loading = true;
                self.table.set_loading(true);
                return KeyResult::Fetch;
            }
            KeyCode::Char('l') | KeyCode::Right if self.kind == ViewType::Forms => {
                self.tab = self.tab.next();
                self.loading = true;
                self.table.set_loading(true);
                return KeyResult::Fetch;
            }
            _ => {}
        }
        KeyResult::None
    }

    fn show_detail(&mut self) {
        let Some(item) = self.items.get(self.table.cursor()) else {
            return;
        };
        let (title, rows) = match self.kind {
            ViewType::Subscribers => subscribers::detail(item),
            ViewType::Campaigns => campaigns::detail(item),
            ViewType::Automations => automations::detail(item),
            ViewType::Groups => groups::detail(item),
            ViewType::Forms => forms::detail(item),
        };
        self.detail.set_title(title);
        self.detail.set_rows(rows);
        self.showing_detail = true;
    }

    pub fn render(&mut self, frame: &mut Frame, area: Rect) {
        if self.showing_detail {
            frame.render_widget(Paragraph::new(self.detail.lines()), area);
            return;
        }

        let table_area = if self.kind == ViewType::Forms {
            let [chrome, rest] =
                Layout::vertical([Constraint::Length(4), Constraint::Min(0)]).areas(area);
            frame.render_widget(Paragraph::new(self.forms_chrome()), chrome);
            rest
        } else {
            area
        };

        self.table
            .set_size(table_area.width as usize, table_area.height as usize);
        frame.render_widget(Paragraph::new(self.table.lines()), table_area);
    }

    fn forms_chrome(&self) -> Vec<Line<'static>> {
        let mut tabs: Vec<Span<'static>> = Vec::new();
        let mut bar_width = 0usize;
        for t in FormTab::ALL {
            let label = format!("  {}  ", t.label());
            bar_width += label.chars().count();
            let style = if t == self.tab {
                Style::new()
                    .fg(theme::PRIMARY)
                    .bg(theme::BG_SELECTED)
                    .add_modifier(Modifier::BOLD)
            } else {
                Style::new()
            };
            tabs.push(Span::styled(label, style));
        }

        vec![
            Line::from(tabs),
            Line::styled("─".repeat(bar_width), Style::new().fg(theme::MUTED)),
            Line::styled(
                format!("← → to switch types | {} forms", self.items.len()),
                Style::new().fg(theme::MUTED),
            ),
            Line::raw(""),
        ]
    }
}
