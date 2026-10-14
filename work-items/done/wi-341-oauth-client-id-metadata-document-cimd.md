---
status: completed
archived_from: ef0c50842dbbf07f02820cb5bec1a53362146304
---

# OAuth Client ID Metadata Documents (CIMD) を DCR の代替クライアント登録経路として実装する

[[ADR-155]] を起票し、CIMD (`draft-ietf-oauth-client-id-metadata-document-00`) を DCR の
代替クライアント解決経路として実装した。`client_id` が `https` + 非空 path の URL 形状で
既存 `OAuth2ClientRepository.FindByID` がヒットしない場合、`client/cimd_http.Fetcher` が
SSRF-safe に文書を fetch・検証し、`ClientRepositoryWithCIMD` デコレータ経由で既存の
`OAuth2Client` 解決経路へ透過的に合流する。`authorize.go`・`push_authorization_request.go`・
`client_auth.go` の呼び出し側コードは無変更(デコレータが `FindByID` だけを上書きし他は
委譲するため)。SSRF 対策(https 限定・DNS 解決後 public IP 限定・redirect 上限・timeout・
応答サイズ上限)は `shared/security/tokens_jose.JWKResolver` から `shared/security/safehttp`
へ抽出し、jwks_uri 解決と CIMD 解決の両方が同じ実装を共有するようにした(既存
`jwks_resolver_test.go` は無変更のまま green)。Discovery に
`client_id_metadata_document_supported: true` を追加。`backend/oauth2/ARCHITECTURE.md` に
設計節を追記し、新規モジュール (`client/cimd_http`、`shared/security/safehttp`) を
`architecture.yaml` 各所へ登録した。

全文は `git show ef0c50842dbbf07f02820cb5bec1a53362146304:<このファイルのパス>` で読む。
