use std::io::IsTerminal;

use anyhow::{bail, Result};

pub fn is_interactive() -> bool {
    std::io::stdin().is_terminal()
}

pub fn input(label: &str, placeholder: &str) -> Result<String> {
    let mut p = inquire::Text::new(label);
    if !placeholder.is_empty() {
        p = p.with_placeholder(placeholder);
    }
    Ok(p.prompt()?.trim().to_string())
}

pub fn confirm(label: &str) -> Result<bool> {
    Ok(inquire::Confirm::new(label).with_default(false).prompt()?)
}

/// Presents `labels` for selection and returns the matching entry of `values`.
pub fn select_labeled(label: &str, labels: Vec<String>, values: Vec<String>) -> Result<String> {
    debug_assert_eq!(labels.len(), values.len());
    let chosen = inquire::Select::new(label, labels.clone()).prompt()?;
    let idx = labels.iter().position(|l| *l == chosen).unwrap_or(0);
    Ok(values[idx].clone())
}

/// Returns `value` if non-empty; otherwise prompts interactively or fails
/// with a "--flag is required" error in non-interactive mode.
pub fn require_arg(value: &str, flag: &str, label: &str) -> Result<String> {
    if !value.is_empty() {
        return Ok(value.to_string());
    }
    if !is_interactive() {
        bail!("--{flag} is required");
    }
    input(label, "")
}

/// Like [`require_arg`] for repeated values; prompts for a comma-separated list.
pub fn require_slice_arg(values: &[String], flag: &str, label: &str) -> Result<Vec<String>> {
    if !values.is_empty() {
        return Ok(values.to_vec());
    }
    if !is_interactive() {
        bail!("--{flag} is required");
    }
    let raw = input(&format!("{label} (comma-separated)"), "")?;
    let result: Vec<String> = raw
        .split(',')
        .map(str::trim)
        .filter(|s| !s.is_empty())
        .map(str::to_string)
        .collect();
    if result.is_empty() {
        bail!("--{flag} is required");
    }
    Ok(result)
}
