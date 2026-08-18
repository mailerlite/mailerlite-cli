use std::collections::BTreeMap;
use std::path::PathBuf;
use std::time::Duration;

use anyhow::{anyhow, bail, Context, Result};
use serde::{Deserialize, Serialize};

pub const OAUTH_CLIENT_ID: &str = "1118";
pub const OAUTH_AUTHORIZE_URL: &str = "https://dashboard.mailerlite.com/oauth/authorize";
pub const OAUTH_TOKEN_URL: &str = "https://dashboard.mailerlite.com/oauth/token";
pub const OAUTH_SCOPES: &str = "manager-app";

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct Profile {
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub api_token: Option<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub oauth_token: Option<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub oauth_refresh_token: Option<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub oauth_expires_at: Option<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub account_id: Option<String>,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct Config {
    #[serde(default)]
    pub active_profile: String,
    #[serde(default)]
    pub profiles: BTreeMap<String, Profile>,
}

pub fn dir() -> Result<PathBuf> {
    if let Ok(xdg) = std::env::var("XDG_CONFIG_HOME") {
        if !xdg.is_empty() {
            return Ok(PathBuf::from(xdg).join("mailerlite"));
        }
    }
    let home = std::env::var_os("HOME")
        .map(PathBuf::from)
        .ok_or_else(|| anyhow!("could not determine home directory"))?;
    Ok(home.join(".config").join("mailerlite"))
}

pub fn path() -> Result<PathBuf> {
    Ok(dir()?.join("config.yaml"))
}

pub fn load() -> Result<Config> {
    let p = path()?;
    let data = match std::fs::read_to_string(&p) {
        Ok(d) => d,
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => return Ok(Config::default()),
        Err(e) => return Err(e).context("failed to read config"),
    };
    let cfg: Config = serde_yaml::from_str(&data).context("failed to parse config")?;
    Ok(cfg)
}

pub fn save(cfg: &Config) -> Result<()> {
    let p = path()?;
    let parent = p.parent().expect("config path has parent");
    std::fs::create_dir_all(parent).context("failed to create config directory")?;
    #[cfg(unix)]
    {
        use std::os::unix::fs::PermissionsExt;
        let _ = std::fs::set_permissions(parent, std::fs::Permissions::from_mode(0o700));
    }
    let data = serde_yaml::to_string(cfg).context("failed to marshal config")?;
    std::fs::write(&p, data).context("failed to write config")?;
    #[cfg(unix)]
    {
        use std::os::unix::fs::PermissionsExt;
        let _ = std::fs::set_permissions(&p, std::fs::Permissions::from_mode(0o600));
    }
    Ok(())
}

/// Returns the active (or first) profile name and profile.
pub fn active_profile(cfg: &Config) -> Result<(String, Profile)> {
    let name = if cfg.active_profile.is_empty() {
        cfg.profiles
            .keys()
            .next()
            .cloned()
            .ok_or_else(|| anyhow!("no profiles configured — run 'mailerlite auth login' or 'mailerlite profile add <name>'"))?
    } else {
        cfg.active_profile.clone()
    };
    let prof = cfg
        .profiles
        .get(&name)
        .cloned()
        .ok_or_else(|| anyhow!("profile {name:?} not found"))?;
    Ok((name, prof))
}

fn resolve_profile(cfg: &Config, profile_override: Option<&str>) -> Result<(String, Profile)> {
    match profile_override {
        Some(name) if !name.is_empty() => {
            let prof = cfg
                .profiles
                .get(name)
                .cloned()
                .ok_or_else(|| anyhow!("profile {name:?} not found"))?;
            Ok((name.to_string(), prof))
        }
        _ => active_profile(cfg),
    }
}

/// Token resolution: env var MAILERLITE_API_TOKEN > profile override > active
/// profile. OAuth tokens close to expiry are refreshed and persisted.
pub fn get_token(profile_override: Option<&str>) -> Result<String> {
    if let Ok(token) = std::env::var("MAILERLITE_API_TOKEN") {
        if !token.is_empty() {
            return Ok(token);
        }
    }

    let mut cfg = load()?;
    let (name, prof) = resolve_profile(&cfg, profile_override)?;

    if let Some(token) = prof.api_token.as_deref() {
        if !token.is_empty() {
            return Ok(token.to_string());
        }
    }
    if let Some(oauth_token) = prof.oauth_token.clone() {
        if let (Some(expires_at), Some(refresh_token)) = (
            prof.oauth_expires_at.as_deref(),
            prof.oauth_refresh_token.clone(),
        ) {
            if let Ok(expires_at) = chrono::DateTime::parse_from_rfc3339(expires_at) {
                let now = chrono::Utc::now();
                if now > expires_at.with_timezone(&chrono::Utc) - chrono::Duration::minutes(5) {
                    match refresh_oauth_token(&refresh_token) {
                        Ok(mut refreshed) => {
                            refreshed.account_id = prof.account_id.clone();
                            let token = refreshed.oauth_token.clone().unwrap_or_default();
                            cfg.profiles.insert(name, refreshed);
                            let _ = save(&cfg);
                            return Ok(token);
                        }
                        Err(refresh_err) => {
                            // If refresh fails but token isn't actually expired yet, use it anyway.
                            if now < expires_at.with_timezone(&chrono::Utc) {
                                return Ok(oauth_token);
                            }
                            bail!("OAuth token expired and refresh failed: {refresh_err}");
                        }
                    }
                }
            }
        }
        return Ok(oauth_token);
    }
    bail!("no token found — run 'mailerlite auth login' or set MAILERLITE_API_TOKEN")
}

/// Returns the stored account ID for the active (or overridden) profile.
pub fn get_account_id(profile_override: Option<&str>) -> Option<String> {
    let cfg = load().ok()?;
    let (_, prof) = resolve_profile(&cfg, profile_override).ok()?;
    prof.account_id.filter(|s| !s.is_empty())
}

/// Exchanges a refresh token for a new access token.
pub fn refresh_oauth_token(refresh_token: &str) -> Result<Profile> {
    let agent = ureq::AgentBuilder::new()
        .timeout(Duration::from_secs(30))
        .build();
    let resp = agent
        .post(OAUTH_TOKEN_URL)
        .send_form(&[
            ("grant_type", "refresh_token"),
            ("client_id", OAUTH_CLIENT_ID),
            ("refresh_token", refresh_token),
        ])
        .map_err(|e| match e {
            ureq::Error::Status(code, _) => anyhow!("refresh failed (HTTP {code})"),
            other => anyhow!("refresh request failed: {other}"),
        })?;

    #[derive(Deserialize)]
    struct TokenResponse {
        access_token: String,
        #[serde(default)]
        refresh_token: String,
        #[serde(default)]
        expires_in: i64,
    }
    let tok: TokenResponse = resp
        .into_json()
        .context("failed to parse refresh response")?;
    if tok.access_token.is_empty() {
        bail!("server returned empty access token on refresh");
    }

    let expires_at = (chrono::Utc::now() + chrono::Duration::seconds(tok.expires_in))
        .to_rfc3339_opts(chrono::SecondsFormat::Secs, true);

    Ok(Profile {
        oauth_token: Some(tok.access_token),
        oauth_refresh_token: Some(tok.refresh_token).filter(|s| !s.is_empty()),
        oauth_expires_at: Some(expires_at),
        ..Profile::default()
    })
}
