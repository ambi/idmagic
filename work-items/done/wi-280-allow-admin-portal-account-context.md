---
status: completed
authors: [tn]
risk: medium
created_at: 2026-07-24
depends_on: [wi-275-account-self-service-api-scopes]
change_kind: bugfix
initial_context:
  scl:
    Authentication:
      - interfaces.GetAccountContext
      - authorization.policies.PortalClientReadAccountContext
      - scenarios.browser bootstrap contextは認証状態とCSRF境界を保持する
  source: [backend/shared/http/support_http/auth.go]
  tests: [backend/shared/http/support_http/auth_test.go]
  stop_before_reading: [frontend]
affected_spec:
  - { context: Authentication, kind: interface, element: GetAccountContext }
  - { context: Authentication, kind: authorization_policy, element: PortalClientReadAccountContext }
  - { context: Authentication, kind: scenario, element: browser bootstrap contextは認証状態とCSRF境界を保持する }
---

# 管理ポータルのアクセストークンで共通アカウントコンテキストを取得できるようにする

## Motivation

`/api/auth/account` は管理 shell とアカウント portal が共有する bootstrap endpoint だが、
WI-275 の account scope enforcement で `idmagic.account` 専用 route に分類された。そのため、
正規の `idmagic.admin` scope を持つ管理ポータルのログイン直後に 403 が返り、管理画面へ入れない。

## Scope

- `spec/contexts/authentication.yaml` の `interfaces.GetAccountContext`、`authorization`、`scenarios`。
- `backend/shared/http/support_http/auth.go` の共通 account context scope 判定。
- `backend/shared/http/support_http/auth_test.go` の portal scope 回帰テスト。
- SCL 派生 artifact の同期。

## Out of Scope

- `/api/account/*` を `idmagic.admin` scope へ開放すること。
- 管理画面またはアカウント portal が要求する OIDC scope の変更。
- API token の scope 語彙や発行フローの変更。

## Plan

- `GetAccountContext` を portal 共通 bootstrap 契約として明記し、Bearer token では
  `idmagic.admin`、`idmagic.account`、`account:read` のいずれかを許可する。
- `/api/account/*` の既存 cross-portal 防止は維持し、例外を `/api/auth/account` だけに限定する。
- 固定 scope 集合と固定 route の判定であり複雑な入力文法ではないため fuzz/property test は追加せず、
  表駆動 adapter test で許可・拒否境界を検証する。

## Tasks

- [x] T001 [SCL] `GetAccountContext` の interface・authorization・scenario を更新し、`just yaml-check-scl` で検証した。
- [x] T002 [Adapter] portal 共通 scope 判定を実装した。RED: `TestAccountContextAcceptsBothPortalScopes` が `idmagic.admin` に対する `insufficient scope: account:read` で先に失敗することを確認（scenario `browser bootstrap contextは認証状態とCSRF境界を保持する`）→ GREEN。
- [x] T003 [Derived] `just scl-render` で SCL 派生 artifact を再生成した。
- [x] T004 [Verify] package test、SCL/WI/ID検証、`just verify` を実行した。

## Verification

- `just yaml-check-scl`
- `just yaml-check-work-items`
- `just check-ids`
- `just test-go-package ./backend/shared/http/support_http`
- `just scl-render`
- `just verify`

## Risk Notes

認可境界の変更であるため risk は medium。許可対象を共通 bootstrap endpoint のみに限定し、
`/api/account/*` に対する `idmagic.admin` の拒否を回帰テストで維持する。

## Completion

- **Completed At**: 2026-07-24
- **Summary**:
  `/api/auth/account` を管理 portal・account portal・自己管理 API client の共通 bootstrap
  endpoint として明文化し、`idmagic.admin`、`idmagic.account`、`account:read` のいずれかで
  取得できるよう修正した。scope 判定は `hasRequiredAccountScope` に分離し、共通 endpoint
  だけの例外を認証フロー本体から隔離した。
- **Affected Guarantees State**:
  管理 portal は正規の `idmagic.admin` token で account context を取得できる。
  `/api/account/*` は引き続き `idmagic.admin` だけでは利用できず、cross-portal 境界を維持する。
  無関係な scope、無効 token、未認証・認証途中の session は従来どおり拒否する。
- **Verification Results**:
  - `just yaml-check-scl` — passed（23 SCL file、18 context reference）
  - `just yaml-check-work-items` — passed（279 work item）
  - `just check-ids` — passed（412 record ID）
  - `just test-go-package ./backend/shared/http/support_http` — passed
  - `just scl-render` — passed（SCL派生 artifactを再生成）
  - `just verify` — passed（Go lint/race tests、UI 77 files・425 tests、typecheck、production build）
- **Evidence**:
  - 実行日: 2026-07-24
  - 実行環境: macOS local workspace
  - 実行主体: Codex
  - 対象ソース版: `a29e71f7` からの作業ツリー
  - 保存先: SCL、生成済み仕様 artifact、Go adapter test、本 work item
  - 要約値: `idmagic.admin` / `idmagic.account` / `account:read` は共通 endpoint で許可、
    無関係 scope と `/api/account/profile` への `idmagic.admin` は拒否
