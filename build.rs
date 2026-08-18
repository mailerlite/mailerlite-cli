use std::process::Command;

fn git(args: &[&str]) -> Option<String> {
    let out = Command::new("git").args(args).output().ok()?;
    if !out.status.success() {
        return None;
    }
    let s = String::from_utf8(out.stdout).ok()?;
    let s = s.trim();
    if s.is_empty() {
        None
    } else {
        Some(s.to_string())
    }
}

fn main() {
    println!("cargo:rerun-if-changed=.git/HEAD");
    let commit = git(&["rev-parse", "--short", "HEAD"]).unwrap_or_else(|| "none".into());
    let date = git(&["log", "-1", "--format=%cd", "--date=format:%Y-%m-%d"])
        .unwrap_or_else(|| "unknown".into());
    println!("cargo:rustc-env=MAILERLITE_COMMIT={commit}");
    println!("cargo:rustc-env=MAILERLITE_BUILD_DATE={date}");
}
