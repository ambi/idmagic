---
status: completed
archived_from: 95be2cb9004aff9d935a702537289810b0d81b5b
---

# アプリケーション単位で claim release を上書きする

ADR-151 を起票し、claim release の fail-closed floor を新規 tenant-default policy row では
なく既存の `UserAttributeDef.visibility` (Private 除外) とする決定を確定した。既存の
per-RP/SP/Client `claim_policy` をそのままアプリ上書きとして読み替え、新しい tenant-default
ストレージは追加していない (ADR-081 の完全置換モデルは不採用)。
- **Domain**: `backend/claimmapping/usecases/floor.go` に `IsAttributeReleasable` (Private
属性・未知 attribute key を fail-closed で拒否)、`IsReservedClaimType`
(iss/sub/aud/exp/iat/nbf/jti/azp/nonce/at_hash/c_hash/acr/amr/sid)、
`ValidateClaimReleaseRules`、`IssueClaimsWithFloor` を実装。
- **Persistence**: `oauth2_clients` に nullable `claim_policy JSONB` を追加し、OIDC にも
WS-Fed/SAML と同じ claim release 上書きの入れ物を用意した (既存 2 テーブルは無変更)。
- **Protocol**: OIDC (ID Token / UserInfo)、SAML assertion、WS-Fed / WS-Trust token の
発行経路をすべて `IssueClaimsWithFloor` に接続した。OIDC は scope 主導の既存
`ClaimsForScopes` の後にアプリ上書き claim を追加でマージし (標準 claim 優先)、
`sub` / NameID の source 属性もアプリ単位で上書きできるようにした。
- **Admin API / UI**: `UpdateApplicationOidcConfig` に `rules` / `sub_source_attribute` を
追加 (WS-Fed/SAML と同型)。3 protocol すべての Update*Config が書き込み時に
`ValidateClaimReleaseRules` で fail-closed floor を検証し (400 拒否)、
`ApplicationClaimMappingUpdated` イベントを発行する。Admin UI に OIDC 向け claim release
editor と、テナント属性定義 (key/type/visibility) を実ユーザー値なしで一覧する非PII
preview (`ClaimReleaseAttributesPreview`) を追加し、OIDC/WS-Fed/SAML の 3 editor 共通で
埋め込んだ。

全文は `git show 95be2cb9004aff9d935a702537289810b0d81b5b:<このファイルのパス>` で読む。
