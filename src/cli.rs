use anyhow::Result;
use clap::{CommandFactory, Parser, Subcommand};
use clap_complete::Shell;

use crate::api::Client;
use crate::commands;
use crate::config;

// cargo-dist requires the git tag to match Cargo.toml's version, so the
// crate version is authoritative. Commit and date are stamped by build.rs.
pub const VERSION: &str = env!("CARGO_PKG_VERSION");
const COMMIT: &str = match option_env!("MAILERLITE_COMMIT") {
    Some(c) => c,
    None => "none",
};
const BUILD_DATE: &str = match option_env!("MAILERLITE_BUILD_DATE") {
    Some(d) => d,
    None => "unknown",
};

/// Global flags shared by every command.
pub struct Ctx {
    pub profile: Option<String>,
    pub verbose: bool,
    pub json: bool,
    pub yes: bool,
}

impl Ctx {
    /// Builds an authenticated API client for the selected profile.
    pub fn client(&self) -> Result<Client> {
        let token = config::get_token(self.profile.as_deref())?;
        let account_id = config::get_account_id(self.profile.as_deref());
        Ok(Client::new(token, account_id, self.verbose))
    }

    /// Confirmation gate: true when --yes was passed or stdin is not a
    /// terminal (parity with the Go CLI); otherwise prompts interactively.
    pub fn confirm(&self, message: &str) -> Result<bool> {
        if self.yes || !crate::prompt::is_interactive() {
            return Ok(true);
        }
        crate::prompt::confirm(message)
    }
}

#[derive(Parser)]
#[command(
    name = "mailerlite",
    version = VERSION,
    about = "MailerLite CLI — manage your email marketing from the terminal",
    long_about = "A command-line interface for the MailerLite API. Manage subscribers, campaigns, automations, groups, forms, and more."
)]
struct Root {
    /// config profile to use
    #[arg(long, global = true)]
    profile: Option<String>,

    /// show HTTP request/response details
    #[arg(long, short = 'v', global = true)]
    verbose: bool,

    /// output as JSON
    #[arg(long, global = true)]
    json: bool,

    /// skip confirmation prompts
    #[arg(long, short = 'y', global = true)]
    yes: bool,

    #[command(subcommand)]
    command: Option<Command>,
}

#[derive(Subcommand)]
enum Command {
    /// Launch the interactive TUI dashboard
    Dashboard,
    /// Manage subscribers
    Subscriber(commands::subscriber::Cmd),
    /// Manage groups
    Group(commands::group::Cmd),
    /// Manage campaigns
    Campaign(commands::campaign::Cmd),
    /// Manage automations
    Automation(commands::automation::Cmd),
    /// Manage forms
    Form(commands::form::Cmd),
    /// Manage subscriber fields
    Field(commands::field::Cmd),
    /// Manage segments
    Segment(commands::segment::Cmd),
    /// Manage webhooks
    Webhook(commands::webhook::Cmd),
    /// List available timezones
    Timezone(commands::timezone::Cmd),
    /// Manage e-commerce shops
    Shop(commands::shop::Cmd),
    /// Manage e-commerce products
    Product(commands::product::Cmd),
    /// Manage e-commerce categories
    Category(commands::category::Cmd),
    /// Manage e-commerce customers
    Customer(commands::customer::Cmd),
    /// Manage e-commerce orders
    Order(commands::order::Cmd),
    /// Manage e-commerce carts
    Cart(commands::cart::Cmd),
    /// Manage e-commerce cart items
    #[command(name = "cart-item")]
    Cartitem(commands::cartitem::Cmd),
    /// Import e-commerce resources
    #[command(name = "import")]
    Import(commands::import_cmd::Cmd),
    /// Manage accounts
    Account(commands::account::Cmd),
    /// Authenticate with MailerLite
    Auth(commands::auth::Cmd),
    /// Manage config profiles
    Profile(commands::profile::Cmd),
    /// Generate shell completion scripts
    Completion {
        /// shell to generate completions for
        #[arg(value_enum)]
        shell: Shell,
    },
    /// Print the version of mailerlite
    Version,
}

pub fn run() -> i32 {
    let root = Root::parse();
    let ctx = Ctx {
        profile: root.profile,
        verbose: root.verbose,
        json: root.json,
        yes: root.yes,
    };

    let Some(command) = root.command else {
        Root::command().print_help().ok();
        return 0;
    };

    let result = match command {
        Command::Dashboard => crate::tui::run(&ctx),
        Command::Subscriber(cmd) => commands::subscriber::run(&ctx, cmd),
        Command::Group(cmd) => commands::group::run(&ctx, cmd),
        Command::Campaign(cmd) => commands::campaign::run(&ctx, cmd),
        Command::Automation(cmd) => commands::automation::run(&ctx, cmd),
        Command::Form(cmd) => commands::form::run(&ctx, cmd),
        Command::Field(cmd) => commands::field::run(&ctx, cmd),
        Command::Segment(cmd) => commands::segment::run(&ctx, cmd),
        Command::Webhook(cmd) => commands::webhook::run(&ctx, cmd),
        Command::Timezone(cmd) => commands::timezone::run(&ctx, cmd),
        Command::Shop(cmd) => commands::shop::run(&ctx, cmd),
        Command::Product(cmd) => commands::product::run(&ctx, cmd),
        Command::Category(cmd) => commands::category::run(&ctx, cmd),
        Command::Customer(cmd) => commands::customer::run(&ctx, cmd),
        Command::Order(cmd) => commands::order::run(&ctx, cmd),
        Command::Cart(cmd) => commands::cart::run(&ctx, cmd),
        Command::Cartitem(cmd) => commands::cartitem::run(&ctx, cmd),
        Command::Import(cmd) => commands::import_cmd::run(&ctx, cmd),
        Command::Account(cmd) => commands::account::run(&ctx, cmd),
        Command::Auth(cmd) => commands::auth::run(&ctx, cmd),
        Command::Profile(cmd) => commands::profile::run(&ctx, cmd),
        Command::Completion { shell } => {
            let mut cmd = Root::command();
            clap_complete::generate(shell, &mut cmd, "mailerlite", &mut std::io::stdout());
            Ok(())
        }
        Command::Version => {
            println!("mailerlite v{VERSION} ({COMMIT}) built {BUILD_DATE}");
            Ok(())
        }
    };

    match result {
        Ok(()) => 0,
        Err(err) => {
            // In --json mode, surface the raw API error body when available.
            if ctx.json {
                if let Some(api_err) = err.downcast_ref::<crate::api::ApiError>() {
                    if let Some(raw) = &api_err.raw_body {
                        let _ = crate::output::json(raw);
                        return 1;
                    }
                }
            }
            crate::output::error(&format!("{err}"));
            1
        }
    }
}
