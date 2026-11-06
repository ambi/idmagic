---
status: completed
---

# SPIFFE の信頼設定では、JWT-SVID を SPIFFE の規則どおりに検証する

信頼設定に検証の種別（`oidc` と `spiffe`）を足し、`spiffe` の信頼設定でだけ SPIFFE の JWT-SVID の規則を検証するようにした（REQ-WORKLOADIDENTITY-011 を追加し、001、002、008 を変更）。
`spiffe` の信頼設定は発行者を省け、交換では `iss` に一致する信頼設定を先に探し、なければ `sub` の SPIFFE ID のトラストドメインで探す。
署名の検証の後に、`sub` の SPIFFE ID の構文（`invalid_spiffe_id`）とトラストドメインの一致（`trust_domain_mismatch`）を確かめる。
JOSE の層では、`spiffe` に限って `kid` を必須にし、`typ` を `JWT` と `JOSE` に限り、`alg` を RS、ES、PS の 256、384、512 に広げた。`oidc` の検証は変えていない。
管理 API は登録で `profile` を受け、省けば `oidc` にする。種別は更新で変えられない。新しい拒否は 422 `workload_trust_bundle_invalid_trust_domain` と 409 `workload_trust_bundle_trust_domain_conflict` である。
スキーマは `profile` の列をデフォルト値 `oidc` で足し、`issuer` を NULL 可にし、`spiffe` のトラストドメインの部分一意索引を足した。既存の行はすべて `oidc` のため、データの移行と重複はない。
案 A（トラストドメインで探す）の根拠は SPIFFE の仕様と参照実装の go-spiffe に置き、他製品との比較（Keycloak と Vault は案 A、Entra ID、PingFederate、Okta、AWS、Google Cloud は汎用の OIDC、Auth0 は組み込みの連携なし）は `decisions.md` に残した。脅威モデルに THREAT-099 を足した。

確認した範囲の限界：実際の SPIRE が発行した JWT-SVID とバンドルでの確認はしていない（wi-76418 が扱う）。ES384 と ES512 は Go の標準ライブラリで作った鍵だけで確かめた。
残るリスク：仕様は `kid` を任意とするため、`kid` のない JWT-SVID を発行する実装は `spiffe` の信頼設定で拒否される。管理画面の信頼設定のページを通るブラウザーの E2E はない。
