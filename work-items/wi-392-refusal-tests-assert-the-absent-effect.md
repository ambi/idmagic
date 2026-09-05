---
depends_on: [wi-490-fold-refusal-coverage-into-one-normative-coverage-rule, wi-491-adopt-markdown-with-gherkin-scenarios]
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-08-22
priority: p1
change_kind: tooling
spec_impact: { kind: none, reason: "既存の拒否例へテストと無作用のアサーションを対応付ける作業であり、製品の振る舞いと公開契約を変えない。検証中に実装の欠陥が見つかった場合は、規範参照を持つ個別の bugfix work item に分ける。" }
---

# 拒否経路のテスト負債を、具体例の被覆と無作用の確認までまとめて解消する

## Motivation

拒否の検証には、拒否応答と、拒否によって防がれた効果の両方が要る。

`mise run report-security-test-gaps` の 2026-09-06 時点の実測では、状態変更を拒否するテスト 158 件のうち 112 件が、拒否後の状態を読み直していない。

一方、[[wi-491-adopt-markdown-with-gherkin-scenarios]] は規範シナリオを Markdown with Gherkin へ移し、被覆の管理単位を規則の `REQ-*` から具体例の `EX-*` へ変更した。

この移行により、Context ごとの旧 work item 12 件が持っていた「規則を名指しするテストを足す」という計画は、そのままでは台帳を縮められなくなった。

既存テストの無作用確認と、まだテストのない拒否例への検証追加は、どちらも「拒否を応答の字面だけで検証済みにしない」という同じ完了条件へ帰着するため、本項目で一つの負債として扱う。

## Scope

- `mise run report-security-test-gaps` が報告する既存の拒否テストを確認し、拒否後に防護対象の状態、発行物、配送、監査記録のいずれかを読み直すアサーションを足す。
- 旧 `wi-475`、`wi-476`、`wi-479` から `wi-486`、`wi-488`、`wi-489` が対象としていた 44 規則を、現在の `scenarios.feature.md` にある拒否の `EX-*` へ対応付ける。
- 対応するテストがない拒否例には、製品と同じ入口から拒否判断へ到達し、応答と防いだ効果を検証するテストを追加する。
- 検証済みになった `EX-*` を `tools/check/example-coverage-debt.json` から削除する。
- 拒否を実装が持っていない、または拒否後に効果が残ることが判明した場合は、本項目で製品コードを直さず、当該 `REQ-*` を `affected_spec` に持つ bugfix work item を作る。
- 作業開始時と完了時に二つの報告タスクを実行し、対象件数と残件を本項目へ記録する。

対象の規則は Application 8 件、WorkloadIdentity 7 件、Provisioning 5 件、DataKeys 4 件、IdentityGovernance 4 件、SharedSignals 4 件、Sourcing 3 件、Audit 2 件、ClaimMapping 2 件、Jobs 2 件、System 2 件、ApiTokens 1 件である。

この件数は移行前の規則数であり、実際の作業単位は各規則に属する拒否の `EX-*` とする。

## Out of Scope

- 拒否ではない正常経路と代替経路の例被覆。
- 行カバレッジ率の目標または閾値。
- 「読み直し呼出しが一つある」といった構文だけによる無作用検査。
- 検証で見つかった製品の欠陥の修正。
- 新しい拒否規則またはエラー型の追加。

## Design

完了条件はテスト名や注記の存在ではなく、誤った実装を区別できることである。

各拒否例について、テストは製品の正式な入口から対象の判断へ到達し、呼び出し元が観測する拒否応答と、防護が無ければ起きたはずの効果が残っていないことを表明する。

状態を読み戻せる依存では拒否前後の値を比較し、読み戻せない外部境界では送出記録が増えていないことを確かめる。

読み取り操作の拒否では、保護対象の表現が応答へ含まれないことを無作用に相当する観測として扱う。

`report-security-test-gaps` は既存テストの構文から候補を挙げる報告であり、検証の意味を保証しない。

`report-coverage-debt` も同じエラー名を持つ近傍テストを候補として示すだけなので、`named` と `nearby` を台帳削除の根拠にせず、対象の `EX-*` の Given、When、Then とテストの入力および観測を一件ずつ照合する。

## Plan

1. 44 規則に属する拒否例の `EX-*` を抽出し、Context、入口、防いだ効果、既存テストの有無を一覧にする。
2. `report-security-test-gaps` の 112 件と一覧を突き合わせ、同じテストを二度直さない作業順を決める。
3. Context ごとに既存テストの無作用確認と未被覆例のテスト追加を行い、確認済みの台帳項目だけを削除する。
4. 拒否処理を一時的に外すか、拒否前に効果を起こす故障を注入し、追加したテストが失敗することを確認する。
5. 製品欠陥は個別の bugfix work item へ移し、本項目のテスト整理と混ぜない。

## Tasks

- [ ] T001 [Inventory] 44 規則を現在の拒否例へ対応付け、入口、防いだ効果、既存テストを記録する。
- [ ] T002 [Baseline] 二つの報告タスクを実行し、重複を除いた対象件数を記録する。
- [ ] T003 [Test] 既存の拒否テストへ、拒否後の無作用を読み戻すアサーションを追加する。
- [ ] T004 [Test] テストのない拒否例へ、正式な入口から応答と無作用を検証するテストを追加する。
- [ ] T005 [Triage] 検出した製品欠陥を個別の bugfix work item へ分ける。
- [ ] T006 [Ledger] 検証済みの `EX-*` を例被覆台帳から削除する。
- [ ] T007 [Verify] 故障注入で検出能力を確かめ、報告と標準検証を通す。

## Verification

- `mise run report-security-test-gaps`
- `mise run report-coverage-debt`
- `mise run check-security-controls`
- `mise run check-spec`
- `mise run test-go-race`
- `mise run verify`

## Risk Notes

リスクは low であり、製品コードは変更しない。

最大の失敗は、対象と関係のないテストへ `EX-*` を追記し、台帳だけを縮めることである。

各テストへ拒否処理を外す故障または拒否前に効果を起こす故障を与え、テストが失敗しなければ被覆済みと扱わない。
