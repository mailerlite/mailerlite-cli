use std::collections::HashMap;
use std::time::{Duration, Instant};

use anyhow::{anyhow, bail, Context, Result};
use base64::engine::general_purpose::URL_SAFE_NO_PAD;
use base64::Engine as _;
use clap::{Args, Subcommand};
use rand::RngCore;
use serde_json::{json, Value};
use sha2::{Digest, Sha256};

use crate::api;
use crate::cli::Ctx;
use crate::config;
use crate::output;
use crate::prompt;
use crate::util::{jint, jstr};

#[derive(Args)]
pub struct Cmd {
    #[command(subcommand)]
    command: Sub,
}

#[derive(Subcommand)]
enum Sub {
    /// Log in to MailerLite
    #[command(long_about = "Authenticate via API token or OAuth browser flow.")]
    Login {
        /// auth method: token or oauth
        #[arg(long, default_value = "")]
        method: String,
        /// API token (for token method)
        #[arg(long, default_value = "")]
        token: String,
    },
    /// Log out and remove stored credentials
    Logout,
    /// Show current authentication status
    Status,
}

pub fn run(ctx: &Ctx, cmd: Cmd) -> Result<()> {
    match cmd.command {
        Sub::Login { method, token } => login(ctx, method, token),
        Sub::Logout => logout(ctx),
        Sub::Status => status(ctx),
    }
}

fn login(ctx: &Ctx, mut method: String, mut token: String) -> Result<()> {
    if method.is_empty() && prompt::is_interactive() {
        method = prompt::select_labeled(
            "Authentication method",
            vec![
                "OAuth (Recommended)".to_string(),
                "API Token (less secure)".to_string(),
            ],
            vec!["oauth".to_string(), "token".to_string()],
        )?;
    }
    if method.is_empty() {
        method = "oauth".to_string();
    }

    let prof_name = ctx
        .profile
        .clone()
        .filter(|s| !s.is_empty())
        .unwrap_or_else(|| "default".to_string());

    let mut cfg = config::load()?;

    match method.as_str() {
        "token" => {
            if token.is_empty() {
                if !prompt::is_interactive() {
                    bail!("--token is required in non-interactive mode");
                }
                token = prompt::input("API Token", "")?;
            }
            if token.is_empty() {
                bail!("token cannot be empty");
            }
            cfg.profiles.insert(
                prof_name.clone(),
                config::Profile {
                    api_token: Some(token),
                    ..Default::default()
                },
            );
        }
        "oauth" => {
            let prof = oauth_browser_flow().map_err(|e| anyhow!("OAuth login failed: {e:#}"))?;
            cfg.profiles.insert(prof_name.clone(), prof);
        }
        other => bail!("unknown auth method: {other} (use 'token' or 'oauth')"),
    }

    cfg.active_profile = prof_name.clone();
    config::save(&cfg)?;

    output::success(&format!("Logged in successfully. Profile: {prof_name}"));

    if method == "oauth" {
        if let Err(e) = select_account(&mut cfg, &prof_name) {
            println!("Warning: could not set account: {e:#}");
        }
    }

    Ok(())
}

fn logout(ctx: &Ctx) -> Result<()> {
    let mut cfg = config::load()?;

    let name = ctx
        .profile
        .clone()
        .filter(|s| !s.is_empty())
        .unwrap_or_else(|| {
            if cfg.active_profile.is_empty() {
                "default".to_string()
            } else {
                cfg.active_profile.clone()
            }
        });

    if !cfg.profiles.contains_key(&name) {
        bail!("profile {name:?} not found");
    }

    cfg.profiles.remove(&name);
    if cfg.active_profile == name {
        cfg.active_profile = cfg.profiles.keys().next().cloned().unwrap_or_default();
    }

    config::save(&cfg)?;

    output::success(&format!("Logged out from profile: {name}"));
    Ok(())
}

fn status(ctx: &Ctx) -> Result<()> {
    let cfg = config::load()?;

    let name = match ctx.profile.clone().filter(|s| !s.is_empty()) {
        Some(n) => n,
        None => match config::active_profile(&cfg) {
            Ok((n, _)) => n,
            Err(_) => {
                output::error("Not logged in. Run 'mailerlite auth login' to authenticate.");
                return Ok(());
            }
        },
    };

    let Some(prof) = cfg.profiles.get(&name) else {
        output::error(&format!("Profile {name:?} not found."));
        return Ok(());
    };

    let has_token = prof.api_token.as_deref().is_some_and(|t| !t.is_empty());
    let has_oauth = prof.oauth_token.as_deref().is_some_and(|t| !t.is_empty());

    if ctx.json {
        return output::json(&json!({
            "profile": name,
            "has_token": has_token,
            "has_oauth": has_oauth,
            "expires_at": prof.oauth_expires_at.clone().unwrap_or_default(),
            "account_id": prof.account_id.clone().unwrap_or_default(),
        }));
    }

    let method = if has_oauth { "OAuth" } else { "API Token" };
    let masked_token = match prof.api_token.as_deref() {
        None | Some("") => "none".to_string(),
        Some(t) if t.len() > 10 => match (t.get(..7), t.get(t.len() - 4..)) {
            (Some(head), Some(tail)) => format!("{head}...{tail}"),
            _ => "***".to_string(),
        },
        Some(_) => "***".to_string(),
    };

    let mut rows = vec![
        vec!["Profile".to_string(), name.clone()],
        vec!["Method".to_string(), method.to_string()],
        vec!["Token".to_string(), masked_token],
        vec!["Active".to_string(), "Yes".to_string()],
    ];

    if let Some(expires) = prof.oauth_expires_at.as_deref().filter(|s| !s.is_empty()) {
        rows.push(vec!["Expires".to_string(), expires.to_string()]);
    }
    if let Some(account_id) = prof.account_id.as_deref().filter(|s| !s.is_empty()) {
        rows.push(vec!["Account ID".to_string(), account_id.to_string()]);
    }

    output::table(&["Field", "Value"], &rows);
    Ok(())
}

/// Full OAuth 2.0 Authorization Code flow with PKCE: starts a local HTTP
/// server, opens the browser to the authorize URL, captures the authorization
/// code, and exchanges it for access/refresh tokens.
fn oauth_browser_flow() -> Result<config::Profile> {
    let state = random_hex(16);
    let (verifier, challenge) = generate_pkce();

    const CALLBACK_PORT: &str = "19821";
    let callback_url = format!("http://127.0.0.1:{CALLBACK_PORT}/callback");

    let server = tiny_http::Server::http(format!("127.0.0.1:{CALLBACK_PORT}"))
        .map_err(|e| anyhow!("failed to start local server on port {CALLBACK_PORT}: {e}"))?;

    let auth_url = format!(
        "{}?client_id={}&redirect_uri={}&response_type=code&scope={}&state={}&code_challenge={}&code_challenge_method=S256",
        config::OAUTH_AUTHORIZE_URL,
        config::OAUTH_CLIENT_ID,
        query_escape(&callback_url),
        query_escape(config::OAUTH_SCOPES),
        state,
        challenge,
    );

    println!("Opening browser for authentication...");
    println!("If the browser doesn't open, visit:\n{auth_url}\n");
    let _ = webbrowser::open(&auth_url);

    let html_header = || {
        tiny_http::Header::from_bytes(&b"Content-Type"[..], &b"text/html; charset=utf-8"[..])
            .expect("static header")
    };

    let deadline = Instant::now() + Duration::from_secs(5 * 60);
    let code = loop {
        let Some(remaining) = deadline
            .checked_duration_since(Instant::now())
            .filter(|d| !d.is_zero())
        else {
            bail!("authentication timed out after 5 minutes");
        };
        let Some(request) = server.recv_timeout(remaining)? else {
            bail!("authentication timed out after 5 minutes");
        };

        let url = request.url().to_string();
        let (path, query) = url.split_once('?').unwrap_or((url.as_str(), ""));
        if path != "/callback" {
            let _ = request.respond(
                tiny_http::Response::from_string("404 page not found\n").with_status_code(404),
            );
            continue;
        }

        let params = parse_query(query);
        if params.get("state").map(String::as_str) != Some(state.as_str()) {
            let _ = request.respond(
                tiny_http::Response::from_string("State mismatch\n").with_status_code(400),
            );
            bail!("state mismatch");
        }

        match params.get("code").filter(|c| !c.is_empty()) {
            Some(code) => {
                let code = code.clone();
                let _ = request.respond(
                    tiny_http::Response::from_string(
                        "<html><body><h2>Authentication successful!</h2><p>You can close this window.</p></body></html>",
                    )
                    .with_header(html_header()),
                );
                break code;
            }
            None => {
                let err_msg = params.get("error").cloned().unwrap_or_default();
                let _ = request.respond(
                    tiny_http::Response::from_string(format!(
                        "<html><body><h2>Authentication failed</h2><p>{err_msg}</p></body></html>"
                    ))
                    .with_header(html_header()),
                );
                bail!("OAuth error: {err_msg}");
            }
        }
    };

    exchange_code_for_tokens(&code, &callback_url, &verifier)
}

/// POSTs to the token endpoint with the authorization code and PKCE verifier
/// to obtain access and refresh tokens.
fn exchange_code_for_tokens(
    code: &str,
    redirect_uri: &str,
    code_verifier: &str,
) -> Result<config::Profile> {
    let resp = ureq::post(config::OAUTH_TOKEN_URL).send_form(&[
        ("grant_type", "authorization_code"),
        ("client_id", config::OAUTH_CLIENT_ID),
        ("redirect_uri", redirect_uri),
        ("code", code),
        ("code_verifier", code_verifier),
    ]);

    let resp = match resp {
        Ok(r) => r,
        Err(ureq::Error::Status(status, r)) => {
            let body: Value = r.into_json().unwrap_or(Value::Null);
            bail!("token exchange failed (HTTP {status}): {body}");
        }
        Err(e) => bail!("token exchange request failed: {e}"),
    };

    let tok: Value = resp.into_json().context("failed to parse token response")?;

    let access_token = jstr(&tok, "access_token");
    if access_token.is_empty() {
        bail!("server returned empty access token");
    }

    let expires_at = (chrono::Utc::now() + chrono::Duration::seconds(jint(&tok, "expires_in")))
        .to_rfc3339_opts(chrono::SecondsFormat::Secs, true);

    Ok(config::Profile {
        oauth_token: Some(access_token),
        oauth_refresh_token: Some(jstr(&tok, "refresh_token")).filter(|s| !s.is_empty()),
        oauth_expires_at: Some(expires_at),
        ..Default::default()
    })
}

/// Creates a PKCE code verifier and its S256 challenge.
fn generate_pkce() -> (String, String) {
    let mut buf = [0u8; 32];
    rand::thread_rng().fill_bytes(&mut buf);
    let verifier = URL_SAFE_NO_PAD.encode(buf);
    let challenge = URL_SAFE_NO_PAD.encode(Sha256::digest(verifier.as_bytes()));
    (verifier, challenge)
}

fn random_hex(n: usize) -> String {
    let mut buf = vec![0u8; n];
    rand::thread_rng().fill_bytes(&mut buf);
    buf.iter().map(|b| format!("{b:02x}")).collect()
}

fn query_escape(s: &str) -> String {
    let mut out = String::with_capacity(s.len());
    for b in s.bytes() {
        match b {
            b'A'..=b'Z' | b'a'..=b'z' | b'0'..=b'9' | b'-' | b'_' | b'.' | b'~' => {
                out.push(b as char)
            }
            b' ' => out.push('+'),
            _ => out.push_str(&format!("%{b:02X}")),
        }
    }
    out
}

fn query_unescape(s: &str) -> String {
    let bytes = s.as_bytes();
    let mut out = Vec::with_capacity(bytes.len());
    let mut i = 0;
    while i < bytes.len() {
        match bytes[i] {
            b'+' => {
                out.push(b' ');
                i += 1;
            }
            b'%' if i + 3 <= bytes.len() => {
                match std::str::from_utf8(&bytes[i + 1..i + 3])
                    .ok()
                    .and_then(|h| u8::from_str_radix(h, 16).ok())
                {
                    Some(v) => {
                        out.push(v);
                        i += 3;
                    }
                    None => {
                        out.push(b'%');
                        i += 1;
                    }
                }
            }
            b => {
                out.push(b);
                i += 1;
            }
        }
    }
    String::from_utf8_lossy(&out).into_owned()
}

fn parse_query(query: &str) -> HashMap<String, String> {
    query
        .split('&')
        .filter(|p| !p.is_empty())
        .map(|p| {
            let (k, v) = p.split_once('=').unwrap_or((p, ""));
            (query_unescape(k), query_unescape(v))
        })
        .collect()
}

/// Fetches the user's accounts and prompts to select one. A single account is
/// selected automatically.
fn select_account(cfg: &mut config::Config, prof_name: &str) -> Result<()> {
    let Some(prof) = cfg.profiles.get(prof_name).cloned() else {
        return Ok(());
    };
    let token = match prof.oauth_token.as_deref() {
        Some(t) if !t.is_empty() => t.to_string(),
        _ => return Ok(()),
    };

    let resp = ureq::get(&format!("{}/accounts", api::DEFAULT_BASE_URL))
        .set("Authorization", &format!("Bearer {token}"))
        .set("Accept", "application/json")
        .call()
        .map_err(|e| match e {
            ureq::Error::Status(status, _) => anyhow!("failed to fetch accounts (HTTP {status})"),
            other => anyhow!("failed to fetch accounts: {other}"),
        })?;

    let body: Value = resp.into_json().context("failed to parse accounts")?;
    let accounts = body
        .get("data")
        .and_then(Value::as_array)
        .cloned()
        .unwrap_or_default();
    if accounts.is_empty() {
        return Ok(());
    }

    let account_id = if accounts.len() == 1 {
        let id = jstr(&accounts[0], "id");
        output::success(&format!(
            "Account selected: {} ({})",
            jstr(&accounts[0], "name"),
            id
        ));
        id
    } else if !prompt::is_interactive() {
        jstr(&accounts[0], "id")
    } else {
        let labels = accounts
            .iter()
            .map(|a| format!("{} ({})", jstr(a, "name"), jstr(a, "id")))
            .collect();
        let values = accounts.iter().map(|a| jstr(a, "id")).collect();
        prompt::select_labeled("Select account", labels, values)?
    };

    let mut prof = prof;
    prof.account_id = Some(account_id);
    cfg.profiles.insert(prof_name.to_string(), prof);
    config::save(cfg)
}
