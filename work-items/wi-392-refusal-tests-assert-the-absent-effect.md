---
depends_on: [wi-490-fold-refusal-coverage-into-one-normative-coverage-rule, wi-491-adopt-markdown-with-gherkin-scenarios]
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-08-22
priority: p1
change_kind: tooling
spec_impact: { kind: none, reason: "既存の拒否テストへ無作用のアサーションを足す作業であり、製品の振る舞いと公開契約を変えない。検証中に実装の欠陥が見つかった場合は、規範参照を持つ個別の bugfix work item に分ける。" }
---

# 被覆済みとして数えられている拒否テスト 112 件に、拒否が防いだ効果の観測を足す

## Motivation

拒否の検証には、拒否応答と、拒否によって防がれた効果の両方が要る。

`mise run report-security-test-gaps` の 2026-09-06 時点の実測では、状態変更を拒否するテスト 159 件のうち 112 件が、拒否後の状態を読み直していない。

**この 112 件は、被覆の台帳には載っていない。** `tools/check/example-coverage-debt.json` が保持するのは「id を名指しするテストが無い」具体例であり、ここで問題にしているのは、名指しがあって被覆済みとして数えられているのに、拒否が防いだ効果を誰も観測していないテストである。台帳を 0 にしても、この 112 件は 1 件も減らない。

分布は次のとおり（2026-09-06 実測）。

| パッケージ | 件数 |
|---|---:|
| `backend/idmanagement/handlers_http` | 15 |
| `backend/shared/http` | 11 |
| `backend/tenancy/handlers_http` | 10 |
| `backend/application/handlers_http` | 10 |
| `backend/sourcing/scim` | 10 |
| `backend/idmanagement/group` | 8 |
| `backend/oauth2/handlers_http` | 8 |
| `backend/application/usecases` | 5 |
| `backend/authentication/handlers_http` | 5 |
| `backend/saml/handlers_http` | 5 |
| `backend/provisioning/client_scim` | 4 |
| `backend/oauth2/client` | 3 |
| `backend/wsfederation/handlers_http` | 3 |
| `backend/idgovernance/usecases` | 3 |
| その他 8 パッケージ | 各 1 から 2、計 12 |

当初この work item は、台帳の消化（旧 `wi-475` などが対象としていた 44 規則の `EX-*` への対応付けと、台帳からの削除）も併せて持っていた。それを [[wi-496-burn-down-the-example-coverage-debt]] へ移し、本項目は既存テストの改善に絞る。理由は Design に書く。

## Scope

- `mise run report-security-test-gaps` が報告する既存の拒否テスト 112 件を確認し、拒否後に防護対象の状態、発行物、配送、監査記録のいずれかを読み直すアサーションを足す。
- 読み直しの形が確立していない境界（外部への送出、読み取り操作の拒否）について、何を無作用の観測とするかを決めて Design へ書く。
- 拒否を実装が持っていない、または拒否後に効果が残ることが判明した場合は、本項目で製品コードを直さず、当該 `REQ-*` を `affected_spec` に持つ bugfix work item を作る。
- 作業開始時と完了時に報告タスクを実行し、対象件数と残件を本項目へ記録する。

## Out of Scope

- `tools/check/example-coverage-debt.json` の 614 件の消化。[[wi-496-burn-down-the-example-coverage-debt]] が持つ。同項目が新しく書く拒否テストには、本項目が定める無作用の規範が効く。
- `tools/check/standards-coverage-debt.json` の 134 件。[[wi-495-burn-down-the-standards-coverage-debt]] が持つ。
- 行カバレッジ率の目標または閾値。
- 「読み直し呼出しが一つある」といった構文だけによる無作用検査。
- 検証で見つかった製品の欠陥の修正。
- 新しい拒否規則またはエラー型の追加。

## Design

完了条件はテスト名や注記の存在ではなく、誤った実装を区別できることである。

各拒否テストは、製品の正式な入口から対象の判断へ到達し、呼び出し元が観測する拒否応答と、防護が無ければ起きたはずの効果が残っていないことを表明する。この形は本項目が直す既存テストだけでなく、[[wi-496-burn-down-the-example-coverage-debt]] が新しく書く拒否テストにも効く規範である。

状態を読み戻せる依存では拒否前後の値を比較し、読み戻せない外部境界では送出記録が増えていないことを確かめる。

読み取り操作の拒否では、保護対象の表現が応答へ含まれないことを無作用に相当する観測として扱う。

`report-security-test-gaps` は既存テストの構文から候補を挙げる報告であり、検証の意味を保証しない。112 件という数は読む対象の数であって、直す対象の数ではない。読んだ結果、既に無作用を観測していると判断した件は、なぜ報告に挙がったかを本項目へ記録して残す。報告の側を直せるならそうする。

**台帳の消化を本項目から外した理由。** 当初は「既存テストの無作用確認」と「テストのない拒否例への検証追加」を一つの負債として扱っていた。完了条件が同じだからである。しかし対象の決め方が違う。前者は既存テストの集合から報告が挙げるものであり、後者は台帳に載る 614 件のうち「拒否である」ものである。後者の「拒否である」を機械的に決める基準は安定しない。結果ステップが `*Error` 型を名指しするもので数えると 164 件だが、型を名指しせずに拒否の語で結果を書いている具体例が 133 件あり、そこには拒否ではないものも混ざる。境界がぶれる基準で所有を分ければ、どちらの work item にも入らない具体例が生まれる。台帳の所有者は wi-496 の 1 つにし、本項目は拒否テストの書き方という規範だけを提供する。

## Plan

1. `report-security-test-gaps` の 112 件を一覧にし、パッケージ、拒否の入口、防いだはずの効果、現在のアサーションを記録する。
2. 読み直しの形が確立していない境界について、無作用の観測を決めて Design へ書く。
3. パッケージ単位で、拒否後の無作用を読み戻すアサーションを追加する。
4. 拒否処理を一時的に外すか、拒否前に効果を起こす故障を注入し、追加したアサーションが失敗することを確認する。
5. 製品欠陥は個別の bugfix work item へ移し、本項目のテスト整理と混ぜない。

## Tasks

- [ ] T001 [Inventory] 112 件を一覧にし、入口、防いだ効果、現在のアサーションを記録する。
- [ ] T002 [Design] 外部への送出と読み取り操作の拒否について、無作用の観測を決める。
- [ ] T003 [Test] 既存の拒否テストへ、拒否後の無作用を読み戻すアサーションを追加する。
- [ ] T004 [Triage] 検出した製品欠陥を個別の bugfix work item へ分ける。
- [ ] T005 [Report] 報告に挙がったが既に無作用を観測していた件を記録し、可能なら報告の側を直す。
- [ ] T006 [Verify] 故障注入で検出能力を確かめ、報告と標準検証を通す。

## Verification

- `mise run report-security-test-gaps` の「拒否後の読み直しが無い」件数が 0 になる。
- `mise run check-security-controls`
- `mise run check-spec`
- `mise run test-go-race`
- `mise run verify`

## Risk Notes

リスクは low であり、製品コードは変更しない。

最大の失敗は、報告の件数を下げるためだけに読み直しの呼び出しを足し、その戻り値を何とも突き合わせないことである。`report-security-test-gaps` は構文を見ているので、それでも件数は下がる。

各テストへ拒否処理を外す故障または拒否前に効果を起こす故障を与え、テストが失敗しなければ無作用を観測したと扱わない。

もう一つの失敗は、112 件を数値目標として扱うことである。読んだ結果「既に観測している」となる件は必ずあり、そのときは報告の側が過剰に挙げている。テストを変えずに記録して次へ進む。
