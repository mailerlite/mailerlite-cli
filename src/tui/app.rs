use std::sync::mpsc::{self, Receiver, Sender};
use std::thread;
use std::time::Duration;

use anyhow::Result;
use crossterm::event::{self, Event, KeyCode, KeyEvent, KeyEventKind, KeyModifiers};
use ratatui::backend::Backend;
use ratatui::layout::{Constraint, Layout, Rect};
use ratatui::style::{Modifier, Style};
use ratatui::text::{Line, Span};
use ratatui::widgets::{Block, Borders, Paragraph};
use ratatui::{Frame, Terminal};
use serde_json::Value;

use crate::api::Client;

use super::components::{help, sidebar::Sidebar, spinner::Spinner, statusbar::StatusBar};
use super::theme;
use super::views::{self, KeyResult, View, ViewType};

pub enum Msg {
    Loaded(ViewType, Vec<Value>),
    Error(ViewType, String),
}

#[derive(PartialEq, Eq)]
enum Focus {
    Sidebar,
    Content,
}

pub struct App {
    client: Client,
    profile: String,
    tx: Sender<Msg>,
    rx: Receiver<Msg>,
    sidebar: Sidebar,
    statusbar: StatusBar,
    spinner: Spinner,
    views: Vec<View>,
    active: ViewType,
    focus: Focus,
    show_help: bool,
    err: Option<String>,
}

impl App {
    pub fn new(client: Client, profile: String) -> Self {
        let (tx, rx) = mpsc::channel();
        Self {
            client,
            profile,
            tx,
            rx,
            sidebar: Sidebar::new(),
            statusbar: StatusBar::new(),
            spinner: Spinner::new("Loading..."),
            views: ViewType::ALL.iter().map(|&k| View::new(k)).collect(),
            active: ViewType::Subscribers,
            focus: Focus::Content,
            show_help: false,
            err: None,
        }
    }

    pub fn run<B: Backend>(&mut self, terminal: &mut Terminal<B>) -> Result<()> {
        self.fetch(self.active);

        loop {
            self.drain_msgs();
            terminal.draw(|frame| self.draw(frame))?;

            if event::poll(Duration::from_millis(100))? {
                match event::read()? {
                    Event::Key(key) if key.kind == KeyEventKind::Press && self.handle_key(key) => {
                        return Ok(());
                    }
                    _ => {}
                }
            }

            self.spinner.tick();
        }
    }

    fn drain_msgs(&mut self) {
        while let Ok(msg) = self.rx.try_recv() {
            match msg {
                Msg::Loaded(kind, items) => {
                    self.views[kind.index()].set_loaded(items);
                }
                Msg::Error(kind, err) => {
                    let view = &mut self.views[kind.index()];
                    view.loading = false;
                    view.table.set_loading(false);
                    self.err = Some(err);
                }
            }
        }
        if self.views[self.active.index()].loading {
            self.spinner.start();
        } else {
            self.spinner.stop();
        }
    }

    fn fetch(&mut self, kind: ViewType) {
        self.err = None;
        let view = &mut self.views[kind.index()];
        view.loading = true;
        view.table.set_loading(true);
        let tab = view.tab;

        self.spinner
            .set_label(format!("Loading {}...", kind.label()));
        self.spinner.start();

        let client = self.client.clone();
        let tx = self.tx.clone();
        thread::spawn(move || {
            let result = match kind {
                ViewType::Subscribers => views::subscribers::fetch(&client),
                ViewType::Campaigns => views::campaigns::fetch(&client),
                ViewType::Automations => views::automations::fetch(&client),
                ViewType::Groups => views::groups::fetch(&client),
                ViewType::Forms => views::forms::fetch(&client, tab),
            };
            let _ = tx.send(match result {
                Ok(items) => Msg::Loaded(kind, items),
                Err(e) => Msg::Error(kind, e.to_string()),
            });
        });
    }

    /// Returns true when the app should quit.
    fn handle_key(&mut self, key: KeyEvent) -> bool {
        let code = key.code;

        if self.show_help {
            if matches!(code, KeyCode::Char('?') | KeyCode::Esc | KeyCode::Backspace) {
                self.show_help = false;
            }
            return false;
        }

        match code {
            KeyCode::Char('c') if key.modifiers.contains(KeyModifiers::CONTROL) => {
                return true;
            }
            KeyCode::Char('q') => return true,
            KeyCode::Char('?') => {
                self.show_help = true;
                return false;
            }
            KeyCode::Tab => {
                self.focus = match self.focus {
                    Focus::Sidebar => Focus::Content,
                    Focus::Content => Focus::Sidebar,
                };
                return false;
            }
            KeyCode::Char('1') => {
                self.switch_view(ViewType::Subscribers);
                return false;
            }
            KeyCode::Char('2') => {
                self.switch_view(ViewType::Campaigns);
                return false;
            }
            KeyCode::Char('3') => {
                self.switch_view(ViewType::Automations);
                return false;
            }
            KeyCode::Char('4') => {
                self.switch_view(ViewType::Groups);
                return false;
            }
            KeyCode::Char('5') => {
                self.switch_view(ViewType::Forms);
                return false;
            }
            _ => {}
        }

        match self.focus {
            Focus::Sidebar => self.handle_sidebar_key(code),
            Focus::Content => {
                if let KeyResult::Fetch = self.views[self.active.index()].handle_key(code) {
                    self.fetch(self.active);
                }
            }
        }

        false
    }

    fn handle_sidebar_key(&mut self, code: KeyCode) {
        match code {
            KeyCode::Char('j') | KeyCode::Down => {
                self.sidebar.next();
                self.switch_view(self.sidebar.active());
            }
            KeyCode::Char('k') | KeyCode::Up => {
                self.sidebar.prev();
                self.switch_view(self.sidebar.active());
            }
            KeyCode::Enter | KeyCode::Char('l') | KeyCode::Right => {
                self.focus = Focus::Content;
            }
            _ => {}
        }
    }

    fn switch_view(&mut self, kind: ViewType) {
        if self.active == kind {
            return;
        }
        self.active = kind;
        self.sidebar.set_active(kind);
        self.fetch(kind);
    }

    fn draw(&mut self, frame: &mut Frame) {
        let area = frame.area();
        if area.width == 0 || area.height == 0 {
            return;
        }

        let [header_area, content_area, status_area] = Layout::vertical([
            Constraint::Length(2),
            Constraint::Min(0),
            Constraint::Length(2),
        ])
        .areas(area);

        self.draw_header(frame, header_area);

        let [sidebar_area, main_area] = Layout::horizontal([
            Constraint::Length(super::components::sidebar::WIDTH),
            Constraint::Min(0),
        ])
        .areas(content_area);

        self.sidebar.set_focused(self.focus == Focus::Sidebar);
        self.sidebar.render(frame, sidebar_area);

        self.draw_main(frame, main_area);
        self.draw_status(frame, status_area);

        if self.show_help {
            help::render(frame, area);
        }
    }

    fn draw_header(&self, frame: &mut Frame, area: Rect) {
        let block = Block::new()
            .borders(Borders::BOTTOM)
            .border_style(Style::new().fg(theme::MUTED));
        let inner = block.inner(area);
        frame.render_widget(block, area);

        let title = " MailerLite Dashboard ";
        let profile = format!("profile: {}", self.profile);
        let gap = (inner.width as usize)
            .saturating_sub(title.chars().count() + profile.chars().count() + 4)
            .max(1);

        let line = Line::from(vec![
            Span::styled(
                title,
                Style::new().fg(theme::PRIMARY).add_modifier(Modifier::BOLD),
            ),
            Span::raw(" ".repeat(gap)),
            Span::styled(profile, Style::new().fg(theme::MUTED)),
        ]);
        frame.render_widget(Paragraph::new(line), inner);
    }

    fn draw_main(&mut self, frame: &mut Frame, area: Rect) {
        let mut inner = area;
        inner.x += 1;
        inner.width = inner.width.saturating_sub(2);

        if let Some(err) = &self.err {
            let [err_area, rest] =
                Layout::vertical([Constraint::Length(2), Constraint::Min(0)]).areas(inner);
            frame.render_widget(
                Paragraph::new(Line::styled(
                    format!("Error: {err}"),
                    Style::new().fg(theme::ERROR),
                )),
                err_area,
            );
            inner = rest;
        }

        let view = &mut self.views[self.active.index()];
        view.table
            .set_focused(self.focus == Focus::Content && !view.showing_detail);
        view.render(frame, inner);
    }

    fn draw_status(&mut self, frame: &mut Frame, area: Rect) {
        self.statusbar.set_profile(self.profile.clone());

        let view = &self.views[self.active.index()];
        if view.loading {
            self.statusbar.set_left(self.active.label());
            self.statusbar.set_loading(true, self.spinner.view());
        } else {
            self.statusbar
                .set_left(format!("{} ({})", self.active.label(), view.items.len()));
            self.statusbar.set_loading(false, "");
        }

        self.statusbar.render(frame, area);
    }
}
