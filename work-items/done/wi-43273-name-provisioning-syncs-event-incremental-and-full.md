---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-25
risk_notes: |
  文脈を見ずに置換すると、ハッシュや属性の突き合わせを指す「照合」まで書き換え、別の意味の文を壊す。Provisioning の同期を指す箇所だけを改める。
priority: p2
depends_on: []
change_kind: docs
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 文書、コメント、管理画面の表示名を改めるだけで、製品の振る舞い、公開契約、設定は変わらない。
  references: []
initial_context:
  specification:
    - docs/domain/provisioning/glossary.md
    - docs/domain/provisioning/internals.md
    - docs/domain/provisioning/scenarios.feature.md
    - docs/domain/scenarios.feature.md#REQ-PLATFORM-003
  typespec: []
  source:
    - frontend/src/features/admin-applications/AdminApplicationProvisioning.i18n.ts
  tests: []
  stop_before_reading: [backend/provisioning/db_postgres, spec]
affected_spec:
  - { path: docs/domain/scenarios.feature.md, requirement: REQ-PLATFORM-003 }
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-003 }
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-004 }
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-005 }
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-011 }
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-013 }
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-016 }
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-017 }
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-019 }
---

# Provisioning の 3 つの同期を、イベント同期、インクリメンタル同期、フル同期と呼ぶ

## 動機

Provisioning は、下流へ反映するプロビジョニングタスクを 3 つの経路で作る。
文書はそれぞれを「書き込み時の捕捉」「照合」「Full Resync」と呼んでいたが、どれも何をするのかが伝わらない。

- 「照合」は reconciliation の訳だが、日本語では「突き合わせる」ことしか表さない。リポジトリの他の場所では、ハッシュや jti を突き合わせる本来の意味でも使っている。
- 「書き込み時の捕捉」は、実装の都合（コミットの後で捕捉ポートを呼ぶ）を名前にしており、利用者から見た振る舞いを表さない。
- 「Full Resync」だけが英語のまま残り、管理画面の日本語 UI は「全体再同期」と別の訳を使っている。「再」に固有の意味は無い。

## 対象範囲

次の対応で、Provisioning の日本語の文書、コード中のコメント、管理画面の表示名、未完了の work item を改める。

| 今の呼び名 | 英語 | 日本語 | 何をするか |
| --- | --- | --- | --- |
| 書き込み時の捕捉 | Event Sync | イベント同期 | User や割り当ての変更を受けて、その対象のプロビジョニングタスクを作る |
| 照合（Reconciliation） | Incremental Sync | インクリメンタル同期 | 周期ごとに、あるべき状態と反映済みの状態の差分をプロビジョニングタスクにする |
| Full Resync | Full Sync | フル同期 | 管理者の操作で、適用範囲の全対象へ `update` のプロビジョニングタスクを作る |

## 対象外

- コードの識別子（`Reconcile*`、`Capture*`、`FullResync*`）、公開 API（`StartFullResync`、`FullResyncCompleted`）、環境変数（`PROVISIONING_RECONCILE_INTERVAL`）、データベースのテーブル名。識別子としては一般的な語であり、公開契約を変えてまで改める利点が無い。用語集が日本語名と識別子の対応を示す。
- 完了済みの work item とリリースノート。当時の記録として残す。
- インクリメンタル同期が全件を読む問題。[[wi-20579-read-only-changes-in-incremental-sync]] が扱う。

## 設計

名前は「カタカナ＋同期」にそろえる。
Microsoft Entra ID は、全件を評価する初回サイクル（initial cycle）と、基準値以降の差分だけを扱う増分サイクル（incremental cycle）を区別しており、インクリメンタル同期とフル同期はこの区別に対応する。
イベント同期は変更イベントを受けて 1 件ずつ作る経路で、Entra ID には対応するものが無い。

「部分同期」は「一部の対象だけ」と読めるので採らない。「リコンサイル同期」は運用者に意味が伝わりにくいので採らない。

## タスク

- [x] T001 [Docs] 用語集に 3 つの同期を定義し、Provisioning の文書と横断シナリオの文言を改める。
- [x] T002 [Code] コードのコメントとテストの説明文を改める。
- [x] T003 [UI] 管理画面の表示名（ja と en）を改める。
- [x] T004 [Verify] `mise run verify` と `mise run test-ui-e2e` が成功した。

## 検証

- `mise run check-spec`
- `mise run verify`

## 完了

- **Completed At**: 2026-09-26
- **Summary**:
  `mise run spec-diff` が示す変更は `REQ-PLATFORM-003` と、`REQ-PROVISIONING-003`、`-004`、`-005`、`-011`、`-013`、`-016`、`-017`、`-019` のシナリオの文言である。
  いずれも「書き込み時の捕捉」「照合」「Full Resync」を「イベント同期」「インクリメンタル同期」「フル同期」へ改めたもので、観測できる振る舞いは変わらない。
  用語集は 3 つの同期を Event Sync、Incremental Sync、Full Sync として定義し、コードの識別子（`Capture*`、`Reconcile*`、`FullResync`）との対応を示す。
  管理画面の表示名は ja を「フル同期」、en を「Full sync」にした。
- **Acceptance RED Evidence**:
  - **Test**: `rg '照合|捕捉|Full Resync'` を Provisioning の文書、横断シナリオ、`backend/provisioning`、worker、未完了の work item に対して実行した。
  - **Requirement**: N/A: 用語の変更であり、製品の振る舞いの要件は変わらない。
  - **Observed Failure**: 変更前は 31 ファイルで旧い用語が見つかった。変更後は 0 件である。
  - **Detection Reason**: 同期を指す旧い用語が 1 か所でも残れば、この検索が見つける。ハッシュや属性の突き合わせを指す「照合」は、対象のファイルに含まれないことを事前に確かめた。
- **Unit RED Evidence**:
  - **Test**: N/A: コメントと表示文字列だけの変更で、単体の境界が無い。
  - **Requirement**: N/A: 振る舞いの要件は変わらない。
  - **Observed Failure**: N/A: 代わりに `mise run verify` と `mise run test-ui-e2e` で、変更がテストと表示を壊していないことを確かめた。
  - **Detection Reason**: UI のテストは表示文字列を辞書から参照するため、表示名の変更で失敗しない。
- **Verification Results**:
  - `mise run verify` - 成功
  - `mise run test-ui-e2e` - 成功（39 件）
