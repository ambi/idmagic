---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-05
change_kind: maintenance
priority: p2
depends_on: []
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 宣言済みの拒否に検証を与えるだけで、製品の振る舞い、公開契約、運用手順のいずれも変わらない。
  references: []
initial_context:
  specification:
    - docs/contexts/saml/scenarios.feature.md#REQ-SAML-001
    - docs/contexts/saml/scenarios.feature.md#REQ-SAML-002
  typespec:
    - IdMagic.Saml.Operations.DownloadSamlSigningCertificate
    - IdMagic.Saml.Operations.PublishSamlMetadata
    - IdMagic.Saml.Operations.SamlSingleSignOn
  source:
    - backend/saml/handlers_http
    - backend/saml/usecases
    - backend/saml/domain
    - backend/wsfederation/tokens_saml
    - backend/signingkeys/keys_memory
    - tools/check/example-coverage-debt.json
  tests:
    - backend/saml/handlers_http
affected_spec:
  - { path: docs/contexts/saml/scenarios.feature.md, requirement: REQ-SAML-001 }
  - { path: docs/contexts/saml/scenarios.feature.md, requirement: REQ-SAML-002 }
---

# SAML が宣言する未検証の拒否 2 件に効果まで確かめるテストを与え、台帳から外す

## Motivation

`tools/check/example-coverage-debt.json` は、どのテストからも id を引用されていない規範 id を保持する、縮むだけの台帳である。

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
- 対応が取れた id を `tools/check/example-coverage-debt.json` から削除する。
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

### 台帳の単位が Rule から具体例へ移ったこと

この項目は `REQ-SAML-001` と `REQ-SAML-002` の 2 件を持つものとして起票された。

その後 [[wi-491-track-tests-per-example]] が台帳を Rule 単位から具体例単位へ移し、`scenarios.feature.md` は Markdown with Gherkin になった。

台帳の現在の単位は `EX-SAML-NNN-MM` であり、Rule の id はもう台帳に載らない。

Rule の採番は移行を越えて保たれているため、起票時に挙げた 2 件は今も同じ拒否を指す。

そこで持ち分を、その 2 Rule が宣言する拒否の具体例へ読み替える。

正常系の具体例は持たない。

### この項目が持つ id

3 件。

| Rule | 具体例 | 宣言している拒否 | 読み直して確かめる対象 |
|---|---|---|---|
| REQ-SAML-001 | EX-SAML-001-02 | 署名資格情報を利用できないとき証明書を返さずエラーを返す | 応答に証明書の要素が無く、空文字や既定の値も返っていない |
| REQ-SAML-002 | EX-SAML-002-02 | 割り当てられていないプロファイルの SSO エンドポイントへの AuthnRequest を拒否 | SAMLResponse が発行されておらず、`SamlSignInRejected` だけが発行されている |
| REQ-SAML-002 | EX-SAML-002-03 | 割り当てられたプロファイルの SSO URL と異なる Destination をフェイルクローズで拒否 | アサーションが 1 通も発行されておらず、SP へのリダイレクトも起きていない |

起票時の表は `REQ-SAML-002` を Destination の 1 件として書いていたが、同じ Rule は「同じリクエストを `profile-b` の SSO エンドポイントに送る」拒否も宣言している。

どちらも「SP は割り当てられた IdP プロファイルだけを利用できる」という同じ規範の具体例なので、両方を持ち分に入れる。

### 解消の条件

1 件の id を台帳から外してよいのは、次の 3 つがすべて成り立つときだけとする。

第 1 に、テストが production の使う入口から入り、宣言された拒否の判断を実際に通ること。

Destination の検証は SSO エンドポイントの経路上にあるため、AuthnRequest を組み立てて HTTP の境界から送る。

第 2 に、拒否が変えなかったものをテスト自身が読み直して確かめること。

第 3 に、具体例の id をテストのソースが引用していること。

### 対照を必ず 1 つ置く

どの拒否のテストにも、同じ入口で同じ操作が通る対照を 1 つ置く。

「拒否されたので何も起きていない」と「そもそも何も起こせない構成だった」は、効果の側だけを読むと同じに見える。

対照が無ければ、配線を落としただけのテストがそのまま緑になる。

### 署名資格情報を利用できない状態の作り方

`REQ-SAML-001` の拒否は `KeyStoreSignerProvider.Resolve` の可用性判定と、それを受ける
`handleSamlSigningCertificate` / `handleSamlMetadata` のフェイルクローズにある。

そこで鍵ストアだけを到達不能にする装飾をテスト側に置き、`Healthy` を false、`GetActiveKey` と
`ListPublicKeys` をエラーにする。外部 provider の停止と同じ形であり、production の
`KeyStoreSignerProvider` と両ハンドラーはそのまま経路に残る。

`FederationSigner` そのものを nil にする案は採らない。production の合成は必ず provider を渡すので、
nil は起こらない状態であり、確かめたい判定の手前で分岐が閉じてしまう。

### `REQ-SAML-002` の「1 通も発行されていない」の読み方

レスポンスボディに SAMLResponse が含まれないことと、SP の Assertion Consumer Service へのリダイレクトが起きないことの両方で読む。

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

1. 3 件の具体例を読み、拒否ごとに入口と読み方を確定する。 (完了、上表)
2. 署名資格情報を利用できない状態を作る手段を確認する。 (完了、鍵ストアを到達不能にする装飾)
3. プロファイル束縛と Destination の 2 件 (002-02、002-03) を、アサーション発行の不在まで確かめる。
4. 証明書の 1 件 (001-02) を、空と既定値の除外まで含めて進める。
5. 各テストについて防護を外すと落ちることを確かめる。落ちないものは入口が誤っているので入口からやり直す。
6. 拒否の実装が欠けていた場合は修正する。
7. 対応の取れた id を台帳から削除し、この項目が持つ id が 1 件も残っていないことを確認する。

## Tasks

- [x] T001 [Inventory] 3 件の入口と読み方を確定し、署名資格情報を欠いた状態を作る手段を決める。
- [x] T002 [Acceptance] プロファイル束縛と Destination の拒否 (002-02、002-03) をアサーション発行の不在まで確かめるテストを書く。
- [x] T003 [Acceptance] 署名資格情報を利用できないときの拒否 (001-02) を、空と既定値を返さないことまで確かめるテストを書く。
- [x] T004 [App] 拒否の実装が欠けていた経路を修正する。
      欠けていた経路は 1 つも見つからなかった。実装の変更は無い。
- [x] T005 [Ledger] 対応の取れた id を `example-coverage-debt.json` から削除する。
- [x] T006 [Verify] 防護を意図的に外すと各テストが落ちることを確かめ、検査を通す。

## Verification

- `mise run test-go-race`
- `mise run check-spec`
- `mise run check-security-controls`
- `mise run report-coverage-debt`
- `mise run check-work-items`
- `mise run check-ids`
- `mise run verify`

## Intended RED checks

この項目は宣言済みの拒否へ検証を足すもので、新しい製品要求を作らない。

そのため通常の Acceptance RED — 「まだ無い振る舞いを求めるテストが落ちる」 — は成立しない。

代わりに、各テストについて防護を外した状態でテストが落ちることを観測する。

これは実装済みの防護に対して取れる唯一の RED であり、同時に medium risk が求める変更耐性の証拠でもある。

Unit RED は、防護が欠けていた経路が見つかった場合にだけ発生する。

## Risk Notes

拒否のテストは、防護に到達する前の入力検証で落ちていても同じ応答を返すため、書いた本人にも成立して見える。

これを避けるため、各テストについて防護を外した状態でテストが落ちることを確かめ、落ちなかったものは入口が誤っていると判断してやり直す。

SAML の応答は XML であり、要素の有無を文字列の一致で確かめると、名前空間の接頭辞の違いで誤って成立する。

要素として解析したうえで確かめる。

台帳は id 順に 1 エントリー 1 id なので、Context ごとの work item が並行しても衝突はエントリー単位に収まる。

各項目は自分が持つ id のエントリーだけを削除し、コメント配列と他の id には触れない。

## Completion

- **Completed At**: 2026-09-06
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返す。
  規範の宣言は 1 件も変わっていない。変わったのは、宣言済みの拒否 3 件が
  「応答だけ確かめられている」状態から「発行の不在まで確かめられている」状態へ
  移ったことと、その 3 件が網羅台帳から消えたことである。
  SAML 分の台帳は 21 件から 18 件へ減った。製品コードの変更は無い。
  防護が欠けている経路は 1 つも見つからなかった。

- **Acceptance RED Evidence**:
  - **Test**: `TestSamlSSORefusesUnassignedIDPProfileAndIssuesNoAssertion`
    (`backend/saml/handlers_http/refusal_effects_test.go`)。
    ほかの 2 例も同じ形で観測した。
  - **Requirement**: REQ-SAML-002
  - **Observed Failure**: `SignInService.Issue` から `sp.MatchesIDPProfile` の判定を外すと、
    `profile-b` の SSO エンドポイントへ送った AuthnRequest が 200 を返し、本文の自動 POST
    フォームに `profile-b` の entityID を Issuer とする署名済み Assertion がそのまま現れた。
    SP に割り当てられていないプロファイルの鍵で署名したアサーションが外へ出る形である。
  - **Detection Reason**: テストは 400 を読むだけでなく、レスポンスボディに `SAMLResponse` が無いこと、
    ACS へのリダイレクトが起きないこと、解析した XML に `Assertion` 要素が無いことを
    重ねて条件にしている。拒否の応答を返しつつ発行だけ済ませている実装も捕まえる。

- **Unit RED Evidence**:
  - **Test**: `N/A: 防護が欠けている経路が 1 つも見つからなかったため、単体の RED は発生しない。`
    代わりに実施したのは、実装済みの防護を 1 つずつ外して対応するテストが落ちることの確認である
    (下記 Change-Resistance Results)。
  - **Requirement**: N/A: この項目は宣言済みの拒否へ検証を足すもので、新しい製品要求を作らない。
  - **Observed Failure**: `N/A`
  - **Detection Reason**: `N/A`

- **Change-Resistance Results**:
  防護を 1 つずつ外し、対応するテストだけが落ちることを確かめた。

  - `SignInService.Issue` の `MatchesIDPProfile` を外すと `002-02` が落ちた
    (上記 Acceptance RED Evidence)。
  - `ValidateSignInAt` の `req.Destination != expectedDestination` を外すと `002-03` が落ち、
    別の IdP 宛に作られた AuthnRequest に対して 200 と署名済み Assertion を返した。
  - `handleSamlSigningCertificate` のフェイルクローズ
    (`err != nil || signer == nil || signer.Certificate() == nil` で 500 を返す判定) を外し、
    証明書が無いときに空の DER を PEM へ包む実装に替えると `001-02` が落ちた。
    応答は 200 で、本文は
    `-----BEGIN CERTIFICATE-----\n-----END CERTIFICATE-----` の空証明書だった。
    シナリオが「空文字や既定の値も返さない」と書いている形がそのまま出た。

  1 件だけ、単一の判定を外しても落ちなかったものがある。
  `KeyStoreSignerProvider.Resolve` の `!p.KeyStore.Healthy(ctx)` を外しても `001-02` は緑のままだった。
  テストが作る到達不能な鍵ストアは `GetActiveKey` と `ListPublicKeys` もエラーにするので、
  健全性の判定を外しても資格情報は解決できない。
  この 3 つは「provider が到達不能」という 1 つの状態の 3 つの現れであって、独立した防護ではない。
  テストが固定しているのは「資格情報を解決できないときに証明書を返さない」ことであり、
  特定の 1 行の有無ではない。取り外せる防護に対する変更耐性は、上記のハンドラー側の
  フェイルクローズで確かめている。

- **Verification Results**:
  - `mise run test-go-race` - passed
  - `mise run check-spec` - passed (711 例中 113 id をテストが名指し。3 項目の実施前は 92)
  - `mise run check-security-controls` - passed
  - `mise run report-coverage-debt` - SAML は 21 件から 18 件へ
  - `mise run check-work-items` - passed
  - `mise run check-ids` - passed
  - `mise run verify` - passed
