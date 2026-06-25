use aes_gcm::aead::{Aead, KeyInit, OsRng};
use aes_gcm::Aes256Gcm;
use aes_gcm::Nonce;
use hmac::{Hmac, Mac};
use pbkdf2::pbkdf2_hmac;
use rand::RngCore;
use serde::{Deserialize, Serialize};
use sha2::Sha256;
use std::io::{self, BufRead, Write};

type HmacSha256 = Hmac<Sha256>;

#[derive(Serialize, Deserialize)]
struct Request {
    id: u64,
    method: String,
    params: serde_json::Value,
}

#[derive(Serialize, Deserialize)]
struct Response {
    id: u64,
    result: Option<serde_json::Value>,
    error: Option<String>,
}

fn derive_key(password: &str, salt: &[u8]) -> Vec<u8> {
    let mut key = vec![0u8; 32];
    pbkdf2_hmac::<Sha256>(password.as_bytes(), salt, 100_000, &mut key);
    key
}

fn encrypt_gcm(plaintext: &[u8], key: &[u8]) -> Result<Vec<u8>, String> {
    let cipher = <Aes256Gcm as KeyInit>::new_from_slice(key).map_err(|e| format!("key: {}", e))?;
    let mut nonce_bytes = [0u8; 12];
    OsRng.fill_bytes(&mut nonce_bytes);
    let nonce = Nonce::from_slice(&nonce_bytes);
    let ciphertext = cipher
        .encrypt(nonce, plaintext)
        .map_err(|e| format!("encrypt: {}", e))?;
    let mut result = nonce_bytes.to_vec();
    result.extend_from_slice(&ciphertext);
    Ok(result)
}

fn decrypt_gcm(data: &[u8], key: &[u8]) -> Result<Vec<u8>, String> {
    if data.len() < 12 {
        return Err("data too short".into());
    }
    let (nonce_bytes, ciphertext) = data.split_at(12);
    let cipher = <Aes256Gcm as KeyInit>::new_from_slice(key).map_err(|e| format!("key: {}", e))?;
    let nonce = Nonce::from_slice(nonce_bytes);
    cipher
        .decrypt(nonce, ciphertext)
        .map_err(|e| format!("decrypt: {}", e))
}

fn hmac_sha256(key: &[u8], data: &[u8]) -> Vec<u8> {
    let mut mac = <HmacSha256 as Mac>::new_from_slice(key).expect("HMAC key");
    mac.update(data);
    mac.finalize().into_bytes().to_vec()
}

fn random_bytes(len: usize) -> Vec<u8> {
    let mut buf = vec![0u8; len];
    OsRng.fill_bytes(&mut buf);
    buf
}

fn handle_request(req: Request) -> Response {
    let result = match req.method.as_str() {
        "random_bytes" => {
            let len = req.params.get("len").and_then(|v| v.as_u64()).unwrap_or(32) as usize;
            let bytes = random_bytes(len);
            serde_json::json!({"hex": hex::encode(&bytes)})
        }
        "derive_key" => {
            let password = req.params.get("password").and_then(|v| v.as_str()).unwrap_or("");
            let salt_hex = req.params.get("salt").and_then(|v| v.as_str()).unwrap_or("");
            let salt = hex::decode(salt_hex).unwrap_or_default();
            let key = derive_key(password, &salt);
            serde_json::json!({"key": hex::encode(&key)})
        }
        "encrypt_gcm" => {
            let plaintext_hex = req.params.get("plaintext").and_then(|v| v.as_str()).unwrap_or("");
            let key_hex = req.params.get("key").and_then(|v| v.as_str()).unwrap_or("");
            let plaintext = hex::decode(plaintext_hex).unwrap_or_default();
            let key = hex::decode(key_hex).unwrap_or_default();
            match encrypt_gcm(&plaintext, &key) {
                Ok(ct) => serde_json::json!({"ciphertext": hex::encode(&ct)}),
                Err(e) => return Response { id: req.id, result: None, error: Some(e) },
            }
        }
        "decrypt_gcm" => {
            let ciphertext_hex = req.params.get("ciphertext").and_then(|v| v.as_str()).unwrap_or("");
            let key_hex = req.params.get("key").and_then(|v| v.as_str()).unwrap_or("");
            let data = hex::decode(ciphertext_hex).unwrap_or_default();
            let key = hex::decode(key_hex).unwrap_or_default();
            match decrypt_gcm(&data, &key) {
                Ok(pt) => serde_json::json!({"plaintext": hex::encode(&pt)}),
                Err(e) => return Response { id: req.id, result: None, error: Some(e) },
            }
        }
        "hmac_sha256" => {
            let key_hex = req.params.get("key").and_then(|v| v.as_str()).unwrap_or("");
            let data_hex = req.params.get("data").and_then(|v| v.as_str()).unwrap_or("");
            let key = hex::decode(key_hex).unwrap_or_default();
            let data = hex::decode(data_hex).unwrap_or_default();
            let mac = hmac_sha256(&key, &data);
            serde_json::json!({"mac": hex::encode(&mac)})
        }
        "ping" => serde_json::json!({"pong": true}),
        _ => return Response { id: req.id, result: None, error: Some(format!("unknown method: {}", req.method)) },
    };
    Response { id: req.id, result: Some(result), error: None }
}

fn main() {
    eprintln!("[systems_agent] started — listening on stdin for JSON requests");
    let stdin = io::stdin();
    let stdout = io::stdout();
    let mut stdout = stdout.lock();

    for line in stdin.lock().lines() {
        let line = match line {
            Ok(l) => l,
            Err(_) => break,
        };
        if line.trim().is_empty() {
            continue;
        }
        let req: Request = match serde_json::from_str(&line) {
            Ok(r) => r,
            Err(e) => {
                let resp = Response {
                    id: 0,
                    result: None,
                    error: Some(format!("parse error: {}", e)),
                };
                let _ = writeln!(stdout, "{}", serde_json::to_string(&resp).unwrap());
                continue;
            }
        };
        let resp = handle_request(req);
        let _ = writeln!(stdout, "{}", serde_json::to_string(&resp).unwrap());
        let _ = stdout.flush();
    }
}
