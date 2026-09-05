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
  - { path: docs/contexts/sharedsignals/scenarios.feature.md, requirement: REQ-SHAREDSIGNALS-003 }
  - { path: docs/contexts/sharedsignals/scenarios.feature.md, requirement: REQ-SHAREDSIGNALS-004 }
  - { path: docs/contexts/sharedsignals/scenarios.feature.md, requirement: REQ-SHAREDSIGNALS-005 }
  - { path: docs/contexts/sharedsignals/scenarios.feature.md, requirement: REQ-SHAREDSIGNALS-008 }
---

# SharedSignals が宣言する未検証の拒否 4 件に効果まで確かめるテストを与え、台帳から外す

## Motivation

`tools/check/example-coverage-debt.json` は、どのテストからも id を引用されていない規範 id を保持する、縮むだけの台帳である。

このうち 102 件は [[wi-390-security-control-test-standard-and-gate]] が拒否専用の台帳へ据え置いた分で、[[wi-490-fold-refusal-coverage-into-one-normative-coverage-rule]] が網羅台帳へ統合した。

その 102 件のうち 4 件が SharedSignals の拒否である。

対象は SET の署名検証、`jti` の重複排除、テナントをまたいだストリームでの受理、無効化したストリームでの受理である。

この Context の受信経路は、**外部から届いた署名付きの主張を、こちらのセッション状態へ反映する**入口である。

拒否が素通りすれば、攻撃者が任意のタイミングでセッション失効やアカウント無効化を起こせる。

逆に言えば、拒否の効果は「反映されていないこと」でしか読めない。

シナリオはいずれも「反映せずに拒否する」「一度だけ反映する」と、副作用の側から宣言している。

にもかかわらず引用するテストが 1 つも無いということは、規範として書かれた「反映しない」が、どこでも確かめられていないことを意味する。

`mise run report-coverage-debt` の分類では 3 件が named だが、named は「同じエラー型名を含む拒否テストが同じパッケージへ到達する」ことしか示さない。

3 件はいずれも `SecurityEventRejectedError` を返すため、型名の一致はどの拒否を確かめたのかを区別しない。

そして注記だけを足す解消は、この項目では認めない。

## Scope

- 台帳の SharedSignals の 4 件それぞれについて、宣言された拒否に production と同じ入口 (SET の受信エンドポイント) から到達するテストを用意する。
- 各テストで 2 つを assert する。呼び出し元が観測する応答と、その SET が反映されなかったこと。
- `jti` の重複排除は、1 回目が反映され 2 回目が反映されないことを対で確かめる。
- 各テストのソースに対応する `REQ-SHAREDSIGNALS-NNN` を書く。
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
- 対応するイベント種別の追加と、送信側ストリームの配送方式の変更。
  [[wi-323-caep-ssf-for-human-user-sessions]] が扱う。

## Design

### 解消の条件

1 件の id を台帳から外してよいのは、次の 3 つがすべて成り立つときだけとする。

第 1 に、テストが production の使う入口から入り、宣言された拒否の判断を実際に通ること。

SET の受信は HTTP の境界であり、署名の検証も `jti` の照合もその経路上にある。

第 2 に、拒否が変えなかったものをテスト自身が読み直して確かめること。

第 3 に、シナリオの id をテストのソースが引用していること。

### 拒否ごとに「変わっていないこと」として何を読むか

| REQ | 宣言している拒否 | 読み直して確かめる対象 |
|---|---|---|
| REQ-SHAREDSIGNALS-003 | 署名が不正な SET を反映せずに拒否 | SET が主張する状態変更が起きていない (対象セッションが有効なまま、対象アカウントが有効なまま) |
| REQ-SHAREDSIGNALS-004 | 同じ `jti` の SET の 2 回目を拒否 | 1 回目の反映だけが残り、2 回目で状態が二重に動いていない |
| REQ-SHAREDSIGNALS-005 | 発行者が一致しても他テナントのストリームでの受理を拒否 | 他テナントの対象に一切の変更が及んでいない |
| REQ-SHAREDSIGNALS-008 | 無効化したストリームでの受理と配送を行わない | 状態が変わっておらず、送信側では配送も試みられていない |

「反映されていないこと」は、SET が主張する種類ごとに読む対象が変わる。

セッション失効を主張する SET なら対象セッションで保護リソースが取得できること、アカウント無効化を主張する SET ならログインが引き続き成功することで確かめる。

状態の列を読むより、その状態が意味する振る舞いで読むほうが、素通りした実装を捕まえる力が強い。

`REQ-SHAREDSIGNALS-004` を対で置くのは、常に拒否する実装を排除するためである。

重複排除は「2 回目を拒否する」防護であり、1 回目の成功を確かめないと、受信そのものが壊れていても成立する。

### 却下した進め方

**named の 3 件を注記だけで閉じる案。**

3 件が同じエラー型を返すため、分類はどの拒否を確かめたのかを区別できない。

**署名検証と `jti` 照合の単体テストで閉じる案。**

検証が正しくても受信経路から呼ばれていなければ素通りする。

この Context は外部から届く入力を扱うため、配線の欠落がそのまま外部からの攻撃面になる。

**拒否の応答コードだけを確かめる案。**

SET の受信は仕様上、反映の有無にかかわらず同じ応答を返しうる。

応答だけでは、拒否と受理を区別できない場合がある。

## Plan

1. 4 件のシナリオを読み、SET の種類ごとに「反映されていないこと」の読み方を確定する。
2. 受信エンドポイントから入る既存テストを調べ、拡張で足りるか新規が要るかを決める。
3. 署名検証の 1 件 (003) から着手し、テストが RED になることを先に観測する。
4. 重複排除の 1 件 (004) を、1 回目の成功と対で進める。
5. テナント境界の 1 件 (005) を進める。
6. 無効化ストリームの 1 件 (008) を、受理と配送の両側で進める。
7. 対応の取れた id を台帳から削除し、この項目が持つ id が 1 件も残っていないことを確認する。

## Tasks

- [ ] T001 [Inventory] 4 件の入口と「反映されていないこと」の読み方を確定する。
- [ ] T002 [Acceptance] 署名が不正な SET の拒否 (003) を反映の不在まで確かめるテストを書く。
- [ ] T003 [Acceptance] `jti` 重複の拒否 (004) を 1 回目の成功と対で確かめるテストを書く。
- [ ] T004 [Acceptance] テナント境界の拒否 (005) を効果まで確かめるテストを書く。
- [ ] T005 [Acceptance] 無効化ストリームの受理と配送の不在 (008) を確かめるテストを書く。
- [ ] T006 [App] 拒否の実装が欠けていた経路を修正する。
- [ ] T007 [Ledger] 対応の取れた id を `example-coverage-debt.json` から削除する。
- [ ] T008 [Verify] 防護を意図的に外すと各テストが落ちることを確かめ、検査を通す。

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

受信が常に失敗する実装は、拒否のテストをすべて通してしまう。

各拒否のテストには、同じ入口で正しい SET が反映される対を必ず置く。

台帳は id 順に 1 エントリー 1 id なので、Context ごとの work item が並行しても衝突はエントリー単位に収まる。

各項目は自分が持つ id のエントリーだけを削除し、コメント配列と他の id には触れない。

## Completion

- **Completed At**: 2026-09-06
- **Summary**:
  取り消す。[[wi-491-adopt-markdown-with-gherkin-scenarios]] により被覆の管理単位が規則の `REQ-*` から具体例の `EX-*` へ変わり、本項目の規則単位の計画は古くなった。対象の棚卸し、拒否テスト、無作用の確認、台帳更新は [[wi-392-refusal-tests-assert-the-absent-effect]] に統合し、製品と仕様には変更を加えずに終了する。
