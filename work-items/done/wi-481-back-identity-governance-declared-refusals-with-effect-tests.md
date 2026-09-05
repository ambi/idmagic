---
status: cancelled
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-05
change_kind: maintenance
priority: p2
depends_on: []
affected_spec:
  - { path: docs/contexts/identity-governance/scenarios.feature.md, requirement: REQ-IDGOVERNANCE-006 }
  - { path: docs/contexts/identity-governance/scenarios.feature.md, requirement: REQ-IDGOVERNANCE-007 }
  - { path: docs/contexts/identity-governance/scenarios.feature.md, requirement: REQ-IDGOVERNANCE-011 }
  - { path: docs/contexts/identity-governance/scenarios.feature.md, requirement: REQ-IDGOVERNANCE-012 }
---

# IdentityGovernance が宣言する未検証の拒否 4 件に効果まで確かめるテストを与え、台帳から外す

## Motivation

`tools/check/example-coverage-debt.json` は、どのテストからも id を引用されていない規範 id を保持する、縮むだけの台帳である。

このうち 102 件は [[wi-390-security-control-test-standard-and-gate]] が拒否専用の台帳へ据え置いた分で、[[wi-490-fold-refusal-coverage-into-one-normative-coverage-rule]] が網羅台帳へ統合した。

その 102 件のうち 4 件が IdentityGovernance の拒否である。

対象は境界条件の保存拒否、未知フィールドと別テナント参照を含むワークフローの有効化拒否、無効化と再試行の競合、テナント境界である。

`REQ-IDGOVERNANCE-012` は「ワークフローは存在しないものとして扱われる」「別テナントの同名リソースへフォールバックしない」と宣言している。

同名フォールバックは、テナント境界の破れ方として最も見つけにくい部類である。

拒否の応答は正しいのに、別テナントのリソースを掴んだまま WorkflowRun が進む実装は、応答だけを見るテストでは合格する。

`REQ-IDGOVERNANCE-011` は無効化と再試行が競合したときに新しいステップを開始しないことを宣言しており、これも効果は「ステップが増えていないこと」でしか読めない。

`mise run report-coverage-debt` の分類では 4 件すべてが named だが、named は「同じエラー型名を含む拒否テストが同じパッケージへ到達する」ことしか示さない。

`InvalidRequestError` はこの Context のほぼすべての検証が返す型なので、この分類はとりわけ当てにならない。

そして注記だけを足す解消は、この項目では認めない。

## Scope

- 台帳の IdentityGovernance の 4 件それぞれについて、宣言された拒否に production と同じ入口から到達するテストを用意する。
- 各テストで 2 つを assert する。呼び出し元が観測する応答 (ステータスとエラー種別) と、その拒否が変えなかった状態。
- テナント境界の拒否は、別テナントの同名リソースへフォールバックしていないことまで確かめる。
- 各テストのソースに対応する `REQ-IDGOVERNANCE-NNN` を書く。
- 対応が取れた id を `tools/check/example-coverage-debt.json` から削除する。
- 拒否を実装が持っていないと判明した場合、その防護をこの項目で実装する。
- 完了時点で、この項目が持つ id が台帳から 1 件残らず消えていることを確認する。

## Out of Scope

- 他 Context の拒否。
  Context ごとに別の work item が持つ。
- 102 件を 1 つの単位として消化すること。
  [[wi-399-burn-down-untested-refusal-debt]] がその形で立てられているが、進める単位を未決のまま残し、注記だけの解消も認めている。本項目はその置き換えである。
- 同じ台帳に載る、この項目が持たない id。
  規則は同じだが、引き取る範囲はここに挙げた id に限る。
- R1、R2、R4 の検査そのものの変更。
  [[wi-390-security-control-test-standard-and-gate]] と [[wi-391-refusal-declaration-floor-and-reinventory]] が持つ。
- 既存テストへ `REQ` id を注記するだけで台帳から外すこと。
  この項目ではこれを解消として扱わない。
- ワークフローのトリガー条件とステップ種別の拡張。
  [[wi-228-lifecycle-workflow-trigger-filter-expressiveness]] が扱う。

## Design

### 解消の条件

1 件の id を台帳から外してよいのは、次の 3 つがすべて成り立つときだけとする。

第 1 に、テストが production の使う入口から入り、宣言された拒否の判断を実際に通ること。

保存と有効化の拒否は管理 API の境界から入り、実行時の拒否は WorkflowRun を進める経路から入る。

第 2 に、拒否が変えなかったものをテスト自身が読み直して確かめること。

第 3 に、シナリオの id をテストのソースが引用していること。

### 拒否ごとに「変わっていないこと」として何を読むか

| REQ | 宣言している拒否 | 読み直して確かめる対象 |
|---|---|---|
| REQ-IDGOVERNANCE-006 | 境界条件に当たる保存を `InvalidRequestError` で拒否 | 保存前の定義がそのまま残っている |
| REQ-IDGOVERNANCE-007 | 未知フィールドと別テナント参照を含むワークフローの有効化を拒否 | ワークフローが有効になっておらず、WorkflowRun も開始していない |
| REQ-IDGOVERNANCE-011 | 無効化済みワークフローの WorkflowRun の再試行を拒否 | 新しいステップが 1 つも開始しておらず、実行の記録が増えていない |
| REQ-IDGOVERNANCE-012 | 別テナントのワークフローとリソースの参照を拒否 | 別テナントのワークフローが応答に現れず、同名リソースへのフォールバックも起きていない |

`REQ-IDGOVERNANCE-012` のフォールバックの不在は、両テナントに同名のリソースを用意したうえで確かめる。

片方にしか存在しない構成では、フォールバックする実装でも拒否と同じ結果になり、区別できない。

これはこの Context に固有の準備であり、テストの構成そのものが規範の要求から決まる。

`REQ-IDGOVERNANCE-011` は競合の再現が要る。

無効化を先に確定させたうえで再試行を投げ、ステップの数が増えないことを読む。

### 却下した進め方

**named の 4 件を注記だけで閉じる案。**

この Context の拒否はほぼすべてが `InvalidRequestError` を返すため、エラー型名の一致は何の証拠にもならない。

**テナント境界を、リポジトリのクエリ条件の単体テストで確かめる案。**

クエリが正しくても、呼び出し側がテナントを渡し忘れていれば素通りする。

境界は入口から確かめる。

**競合の再現を諦め、無効化済みワークフローの再試行だけを確かめる案。**

シナリオが宣言しているのは競合時の振る舞いであり、順序が確定した状態での再試行は別の条件である。

## Plan

1. 4 件のシナリオを読み、拒否ごとに入口と「変わっていないこと」を確定する。
2. 両テナントに同名リソースを用意する固定具を整える。
3. 保存と有効化の 2 件 (006、007) から着手し、テストが RED になることを先に観測する。
4. テナント境界の 1 件 (012) を、同名フォールバックの不在まで含めて進める。
5. 競合の 1 件 (011) を進める。
6. 拒否の実装が欠けていた場合は修正する。
7. 対応の取れた id を台帳から削除し、この項目が持つ id が 1 件も残っていないことを確認する。

## Tasks

- [ ] T001 [Inventory] 4 件の入口と「変わっていないこと」を確定し、同名リソースの固定具を用意する。
- [ ] T002 [Acceptance] 保存と有効化の拒否 (006、007) を効果まで確かめるテストを書く。
- [ ] T003 [Acceptance] テナント境界の拒否 (012) をフォールバックの不在まで確かめるテストを書く。
- [ ] T004 [Acceptance] 無効化と再試行の競合の拒否 (011) をステップ数まで確かめるテストを書く。
- [ ] T005 [App] 拒否の実装が欠けていた経路を修正する。
- [ ] T006 [Ledger] 対応の取れた id を `example-coverage-debt.json` から削除する。
- [ ] T007 [Verify] 防護を意図的に外すと各テストが落ちることを確かめ、検査を通す。

## Verification

- `mise run test-go-race`
- `mise run check-spec`
- `mise run check-security-controls`
- `mise run report-coverage-debt`
- `mise run check-work-items`
- `mise run check-ids`
- `mise run verify`

## Risk Notes

拒否のテストは、防護に到達する前の入力検証で落ちていても同じ応答を返すため、書いた本人にも成立して見える。

これを避けるため、各テストについて防護を外した状態でテストが落ちることを確かめ、落ちなかったものは入口が誤っていると判断してやり直す。

競合の再現に時間依存の待ちを入れると、CI で不安定になる。

無効化の確定を観測してから再試行を投げる形にし、待ち時間には依存させない。

台帳は id 順に 1 エントリー 1 id なので、Context ごとの work item が並行しても衝突はエントリー単位に収まる。

各項目は自分が持つ id のエントリーだけを削除し、コメント配列と他の id には触れない。

## Completion

- **Completed At**: 2026-09-06
- **Summary**:
  取り消す。[[wi-491-adopt-markdown-with-gherkin-scenarios]] により被覆の管理単位が規則の `REQ-*` から具体例の `EX-*` へ変わり、本項目の規則単位の計画は古くなった。対象の棚卸し、拒否テスト、無作用の確認、台帳更新は [[wi-392-refusal-tests-assert-the-absent-effect]] に統合し、製品と仕様には変更を加えずに終了する。
