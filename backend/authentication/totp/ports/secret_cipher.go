package ports

import "context"

// SecretCipher envelope-encrypts the TOTP seed at rest. The
// concrete implementation is backend/datakeys.FieldCipher; this port exists
// so db_postgres does not depend on DataKeys' internal packages directly
// (mirrors backend/authentication/password/ports.PasswordHasher).
type SecretCipher interface {
	Encrypt(ctx context.Context, tenantID, recordContext, table, recordID, field, plaintext string) (keyVersion int, ciphertext []byte, err error)
	Decrypt(ctx context.Context, tenantID, recordContext, table, recordID, field string, keyVersion int, ciphertext []byte) (plaintext string, err error)
}
