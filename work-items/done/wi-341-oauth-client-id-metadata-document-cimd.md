---
status: completed
authors: [claude]
risk: medium
created_at: 2026-08-08
depends_on: [wi-56-mcp-authorization-server]
---

# OAuth Client ID Metadata Documents (CIMD) を DCR の代替クライアント登録経路として実装する

## Motivation

[[wi-322-mcp-authorization-spec-repin-2026-07-audit]] の差分監査で判明した積み残し。MCP
authorization 仕様 2026-07-28 改訂は、クライアント登録の優先順位を
「pre-registration → **Client ID Metadata Documents (CIMD)** → Dynamic Client Registration
(RFC 7591、fallback 用) → 手動入力」に変更した。DCR は明示的に **deprecated** 扱いで、
「CIMD 未対応の authorization server との後方互換のためだけに残す」位置づけになっている。

idmagic は [[ADR-055]] のもと RFC 7591 DCR のみで MCP クライアントの自動オンボーディングを
提供しており([[ADR-011]] の Discovery 方針に沿う)、CIMD には未対応。CIMD 自体は
`client_id` を client がホストする HTTPS URL にし、authorization server がその URL を
fetch してメタデータ(`client_id` / `client_name` / `redirect_uris` 等)を検証する方式で、
事前関係のない client/server 間の自動オンボーディングという MCP の中心的なユースケースに
DCR より適している(登録リクエストという往復が不要)。

根拠仕様は IETF `draft-ietf-oauth-client-id-metadata-document-**00**`(2026年8月時点でごく
初期のドラフト、RFC 化されていない)。[[ADR-055]] が MCP 認可仕様自体について「版差が
大きいため対象改訂を固定する」と同じ理由で、ここも新しい ADR で対象 draft のバージョンと
追従方針を明示的に固定すべき規模の決定である。また、client が指定する任意の HTTPS URL を
authorization server が fetch するという新しい外部入力面が加わるため、SSRF 対策の設計判断が
本 wi の中心的な論点になる。

## Scope

- **decision**: 新規 ADR。以下を確定する。
  - 対象 draft のバージョン pin(`-00` のまま追従するか、内容が固まるまで待つか)と、
    draft 改訂時の再ピン留め方針([[ADR-055]] と同様の運用にするか)。
  - CIMD で解決した client 情報を `Client` として永続化するか、都度解決(request-scoped、
    非永続)にするかのデータモデル。
  - fetch 時の SSRF 対策(private / loopback / link-local IP 拒否、DNS rebinding 対策、
    リダイレクト制限、タイムアウト、レスポンスサイズ上限)の具体的な境界。
  - 既存 RFC 7591 DCR との共存方針(両方式を並行提供し、`client_id` の形式で分岐)。
  - `redirect_uri` 検証・キャッシュ方針(HTTP cache header 準拠 vs 上限 TTL)。
  - CIMD 由来 client の client 認証方式(`token_endpoint_auth_method: none` を許容するか、
    `private_key_jwt` を要求するか)。
- **scl**: `spec/contexts/oauth2.yaml` に以下を追加。
  - standards: `draft-ietf-oauth-client-id-metadata-document` の requirements 一覧。
  - AS Metadata (RFC 8414) response へ `client_id_metadata_document_supported` フィールドを追加。
  - Authorize / PAR / Token の `client_id` 解釈に CIMD 解決分岐を追加(既存 `RegisterClient`
    interface とは別の、実行時解決パス)。
  - 新規イベント(例: `ClientIdMetadataDocumentResolved` / `ClientIdMetadataDocumentFetchRejected`)、
    新規エラー(メタデータ取得失敗・JSON 形式不正・`client_id` 不一致・`redirect_uri` 不一致は
    いずれも fail-closed で `invalid_client` 系)。
- **go**: HTTPS-only の SSRF-safe fetcher、キャッシュ、`/authorize`・`/par`・`/token` の
  client 解決ロジックへの統合。
- **http**: 新規エンドポイントは追加しない。既存 discovery 応答
  (`/.well-known/oauth-authorization-server`)に `client_id_metadata_document_supported` を追加。

## Out of Scope

- 実際の MCP クライアント実装(Claude Desktop 等)との相互運用テスト。
- RFC 7591 DCR の廃止・削除(MCP 仕様どおり後方互換として残す。[[ADR-055]] の決定 5 は変更しない)。
- Pre-registration(静的 client 登録)方式自体の変更。
- `application_type` (OIDC DCR) 対応 — [[wi-322-mcp-authorization-spec-repin-2026-07-audit]] で
  影響なしと判断済みの別項目であり、本 wi のスコープ外。

## Design

[[ADR-155]] で確定。要点:

- CIMD 解決結果は永続化しない。`client_id` が `https` + 非空 path の URL 形状で、既存の
  `OAuth2ClientRepository.FindByID` がヒットしないときだけ CIMD ドキュメントを fetch して
  都度解決する。解決結果は既存 `domain.OAuth2Client` にマップし、`/authorize`・`/par`・`/token`
  の既存ロジック(redirect_uri 照合・PKCE・consent 表示・scope 検証)をそのまま再利用する。
  実装は `OAuth2ClientRepository` を埋め込み `FindByID` だけを上書きするデコレータ
  (`ClientRepositoryWithCIMD`)として `backend/oauth2/client/cimd_http/` に置き、composition
  root で 1 箇所だけ差し替える(`authorize.go`/`push_authorization_request.go`/
  `client_auth.go`/`token_handler.go` 側の呼び出しコードは無変更)。
- fetch は `shared/security/tokens_jose` の `JWKResolver` が持つ SSRF-safe dialer
  (https 限定・DNS 解決後 public IP 限定・redirect 上限・timeout・応答サイズ上限)を
  共通ヘルパーへ抽出して再利用する。
- MVP は `token_endpoint_auth_method` 未指定 or `none` の文書のみ受理(それ以外は fail-closed
  で拒否)。`client_id` はフィールドと fetch URL の厳密一致必須、`redirect_uris` は必須・非空。
  `scope` は文書の自己申告値(未指定時は `openid` のみ)— 既存 DCR の自己申告モデルと同じ扱い。
- Discovery に `client_id_metadata_document_supported: true` を追加。
- `Application` 割当ゲートは DCR 自己登録クライアントと同じ現行挙動(未登録 Application =
  許可)に揃え、CIMD 専用の新しい割当モデルは作らない。

## Plan

1. `spec/contexts/oauth2.yaml` を更新: 新規 standards エントリ、`DiscoveryDocument` へ
   `client_id_metadata_document_supported`、`Client` 語彙定義への CIMD 注記、新規イベント。
2. Domain: `client/domain/cimd.go` に URL 形状判定・ドキュメントの純粋パース/検証関数を
   test-first で実装する(ネットワーク非依存)。
3. Adapters: `client/cimd_http/` に SSRF-safe fetcher と `ClientRepositoryWithCIMD` デコレータを
   test-first で実装する(fetcher の SSRF ケースは既存 `jwks_resolver_test.go` と同水準、
   デコレータの分岐はモック fetcher で純粋にテストする)。
4. Infrastructure: composition root で `ClientRepo` をデコレータでラップして差し替える。
5. Discovery builder (`shared/spec/discovery.go`) に boolean を追加する。
6. `just verify` を通し、`## Completion` を追記して `work-items/done/` へ移す。

## Tasks

- [x] T001 [ADR] [[ADR-155]] を起票し、draft pin・永続化モデル・SSRF 対策境界・DCR 共存方針を確定した。
- [x] T002 [SCL] ADR-155 の決定に沿って `spec/contexts/oauth2.yaml` を更新した(新規 standards
      `ClientIDMetadataDocumentDraft00`、`DiscoveryDocument.client_id_metadata_document_supported`、
      新規 value_object `ClientIDMetadataDocument`、新規 event `ClientIdMetadataDocumentResolved`/
      `ClientIdMetadataDocumentRejected`、`Authorize`/`PushAuthorizationRequest`/`Token` の
      `emits` 追加、`Client` 語彙定義への CIMD 注記)。`just check-scl` / `just check` は green。
- [x] T003 [Domain] `client/domain/cimd.go` — RED: `TestIsClientIDMetadataDocumentURL`・
      `TestParseClientIDMetadataDocument_*` を先に fail 確認(`undefined: IsClientIDMetadataDocumentURL`
      等のビルド失敗)(standards `ClientIDMetadataDocumentDraft00`、value_object
      `ClientIDMetadataDocument`)→ GREEN。
- [x] T004 [Adapters] SSRF-safe fetcher と `ClientRepositoryWithCIMD` を test-first (RED→GREEN) で実装した:
      RED: `shared/security/safehttp` の `TestIsPublicIP*`/`TestSafeIPs*` を先に fail 確認
      (`undefined: IsPublicIP`/`SafeIPs`) → GREEN (`tokens_jose.JWKResolver` を委譲にリファクタ、
      既存テスト無変更で green 維持)。
      RED: `cimd_http` の `TestClientRepositoryWithCIMD_*` を先に fail 確認
      (`undefined: ClientRepositoryWithCIMD`) → GREEN。
      RED: `cimd_http` の `TestFetcher_*` を先に fail 確認 (httptest.NewServer が http のため
      https-only 検証で意図通り reject、テストの意図とズレていたため httptest.NewTLSServer に
      修正して正しい理由での RED を確認) → GREEN。
- [x] T005 [Infrastructure] composition root で `ClientRepo` をデコレータに差し替えた
      (`bootstrap/memory.go`・`bootstrap/postgres.go`)。`Emit` は起動順序上
      `NewEmitFunc` より前に組み立てられないため、`cmd/idmagic/server.go` で
      `sessionManager.Emit` と同じパターンの事後設定にした
      (worker/batch プロセスは `/authorize` 系を提供しないため未配線のままで問題ない、
      デコレータの `emit()` は `Emit == nil` の場合 no-op)。`go build ./backend/...` /
      `go test ./backend/cmd/...` は green。
- [x] T006 [Adapters] `shared/spec/discovery.go` — RED:
      `TestBuildDiscoveryDocument_AdvertisesClientIDMetadataDocumentSupport` を先に fail 確認
      (`client_id_metadata_document_supported = <nil>`)(SCL `DiscoveryDocument.client_id_metadata_document_supported`)
      → GREEN。
- [x] T007 [Verify] `just verify` / `just verify-go` (lint + race テスト) を通した。
      ついでに `backend/oauth2/ARCHITECTURE.md` へ CIMD の設計節を追記し、
      新規モジュール (`client/cimd_http`、`shared/security/safehttp`) を
      `architecture.yaml` 各所に登録して `just check-architecture` を通した。

## Verification

- `just test-go` / `just lint-go` / `just build-go`
- SSRF 対策のケース(private/loopback IP、リダイレクトループ、巨大レスポンス、タイムアウト)を
  fail-closed で拒否することを test-go で確認する。
- `client_id` と fetch したメタデータ内 `client_id` の不一致、`redirect_uri` の不一致を拒否する
  ことを確認する。

## Risk Notes

リスクは medium。最大の懸念は **SSRF**: client が任意の HTTPS URL を `client_id` として提示し、
authorization server がそれを fetch する新しい外部入力面が加わる。private/loopback IP・
DNS rebinding・リダイレクト・応答サイズを fail-closed で制限しないと、内部ネットワークへの
到達性探索やリソース枯渇に使われうる。次点のリスクは仕様側: 根拠 draft がまだ `-00` の
ごく初期段階でありRFC化されていないため、[[ADR-055]] と同様に対象改訂を明示的に固定し、
draft が動いた場合の追従方針を決めておく必要がある。

## Completion

- **Completed At**: 2026-08-08
- **Summary**:
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
- **対応していないこと(開示)**:
  - `token_endpoint_auth_method` が `none` 以外(特に `private_key_jwt` 経由の CIMD 対応)は
    `adoption: excluded` として MVP スコープ外にした。ドキュメント側の `jwks`/`jwks_uri` への
    追加 fetch が絡み SSRF 経路が複合するため、別途 work item で扱う。
  - キャッシュは HTTP cache header (`Cache-Control` 等) を解釈せず固定 5 分 TTL とした
    (`CIMD00-CACHE` は `adoption: partial`)。仕様は SHOULD でヘッダ準拠を求めるが、
    実装複雑度に対し優先度が低いと判断し見送った。文書更新の反映が最大 5 分遅延しうる。
  - Out of Scope どおり、実際の MCP クライアント実装(Claude Desktop 等)との相互運用テスト、
    RFC 7591 DCR の廃止、pre-registration 方式の変更、`application_type` 対応は行っていない。
- **Verification Results**:
  - `just check` - passed(SCL・work-item・architecture ledger・traceability を含め all green)。
  - `just verify` - passed(check / traceability-strict / test-tools / typecheck-tools /
    lint-go / test-go / format-check-ui / lint-ui / test-ui-unit / build-ui すべて green。
    UI は本 wi で変更していない)。
  - `just verify-go` (lint-go + test-go-race) - passed。
  - RED→GREEN の自己証跡は各 Tasks 項目に記載
    (`TestIsClientIDMetadataDocumentURL`、`TestParseClientIDMetadataDocument_*`、
    `TestIsPublicIP*`/`TestSafeIPs*`、`TestClientRepositoryWithCIMD_*`、`TestFetcher_*`、
    `TestBuildDiscoveryDocument_AdvertisesClientIDMetadataDocumentSupport`)。
  - 手動 E2E は実施せず(Out of Scope の相互運用テストと同様、実クライアントへの接続確認は
    別途)。
