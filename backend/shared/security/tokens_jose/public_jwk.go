package tokens_jose

import (
	"crypto/rsa"
	"errors"
	"fmt"
)

// minimumRSAModulusBits は、利用者が登録する RSA の署名鍵に求める鍵長の下限。
const minimumRSAModulusBits = 2048

// privateJWKMembers は、JWK が秘密鍵または対称鍵を運ぶときにだけ現れるメンバー
// (RFC 7518 の 6.2.2 節、6.3.2 節、6.4.1 節)。
var privateJWKMembers = []string{"d", "p", "q", "dp", "dq", "qi", "oth", "k"}

// ValidatePublicSigningJWK は、利用者が登録する署名の検証用の公開鍵として JWK を受け入れて
// よいかを確かめ、RFC 7638 のサムプリントを返す。受け入れるのは、ES256 に使う P-256 の
// EC 鍵と、PS256 に使う 2048 ビット以上の RSA 鍵だけで、秘密鍵の成分を含む JWK は拒否する。
func ValidatePublicSigningJWK(jwk map[string]any) (string, error) {
	for _, member := range privateJWKMembers {
		if _, ok := jwk[member]; ok {
			return "", fmt.Errorf("jwk: private member %q is not allowed", member)
		}
	}
	key, err := publicKeyFromJWK(jwk)
	if err != nil {
		return "", fmt.Errorf("jwk: %w", err)
	}
	if rsaKey, ok := key.(*rsa.PublicKey); ok {
		if rsaKey.N.BitLen() < minimumRSAModulusBits {
			return "", fmt.Errorf("jwk: RSA modulus must be at least %d bits", minimumRSAModulusBits)
		}
		if rsaKey.E < 3 || rsaKey.E%2 == 0 {
			return "", errors.New("jwk: RSA public exponent is invalid")
		}
	}
	return jwkThumbprint(jwk)
}
