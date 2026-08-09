---
status: completed
authors: [tn]
risk: medium
created_at: 2026-08-10
depends_on: [wi-341-oauth-client-id-metadata-document-cimd]
change_kind: bugfix
initial_context:
  scl: { OAuth2: [scenarios.Client metadata fetchは公開IPへ直接接続する] }
  source: [backend/shared/security/safehttp/client.go, backend/oauth2/client/cimd_http/fetcher.go]
  tests: [backend/shared/security/safehttp/client_test.go, backend/oauth2/client/cimd_http/fetcher_test.go, backend/shared/security/tokens_jose/jwks_resolver_test.go]
  stop_before_reading: [frontend, infra]
affected_spec:
  - { context: OAuth2, kind: scenario, element: Client metadata fetchは公開IPへ直接接続する }
---

# 外部 client metadata の fetch を検査済み公開 IP への直接接続に限定する

## Motivation
CodeQL alert 14 は CIMD の client_id URL が HTTP request に到達する経路を SSRF として検出した。
既存の `safehttp` は URL、DNS 解決結果、redirect を検証しているが、環境 proxy が有効な場合は
custom dialer が最終宛先ではなく proxy を検査する余地があり、CGNAT の拒否も
`100.64.0.0/10` 全体ではなく文字列 prefix の一部に限られている。独自 transport を CodeQL が
sanitizer として認識しない点と実際の防御不足を分け、実装上の境界を先に完全にする。

## Scope
- `spec/contexts/oauth2.yaml` の `scenarios.Client metadata fetchは公開IPへ直接接続する`
- `backend/shared/security/safehttp` の direct-connection policy と CGNAT 判定
- `backend/oauth2/ARCHITECTURE.md` の SSRF-safe fetch 設計
- GitHub CodeQL alert 14 の query と現在状態の確認

## Out of Scope
- Client metadata host の固定 allow-list 化
- SSRF-safe client での outbound HTTP(S) proxy 対応
- CodeQL query pack または repository-local sanitizer model の追加
- CGNAT 以外の special-use address policy の拡張
- GitHub 上の alert 状態または dismissal comment の変更

## Design
`safehttp.NewClient` が生成する transport は環境 proxy を参照せず、DNS 検査済み IP へ直接 dial する。
CGNAT は文字列比較ではなく `100.64.0.0/10` の CIDR membership で拒否する。この package は CIMD と
JWK resolver の双方が共有するため、公開 interface を変えず両方へ同じ保証を適用する。

CodeQL の `go/request-forgery` は custom dialer の接続制約を追跡しない。alert は既に false positive として
dismiss 済みのため GitHub 上の状態は変更せず、repository 内のテストと設計記録を防御根拠とする。CIMD は
client 所有の任意 HTTPS host を解決する仕様なので、固定 allow-list は採用しない。小さな CIDR 判定と
transport policy の変更であり複雑な parser や認証・認可判断を追加しないため、fuzz/property test は
追加しない。

## Plan
1. SCL に公開 IP への直接接続と CGNAT 拒否の受け入れシナリオを追加し、検証・再生成する。
2. proxy 無効化と CGNAT `/10` 境界の回帰テストを先に追加して RED を確認する。
3. `safehttp` を最小変更して GREEN にし、CIMD/JWK の既存回帰テストを通す。
4. OAuth2 Architecture の現在設計を同期し、workspace 全体を検証する。
5. CodeQL alert 14 の query と現在の disposition を確認し、GitHub 上の状態は変更しない。

## Tasks
- [x] T001 [SCL] `Client metadata fetchは公開IPへ直接接続する` scenario を追加し、SCL と派生物を同期した。
- [x] T002 [Adapters/Test] proxy 無効化と CGNAT `/10` 境界の回帰テストを追加。RED: `TestIsPublicIPRejectsEntireCGNATRange` が `100.127.255.255` の誤受理で、`TestNewClientDoesNotUseEnvironmentProxy` が設定済み proxy で先に fail することを確認（scenario `Client metadata fetchは公開IPへ直接接続する`）。
- [x] T003 [Adapters] `safehttp` を direct-only transport と正しい CGNAT CIDR 判定へ変更し、追加テストと CIMD/JWK resolver の package test で GREEN を確認した。
- [x] T004 [Architecture] OAuth2 の SSRF-safe fetch 設計正本を同期した。
- [x] T005 [Verify] Go、SCL、Architecture、workspace 全体を検証した。CodeQL alert 14 は既に `false positive` で dismiss 済みであることを確認し、追加コメントは行わない。

## Verification
- `just check-scl`
- `just scl-render`
- `just test-go-package ./backend/shared/security/safehttp`
- `just test-go-package ./backend/oauth2/client/cimd_http`
- `just test-go-package ./backend/shared/security/tokens_jose`
- `just check`
- `just verify-go`
- `just verify`
- `git diff --check`

## Risk Notes
環境 proxy を必要とする deployment では CIMD/JWKS の outbound fetch が失敗する可能性があるが、proxy が
最終宛先を代理解決する構成ではこの transport 単独で SSRF 境界を保証できないため fail-closed を優先する。
将来 proxy が必要になった場合は、信頼済み proxy と最終接続先の双方を保証できる別設計として扱う。

## Completion
- **Completed At**: 2026-08-10
- **Summary**:
  `safehttp` の outbound request を DNS 検査済み IP への直接接続に限定し、環境 proxy による
  SSRF 境界の迂回を防いだ。CGNAT 判定を `100.64.0.0/10` 全体の CIDR membership に修正し、
  SCL scenario、OAuth2 Architecture、回帰テスト、SCL 派生物を同期した。CodeQL alert 14 は
  custom transport を認識しない `go/request-forgery` の false positive として既に dismiss 済みだった。
- **Verification Results**:
  - `just check-scl` - passed（27 SCL files）
  - `just scl-render` - passed（再実行後も派生物は同期）
  - `just test-go-package ./backend/shared/security/safehttp` - passed
  - `just test-go-package ./backend/oauth2/client/cimd_http` - passed
  - `just test-go-package ./backend/shared/security/tokens_jose` - passed
  - `just check` - passed（SCL、work item、ID、Architecture、traceability）
  - `just verify-go` - passed（lint、全 Go race tests）
  - `just verify` - passed（全 workspace checks）
  - `git diff --check` - passed
  - GitHub API - CodeQL alert 14 は `dismissed_reason: false positive` を確認（追加コメントなし）
