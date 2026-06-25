import hashlib
import hmac
import os

import pyaes


class EmeraldCryptoVault:
    KEY_LENGTH = 32
    NONCE_LENGTH = 12
    SALT_LENGTH = 16
    MAC_LENGTH = 32
    ITERATIONS = 100000

    def __init__(self):
        self.secret_key = os.getenv("EMERALD_MASTER_SECURE_KEY")
        if not self.secret_key:
            raise ValueError("CRITICAL_SECURITY_ERROR_MASTER_KEY_MISSING")
        self._master = bytes.fromhex(self.secret_key)

    def _derive_keys(self, salt: bytes):
        raw = hashlib.pbkdf2_hmac(
            "sha256", self._master, salt,
            self.ITERATIONS, dklen=self.KEY_LENGTH * 2,
        )
        return raw[:32], raw[32:]

    def encrypt_data_payload(self, plain_text_data: str) -> bytes:
        nonce = os.urandom(self.NONCE_LENGTH)
        salt = os.urandom(self.SALT_LENGTH)
        enc_key, mac_key = self._derive_keys(salt)

        ctr_val = int.from_bytes(nonce, "big")
        counter = pyaes.Counter(initial_value=ctr_val)
        aes = pyaes.AESModeOfOperationCTR(enc_key, counter=counter)
        ciphertext = aes.encrypt(plain_text_data.encode("utf-8"))

        mac = hmac.new(mac_key, nonce + salt + ciphertext, hashlib.sha256).digest()
        return nonce + salt + mac + ciphertext

    def decrypt_data_payload(self, encrypted_payload: bytes) -> str:
        min_len = self.NONCE_LENGTH + self.SALT_LENGTH + self.MAC_LENGTH
        if len(encrypted_payload) < min_len:
            raise ValueError("CRITICAL_SECURITY_ERROR_TAMPERED_PAYLOAD")

        nonce = encrypted_payload[:12]
        salt = encrypted_payload[12:28]
        mac = encrypted_payload[28:60]
        ciphertext = encrypted_payload[60:]

        enc_key, mac_key = self._derive_keys(salt)

        expected_mac = hmac.new(mac_key, nonce + salt + ciphertext, hashlib.sha256).digest()
        if not hmac.compare_digest(expected_mac, mac):
            raise ValueError("CRITICAL_SECURITY_ERROR_TAMPERED_PAYLOAD")

        ctr_val = int.from_bytes(nonce, "big")
        counter = pyaes.Counter(initial_value=ctr_val)
        aes = pyaes.AESModeOfOperationCTR(enc_key, counter=counter)
        plaintext = aes.decrypt(ciphertext)
        return plaintext.decode("utf-8")


class SecureVaultFileManager:
    def __init__(self, vault: EmeraldCryptoVault):
        self.vault = vault

    def encrypt_file(self, file_path: str, output_path: str = None) -> str:
        with open(file_path, "r") as f:
            data = f.read()
        encrypted = self.vault.encrypt_data_payload(data)
        out = output_path or file_path + ".encrypted"
        with open(out, "wb") as f:
            f.write(encrypted)
        return out

    def decrypt_file(self, encrypted_path: str) -> str:
        with open(encrypted_path, "rb") as f:
            data = f.read()
        return self.vault.decrypt_data_payload(data)
