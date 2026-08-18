pub mod detail;
pub mod help;
pub mod sidebar;
pub mod spinner;
pub mod statusbar;
pub mod table;

pub fn pad_right(s: &str, width: usize) -> String {
    let len = s.chars().count();
    if len >= width {
        return s.to_string();
    }
    let mut out = String::with_capacity(s.len() + width - len);
    out.push_str(s);
    for _ in len..width {
        out.push(' ');
    }
    out
}
