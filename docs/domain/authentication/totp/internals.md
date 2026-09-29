# TOTPの内部設計

## 永続化

`mfa_factors.secret` は以前からある平文の TOTP の種のカラムであり、既存のレコードが読めるようにするためだけに残している (二重読み)。新しい書き込みは `secret_key_version` と `secret_ciphertext` を埋め、`secret` は `NULL` のままにする。残りの平文のレコードは保留中の埋め戻しで移行し、その後 `secret` を削除する。
