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
  - { path: docs/contexts/saml/scenarios.md, requirement: REQ-SAML-001 }
  - { path: docs/contexts/saml/scenarios.md, requirement: REQ-SAML-002 }
---

# SAML が宣言する未検証の拒否 2 件に効果まで確かめるテストを与え、台帳から外す

## Motivation

`tools/check/scenario-coverage-debt.json` は、どのテストからも id を引用されていない規範 id を保持する、縮むだけの台帳である。

このうち 102 件は [[wi-390-security-control-test-standard-and-gate]] が拒否専用の台帳へ据え置いた分で、[[wi-490-fold-refusal-coverage-into-one-normative-coverage-rule]] が網羅台帳へ統合した。

その 102 件のうち 2 件が SAML の拒否である。

`REQ-SAML-002` は、`profile-a` の SSO URL と異なる Destination を指定した AuthnRequest をフェイルクローズで拒否することを宣言している。

Destination の検証は、SAML において受け取ったリクエストが自分宛であることを確かめる手段であり、素通りすれば、別の宛先向けに作られたリクエストに対してこちらがアサーションを発行する。

発行されたアサーションは署名付きで外へ出るため、拒否が遅れた分は取り消せない。

`REQ-SAML-001` は、フェデレーション署名資格情報を利用できないときに証明書を返さずエラーを返すことを宣言している。

これは可用性ではなくフェイルクローズの宣言であり、空の証明書や既定値を返さないことを求めている。

この 2 件はいずれも nearby に分類され、同じエラー型名を引用しているテストは 1 件も無い。

そして注記だけを足す解消は、この項目では認めない。

拒否の応答だけを確かめるテストに id を書き足せば、検査は通り、台帳は縮み、防護が効いていることは何ひとつ確かめられないまま「検証済み」の見た目だけが残る。

## Scope

- 台帳の SAML の 2 件それぞれについて、宣言された拒否に production と同じ入口 (SSO エンドポイントとメタデータの公開経路) から到達するテストを用意する。
- `REQ-SAML-002` は、拒否のときにアサーションが 1 通も発行されていないことを確かめる。
- `REQ-SAML-001` は、証明書として空や既定値が返っていないことを確かめる。
- 各テストのソースに対応する `REQ-SAML-NNN` を書く。
- 対応が取れた id を `tools/check/scenario-coverage-debt.json` から削除する。
- 拒否を実装が持っていないと判明した場合、その防護をこの項目で実装する。
- 完了時点で、この項目が持つ id が台帳から 1 件残らず消えていることを確認する。

## Out of Scope

- 他 Context の拒否。
  Context ごとに別の work item が持つ。
- 102 件を 1 つの単位として消化すること。
  [[wi-399-burn-down-untested-refusal-debt]] がその形で立てられているが、進める単位を未決のまま残し、注記だけの解消も認めている。本項目はその置き換えである。
- Application 側の SAML プロトコル設定の拒否。
  [[wi-475-back-application-declared-refusals-with-effect-tests]] が `REQ-APPLICATION-005` として扱う。
- 同じ台帳に載る、この項目が持たない id。
  規則は同じだが、引き取る範囲はここに挙げた id に限る。
- R1、R2、R4 の検査そのものの変更。
  [[wi-390-security-control-test-standard-and-gate]] と [[wi-391-refusal-declaration-floor-and-reinventory]] が持つ。
- 既存テストへ `REQ` id を注記するだけで台帳から外すこと。
  この項目ではこれを解消として扱わない。
- SAML の対応プロファイルとバインディングの拡張。

## Design

### 解消の条件

1 件の id を台帳から外してよいのは、次の 3 つがすべて成り立つときだけとする。

第 1 に、テストが production の使う入口から入り、宣言された拒否の判断を実際に通ること。

Destination の検証は SSO エンドポイントの経路上にあるため、AuthnRequest を組み立てて HTTP の境界から送る。

第 2 に、拒否が変えなかったものをテスト自身が読み直して確かめること。

第 3 に、シナリオの id をテストのソースが引用していること。

### 拒否ごとに何を読むか

| REQ | 宣言している拒否 | 読み直して確かめる対象 |
|---|---|---|
| REQ-SAML-001 | 署名資格情報を利用できないとき証明書を返さずエラーを返す | 応答に証明書の要素が無く、空文字や既定の値も返っていない |
| REQ-SAML-002 | 割り当てられたプロファイルの SSO URL と異なる Destination をフェイルクローズで拒否 | アサーションが 1 通も発行されておらず、SP へのリダイレクトも起きていない |

`REQ-SAML-002` の「1 通も発行されていない」は、応答本文に SAMLResponse が含まれないことと、SP の Assertion Consumer Service へのリダイレクトが起きないことの両方で読む。

拒否の応答を返しつつ、裏で発行だけ済ませている実装は、片方だけでは検出できない。

`REQ-SAML-001` の「空文字や既定の値も返っていない」は、フェイルクローズの宣言に対応する。

証明書が空の文字列で返ると、受け取った SP 側は署名検証を諦めるか、検証なしで受理する実装に当たる。

エラーを返すことと、空を返さないことは別の要求である。

### 却下した進め方

**Destination の比較関数の単体テストで閉じる案。**

比較が正しくても SSO の経路から呼ばれていなければ素通りする。

**アサーションの発行を、応答コードだけで確かめる案。**

拒否の応答を返しながら発行が済んでいる実装を検出できない。

**署名資格情報の不在を、設定の読み出しだけで確かめる案。**

拒否の効果はメタデータの応答内容であり、設定の状態はその原因にすぎない。

## Plan

1. 2 件のシナリオを読み、拒否ごとに入口と読み方を確定する。
2. 署名資格情報を利用できない状態を作る手段を確認する。
3. Destination の 1 件 (002) から着手し、テストが RED になることを先に観測する。
4. 証明書の 1 件 (001) を、空と既定値の除外まで含めて進める。
5. 拒否の実装が欠けていた場合は修正する。
6. 対応の取れた id を台帳から削除し、この項目が持つ id が 1 件も残っていないことを確認する。

## Tasks

- [ ] T001 [Inventory] 2 件の入口と読み方を確定し、署名資格情報を欠いた状態を作る手段を決める。
- [ ] T002 [Acceptance] Destination 不一致の拒否 (002) をアサーション発行の不在まで確かめるテストを書く。
- [ ] T003 [Acceptance] 署名資格情報を利用できないときの拒否 (001) を、空と既定値を返さないことまで確かめるテストを書く。
- [ ] T004 [App] 拒否の実装が欠けていた経路を修正する。
- [ ] T005 [Ledger] 対応の取れた id を `scenario-coverage-debt.json` から削除する。
- [ ] T006 [Verify] 防護を意図的に外すと各テストが落ちることを確かめ、検査を通す。

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

SAML の応答は XML であり、要素の有無を文字列の一致で確かめると、名前空間の接頭辞の違いで誤って成立する。

要素として解析したうえで確かめる。

台帳は id 順に 1 エントリー 1 id なので、Context ごとの work item が並行しても衝突はエントリー単位に収まる。

各項目は自分が持つ id のエントリーだけを削除し、コメント配列と他の id には触れない。
