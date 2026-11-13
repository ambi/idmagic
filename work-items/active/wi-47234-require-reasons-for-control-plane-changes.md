---
status: pending
authors: [tn]
risk: high
reversibility: irreversible
created_at: 2026-11-14
change_kind: feature
priority: p2
depends_on: [wi-468-system-console-privileged-session-assurance]
affected_spec:
  - { path: docs/modules/tenancy/features/resolution.md, requirement: REQ-TENANCY-011 }
  - { path: docs/modules/tenancy/features/lifecycle.md, requirement: REQ-TENANCY-027 }
  - { path: docs/modules/jobs/features/admin.md, requirement: REQ-JOBS-015 }
  - { path: docs/modules/data-keys/features/lifecycle.md, requirement: REQ-DATAKEYS-003 }
  - { path: docs/modules/data-keys/features/lifecycle.md, requirement: REQ-DATAKEYS-008 }
  - { path: spec/modules/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.SetTenantEndpointStyle }
  - { path: spec/modules/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.DisableTenant }
  - { path: spec/modules/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.EnableTenant }
  - { path: spec/modules/jobs/main.tsp, symbol: IdMagic.Jobs.Operations.CancelSystemJob }
  - { path: spec/modules/data-keys/main.tsp, symbol: IdMagic.DataKeys.Operations.RotateTenantDataKey }
  - { path: spec/modules/data-keys/main.tsp, symbol: IdMagic.DataKeys.Operations.DisableTenantDataKey }
---

# 影響の大きい制御面の操作に理由と確認を求め、監査に残す

## 動機

テナントの停止、再開、正規ロケーションの切替、別テナントのジョブの取消し、DEK のローテーションと無効化は、対象テナントの認証やデータの読み出しに大きく影響する。
現在の要求は理由を持たず、監査イベントにも理由と再認証の方式が残らない。
`JobCanceled` は操作者すら持たない。
誰がどの判断で変更したかを、監査から再構成できない。
[[wi-468-system-console-privileged-session-assurance]] から切り出した（2026-11-14 に利用者と決定）。

## 対象範囲

- `SetTenantEndpointStyle`、`DisableTenant`、`EnableTenant`、`CancelSystemJob`、`RotateTenantDataKey`、`DisableTenantDataKey` に、空でない操作理由を要求する。`SetTenantEndpointStyle` 以外は現在本文を持たないので、本文を加える。
- 理由の欠落は `400 invalid_request` で拒否し、対象の状態を変えない。
- API アクセストークンから届く操作（`CancelSystemJob`）も理由を要求する。
- 監査イベントへ、操作者、対象テナント、操作理由、再認証の方式（`amr`）を記録する。
- システムコンソールは、対象の `tenant_id`、realm、操作、影響、入力した理由を確認画面に示してから送る。日英で実装する。

## 対象外

- 理由の内容の検証（チケット番号の形式など）と、外部のチケットシステムとの連携。
- テナント管理者の操作へ理由を広げること。

## 設計

操作理由を監査イベントだけへ後付けする案は採らない。
要求の時点で理由を必須にしなければ、誰がどの判断で影響の大きい変更を行ったかを監査から再構成できないためである。

理由の長さの上限、監査イベントの形（既存のイベントへ欄を足すか、操作の監査の共通の欄にするか）は、着手時に決める。

## 計画

1. 理由の欄、上限、監査イベントの形を仕様に書く。
2. 要求、ユースケース、監査イベントへ理由と方式を通す。
3. 確認画面と理由の入力を実装する。

## タスク

- [ ] T001 [Spec] 理由の欄、上限、拒否、監査イベントの欄を規範化する。
- [ ] T002 [Acceptance] 理由のない要求が現在は受け付けられることを観測し、RED にする。
- [ ] T003 [App] 理由を要求モデル、ユースケース、監査イベントへ伝搬する。
- [ ] T004 [UI] 対象と影響の確認と理由の入力を日英で実装する。
- [ ] T005 [Verify] 拒否で対象の状態が変わらないことを確かめ、変更を検証する。

## 検証

- 各操作の受け入れテストで、理由の欠落の拒否と、監査イベントの理由と方式を確かめる。
- `mise run test-ui-e2e-file`：システムコンソールの確認画面と理由の入力。
- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>` とする。

## リスク

本文を必須にするので、既存の呼び出し元は理由を送るまで失敗する。公開前（`unpublished`）なので移行措置は設けない。
