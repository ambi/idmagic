---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-04
priority: p1
depends_on: [wi-17076-limit-specifications-to-external-contracts-written-once, wi-89346-detect-unspecified-behavior-mechanically]
change_kind: docs
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  reason: IdManagement の 10 機能の機能仕様の書き方を改め、実装がすでに返している結果を要件に書き足すだけで、製品の振る舞い、公開 API、設定、運用手順は変えないので、リリースの読者へ知らせる差分がない。
  references: []
initial_context:
  specification:
    - SPECIFICATION_FORMAT.md
    - docs/development/specification-first-workflow.md#仕様にない振る舞いの分類
    - docs/domain/identity-management/user/README.md
    - docs/domain/identity-management/user/lifecycle.md
  typespec:
    - spec/contexts/identity-management/models.tsp
    - spec/contexts/identity-management/main.tsp
  source:
    - tools/check/src/unspecified-vocabulary.ts
    - tools/check/src/specification-doc.ts
    - tools/check/unspecified-vocabulary-debt.json
    - backend/idmanagement/agent/usecases/admin_agents.go
    - backend/idmanagement/usecases/data_export.go
    - backend/idmanagement/group/usecases/dynamic_groups.go
    - backend/idmanagement/user/handlers_http/admin_user_import_handler.go
    - backend/jobs/usecases/runner.go
  tests:
    - backend/idmanagement/handlers_http/export_refusal_effects_test.go
    - backend/idmanagement/user/usecases/user_import_examples_test.go
  stop_before_reading: [frontend]
affected_spec:
  - { path: docs/domain/identity-management/user/README.md, requirement: REQ-IDMANAGEMENT-047 }
  - { path: docs/domain/identity-management/account/README.md, requirement: REQ-IDMANAGEMENT-002 }
  - { path: docs/domain/identity-management/account/README.md, requirement: REQ-IDMANAGEMENT-003 }
  - { path: docs/domain/identity-management/account/README.md, requirement: REQ-IDMANAGEMENT-016 }
  - { path: docs/domain/identity-management/account/README.md, requirement: REQ-IDMANAGEMENT-017 }
  - { path: docs/domain/identity-management/account/README.md, requirement: REQ-IDMANAGEMENT-018 }
  - { path: docs/domain/identity-management/account/README.md, requirement: REQ-IDMANAGEMENT-019 }
  - { path: docs/domain/identity-management/account/README.md, requirement: REQ-IDMANAGEMENT-051 }
  - { path: docs/domain/identity-management/account/README.md, requirement: REQ-IDMANAGEMENT-052 }
  - { path: docs/domain/identity-management/account/README.md, requirement: REQ-IDMANAGEMENT-053 }
  - { path: docs/domain/identity-management/account/README.md, requirement: REQ-IDMANAGEMENT-054 }
  - { path: docs/domain/identity-management/admin-access/README.md, requirement: REQ-IDMANAGEMENT-014 }
  - { path: docs/domain/identity-management/admin-access/README.md, requirement: REQ-IDMANAGEMENT-025 }
  - { path: docs/domain/identity-management/agent/README.md, requirement: REQ-IDMANAGEMENT-009 }
  - { path: docs/domain/identity-management/agent/README.md, requirement: REQ-IDMANAGEMENT-073 }
  - { path: docs/domain/identity-management/agent/README.md, requirement: REQ-IDMANAGEMENT-074 }
  - { path: docs/domain/identity-management/agent/README.md, requirement: REQ-IDMANAGEMENT-075 }
  - { path: docs/domain/identity-management/agent/README.md, requirement: REQ-IDMANAGEMENT-076 }
  - { path: docs/domain/identity-management/agent/README.md, requirement: REQ-IDMANAGEMENT-077 }
  - { path: docs/domain/identity-management/agent/README.md, requirement: REQ-IDMANAGEMENT-078 }
  - { path: docs/domain/identity-management/agent/README.md, requirement: REQ-IDMANAGEMENT-081 }
  - { path: docs/domain/identity-management/agent/README.md, requirement: REQ-IDMANAGEMENT-082 }
  - { path: docs/domain/identity-management/csv-transfer/README.md, requirement: REQ-IDMANAGEMENT-034 }
  - { path: docs/domain/identity-management/csv-transfer/README.md, requirement: REQ-IDMANAGEMENT-035 }
  - { path: docs/domain/identity-management/csv-transfer/README.md, requirement: REQ-IDMANAGEMENT-036 }
  - { path: docs/domain/identity-management/csv-transfer/README.md, requirement: REQ-IDMANAGEMENT-037 }
  - { path: docs/domain/identity-management/csv-transfer/README.md, requirement: REQ-IDMANAGEMENT-038 }
  - { path: docs/domain/identity-management/csv-transfer/README.md, requirement: REQ-IDMANAGEMENT-080 }
  - { path: docs/domain/identity-management/data-export/README.md, requirement: REQ-IDMANAGEMENT-039 }
  - { path: docs/domain/identity-management/data-export/README.md, requirement: REQ-IDMANAGEMENT-040 }
  - { path: docs/domain/identity-management/data-export/README.md, requirement: REQ-IDMANAGEMENT-041 }
  - { path: docs/domain/identity-management/data-export/README.md, requirement: REQ-IDMANAGEMENT-079 }
  - { path: docs/domain/identity-management/data-export/README.md, requirement: REQ-IDMANAGEMENT-086 }
  - { path: docs/domain/identity-management/data-export/README.md, requirement: REQ-IDMANAGEMENT-087 }
  - { path: docs/domain/identity-management/data-export/README.md, requirement: REQ-IDMANAGEMENT-088 }
  - { path: docs/domain/identity-management/dynamic-group/README.md, requirement: REQ-IDMANAGEMENT-020 }
  - { path: docs/domain/identity-management/dynamic-group/README.md, requirement: REQ-IDMANAGEMENT-021 }
  - { path: docs/domain/identity-management/dynamic-group/README.md, requirement: REQ-IDMANAGEMENT-022 }
  - { path: docs/domain/identity-management/dynamic-group/README.md, requirement: REQ-IDMANAGEMENT-023 }
  - { path: docs/domain/identity-management/dynamic-group/README.md, requirement: REQ-IDMANAGEMENT-065 }
  - { path: docs/domain/identity-management/dynamic-group/README.md, requirement: REQ-IDMANAGEMENT-066 }
  - { path: docs/domain/identity-management/dynamic-group/README.md, requirement: REQ-IDMANAGEMENT-067 }
  - { path: docs/domain/identity-management/dynamic-group/README.md, requirement: REQ-IDMANAGEMENT-068 }
  - { path: docs/domain/identity-management/dynamic-group/README.md, requirement: REQ-IDMANAGEMENT-069 }
  - { path: docs/domain/identity-management/dynamic-group/README.md, requirement: REQ-IDMANAGEMENT-070 }
  - { path: docs/domain/identity-management/dynamic-group/README.md, requirement: REQ-IDMANAGEMENT-071 }
  - { path: docs/domain/identity-management/group-csv/README.md, requirement: REQ-IDMANAGEMENT-008 }
  - { path: docs/domain/identity-management/group-csv/README.md, requirement: REQ-IDMANAGEMENT-026 }
  - { path: docs/domain/identity-management/group-csv/README.md, requirement: REQ-IDMANAGEMENT-027 }
  - { path: docs/domain/identity-management/group-csv/README.md, requirement: REQ-IDMANAGEMENT-028 }
  - { path: docs/domain/identity-management/group-csv/README.md, requirement: REQ-IDMANAGEMENT-029 }
  - { path: docs/domain/identity-management/group-csv/README.md, requirement: REQ-IDMANAGEMENT-030 }
  - { path: docs/domain/identity-management/group-csv/README.md, requirement: REQ-IDMANAGEMENT-031 }
  - { path: docs/domain/identity-management/group-csv/README.md, requirement: REQ-IDMANAGEMENT-072 }
  - { path: docs/domain/identity-management/group/README.md, requirement: REQ-IDMANAGEMENT-015 }
  - { path: docs/domain/identity-management/group/README.md, requirement: REQ-IDMANAGEMENT-024 }
  - { path: docs/domain/identity-management/group/README.md, requirement: REQ-IDMANAGEMENT-060 }
  - { path: docs/domain/identity-management/group/README.md, requirement: REQ-IDMANAGEMENT-061 }
  - { path: docs/domain/identity-management/group/README.md, requirement: REQ-IDMANAGEMENT-062 }
  - { path: docs/domain/identity-management/group/README.md, requirement: REQ-IDMANAGEMENT-063 }
  - { path: docs/domain/identity-management/group/README.md, requirement: REQ-IDMANAGEMENT-064 }
  - { path: docs/domain/identity-management/group/README.md, requirement: REQ-IDMANAGEMENT-084 }
  - { path: docs/domain/identity-management/group/README.md, requirement: REQ-IDMANAGEMENT-085 }
  - { path: docs/domain/identity-management/roles/README.md, requirement: REQ-IDMANAGEMENT-032 }
  - { path: docs/domain/identity-management/roles/README.md, requirement: REQ-IDMANAGEMENT-033 }
  - { path: docs/domain/identity-management/user-csv/README.md, requirement: REQ-IDMANAGEMENT-004 }
  - { path: docs/domain/identity-management/user-csv/README.md, requirement: REQ-IDMANAGEMENT-006 }
  - { path: docs/domain/identity-management/user-csv/README.md, requirement: REQ-IDMANAGEMENT-007 }
  - { path: docs/domain/identity-management/user-csv/README.md, requirement: REQ-IDMANAGEMENT-055 }
  - { path: docs/domain/identity-management/user-csv/README.md, requirement: REQ-IDMANAGEMENT-056 }
  - { path: docs/domain/identity-management/user-csv/README.md, requirement: REQ-IDMANAGEMENT-057 }
  - { path: docs/domain/identity-management/user-csv/README.md, requirement: REQ-IDMANAGEMENT-058 }
  - { path: docs/domain/identity-management/user-csv/README.md, requirement: REQ-IDMANAGEMENT-059 }
---

# IdManagement の残りの機能仕様を、EARS 形式の要件だけで書く方式へ書き直す

## 動機

wi-17076 は、仕様に外部から観測できる振る舞いだけを EARS 形式の要件として一か所に書く方式を定め、`user` で試した。
試行では `user` の 4 文書が 719 行から 393 行になり、未記載の振る舞いが 1 件見つかった。
IdManagement の残りの 10 機能は、新しい構成（機能ノード）へは移ったが、書き方は旧いままである。

| 旧い書き方の名残 | 残る機能の数 |
| --- | --- |
| 操作ごとの要約表（「成功時の作用」など） | 10 |
| 担保手段、要判断、話題の索引の欄 | 10 |

wi-89346 の `check-unspecified-vocabulary` は、IdManagement のエラーコードとドメインイベント 72 件のうち 17 件を、どの要件にも現れないものとして許容リストに載せた。
多くは、廃止した要約表にだけ書かれている。

## 対象範囲

- account、admin-access、agent、csv-transfer、data-export、dynamic-group、group、group-csv、roles、user-csv の機能仕様を、`user` と同じ方式で書き直す。
  - 要件の ID は変えず、本文を EARS 形式に改める。
  - 操作ごとの要約表、担保手段、要判断、話題の索引を消し、要約表にだけあった事実は要件へ移す。
  - 状態機械（AgentLifecycle、DataExportLifecycle、DynamicMembershipEvaluationLifecycle）に状態遷移表（マトリクス形式）を加える。
- 書き直しで要件に現れたエラーコードとイベントを、`tools/check/unspecified-vocabulary-debt.json` から消す。
  要件に書けないものは、仕様にない振る舞いの分類に従って (a)(b)(c) に分け、(c) は起票する。
- 書き直しと状態遷移表の作成で見つけた未記載の振る舞いを、同じ分類で扱う。
- 書き直しの前後の行数を測り、完了の節に残す。

## 対象外

- 状態遷移表から予測するモデルベースのテストの追加。
  wi-86874 が扱う。
- (c) に分類した振る舞いの実装の変更。
  分類ごとに起票する work item が扱う。

## 設計

書き方は `SPECIFICATION_FORMAT.md` と、`user` の書き直しの結果に従う。
要件の義務を変える判断は、書き直しに紛れ込ませず、(a)(b)(c) の分類として利用者に示す。

着手時に決めたこと：

- 要件の ID とタイトルは変えない。例の付録の `Rule` のタイトルと一致させ続けるためである。
- 要約表にだけあった事実（拒否が作用を起こさないこと、下流への通知、状態を変えない参照など）は、その操作の要件へ移した。
- 要判断の欄 11 件は、未決定の点がすべて wi-97149 の表に載っているので、欄だけを消した。
- 状態遷移表のセルは、成功しても状態を変えずにイベントを発行する操作（更新、束縛、ダウンロードなど）を、自己遷移と遷移の表の行で書いた。セルの値の `何もしない` はイベントを発行しないことを含むためである。
- Agent の削除は記録を消すので、遷移先として終端の状態 `Deleted` を加えた。以後の操作は 404 `agent_not_found` で拒否する。
- 要件の文が実装とテストの応答と食い違っていた拒否（`InvalidRequestError` と書いていたもの）は、テストが固定している応答に書き直し、TypeSpec が宣言していない点を (c) として起票した。

### 未記載の振る舞いの分類

| # | 見つけた振る舞い | 見つけた工程 | 分類 | 扱い |
| --- | --- | --- | --- | --- |
| 1 | Agent の登録は `AgentRegistered` を、束縛の解除は `AgentCredentialUnbound` を発行する | 許容リスト | (a) | REQ-IDMANAGEMENT-073、074 |
| 2 | 削除した Agent、存在しない Agent、別のテナントの Agent への操作は 404 `agent_not_found` で拒否する | 許容リスト、状態遷移表 | (a) | REQ-IDMANAGEMENT-078、AgentLifecycle の `Deleted` |
| 3 | 空白だけの所有者を指定した Agent の更新は 422 `agent_owner_required` で拒否する | 許容リスト | (a) | REQ-IDMANAGEMENT-075 |
| 4 | メールアドレスの変更の起票は `EmailChangeRequested` を発行する | 許容リスト | (a) | REQ-IDMANAGEMENT-053 |
| 5 | 管理者による必須操作の付与と解除は `UserRequiredActionSet` と `UserRequiredActionCleared` を発行する | 許容リスト | (a) | REQ-IDMANAGEMENT-047 |
| 6 | 規則の保存、有効化、無効化は `DynamicGroupRuleUpdated`、`DynamicGroupRuleEnabled`、`DynamicGroupRuleDisabled` を発行する | 許容リスト | (a) | REQ-IDMANAGEMENT-066、068 |
| 7 | インポートのジョブの参照と適用は、404 `*_import_not_found`、409 `preview_not_ready`、409 `preview_digest_mismatch` を返す。要件は `InvalidRequestError` と書いていた | 許容リスト | (a) と (c) | REQ-IDMANAGEMENT-004、026、029 に現在の応答を書いた。TypeSpec の宣言の漏れは wi-17379 |
| 8 | エクスポートのダウンロードの拒否は 409 `data_export_not_downloadable`、別の種類、Group、テナントのパスは 404 `data_export_not_found` を返す。要件は `InvalidRequestError` と書いていた | 状態遷移表 | (a) と (c) | REQ-IDMANAGEMENT-008、087、088 と例の付録を直した。TypeSpec の宣言の漏れは wi-17379 |
| 9 | `invalid_target`、三つの `*_import_unavailable`、`FederationLinked`、`FederationUnlinked` は、どの経路からも起きない | 許容リスト | (c) | wi-45298。許容リストに残した |
| 10 | エクスポートの生成と動的グループの全件の再評価は、失敗すると `Jobs` の試行の上限まで `queued` に戻る | 状態遷移表 | (a) | REQ-IDMANAGEMENT-086、DataExportLifecycle、DynamicMembershipEvaluationLifecycle |
| 11 | 再試行に戻る生成の失敗と、取り消した後に終わった生成が、`DataExportFailed` または `DataExportSucceeded` を発行する | 状態遷移表 | (c) | wi-74002。REQ-IDMANAGEMENT-086 と表には現在の挙動を書いた |
| 12 | 成果物のダイジェストが一致しないダウンロードは 409 `data_export_not_downloadable` で拒否する | 要約表 | (a) | REQ-IDMANAGEMENT-087 |

候補のうち漏れでないと判断したもの：

- 遷移の表の `DataExportExpired`、`DynamicMembershipEvaluationStarted`、`DynamicMembershipEvaluationFailed` はドメインイベントとして発行しない。遷移の契機の名前であり、`Effects` の列を空にして区別した。
- Agent の登録で、名前の検証が区分の検証より先に行われる順序は (b) とした。どちらの拒否を先に返しても、Agent を作らない結果は同じである。

## 計画

1. 機能ごとに、旧い本文と要約表の事実を要件へ対応づけてから書き直す。
2. 状態機械に状態遷移表を加え、空のセルを分類する。
3. 許容リストの項目を消し、残る項目を分類する。

## タスク

- [x] T001 [Spec] 10 機能の機能仕様を書き直す。検査：`mise run check-spec`。
- [x] T002 [Spec] 三つの状態機械に状態遷移表（マトリクス形式）を加える。検査：`mise run check-spec`。
- [x] T003 [Tooling] 許容リストの項目を消す。RED：`mise run check-unspecified-vocabulary` が、要件に現れた 11 件を消すよう求めて失敗した。
- [x] T004 [Verify] 変更を検証する。

## 検証

- `mise run check-spec`、`mise run check-unspecified-vocabulary`
- `mise run spec-diff` で、要件の差分が本文の書き直しと、状態遷移表の追加だけであること。
- `mise run verify`

## リスク

- 書き直しで、要件の義務が落ちる。
  要件ごとに書き直しの前後の義務を突き合わせ、`spec-diff` で確かめる。

## 完了
- **Completed At**: 2026-10-04
- **Summary**:
  `mise run spec-diff -- main` の結果、規範の差分は REQ-IDMANAGEMENT-002 から 088 のうち対象の 10 機能の要件と REQ-IDMANAGEMENT-047 の本文の書き直しと、AgentLifecycle、DataExportLifecycle、DynamicMembershipEvaluationLifecycle の状態遷移の変更（状態遷移表（マトリクス形式）の追加と、自己遷移、再試行、Agent の `Deleted` の追加）だけであり、要件の追加と削除はない。
  10 機能の `README.md` は 1,376 行から 1,095 行になり、各機能の全文書は 3,516 行から 3,235 行になった。三つの状態遷移表を加えた後の値である。
  操作ごとの要約表、担保手段の欄、要判断の欄 11 件を消した。要約表にだけあった事実は要件へ移し、要判断の未決定の点は wi-97149 に残っている。
  `tools/check/unspecified-vocabulary-debt.json` の 17 件のうち、要件に書いた 11 件を消した。残る 6 件は (c) として wi-45298 に起票した。
  状態遷移表と許容リストから見つけた未記載の振る舞いを設計の表のとおりに分類し、(c) を wi-45298、wi-17379、wi-74002 として起票した。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-unspecified-vocabulary`
  - **Requirement**: N/A: 仕様の書き方の変更であり、製品の要件の義務を変えない。
  - **Observed Failure**: 書き直しの後、許容リストを変える前の検査は、`agent_owner_required`、`agent_not_found`、`user_import_not_found`、`group_import_not_found`、`EmailChangeRequested`、`UserRequiredActionSet`、`AgentRegistered`、`AgentCredentialUnbound`、`DynamicGroupRuleUpdated`、`DynamicGroupRuleEnabled`、`DynamicGroupRuleDisabled` の 11 件を「appears in a requirement now」として失敗した。
  - **Detection Reason**: 要件に書いた語が許容リストに残ることと、書き直しで要件から語が落ちることの両方を、要件の本文と遷移の表から区別する。
- **Unit RED Evidence**:
  - **Test**: `mise run check-spec`
  - **Requirement**: N/A: 仕様の書き方の変更であり、製品の要件の義務を変えない。
  - **Observed Failure**: group の書き直しで置いた要件の見出しへのリンクを、生成サイトの検査が `links to missing fragment` で拒否した。
  - **Detection Reason**: 生成サイトに存在しない見出しへの参照を、文書の文字列からではなく生成したページの ID から検出する。
- **Change-Resistance Results**:
  AgentLifecycle の状態遷移表に、空のセル（`Deleted` × 更新）と、遷移の表の行の欠落を注入すると、`mise run check-spec` が `state matrix gives no outcome for Deleted × 更新` と `state matrix moves Killed to Killed, which the transition table does not list` で失敗した。
  表と実装の食い違いを実行時に検出するテストはなく、wi-86874 が扱う。
- **Verification Results**:
  - `mise run check-spec` - 成功
  - `mise run check-unspecified-vocabulary` - 成功
  - `mise run check-work-items` - 成功
  - `mise run verify` - 成功（1 回目は許容リストの JSON の整形で `lint-tools` が失敗し、整形を直して成功）
