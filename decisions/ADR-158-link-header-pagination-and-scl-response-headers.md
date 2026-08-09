---
status: accepted
authors: [tn]
created_at: 2026-08-09
superseded_by: [ADR-159]
---

# ADR-158: 一覧 API のページングは RFC 8288 `Link` ヘッダで表現する

## コンテキスト
wi-159 で admin/API の主要一覧 (User、Group、Agent、Application、Consent、AuditEvent、ProvisioningDelivery 等) を keyset (cursor) pagination へ揃える。ページング状態をどこに載せるかは、client 実装の安全性・既存 RFC-first 方針・SCL の表現力の 3 点にまたがる cross-context な決定であり、着手前に固定する必要がある。

現状 `bindings.kind: http` の `headers` 語彙はリクエストヘッダの入力表現専用 (`Introspect.headers.dpop` 等) で、レスポンスヘッダを型付き契約として宣言する語彙が SCL に無い。また `ListProvisioningDeliveries` は `stability: stable` だが未リリースであり、ADR-156 の `stable` は API access token での到達可能性の分類であって外部互換保証ではない。

## 決定
- 一覧 API のページング状態は body ではなく RFC 8288 `Link` レスポンスヘッダ (`rel="next"`、GitHub REST API と同型) で表す。body は常に純粋なドメインデータのみとし、`next_cursor`/`has_more` は持たせない。`rel="first"/"prev"/"last"` は返さない。
- 共通契約: input は `cursor: String, optional` + `limit: Integer, optional` (+ interface 固有の sort/filter)。output は `<items>: T[]` のみ。`limit` の既定値・上限は interface ごとに明記する。命名は `page_size` ではなく既存 `ListProvisioningDeliveries`/`AuditEventQuery` にある `limit` に一本化する。
- cursor は opaque token とし、tenant_id・filter/sort・最終行の keyset (sort key + id)・expiry を含めて署名する (HMAC-SHA256 + base64url)。検証失敗は既存の `InvalidRequestError` を返す。

cursor の expiry と `rel="prev"` 非対応は [ADR-159](ADR-159-addressable-bidirectional-cursor-pages.md) が部分的に上書きする。body ではなく RFC 8288 `Link` ヘッダを使う決定は引き続き有効である。
- `SPECIFICATION_CORE_LANGUAGE.md` に `output.headers` のレスポンスヘッダ宣言語彙を新設し、`tools/scl-to-openapi` (`responses.headers`)・`tools/scl-to-jsonschema`・`tools/check` を対応させる。
- `ListProvisioningDeliveries` (`stability: stable`) を含め、対象の全 list interface をこの契約に合わせて改修する。pre-release のため互換維持は理由にならない。
- 絶対 URL の組み立ては `backend/shared/http/support_http` の `Deps.CanonicalLocation` を再利用する。

詳細な理由・比較検討は `work-items/wi-159-admin-resource-cursor-pagination.md` の `## Design` に記載済み。

## 却下した代替案
- body-embedded `next_cursor` (AIP-158 方式): 既存踏襲・admin SPA 専用・SCL 表現力欠如のいずれも実質的な根拠にならない。加えて client が次ページ要求時に元の filter/sort を再送し忘れる/書き換えるバグを構造的に許してしまう。
- レスポンスボディの `has_more: Boolean`: `Link` ヘッダの有無と冗長になる。
- SCL の表現力不足を理由に body で妥協する案: SCL 側の欠陥は SCL を直す理由であり、body に逃げる理由にはならない。

## 影響
- SCL: `SPECIFICATION_CORE_LANGUAGE.md` に `output.headers` 語彙を追加 (コア言語拡張)。`interfaces.ListAdminUsers` / group / agent / application / consent / `ListAdminAuditEvents` / `ListProvisioningDeliveries` 等の list interface の input/output 契約。
- ツール: `tools/scl-to-openapi`、`tools/scl-to-jsonschema`、`tools/check`。
- Go: 対象 usecase/handler の input/output、cursor encode/decode・署名検証、`Link` header 組み立て。
- Persistence: keyset pagination 化 (offset 廃止)。
