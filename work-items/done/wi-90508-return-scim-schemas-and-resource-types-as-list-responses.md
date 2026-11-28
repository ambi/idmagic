---
status: completed
---

# SCIM の /Schemas と /ResourceTypes の一覧を ListResponse で返す

REQ-SOURCING-029 を追加した。`GetScimSchemas` と `GetScimResourceTypes` は、全件を一ページの ListResponse（`ScimSchemaListResponse`、`ScimResourceTypeListResponse`）で返す。
ソートとページングの引数は無視し、`filter` の引数には値が空でも 403 の SCIM エラー（`ScimDiscoveryFilterRefused`）を返す。認証の失敗はそれより先に判定する。
filter の 403 は RFC 7644 §4 の SHOULD であり、確かめた製品の文書には扱いがなかったので、利用者の確認を経て採った。
Entra ID の記録の `discover-schema` を再生の列へ入れた。
