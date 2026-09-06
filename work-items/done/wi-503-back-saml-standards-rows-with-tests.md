---
depends_on: []
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-06
priority: p2
change_kind: maintenance
spec_impact: { kind: none, reason: "宣言済みの標準行に、その id を名指しするテストを対応付ける作業である。standards.md の行そのものも製品の振る舞いも変えない。テストが書けない行が見つかった場合、それは製品が宣言した採用を満たしていないということなので、欠陥として個別の work item に切り出す。" }
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 標準の行にも製品の振る舞いにも変更が無く、増えたのはテストと注記だけなので、リリースの読み手に見えるものが無い。
  references: []
initial_context:
  specification:
    - docs/contexts/saml/standards.md
    - docs/contexts/saml/scenarios.feature.md
  typespec: []
  source:
    - backend/saml/domain/authnrequest.go
    - backend/saml/domain/service_provider.go
    - backend/saml/usecases/signin.go
    - backend/saml/handlers_http/sso_handler.go
    - backend/saml/handlers_http/routes.go
    - backend/saml/handlers_http/admin_service_provider_handler.go
    - backend/saml/handlers_http/metadata_handler.go
    - backend/saml/handlers_http/slo_handler.go
    - backend/saml/metadata_saml/idp_metadata.go
    - backend/saml/responses_saml/response.go
    - backend/cmd/idmagic/server.go
    - tools/check/standards-coverage-debt.json
  tests:
    - backend/saml/handlers_http/saml_handler_test.go
    - backend/saml/handlers_http/refusal_effects_test.go
    - backend/saml/metadata_saml/idp_metadata_test.go
    - backend/saml/responses_saml/response_test.go
  stop_before_reading:
    - backend/oauth2
    - backend/wsfederation/requests_wstrust
    - frontend
---

# SAML が宣言する標準 6 行にテストを対応付け、標準の被覆台帳から外す

## Motivation

[[wi-495-burn-down-the-standards-coverage-debt]] は標準の被覆台帳へ受入集合を入れ、消化の単位を所有文書と決めた。本項目はそのうち `docs/contexts/saml/standards.md` の 6 行を引き取る。この文書は 7 行のうち 6 行が名指しを持たない。

SAML の行 id は `SAML2Core-BearerAssertion` のように大文字と小文字が混じる。[[wi-418-normative-coverage-gates]] の当初の実装は「ハイフンで繋いだ大文字の並び」を id の形と決め打ちしていたため、SAML と WS-Federation の 13 行はどれほど明白に名指されても被覆と認められなかった。この形の推測は既に取り除かれ、宣言された id そのものを探すようになっている。したがって本項目の消化は、名指しさえ書けば必ず届く。

## Scope

- 次の 6 行を消化する。

| ID | Adoption |
|---|---|
| `SAML2Profile-WebBrowserSSO` | required |
| `SAML2Bindings-RedirectPost` | required |
| `SAML2Metadata-IDPSSODescriptor` | required |
| `SAML2Metadata-WantAuthnRequestsSigned` | optional |
| `SAML2Core-EncryptedAssertion` | excluded |
| `SAML2Profile-ECP` | excluded |

- 1 行ごとに、その行の `Statement` を区別できる入力と観測を持つテストを対応付け、`// <ID>: <成り立つ性質>` の注記を足す。注記は性質を平叙文で述べる。
- 観測の形は行の `Adoption` に従う。型は [[wi-495-burn-down-the-standards-coverage-debt]] の Design が定める。
- 消化した id を `tools/check/standards-coverage-debt.json` から外す。

## Out of Scope

- 他の文書が宣言する標準行。文書ごとに別の work item が持つ。
- `standards.md` の行の追加、削除、`Adoption` および `Strength` の変更。
- 見つかった実装の欠陥の修正。欠陥として切り出した先で扱う。
- 受入集合の導入と、台帳ファイルそのものの削除。親の [[wi-495-burn-down-the-standards-coverage-debt]] が持つ。
- 具体例の被覆負債。[[wi-496-burn-down-the-example-coverage-debt]] が持つ。
- SAML の適合範囲そのものの拡張。`excluded` の 2 行を提供へ変えるのは規範の変更であり、別の work item が扱う。
- id を名指しする文字列があるだけを根拠にした台帳からの削除。

## Design

6 行のうち 2 行が `excluded` である。台帳全体で `excluded` は 15 行あり、そのうち 2 行がここにある。**`excluded` の観測は `required` と逆向きになる。** 行の `Statement` は製品の振る舞いではなく標準側の機能を書いているので、テストはそれを満たすことではなく満たさないことを観測する。

`SAML2Core-EncryptedAssertion` は暗号化アサーションを提供しないという行である。観測は、暗号化アサーションを要求または提示されたときに製品が何をするかである。拒否するなら、その拒否と、拒否が防いだ効果を観測する。単に提供していないだけなら、その機能への入口が存在しないことを観測する。どちらであるかは実装を読んで決める。

`SAML2Profile-ECP` は ECP プロファイルを提供しないという行である。ECP は `PAOS` バインディングを使うので、入口の有無は要求の形で区別できる。

`SAML2Metadata-WantAuthnRequestsSigned` は `optional` である。提供しているならその振る舞い、つまりメタデータに `WantAuthnRequestsSigned` を出し、署名の無い `AuthnRequest` をどう扱うかを観測する。提供していないなら行の `Adoption` が誤っているので規範の変更として切り出す。

`required` の 3 行は、宣言した振る舞いが製品の正式な入口から到達できることを観測する。`SAML2Bindings-RedirectPost` は Redirect と POST の 2 つのバインディングを名指しているので、片方だけを観測しても行を区別できない。両方を観測する。

### 入口とハーネスの組み立て

観測は `httpadapter.Register` が組み立てた `/saml/*` を通す。ハーネスの配線は `backend/cmd/idmagic/server.go`（`Saml: deps.Saml`、`FederationSigner: samltoken.KeyStoreSignerProvider{...}`）と同じ形にした。**組み立てが本番と違えば、入口を通したという事実そのものが根拠にならない。** [[wi-500-back-api-tokens-standards-rows-with-tests]] はここを間違えて、製品ではなく自分で組み立てたスタックを測り、存在しない欠陥を報告した。本項目では配線を突き合わせてから始めた。

### `excluded` の観測の型（本文書で決めた）

**その機能の入口が存在しないことを、公開している契約と実際の応答の両方から観測する。対照として、提供している側の経路が成立することを併せて読む。**

SAML の 2 行はどちらも、製品がその機能を実装していない種類の `excluded` である。拒否の応答が返るわけではないので、[[wi-495-burn-down-the-standards-coverage-debt]] が挙げた「拒否と、拒否が防いだ効果」の型は使えない。代わりに次を読む。

- メタデータがその機能を広告していない（`SAML2Profile-ECP` なら PAOS / SOAP の `SingleSignOnService`、`SAML2Core-EncryptedAssertion` なら `KeyDescriptor use="encryption"`）。
- 実際の応答がその機能の成果物を持たない（ECP の SOAP エンベロープ、`EncryptedAssertion`）。
- 有効化する入口が無い（SP 登録に暗号化の設定が入らない）。

対照が要る理由は、広告が空でも、応答が空でも、上の 3 つは通ってしまうからである。提供している 2 つのバインディングが広告されていること、平文の Assertion が実際に発行されること、署名鍵は広告されていることを併せて読む。

[[wi-501-back-authentication-standards-rows-with-tests]] の型（「採用した実装なら拒否する入力が受理される」）とも、[[wi-500-back-api-tokens-standards-rows-with-tests]] の型（「その提示の形では通らず、状態も変わらない」）とも違う。**`excluded` に 1 つの型は無い。行の `Statement` が製品の制約を書いているのか、標準側の機能を書いているのか、その機能を実装していないのかで観測は変わる。**

## Plan

1. ~~`SAML2Profile-WebBrowserSSO` から着手し、`required` の観測の型を決める。~~ 完了。
2. ~~`SAML2Bindings-RedirectPost` を、Redirect と POST の両方で消化する。~~ 完了。
3. ~~`SAML2Metadata-IDPSSODescriptor` を、生成されたメタデータを読む形で消化する。~~ 完了。
4. ~~`SAML2Core-EncryptedAssertion` の実装を読み、`excluded` の観測の型をここで決める。~~ 完了。型は Design の該当節。
5. ~~その型で `SAML2Profile-ECP` を消化する。~~ 完了。
6. ~~`SAML2Metadata-WantAuthnRequestsSigned` の実装の有無を確かめ、観測するか切り出すかを決める。~~ 実装あり。観測した。
7. ~~解決した id を台帳から外す。~~ 完了。6 件すべて。

## Tasks

- [x] T001 [Acceptance] `SAML2Profile-WebBrowserSSO` を消化し、`required` の型を決める。
  行が 3 つの句を持つので観測も分けた。SP 起点と IdP 起点は既存の
  `TestSamlSSO_SPInitiatedAuthenticatedIssuesPostForm` と `TestSamlSSO_IdPInitiatedIssuesPostForm` に
  注記を足した。フェイルクローズの拒否は `TestSamlWebBrowserSSOFailsClosedOnUnsupportedRequestParameters`、
  `IsPassive` は `TestSamlWebBrowserSSOReturnsNoPassiveWhenLoginIsRequired` を新設した。
  recipe: `mise run test-go-package -- ./backend/saml/handlers_http`
- [x] T002 [Acceptance] `SAML2Bindings-RedirectPost` を、Redirect と POST の両方で消化する。
  `TestSamlAcceptsRedirectAndPostBindingsAndRepliesByPost`。両バインディングの受理、HTTP-POST 以外の
  `ProtocolBinding` の拒否、SAMLResponse と返信可能なプロトコルエラーが POST で返ることの 5 事例。
  recipe: `mise run test-go-package -- ./backend/saml/handlers_http`
- [x] T003 [Acceptance] `SAML2Metadata-IDPSSODescriptor` を、生成されたメタデータから消化する。
  `TestSamlMetadataPublishesTheIDPSSODescriptorContract`。文字列の有無ではなく、`Binding` ごとの
  `Location` が実際のエンドポイントと一致すること、広告した NameID 形式の集合が製品が受理する集合と
  一致することを読む。
  recipe: `mise run test-go-package -- ./backend/saml/handlers_http`
- [x] T004 [Type] `SAML2Core-EncryptedAssertion` で `excluded` の観測の型を決め、消化する。
  `TestSamlDoesNotOfferEncryptedAssertions`。型は Design の「`excluded` の観測の型」節。
  recipe: `mise run test-go-package -- ./backend/saml/handlers_http`
- [x] T005 [Acceptance] `SAML2Profile-ECP` を同じ型で消化する。
  `TestSamlDoesNotOfferTheECPProfile`。
  recipe: `mise run test-go-package -- ./backend/saml/handlers_http`
- [x] T006 [Inventory] `SAML2Metadata-WantAuthnRequestsSigned` の実装の有無を確かめ、観測するか切り出すかを決める。
  実装がある。`domain.ValidateRequestSignature` が Redirect と POST の両方の署名を検証し、管理 API は
  証明書を伴う有効化だけを受理する。`TestSamlServiceProviderTrustPolicyCanRequireSignedAuthnRequests` で
  観測した。切り出しは不要。
  recipe: `mise run test-go-package -- ./backend/saml/handlers_http`
- [x] T007 [Ledger] 解決した id を `standards-coverage-debt.json` から外す。
  6 件すべてを外した（110 → 104）。
  recipe: `mise run check-spec`
- [x] T008 [Verify] `mise run verify`。

## Verification

- 本項目が持つ 6 件が `tools/check/standards-coverage-debt.json` から 1 件残らず消えている。
- 注記を足した各テストについて、対応する production の判断を崩すとそのテストが落ちる。
- `mise run check-spec`
- `mise run verify`

## Completion

- **Completed At**: 2026-09-07
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返す。規範の変更は無い。
  差分は `docs/contexts/saml/standards.md` の 7 行のうち、名指しを持たなかった 6 行に対する被覆の状態で
  ある。6 行すべてがその行の `Statement` を区別できる入力と観測を持つテストを得て
  `tools/check/standards-coverage-debt.json` から消え、台帳は 110 件から 104 件になった。
  新設したテストは `backend/saml/handlers_http/saml_standards_test.go` の 6 件で、既存の 2 件へ注記を
  足した。製品コードは 1 行も変わっていない。欠陥は 1 件も見つからなかった。
  `excluded` の観測の型を 1 つ決めた（Design の該当節）。**この型は既に決まっていた 2 つの型のどちらとも
  違う。** 標準側の機能を実装していない行は、拒否ではなく「入口が無いこと」を、公開している契約と実際の
  応答の両方から読む。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`（6 件を台帳から外し、テストを書く前の状態で）
  - **Requirement**: N/A: 標準の被覆はテストの有無についての性質であり、製品の規範要求ではない。
  - **Observed Failure**: exit 1。`docs/contexts/saml/standards.md` の 6 行それぞれについて
    `<ID> is declared, but no test names it. Cite the id from the test that exercises it, or list it in
    tools/check/standards-coverage-debt.json with a reason.`（10 行目 `SAML2Core-EncryptedAssertion` から
    36 行目 `SAML2Metadata-WantAuthnRequestsSigned` まで 6 件）
  - **Detection Reason**: 検査は、宣言された id・テストが名指す id・台帳の 3 つを突き合わせる。台帳から
    外した id は、名指すテストが実在しない限り必ず報告される。消化後の同じコマンドは
    `ok normative coverage (156 standard(s), 311 rule(s), 744 example(s), 181 id(s) named by a test)` を返す。
- **Unit RED Evidence**:
  - **Test**: 各行に対応付けたテストへの故障注入（Change-Resistance Results を参照）
  - **Requirement**: N/A: 行ごとの観測は標準の `Statement` に対応し、`REQ` 番号には対応しない。
  - **Observed Failure**: 14 件の変異のうち 13 件を検出し、1 件は等価変異だった。内訳は表。
  - **Detection Reason**: `checkNormativeCoverage` は文字列の一致しか見ないので、id を書くだけで検査は
    通ってしまう。名指しが実在の検証に付いていることの担保は、production を崩したときにそのテストが
    落ちるという観測しかない。
  - **見つけた不足**: 最初に書いた `TestSamlWebBrowserSSOFailsClosedOnUnsupportedRequestParameters` は、
    ACS インデックスの検査を外す変異を生き残った。**全事例が同じ `AuthnRequest` の `ID` を使っていたため、
    対照の 1 件目がリプレイ表を消費し、2 件目以降は確かめたい検査ではなくリプレイ検査で拒否されていた。**
    拒否だけを並べたテストは、別の理由で落ちていても通ってしまう。要求ごとに `ID` を変えたところ、
    この変異を検出するようになった。故障注入をしなければ、6 行のうち 1 行はテストがあるように見えて
    何も観測していない状態のまま完了していた。
- **Change-Resistance Results**:
  6 行すべてについて、行が言っていることを production 側で崩し、対応するテストが落ちることを観測した。

  | 行 | 注入した故障 | 落ちたテストと観測 |
  |---|---|---|
  | `SAML2Profile-WebBrowserSSO` | `handleSamlSSORedirect` が SP 起点の要求を IdP 起点として扱う | `TestSamlSSO_SPInitiatedAuthenticatedIssuesPostForm` |
  | 〃 | `handleIdPInitiated` が常に拒否する | `TestSamlSSO_IdPInitiatedIssuesPostForm` |
  | 〃 | `ACSIndexSpecified` の拒否を外す | `.../AssertionConsumerServiceIndex`: 索引つき要求で Assertion が発行される |
  | 〃 | NameID 形式の検査を 2 か所とも外す | `.../unsupported_NameIDPolicy_format`: 未対応形式で Assertion が発行される |
  | 〃 | `IsPassive` の分岐を外してログインへ倒す | `TestSamlWebBrowserSSOReturnsNoPassiveWhenLoginIsRequired`: NoPassive が返らない |
  | `SAML2Bindings-RedirectPost` | `ProtocolBinding` の検査を外す | `.../a_response_ProtocolBinding_other_than_HTTP-POST_is_refused` |
  | 〃 | `handleSamlSSOPost` が常に拒否する | `.../HTTP-POST_binding_is_accepted` |
  | `SAML2Metadata-IDPSSODescriptor` | SLO エンドポイントを公開しない | `TestSamlMetadataPublishesTheIDPSSODescriptorContract` |
  | 〃 | NameID 形式を公開しない | 同テスト |
  | 〃 | 署名証明書を公開しない | 同テスト |
  | `SAML2Metadata-WantAuthnRequestsSigned` | `ValidateRequestSignature` が常に nil を返す | `TestSamlServiceProviderTrustPolicyCanRequireSignedAuthnRequests`: 署名の無い要求が通る |
  | 〃 | 証明書の無い有効化を管理 API が受理する | 同テスト |
  | `SAML2Core-EncryptedAssertion` | メタデータの `KeyDescriptor` を `use="encryption"` にする | `TestSamlDoesNotOfferEncryptedAssertions` |
  | `SAML2Profile-ECP` | メタデータへ PAOS の `SingleSignOnService` を足す | `TestSamlDoesNotOfferTheECPProfile` |

  等価変異が 1 件ある。`ValidateSignInAt` の NameID 形式の検査は 2 か所にあり、`NameIDPolicy` 側の検査
  だけを外しても、直後の「解決後の形式が受理できるか」の検査が同じ入力を捕まえる。片側だけの変異は
  どのテストも殺さないが、それは観測の穴ではなく実装が二重に守っているということである。2 か所とも
  外した変異は検出される。
- **Verification Results**:
  - `mise run check-spec` - passed（`ok normative coverage (156 standard(s), ..., 181 id(s) named by a test)`）
  - `mise run check-work-items` - passed
  - `mise run check-ids` - passed
  - `mise run verify` - passed
  - `mise run test-ui-e2e` - `N/A: 変更は Go のテストと tools の JSON だけで、ブラウザーへ到達する経路が無い。`

## Risk Notes

- **`excluded` に書けるテストが無い。** 製品がその機能をそもそも実装していないなら、観測できるのは「入口が存在しない」ことだけになりうる。この場合に何を観測とするかは T004 で決める。決められない行は、台帳へ残す理由を `present when the check was introduced` から具体的な理由へ書き換えたうえで残す。理由が更新されていれば、判断済みであることが後から読める。
- **`SAML2Bindings-RedirectPost` を片方のバインディングだけで消化する。** 片方だけでは、もう片方を提供していない実装と区別できない。両方を観測する。
- **台帳の同時編集。** 自分が持つ id のエントリーだけを削除し、コメント配列と他の id には触れない。
