---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-04
priority: p1
depends_on: [wi-17076-limit-specifications-to-external-contracts-written-once, wi-89346-detect-unspecified-behavior-mechanically]
change_kind: docs
affected_spec:
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

## 計画

1. 機能ごとに、旧い本文と要約表の事実を要件へ対応づけてから書き直す。
2. 状態機械に状態遷移表を加え、空のセルを分類する。
3. 許容リストの項目を消し、残る項目を分類する。

## タスク

- [ ] T001 [Spec] 10 機能の機能仕様を書き直す。
- [ ] T002 [Spec] 三つの状態機械に状態遷移表（マトリクス形式）を加える。
- [ ] T003 [Tooling] 許容リストの項目を消す。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `mise run check-spec`、`mise run check-unspecified-vocabulary`
- `mise run spec-diff` で、要件の差分が本文の書き直しと、状態遷移表の追加だけであること。
- `mise run verify`

## リスク

- 書き直しで、要件の義務が落ちる。
  要件ごとに書き直しの前後の義務を突き合わせ、`spec-diff` で確かめる。
