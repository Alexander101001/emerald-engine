use std::collections::HashMap;
use std::io::{self, BufRead, Write};

fn analyze_trends(data: &str) -> HashMap<String, f64> {
    let mut scores = HashMap::new();
    for line in data.lines() {
        let parts: Vec<&str> = line.split(',').collect();
        if parts.len() >= 3 {
            let keyword = parts[0].trim().to_lowercase();
            let mentions: f64 = parts[1].trim().parse().unwrap_or(0.0);
            let growth: f64 = parts[2].trim().parse().unwrap_or(0.0);
            let score = mentions * growth * 100.0;
            scores.insert(keyword, score);
        }
    }
    scores
}

fn main() {
    let stdin = io::stdin();
    let mut input = String::new();
    for line in stdin.lock().lines() {
        match line {
            Ok(l) => {
                if l.trim() == "EOF" { break; }
                input.push_str(&l);
                input.push('\n');
            }
            Err(e) => {
                eprintln!("rust: read error {}", e);
                break;
            }
        }
    }
    let results = analyze_trends(&input);
    let mut output = serde_json::to_string(&results).unwrap_or_else(|_| "{}".to_string());
    println!("{}", output);
    io::stdout().flush().unwrap();
}
