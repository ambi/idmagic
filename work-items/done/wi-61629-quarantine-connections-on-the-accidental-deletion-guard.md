---
status: completed
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-09-24
risk_notes: |
  ガードが効かないと、誤った設定や取り込みによる一括の無効化と削除が下流へそのまま届き、外部システムのアカウントを失わせる。逆にガードが誤って効くと、正当な deprovision が止まり、退職者のアカウントが下流で有効なまま残る。
priority: p2
depends_on: []
change_kind: bugfix
evidence_policy: risk-based-v3
documentation_impact:
  level: release_note
  reason: 設定済みの誤削除ガードの閾値が、照合で実際に効くようになる。閾値を設定している運用者は、超えたときに接続が隔離されることを知る必要がある。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-61629-quarantine-connections-on-the-accidental-deletion-guard.md }
initial_context:
  specification:
    - docs/domain/provisioning/scenarios.feature.md#REQ-PROVISIONING-011
    - docs/domain/provisioning/internals.md
    - docs/domain/provisioning/glossary.md
  typespec:
    - IdMagic.Contract.DeprovisionPolicy
    - IdMagic.Contract.ConnectionQuarantined
  source:
    - backend/provisioning/domain/mapping.go
    - backend/provisioning/domain/reconcile.go
    - backend/provisioning/usecases/reconcile.go
    - backend/provisioning/usecases/job_handler.go
    - backend/provisioning/usecases/admin.go
    - backend/provisioning/usecases/execute_task.go
    - backend/provisioning/module.go
    - backend/cmd/idmagic-worker/worker.go
  tests:
    - backend/provisioning/domain/reconcile_test.go
    - backend/provisioning/e2e_reconcile_test.go
    - backend/provisioning/e2e_full_resync_test.go
  stop_before_reading: [frontend, backend/provisioning/client_scim, backend/provisioning/db_postgres]
primary_use_cases:
  - id: reconcile-guard-quarantines
    requirement: REQ-PROVISIONING-011
    observable_result: 閾値を超える deprovision を計画した照合は、プロビジョニングタスクを作らず、接続を隔離して ConnectionQuarantined を発行する。
    unit_test: { path: backend/provisioning/domain/reconcile_test.go, name: TestPlanReconciliation_QuarantinesInsteadOfDeprovisioningOverTheGuard, task: test-go-race }
    e2e_test: { path: backend/provisioning/e2e_reconcile_test.go, name: TestE2E_ReconcileOverTheGuardQuarantinesWithoutDeprovisioning, task: test-go-race }
    unit_fault_model: 照合の計算が、1 回あたりの上限で打ち切った後の件数を閾値と比べ、上限より多い deprovision を見逃す。
    e2e_fault_model: 照合のユースケースが隔離の理由を受け取っても、接続を保存せずイベントを発行しない。
affected_spec:
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-011 }
---

# 照合が誤削除ガードの閾値を超える deprovision を計画したら、何も作らずに接続を隔離する

## 動機

`REQ-PROVISIONING-011` は、1 回の同期で deactivate または delete の対象が `accidental_deletion_count_threshold` を超えたら、deprovision を実行せず `ConnectionQuarantined` を発行して接続を隔離し、`notification_email` へ通知すると宣言する。

`DeprovisionPolicy` は `accidental_deletion_count_threshold` と `accidental_deletion_percent_threshold` を保存し検証するが、それを読む処理は無い。
そのため、閾値を設定しても誤った一括の無効化や削除は止まらず、下流のアカウントがそのまま無効化または削除され得る。

隔離の状態遷移と `ConnectionQuarantined` の発行は、連続失敗の経路で [[wi-85060-publish-provisioning-lifecycle-events]] が実装した。
[[wi-92540-guarantee-provisioning-deliveries-by-reconciliation]] は、照合が作る無効化と削除にもガードを効かせると決めた。

## 対象範囲

- 照合の計画で deprovision を数え、閾値を超えたらその照合では何も作らずに接続を隔離する。
- `REQ-PROVISIONING-011` の具体例を、ガードの置き場所である照合の文面へ合わせ、割合の閾値と閾値ちょうどの例を足す。`internals.md` にガードの節を書く。

## 対象外

- 隔離の解除（`ResumeProvisioningConnection`）。既に実装されている。
- 隔離を `notification_email` へ通知すること。連続失敗による隔離も通知しておらず、隔離一般の欠陥として [[wi-33641-notify-quarantine-to-the-notification-email]] が扱う。`EX-PROVISIONING-011-01` と `-02` は通知の行を含むので、台帳では同項目を `blocked_by` とする。
- 隔離した接続のプロビジョニングタスクをディスパッチャーとジョブハンドラーが送り続けること。連続失敗による隔離にも共通する欠陥として [[wi-27022-stop-provisioning-on-quarantined-connections]] が扱う。

## 設計

この機能は、誤った設定や取り込みという例外的な状況のための安全装置である。
機能の強さよりも、実装が小さく、読みやすく、通常の照合の性能を変えないことを優先する。

### ガードを照合だけに置く

deprovision を作る経路は 4 つある。

| 経路 | 作り方 | ガード |
| --- | --- | --- |
| 照合 | 差分を計画し、`deactivate`、`delete`、猶予期間つき削除の予約を作る | 置く |
| Full Resync | スコープ内の対象へ `update` を作る | 置かない。管理者が明示的に起動する操作である |
| 書き込み時の捕捉 | 変更のたびに 1 件ずつ作る | 置かない。数える単位を持たない |
| 猶予期間つき削除の実体化 | ディスパッチャーが期限に達した予約を `delete` にする | 置かない。予約を作る時点で照合が数えている |

照合は、捕捉が漏れる経路の変更も、スコープや方針の設定変更による一括の差分も、1 回の計画としてまとめて扱う唯一の経路である。
全 User とリンクをすでに読み込んで 1 人ずつ計画しているので、数えるための追加の読み取りは無い。

採らなかった案と理由は次のとおり。

| 案 | 採らなかった理由 |
| --- | --- |
| Full Resync にも置く | Full Resync は `deactivate` と `delete` を作らず、無効な User への `update` を数える専用の規則と、管理 API へのリンクの保存先の配線が要る |
| Full Resync を照合の計算へ載せ替える | ガードのために Full Resync の作り直しを抱え込む |
| ディスパッチャーで送る前に数える | 作成済みのタスクの処分という新しい規範の判断が要る |

### 判定

| 項目 | 決定 |
| --- | --- |
| 数える deprovision | 計画した `deactivate` と `delete`。猶予期間つき削除の予約を含む。決着を待つタスクがあるために飛ばした User は数えない |
| 件数の閾値 | 件数が閾値より大きければ超過。等しければ超過ではない |
| 割合の閾値 | 件数が接続の User のリンク数の閾値 % より大きければ超過 |
| 両方の閾値 | どちらかを超えたら超過。どちらも未設定ならガードは無い |
| 上限との関係 | 1 回あたりの上限で打ち切る前の件数を比べる |
| 超えたとき | その照合では何も作らない。接続を隔離し、保存の後に `ConnectionQuarantined` を発行する |

超えたときに deprovision 以外も作らないのは、隔離が接続のプロビジョニング全体を止める状態だからである。
止めたものはどこにも残らないので、解除の後は次の照合があるべき状態から作り直す。

### 型と操作

```go
// domain
type ReconcilePlan struct {
    Actions          []ReconcileAction
    QuarantineReason string // 空でなければガードを超えており、Actions は空
}
func PlanReconciliation(in ReconcileInput) ReconcilePlan

// usecases
func quarantineConnection(ctx context.Context, repo ports.ProvisioningConnectionRepository, emitFn func(spec.DomainEvent), conn *domain.ProvisioningConnection, reason string, now time.Time) error
```

`quarantineConnection` は、連続失敗の経路が持っていた「隔離して保存し、保存の後に発行する」処理を取り出したものであり、両方の経路が使う。
`ReconcileDeps` に `Emit` を足し、`Module.ReconcileDeps` の引数にして、worker が発行先を必ず決めるようにする。
時刻は `now` 引数から、イベントの発行は `Emit` ポートから入る。

## 計画

1. `spec-change` で具体例と `internals.md` を更新する。
2. 構造変更: `PlanReconciliation` の戻り値を `ReconcilePlan` にする（振る舞いは変えない）。
3. 振る舞い: 照合の計画 → 照合のユースケース → worker の配線。

## タスク

- [x] T001 [Spec] `EX-PROVISIONING-011-01` と `-02` を照合の文面へ合わせ、`-03`（割合）と `-04`（閾値ちょうど）を足した。`internals.md` へガードの節を書いた。
- [x] T002 [Structure] `PlanReconciliation` の戻り値を `ReconcilePlan` にした。`mise run test-go-package -- ./backend/provisioning/...` が GREEN。
- [x] T003 [Acceptance] E2E の RED を確認した。`TestE2E_ReconcileOverTheGuardQuarantinesWithoutDeprovisioning` が `created = 6, want 0` で失敗した。recipe は `mise run test-go-test -- ./backend/provisioning TestE2E_Reconcile`。
- [x] T004 [Domain] 照合の計画で数えて判定した。recipe は `mise run test-go-test -- ./backend/provisioning/domain TestPlanReconciliation`。
- [x] T005 [UseCase] 照合で隔離した。連続失敗の経路から `quarantineConnection` を取り出して共有した。recipe は `mise run test-go-package -- ./backend/provisioning/usecases`。
- [x] T006 [Adapter] `Module.ReconcileDeps` の引数と worker の配線を足した。
- [x] T007 [Evidence] `mise run test-go-mutation -- backend/provisioning/domain` と、配線を外すフォールト注入。
- [x] T008 [Verify] `mise run verify` が成功した。

## 検証

- `mise run test-go-package -- ./backend/provisioning/...`
- `mise run verify`

## リスク

閾値の判定を件数だけで行うと、小さなテナントでは割合の閾値が効かず、大きなテナントでは件数の閾値が正当な一括処理を止める。両方の閾値の組み合わせを domain のテストで固定する。

ガードが誤って効く側の誤りは、正当な deprovision を止める。閾値ちょうどの件数と、作成と更新を数えないことを、超過しない側の例として固定する。

## 完了

- **Completed At**: 2026-09-25
- **Summary**:
  `mise run spec-diff` が示す規範の差分は `REQ-PROVISIONING-011` だけである。
  `EX-PROVISIONING-011-01` と `-02` は、誤削除ガードが検出する場所を「full resync」から「照合」へ改めた。
  `EX-PROVISIONING-011-03`（割合の閾値を超える）と `-04`（閾値ちょうどでは隔離しない）を足した。
  実装では、照合の計画が上限で打ち切る前の deprovision を数え、件数または割合の閾値を超えたら何も作らずに隔離の理由を返す。
  照合のユースケースは最新の接続を隔離し、保存の後に `ConnectionQuarantined` を発行する。
  隔離の保存と発行は連続失敗の経路と共有する `quarantineConnection` に取り出した。
  Full Resync、書き込み時の捕捉、猶予期間つき削除の実体化にはガードを置かない。その理由は `internals.md` の「誤削除ガード」節にある。
  `EX-PROVISIONING-011-01` は通知の行を含むので、台帳の `blocked_by` を [[wi-33641-notify-quarantine-to-the-notification-email]] へ付け替えた。
- **Primary Use Case Evidence**:
  - id: reconcile-guard-quarantines
    unit_red: 照合の計画が隔離の理由を返さず deactivate を作ったため、TestPlanReconciliation_QuarantinesInsteadOfDeprovisioningOverTheGuard が失敗した。割合の閾値と猶予期間つき削除の予約を数えるテストも、同じく隔離の理由が空で失敗した。
    e2e_red: 照合が閾値を超えても 6 件の deactivate を作ったため、TestE2E_ReconcileOverTheGuardQuarantinesWithoutDeprovisioning が created = 6, want 0 で失敗した。
    unit_fault_injection: mise run test-go-mutation で reconcile.go の変異体 23 件をすべて検出した。閾値の境界、割合の乗算、上限の比較、deprovision の数え上げの変異を含む。生き残った 2 件は connection.go と task.go の既存の検証であり、この変更の範囲外である。
    e2e_fault_injection: Module.ReconcileDeps から Emit の配線を外すと ConnectionQuarantined が発行されず、照合のユースケースの隔離の分岐を外すと接続の health が ok のままになり、どちらも TestE2E_ReconcileOverTheGuardQuarantinesWithoutDeprovisioning が失敗した。
- **Change-Resistance Results**:
  変異テストと、配線と分岐の手作業のフォールト注入は、上の Primary Use Case Evidence のとおりである。
  閾値ちょうどでは隔離しないこと（`TestE2E_ReconcileAtTheGuardStillDeprovisions`、`TestPlanReconciliation_DeprovisionsAtTheCountThreshold`）と、作成と更新を数えないこと（`TestPlanReconciliation_DoesNotCountProvisioningTowardTheGuard`）を、ガードが誤って効く側の対照として固定した。
- **Verification Results**:
  - `mise run test-go-changed` - 成功
  - `mise run lint-go` - 成功
  - `mise run check-spec` - 成功
  - `mise run verify` - 成功
