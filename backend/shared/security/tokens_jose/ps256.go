package tokens_jose

// SignPS256 は鍵の所有モジュールを知らずに JWT へ署名する。
// crypto.Signer を使うため、ローカル鍵とリモートの署名器を同じ経路で扱う。
func SignPS256(privateKey any, kid string, extraHeader map[string]string, claims map[string]any) (string, error) {
	return SignJWT(privateKey, "PS256", keyedHeader(kid, extraHeader), claims)
}

// keyedHeader は kid と追加のヘッダーを一つの保護ヘッダーにする。追加のヘッダーの kid が優先する。
func keyedHeader(kid string, extraHeader map[string]string) map[string]any {
	header := map[string]any{"kid": kid}
	for k, v := range extraHeader {
		header[k] = v
	}
	return header
}
