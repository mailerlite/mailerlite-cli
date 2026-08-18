use serde::Serialize;
use unicode_width::UnicodeWidthStr;

fn no_color() -> bool {
    std::env::var_os("NO_COLOR").is_some_and(|v| !v.is_empty())
}

fn paint(code: &str, text: &str) -> String {
    if no_color() {
        text.to_string()
    } else {
        format!("\x1b[{code}m{text}\x1b[0m")
    }
}

/// Bold bright blue — table headers.
pub fn header(text: &str) -> String {
    paint("1;94", text)
}

/// Bright green — success messages (stdout).
pub fn success(msg: &str) {
    println!("{}", paint("92", msg));
}

/// Bright red — error messages (stderr).
pub fn error(msg: &str) {
    eprintln!("{}", paint("91", msg));
}

/// Bright black (gray) — dim text.
pub fn dim(text: &str) -> String {
    paint("90", text)
}

/// Pretty-prints a value as 2-space-indented JSON to stdout.
pub fn json<T: Serialize>(v: &T) -> anyhow::Result<()> {
    println!("{}", serde_json::to_string_pretty(v)?);
    Ok(())
}

/// Renders a bordered table (plain aligned columns when NO_COLOR is set).
/// Prints "No results found." when rows is empty.
pub fn table(headers: &[&str], rows: &[Vec<String>]) {
    if rows.is_empty() {
        println!("{}", dim("No results found."));
        return;
    }
    if no_color() {
        plain_table(headers, rows);
        return;
    }

    let widths = column_widths(headers, rows);
    let border = |s: &str| dim(s);

    let rule = |left: &str, mid: &str, right: &str| {
        let mut line = String::from(left);
        for (i, w) in widths.iter().enumerate() {
            if i > 0 {
                line.push_str(mid);
            }
            line.push_str(&"─".repeat(w + 2));
        }
        line.push_str(right);
        border(&line)
    };

    let sep = border("│");
    let render_row = |cells: &[String], style: fn(&str) -> String| {
        let mut line = String::new();
        line.push_str(&sep);
        for (i, w) in widths.iter().enumerate() {
            let cell = cells.get(i).map(String::as_str).unwrap_or("");
            let pad = w - cell.width();
            line.push(' ');
            line.push_str(&style(cell));
            line.push_str(&" ".repeat(pad + 1));
            line.push_str(&sep);
        }
        line
    };

    println!("{}", rule("┌", "┬", "┐"));
    let hdr: Vec<String> = headers.iter().map(|h| h.to_string()).collect();
    println!("{}", render_row(&hdr, header));
    println!("{}", rule("├", "┼", "┤"));
    for row in rows {
        println!("{}", render_row(row, |s| s.to_string()));
    }
    println!("{}", rule("└", "┴", "┘"));
}

fn plain_table(headers: &[&str], rows: &[Vec<String>]) {
    let widths = column_widths(headers, rows);
    let mut line = String::new();
    for (i, h) in headers.iter().enumerate() {
        let h = h.to_uppercase();
        line.push_str(&h);
        line.push_str(&" ".repeat(widths[i] + 2 - h.width()));
    }
    println!("{}", line.trim_end());
    for row in rows {
        let mut line = String::new();
        for (i, w) in widths.iter().enumerate() {
            let cell = row.get(i).map(String::as_str).unwrap_or("");
            line.push_str(cell);
            line.push_str(&" ".repeat(w + 2 - cell.width()));
        }
        println!("{}", line.trim_end());
    }
}

fn column_widths(headers: &[&str], rows: &[Vec<String>]) -> Vec<usize> {
    let mut widths: Vec<usize> = headers.iter().map(|h| h.width()).collect();
    for row in rows {
        for (i, cell) in row.iter().enumerate() {
            if i < widths.len() && cell.width() > widths[i] {
                widths[i] = cell.width();
            }
        }
    }
    widths
}

/// Truncates to at most `max` characters, appending "..." when trimmed.
pub fn truncate(s: &str, max: usize) -> String {
    let count = s.chars().count();
    if count <= max {
        return s.to_string();
    }
    if max <= 3 {
        return s.chars().take(max).collect();
    }
    let mut out: String = s.chars().take(max - 3).collect();
    out.push_str("...");
    out
}
