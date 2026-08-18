mod app;
mod components;
mod keys;
mod theme;
mod views;

use std::io;

use anyhow::Result;
use crossterm::execute;
use crossterm::terminal::{
    disable_raw_mode, enable_raw_mode, EnterAlternateScreen, LeaveAlternateScreen,
};
use ratatui::backend::CrosstermBackend;
use ratatui::Terminal;

use crate::cli::Ctx;

pub fn run(ctx: &Ctx) -> Result<()> {
    let client = ctx.client()?;
    let profile = ctx.profile.clone().unwrap_or_else(|| "default".to_string());

    enable_raw_mode()?;
    execute!(io::stdout(), EnterAlternateScreen)?;

    let original_hook = std::panic::take_hook();
    std::panic::set_hook(Box::new(move |info| {
        restore_terminal();
        original_hook(info);
    }));

    let mut terminal = Terminal::new(CrosstermBackend::new(io::stdout()))?;
    let mut app = app::App::new(client, profile);
    let result = app.run(&mut terminal);

    restore_terminal();
    result
}

fn restore_terminal() {
    let _ = disable_raw_mode();
    let _ = execute!(io::stdout(), LeaveAlternateScreen, crossterm::cursor::Show);
}
