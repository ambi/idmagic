---
status: cancelled
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-05
change_kind: maintenance
priority: p1
depends_on: []
affected_spec:
  - { path: docs/contexts/data-keys/scenarios.feature.md, requirement: REQ-DATAKEYS-001 }
  - { path: docs/contexts/data-keys/scenarios.feature.md, requirement: REQ-DATAKEYS-003 }
  - { path: docs/contexts/data-keys/scenarios.feature.md, requirement: REQ-DATAKEYS-004 }
  - { path: docs/contexts/data-keys/scenarios.feature.md, requirement: REQ-DATAKEYS-005 }
---

# DataKeys が宣言する未検証の拒否 4 件に効果まで確かめるテストを与え、台帳から外す

## Motivation

`tools/check/example-coverage-debt.json` は、どのテストからも id を引用されていない規範 id を保持する、縮むだけの台帳である。

このうち 102 件は [[wi-390-security-control-test-standard-and-gate]] が拒否専用の台帳へ据え置いた分で、[[wi-490-fold-refusal-coverage-into-one-normative-coverage-rule]] が網羅台帳へ統合した。

その 102 件のうち 4 件が DataKeys の拒否である。

台帳に載っている DataKeys の拒否は、すべてフェイルクローズか鍵の状態遷移の防護である。

MasterKey プロバイダーへ到達できないときに DEK を作らないこと、retiring の DEK での復号を止めること、active の DEK を直接 disable させないこと、参照が残る DEK を destroy させないこと。

前 2 者が素通りすれば、使えない鍵や無効化したはずの鍵で暗号文が読み書きされる。

後 2 者が素通りすれば、参照の残る鍵が消え、暗号文が**復元できなくなる**。

拒否が守っているのは可用性そのものであり、被害は元に戻らない。

この 4 件はいずれも nearby に分類され、同じエラー型名を引用しているテストは 1 件も無い。

そして注記だけを足す解消は、この項目では認めない。

拒否の応答だけを確かめるテストに id を書き足せば、検査は通り、台帳は縮み、防護が効いていることは何ひとつ確かめられないまま「検証済み」の見た目だけが残る。

## Scope

- 台帳の DataKeys の 4 件それぞれについて、宣言された拒否に production と同じ入口から到達するテストを用意する。
- 各テストで 2 つを assert する。呼び出し元が観測するエラー種別と、その拒否が変えなかった鍵の状態。
- 鍵の状態遷移の拒否は、拒否の後に鍵のバージョンと状態を読み直して確かめる。
- 各テストのソースに対応する `REQ-DATAKEYS-NNN` を書く。
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
- 鍵の状態機械そのものの変更、ローテーションの運用手順。
  [[wi-307-datakeys-rotation-lifecycle-operations]] が扱う。

## Design

### 解消の条件

1 件の id を台帳から外してよいのは、次の 3 つがすべて成り立つときだけとする。

第 1 に、テストが production の使う入口から入り、宣言された拒否の判断を実際に通ること。

DataKeys の入口は use case であり、`BootstrapTenantDataKey`、`DisableTenantDataKey`、`DestroyTenantDataKey`、および復号の要求経路である。

第 2 に、拒否が変えなかったものをテスト自身が読み直して確かめること。

第 3 に、シナリオの id をテストのソースが引用していること。

### 拒否ごとに「変わっていないこと」として何を読むか

| REQ | 宣言している拒否 | 読み直して確かめる対象 |
|---|---|---|
| REQ-DATAKEYS-001 | MasterKey プロバイダー到達不能時の DEK 生成をフェイルクローズで拒否 | テナントに DEK が 1 件も作成されておらず、半端な状態の鍵も残っていない |
| REQ-DATAKEYS-003 | retiring の DEK で暗号化された EncryptedSecret の復号を拒否 | 平文が返っておらず、対象の EncryptedSecret も変更されていない |
| REQ-DATAKEYS-004 | active の DEK の直接 disable を拒否 | 対象バージョンが `active` のままである |
| REQ-DATAKEYS-005 | 未移行の参照が残る DEK の destroy を拒否 | 対象バージョンが `retiring` のままであり、鍵素材が失われていない |

`REQ-DATAKEYS-001` の「半端な状態の鍵も残っていない」は、フェイルクローズの拒否に特有の確認である。

プロバイダーへの到達に失敗する場所によっては、行だけ作られて鍵素材が入らない状態が残りうる。

その状態のテナントは、以後の起動でも復旧できない。

`REQ-DATAKEYS-005` の「鍵素材が失われていない」は、拒否の後に実際に復号が成功することで確かめる。

状態の列だけを読むと、状態は `retiring` のまま素材だけ消えている実装を見逃す。

### 却下した進め方

**4 件を状態遷移の判定関数の単体テストで確かめる案。**

判定が正しくても use case から呼ばれていなければ素通りする。

鍵の破壊は取り消せないため、この Context では特に、配線まで確かめる必要がある。

**復号の拒否を、鍵の状態を読む関数だけで確かめる案。**

拒否の効果は平文が返らないことであり、状態の読み出しはその手段の 1 つにすぎない。

**プロバイダー到達不能を、実際のプロバイダーを落として確かめる案。**

再現が環境に依存し、CI で安定しない。

プロバイダーの境界で失敗を注入する。

## Plan

1. 4 件のシナリオを読み、拒否ごとに入口と「変わっていないこと」を確定する。
2. MasterKey プロバイダーの境界で失敗を注入できる形になっているかを確認する。
3. 状態遷移の 2 件 (004、005) から着手し、テストが RED になることを先に観測する。
4. 復号のロックアウト 1 件 (003) を進める。
5. フェイルクローズの 1 件 (001) を、失敗注入で進める。
6. 拒否の実装が欠けていた場合は修正する。
7. 対応の取れた id を台帳から削除し、この項目が持つ id が 1 件も残っていないことを確認する。

## Tasks

- [ ] T001 [Inventory] 4 件の入口と「変わっていないこと」を確定し、失敗注入の位置を決める。
- [ ] T002 [Acceptance] 状態遷移の拒否 (004、005) を鍵の状態と復号可否まで確かめるテストを書く。
- [ ] T003 [Acceptance] retiring の DEK の復号拒否 (003) を効果まで確かめるテストを書く。
- [ ] T004 [Acceptance] プロバイダー到達不能時のフェイルクローズ (001) を、半端な状態が残らないことまで確かめるテストを書く。
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

鍵の破壊を伴う経路のテストは、取り違えると開発環境のデータを失う。

destroy の拒否を確かめるテストは、拒否されることを先に確認し、成功経路のテストは専用のテナントに閉じる。

フェイルクローズの拒否は、失敗の注入位置によって「拒否した」ように見えるだけの結果になりうる。

注入位置を変えた 2 通りで確かめ、いずれでも DEK が残らないことを読む。

台帳は id 順に 1 エントリー 1 id なので、Context ごとの work item が並行しても衝突はエントリー単位に収まる。

各項目は自分が持つ id のエントリーだけを削除し、コメント配列と他の id には触れない。

## Completion

- **Completed At**: 2026-09-06
- **Summary**:
  取り消す。[[wi-491-adopt-markdown-with-gherkin-scenarios]] により被覆の管理単位が規則の `REQ-*` から具体例の `EX-*` へ変わり、本項目の規則単位の計画は古くなった。対象の棚卸し、拒否テスト、無作用の確認、台帳更新は [[wi-392-refusal-tests-assert-the-absent-effect]] に統合し、製品と仕様には変更を加えずに終了する。
