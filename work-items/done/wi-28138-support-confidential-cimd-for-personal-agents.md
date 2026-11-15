---
status: completed
---

# CIMD の秘密鍵認証と client assertion の PS384 を追加する

CIMD の `private_key_jwt` を採用し、公開鍵だけのインライン JWKS または HTTPS の JWKS URI のいずれか一方から、永続登録しない confidential クライアントを解決できるようにした。
認可コードと refresh のグラントだけを許し、PKCE を必須のまま保持し、不正な鍵・秘密素材・鍵源の二重指定は公開クライアントへ降格せず拒否する。
client assertion に PS384 の SHA-384・RSA-PSS 検証を追加し、登録済みクライアントにも提供する。Discovery と AS メタデータの広告を合わせ、発行トークンと DPoP の署名方式は広げていない。
仕様差分は REQ-OAUTH2-005・014・017・028・030、CIMD の任意機能の採用、署名方式の用途別の規則、CIMD と Discovery の TypeSpec に反映した。
Poppy の企業・issuer・付与のプロファイルは後続の WI-10861 が扱い、今回の基盤だけで Poppy 適合や実製品との相互運用を保証しない。
