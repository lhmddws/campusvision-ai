package com.sims.dormitory.util;

import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.*;

@DisplayName("CryptoService AES-256-GCM Tests")
class CryptoServiceTest {

    private static final byte[] KEY_1 = "01234567890123456789012345678901".getBytes();
    private static final byte[] KEY_2 = "abcdefghijklmnopqrstuvwxyz123456".getBytes();

    @Test
    @DisplayName("round-trip encrypt and decrypt returns original password")
    void roundTrip() {
        CryptoService cs = new CryptoService(KEY_1);
        String password = "admin123";

        CryptoService.EncryptedPassword ep = cs.encryptPassword(password);
        String decrypted = cs.decryptPassword(ep.ciphertext(), ep.nonce());

        assertEquals(password, decrypted);
    }

    @Test
    @DisplayName("decrypting with wrong key throws exception")
    void wrongKeyFails() {
        CryptoService cs1 = new CryptoService(KEY_1);
        CryptoService cs2 = new CryptoService(KEY_2);

        CryptoService.EncryptedPassword ep = cs1.encryptPassword("admin123");

        assertThrows(RuntimeException.class,
            () -> cs2.decryptPassword(ep.ciphertext(), ep.nonce()),
            "crypto: decryption failed");
    }

    @Test
    @DisplayName("decrypting with invalid base64 input throws exception")
    void invalidBase64() {
        CryptoService cs = new CryptoService(KEY_1);

        assertThrows(RuntimeException.class,
            () -> cs.decryptPassword("!!!not-base64!!!", "dGVzdA=="));
    }

    @Test
    @DisplayName("constructor rejects short key")
    void keyLengthValidation() {
        byte[] shortKey = new byte[16];

        IllegalArgumentException ex = assertThrows(IllegalArgumentException.class,
            () -> new CryptoService(shortKey));
        assertTrue(ex.getMessage().contains("32 bytes"));
    }

    @Test
    @DisplayName("no-arg constructor falls back to dev key without throwing")
    void devModeFallback() {
        assertDoesNotThrow(() -> {
            CryptoService cs = new CryptoService();
            CryptoService.EncryptedPassword ep = cs.encryptPassword("dev-test");
            String decrypted = cs.decryptPassword(ep.ciphertext(), ep.nonce());
            assertEquals("dev-test", decrypted);
        });
    }

    @Test
    @DisplayName("different plaintexts produce different ciphertexts")
    void differentInputsDifferentOutput() {
        CryptoService cs = new CryptoService(KEY_1);

        CryptoService.EncryptedPassword ep1 = cs.encryptPassword("password1");
        CryptoService.EncryptedPassword ep2 = cs.encryptPassword("password2");

        assertNotEquals(ep1.ciphertext(), ep2.ciphertext());
    }

    @Test
    @DisplayName("same plaintext produces different ciphertext each time (nonce randomness)")
    void nonceRandomness() {
        CryptoService cs = new CryptoService(KEY_1);

        CryptoService.EncryptedPassword ep1 = cs.encryptPassword("same-password");
        CryptoService.EncryptedPassword ep2 = cs.encryptPassword("same-password");

        assertNotEquals(ep1.ciphertext(), ep2.ciphertext());
        assertNotEquals(ep1.nonce(), ep2.nonce());
    }
}
