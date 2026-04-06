use std::collections::HashMap;
use regex::Regex;

/// A named variable sweeping linearly from start to end.
pub struct VarRange {
    pub name: String,
    pub start: f64,
    pub end: f64,
}

/// Replace $name and $name:default tokens in raw YAML text.
/// Provided vars override defaults. Unresolved vars without defaults are left as-is.
pub fn substitute_vars(text: &str, vars: &HashMap<String, f64>) -> String {
    let re = Regex::new(r"\$([a-zA-Z_]\w*)(?::([+-]?(?:\d+\.?\d*|\.\d+)(?:[eE][+-]?\d+)?))?")
        .unwrap();
    re.replace_all(text, |caps: &regex::Captures| {
        let name = &caps[1];
        if let Some(&val) = vars.get(name) {
            format!("{:.10}", val).trim_end_matches('0').trim_end_matches('.').to_string()
        } else if let Some(default) = caps.get(2) {
            default.as_str().to_string()
        } else {
            caps[0].to_string()
        }
    }).to_string()
}

/// Parse "name:start:end" into a VarRange.
pub fn parse_range(s: &str) -> Result<VarRange, String> {
    let parts: Vec<&str> = s.splitn(3, ':').collect();
    if parts.len() != 3 {
        return Err(format!("invalid range '{}': expected name:start:end", s));
    }
    let start: f64 = parts[1].parse().map_err(|_| format!("invalid range '{}': bad start", s))?;
    let end: f64 = parts[2].parse().map_err(|_| format!("invalid range '{}': bad end", s))?;
    Ok(VarRange { name: parts[0].to_string(), start, end })
}

/// Parse "name=value" into (name, value).
pub fn parse_var(s: &str) -> Result<(String, f64), String> {
    let parts: Vec<&str> = s.splitn(2, '=').collect();
    if parts.len() != 2 {
        return Err(format!("invalid var '{}': expected name=value", s));
    }
    let val: f64 = parts[1].parse().map_err(|_| format!("invalid var '{}': bad value", s))?;
    Ok((parts[0].to_string(), val))
}

/// Compute variable values at parameter t ∈ [0, 1].
pub fn interpolate_vars(ranges: &[VarRange], t: f64) -> HashMap<String, f64> {
    ranges.iter().map(|r| {
        (r.name.clone(), r.start + t * (r.end - r.start))
    }).collect()
}
