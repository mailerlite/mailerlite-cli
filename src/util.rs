use anyhow::{bail, Result};
use serde_json::{Map, Value};

/// Parses repeated `key=value` pairs into a JSON object.
pub fn parse_kv_fields(pairs: &[String]) -> Result<Map<String, Value>> {
    let mut fields = Map::new();
    for pair in pairs {
        match pair.split_once('=') {
            Some((k, v)) if !k.is_empty() => {
                fields.insert(k.trim().to_string(), Value::String(v.trim().to_string()));
            }
            _ => bail!("invalid field {pair:?}: use key=value format"),
        }
    }
    Ok(fields)
}

/// Extracts a string field from a JSON object ("" when absent). Numbers are
/// rendered with `to_string` so numeric IDs display cleanly.
pub fn jstr(v: &Value, key: &str) -> String {
    match v.get(key) {
        Some(Value::String(s)) => s.clone(),
        Some(Value::Number(n)) => n.to_string(),
        Some(Value::Bool(b)) => b.to_string(),
        _ => String::new(),
    }
}

/// Extracts a nested string via a dotted path (e.g. "stats.sent").
pub fn jpath(v: &Value, path: &str) -> String {
    let mut cur = v;
    for part in path.split('.') {
        match cur.get(part) {
            Some(next) => cur = next,
            None => return String::new(),
        }
    }
    match cur {
        Value::String(s) => s.clone(),
        Value::Number(n) => n.to_string(),
        Value::Bool(b) => b.to_string(),
        Value::Null => String::new(),
        other => other.to_string(),
    }
}

/// Extracts an integer field (0 when absent).
pub fn jint(v: &Value, key: &str) -> i64 {
    v.get(key).and_then(Value::as_i64).unwrap_or(0)
}

/// Extracts a float field (0.0 when absent).
pub fn jfloat(v: &Value, key: &str) -> f64 {
    v.get(key).and_then(Value::as_f64).unwrap_or(0.0)
}
