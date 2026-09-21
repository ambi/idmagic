---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-21
priority: p2
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-015 }
---

# テナントをまたぐ Provisioning 管理参照を公開 HTTP 境界で NotFound にする

## 動機

`EX-PROVISIONING-015-01` は tenant-b の管理者が tenant-a の接続または配信を参照した場合に NotFound を返すと宣言する。
リポジトリの tenant 分離テストはあるが、管理者の認証・テナント解決・HTTP 応答までを通した観測がない。

## 対象範囲

- tenant-a の Provisioning 接続と配信を、tenant-b の認証済み管理者が HTTP で参照する受け入れテストを追加する。
- 接続と配信の双方で NotFound と情報非開示を検証する。
- 現行の公開境界が別の結果を返す場合は、契約に合わせて修正する。

## 対象外

- API トークンのクロステナント拒否。[[wi-37560-return-access-denied-for-cross-tenant-provisioning-api-tokens]] が扱う。
- 他 Context のテナント境界テスト。

## 設計

この規範の actor は API トークンではなく tenant-b の管理者である。
テストは両テナントに管理者セッションを作り、正式な realm 入口を通す。保存層だけのテストは middleware や応答変換を迂回するため受け入れ証拠にしない。

## 計画

1. tenant ごとの管理者セッションを組み立てる既存 fixture を選ぶ。
2. 接続と配信の NotFound RED を確認する。
3. 必要な認証・テナント配線またはハンドラーを修正し検証する。

## タスク

- [ ] T001 [Readiness] 複数 tenant の管理者セッション fixture を決める。
- [ ] T002 [Acceptance] `EX-PROVISIONING-015-01` の接続・配信参照 RED を確認する。
- [ ] T003 [Adapter] NotFound と情報非開示を実装または確認する。
- [ ] T004 [Verify] 対象パッケージと `mise run verify` を実行する。

## 検証

- `mise run test-go-package -- ./backend/shared/http/server_http`
- `mise run verify`

## リスク

別 tenant のリソースが見えるとテナント境界を破る。接続と配信を別々に観測し、一方だけのフィルタでは成立しないようにする。
