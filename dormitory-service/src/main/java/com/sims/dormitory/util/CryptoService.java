package com.sims.dormitory.util;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Component;

import javax.crypto.Cipher;
import javax.crypto.spec.GCMParameterSpec;
import javax.crypto.spec.SecretKeySpec;
import java.security.SecureRandom;
import java.util.Base64;

@Component
public class CryptoService {

    private static final Logger log = LoggerFactory.getLogger(CryptoService.class);

    private static final String ALGORITHM = "AES/GCM/NoPadding";
    private static final int GCM_IV_LENGTH = 12;
    private static final int GCM_TAG_LENGTH = 128;
    private static final int KEY_LENGTH = 32;
    private static final String ENV_KEY = "CAMERA_ENCRYPTION_KEY";

    private static final byte[] DEV_KEY = "01234567890123456789012345678901".getBytes();

    private final SecretKeySpec masterKey;

    public CryptoService() {
        String keyStr = System.getenv(ENV_KEY);
        if (keyStr == null || keyStr.isEmpty()) {
            log.warn("[WARN] crypto: {} not set, using DEV key (INSECURE — for development only)", ENV_KEY);
            this.masterKey = new SecretKeySpec(DEV_KEY, "AES");
        } else {
            byte[] key = keyStr.getBytes();
            if (key.length != KEY_LENGTH) {
                throw new IllegalArgumentException(
                    "crypto: key must be " + KEY_LENGTH + " bytes, got " + key.length
                );
            }
            this.masterKey = new SecretKeySpec(key, "AES");
        }
    }

    public CryptoService(byte[] key) {
        if (key.length != KEY_LENGTH) {
            throw new IllegalArgumentException(
                "crypto: key must be " + KEY_LENGTH + " bytes, got " + key.length
            );
        }
        this.masterKey = new SecretKeySpec(key, "AES");
    }

    /**
     * Encrypt a password using AES-256-GCM.
     * @param password plaintext password
     * @return encrypted result with ciphertext and nonce (both base64-encoded)
     */
    public EncryptedPassword encryptPassword(String password) {
        try {
            byte[] plaintext = password.getBytes(java.nio.charset.StandardCharsets.UTF_8);
            byte[] nonce = new byte[GCM_IV_LENGTH];
            SecureRandom.getInstanceStrong().nextBytes(nonce);

            Cipher cipher = Cipher.getInstance(ALGORITHM);
            GCMParameterSpec spec = new GCMParameterSpec(GCM_TAG_LENGTH, nonce);
            cipher.init(Cipher.ENCRYPT_MODE, masterKey, spec);

            byte[] ciphertext = cipher.doFinal(plaintext);

            return new EncryptedPassword(
                Base64.getEncoder().withoutPadding().encodeToString(ciphertext),
                Base64.getEncoder().withoutPadding().encodeToString(nonce)
            );
        } catch (Exception e) {
            throw new RuntimeException("crypto: encryption failed", e);
        }
    }

    /**
     * Decrypt a password using AES-256-GCM.
     * @param ciphertextBase64 base64-encoded ciphertext
     * @param nonceBase64 base64-encoded nonce
     * @return decrypted plaintext password
     */
    public String decryptPassword(String ciphertextBase64, String nonceBase64) {
        try {
            byte[] nonce = Base64.getDecoder().decode(nonceBase64);
            byte[] ciphertext = Base64.getDecoder().decode(ciphertextBase64);

            Cipher cipher = Cipher.getInstance(ALGORITHM);
            GCMParameterSpec spec = new GCMParameterSpec(GCM_TAG_LENGTH, nonce);
            cipher.init(Cipher.DECRYPT_MODE, masterKey, spec);

            byte[] plaintext = cipher.doFinal(ciphertext);
            return new String(plaintext, java.nio.charset.StandardCharsets.UTF_8);
        } catch (Exception e) {
            throw new RuntimeException("crypto: decryption failed", e);
        }
    }

    public record EncryptedPassword(String ciphertext, String nonce) {}
}
