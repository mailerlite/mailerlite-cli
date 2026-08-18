mod api;
mod cli;
mod commands;
mod config;
mod output;
mod prompt;
mod tui;
mod util;

fn main() {
    // Rust ignores SIGPIPE by default; restore the conventional CLI behavior
    // so piping into `head` etc. terminates silently instead of panicking.
    #[cfg(unix)]
    unsafe {
        libc::signal(libc::SIGPIPE, libc::SIG_DFL);
    }
    std::process::exit(cli::run());
}
