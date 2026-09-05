---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-05
change_kind: maintenance
priority: p2
depends_on: []
affected_spec:
  - { path: docs/contexts/sourcing/scenarios.md, requirement: REQ-SOURCING-001 }
  - { path: docs/contexts/sourcing/scenarios.md, requirement: REQ-SOURCING-003 }
  - { path: docs/contexts/sourcing/scenarios.md, requirement: REQ-SOURCING-004 }
---

# Sourcing が宣言する未検証の拒否 3 件に効果まで確かめるテストを与え、台帳から外す

## Motivation

`tools/check/scenario-coverage-debt.json` は、どのテストからも id を引用されていない規範 id を保持する、縮むだけの台帳である。

このうち 102 件は [[wi-390-security-control-test-standard-and-gate]] が拒否専用の台帳へ据え置いた分で、[[wi-490-fold-refusal-coverage-into-one-normative-coverage-rule]] が網羅台帳へ統合した。

その 102 件のうち 3 件が Sourcing の拒否である。

対象は SCIM の検索条件の拒否、PUT の必須属性欠落の拒否、未対応の PATCH パスと読み取り専用属性への書き込みの拒否である。

`REQ-SOURCING-001` はプロビジョニングトークンのテナント不一致の拒否も含んでおり、これはテナント境界そのものである。

`REQ-SOURCING-003` と `REQ-SOURCING-004` は、いずれもシナリオが「リソースを変更しない」と副作用の側から宣言している。

SCIM の PATCH は部分更新であり、拒否が遅れれば「一部だけ適用された」状態が残る。

読み取り専用の `id`、`meta`、`schemas` への書き込みが素通りすれば、外部 IdP がこちらの識別子を書き換えられる。

拒否の応答だけを確かめるテストは、この「一部だけ適用された」状態を検出できない。

この 3 件はいずれも nearby に分類され、同じエラー型名を引用しているテストは 1 件も無い。

そして注記だけを足す解消は、この項目では認めない。

## Scope

- 台帳の Sourcing の 3 件それぞれについて、宣言された拒否に production と同じ入口 (SCIM のエンドポイント) から到達するテストを用意する。
- 各テストで 2 つを assert する。SCIM プロトコルエラーの `scimType` と、その拒否がリソースを変えていないこと。
- 複数の操作を含む PATCH は、拒否されたときに 1 つも適用されていないことを確かめる。
- 各テストのソースに対応する `REQ-SOURCING-NNN` を書く。
- 対応が取れた id を `tools/check/scenario-coverage-debt.json` から削除する。
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
- 対応する SCIM 機能の拡張。
  [[wi-248-scim-complex-value-filter-bracket-syntax]]、[[wi-249-scim-bulk-operations]]、[[wi-250-scim-sort-and-etag]] が扱う。

## Design

### 解消の条件

1 件の id を台帳から外してよいのは、次の 3 つがすべて成り立つときだけとする。

第 1 に、テストが production の使う入口から入り、宣言された拒否の判断を実際に通ること。

SCIM はプロビジョニングトークンを付けた HTTP の境界から入る。

第 2 に、拒否が変えなかったものをテスト自身が読み直して確かめること。

第 3 に、シナリオの id をテストのソースが引用していること。

### 拒否ごとに「変わっていないこと」として何を読むか

| REQ | 宣言している拒否 | 読み直して確かめる対象 |
|---|---|---|
| REQ-SOURCING-001 | 不正な `filter`、解釈できない `startIndex`/`count`、テナント不一致を拒否 | 応答にリソースが 1 件も含まれず、他テナントのリソースも返らない |
| REQ-SOURCING-003 | 必須属性を欠く PUT を `invalidValue` で拒否 | 対象リソースが置換前の値をすべて保っている |
| REQ-SOURCING-004 | 許可外パス、読み取り専用属性、未対応 `op` を拒否 | 対象リソースが変更されておらず、同じ PATCH に含まれる他の操作も適用されていない |

`REQ-SOURCING-003` の「置換前の値をすべて保っている」は、PUT が完全置換であることに由来する。

拒否が遅れて先に既存属性を消す実装は、応答だけを見るテストでは成功と区別できず、しかも被害はリソース全体に及ぶ。

`REQ-SOURCING-004` は、拒否された操作を含む PATCH に有効な操作も混ぜて投げる。

不正な操作だけを捨てて残りを適用する実装は、SCIM の要求に反するが、単独の不正操作だけを送るテストでは検出できない。

### 却下した進め方

**フィルターと PATCH パスのパーサーの単体テストで閉じる案。**

パーサーが正しくても呼ばれていなければ素通りする。

またパーサーは「リソースを変更しない」という効果を持たないため、規範の半分しか確かめられない。

**不正な操作だけを含む PATCH で確かめる案。**

上に述べたとおり、部分適用する実装を検出できない。

**テナント不一致を、トークンの解決処理の単体テストで確かめる案。**

境界は入口から確かめる。

## Plan

1. 3 件のシナリオを読み、拒否ごとに入口と「変わっていないこと」を確定する。
2. 有効な操作と不正な操作を混在させた PATCH の固定具を用意する。
3. 検索条件とテナント境界の 1 件 (001) から着手し、テストが RED になることを先に観測する。
4. PUT の 1 件 (003) を、置換前の値の保持まで含めて進める。
5. PATCH の 1 件 (004) を、混在した操作で進める。
6. 拒否の実装が欠けていた場合は修正する。
7. 対応の取れた id を台帳から削除し、この項目が持つ id が 1 件も残っていないことを確認する。

## Tasks

- [ ] T001 [Inventory] 3 件の入口と「変わっていないこと」を確定し、混在 PATCH の固定具を用意する。
- [ ] T002 [Acceptance] 検索条件とテナント境界の拒否 (001) を効果まで確かめるテストを書く。
- [ ] T003 [Acceptance] PUT の必須属性欠落の拒否 (003) を置換前の値の保持まで確かめるテストを書く。
- [ ] T004 [Acceptance] PATCH の拒否 (004) を部分適用の不在まで確かめるテストを書く。
- [ ] T005 [App] 拒否の実装が欠けていた経路を修正する。
- [ ] T006 [Ledger] 対応の取れた id を `scenario-coverage-debt.json` から削除する。
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

SCIM のプロトコルエラーは `scimType` で種類を区別するため、これを assert しないと別の理由で落ちても成立する。

各テストは `scimType` の値まで固定する。

台帳は id 順に 1 エントリー 1 id なので、Context ごとの work item が並行しても衝突はエントリー単位に収まる。

各項目は自分が持つ id のエントリーだけを削除し、コメント配列と他の id には触れない。
