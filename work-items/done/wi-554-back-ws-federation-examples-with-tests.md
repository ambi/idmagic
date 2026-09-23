---
depends_on: [wi-565-make-backing-declared-examples-cheap]
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-12
priority: p3
change_kind: maintenance
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 宣言済みの具体例にテストを対応付ける作業であり、製品の振る舞いも公開契約も運用手順も変えない。
  references: []
initial_context:
  specification:
    - docs/domain/ws-federation/scenarios.feature.md#REQ-WSFEDERATION-001
    - docs/domain/ws-federation/scenarios.feature.md#REQ-WSFEDERATION-002
    - docs/domain/ws-federation/scenarios.feature.md#REQ-WSFEDERATION-003
    - docs/domain/ws-federation/scenarios.feature.md#REQ-WSFEDERATION-004
    - docs/domain/ws-federation/scenarios.feature.md#REQ-WSFEDERATION-005
  typespec:
    - IdMagic.WsFederation.Operations.WsTrustIssue
    - IdMagic.WsFederation.Operations.WsFederationSignIn
    - IdMagic.WsFederation.Operations.RegisterWsFedRelyingParty
    - IdMagic.WsFederation.Operations.ConfigureEntraFederation
  source:
    - backend/wsfederation/handlers_http/wstrust_handler.go
    - backend/wsfederation/handlers_http/wsfed_handler.go
    - backend/wsfederation/handlers_http/admin_entra_handler.go
    - backend/wsfederation/handlers_http/service.go
    - backend/wsfederation/usecases/signin.go
    - backend/wsfederation/requests_wstrust/rst.go
    - backend/shared/http/support_http/application_gate.go
    - backend/shared/http/testing_stack/stack.go
    - tools/check/example-coverage-debt.json
  tests:
    - backend/wsfederation/handlers_http/wsfed_handler_test.go
    - backend/shared/http/support_http/admin_scope_test.go
    - backend/shared/http/server_http/saml_scope_examples_test.go
    - backend/shared/http/server_http/application_api_token_tenant_test.go
    - backend/saml/handlers_http/scenario_examples_test.go
  stop_before_reading:
    - backend/wsfederation/db_postgres
    - backend/wsfederation/tokens_saml
    - backend/wsfederation/metadata_wsfederation
    - infra
spec_impact: { kind: none, reason: "宣言済みの具体例に、その id を名指しするテストを対応付ける作業である。シナリオも製品の振る舞いも変えない。テストが書けない具体例が見つかった場合、それは実装が具体例のとおりに振る舞っていないということなので、欠陥として個別の work item に切り出す。" }
---

# WsFederation が宣言する具体例 11 件にテストを対応付け、被覆台帳から外す

## Motivation

[[wi-496-burn-down-the-example-coverage-debt]] は具体例の被覆台帳の消化単位を Context と決め、測定のうえで残りを Context ごとの子 work item へ割った。本項目はそのうち `docs/domain/ws-federation/scenarios.feature.md` が宣言する 11 件を引き取る。

親項目が `claim-mapping` の 3 件と `workloadidentity` の 13 件で測った結果は、**16 件のうち注記だけで済んだのは 4 件だけ**だというものである。3 件はテストが 1 つも無く、9 件は既存テストへ新しい観測を足す必要があった。件数は作業量の目安にならない。

## Scope

- 11 件を 1 件ずつ確認し、次のいずれかに解決して `tools/check/example-coverage-debt.json` から外す。
  - 当の具体例を検証しているテストが実在する → そのテストに `// EX-WSFEDERATION-NNN-MM: <この具体例の何を固定しているか>` の注記を足す。
  - 当の具体例を検証しているテストが無い → 書く。具体例が拒否なら、[[wi-392-refusal-tests-assert-the-absent-effect]] が定める形（拒否応答と、拒否が防いだ効果の双方を観測する）で書く。
  - 具体例が宣言されなくなっている → 台帳から外す（検査が落ちて教える）。
- 注記は「何を固定しているか」を書く。id だけの注記は禁止する。
- 実装にバグがあって具体例のとおりに振る舞っていないことが分かった場合は、難しくない修正なら本 work item で修正する。難しい修正なら別 work-item に切り出す。

## Out of Scope

- 他の Context が宣言する具体例。Context ごとに別の work item が持つ。
- `tools/check/standards-coverage-debt.json`。[[wi-495-burn-down-the-standards-coverage-debt]] が消化済みである。
- 既に台帳に載っていない拒否テストが、防いだ効果を観測していない件。[[wi-392-refusal-tests-assert-the-absent-effect]] が持つ。本項目が新しく書く拒否テストは、その規範を満たす形で書く。既存テストに注記を足すだけの件で、そのテストが効果の不在を見ていない場合は、注記を足したうえで wi-392 の対象として残す。
- シナリオと具体例の追加、削除、書き換え。具体例の記述が実装と食い違うことが判明した場合は規範の変更なので、別の work item が扱う。
- 行カバレッジ率の目標または閾値。
- `report-coverage-debt` の分類（`named`、`nearby`、`none`）を根拠にした台帳からの削除。分類は読む順を決める材料であり、台帳から外す根拠にはならない。

## Design

### 消化の対象と観測の位置

11 件は 3 つの入口に分かれる。入口ごとに観測の位置を決め、具体例 1 件に `//spec:covers` の注記 1 つを対応させる。

| 入口 | 具体例 | 観測の位置 |
| --- | --- | --- |
| 管理 API の粒度スコープ | EX-WSFEDERATION-001-01、001-02、001-03 | `backend/wsfederation/handlers_http` の新しいテストファイル。`testing_stack` の `WithApiTokens` と `WithWsFederation` で建て、実際に発行した API アクセストークンで `/api/admin/v1/wsfed/*` を叩く |
| パッシブサインイン `/wsfed` | EX-WSFEDERATION-002-01、002-02、002-03、003-01 | `backend/wsfederation/handlers_http/wsfed_handler_test.go` の既存 fixture |
| 能動 STS `/trust/usernamemixed` | EX-WSFEDERATION-004-01、004-02、004-03、005-01 | 同上 |

パッシブと能動の入口を `testing_stack` へ移さないのは、`testing_stack` に認証済みの利用者文脈を差し込む option が無いためである。パッシブサインインは認証済みのセッションを前提にし、既存 fixture はそれを固定の認証文脈で与えている。`testing_stack` へ差し込み口を足すと、製品の組み立てと一致させるという基盤の既定から外れる。WI が言う「触る必要が出た範囲だけ移す」に従い、既存 fixture を割り当てゲートの配線が足せる形に広げるだけにとどめる。

`Then` の数だけ観測を置く。拒否の具体例では、拒否の応答に加えて、トークンが出ていないこと（応答本文と発行イベントの双方）と、拒否イベントが出ていることを読む。拒否のたびに、同じ入口で崩していない要求が通る対照を置く。

### 具体例ごとの判断

| 具体例 | 既存テストの状態 | 行うこと |
| --- | --- | --- |
| 001-01 | 判定関数だけを呼ぶテストがあり、入口を通していない | 新しく書く。read は一覧だけに届き、write は RP の登録・削除と Entra の構成に届き一覧に届かないことを、保存先の読み直しとともに見る |
| 001-02 | 同上 | 新しく書く。read での登録・削除・Entra 構成が 403 で拒否され、保存先が変わらないことを見る |
| 001-03 | 無い | 実測し、台帳に残す（下の「直さない食い違い」） |
| 002-01 | 署名検証まで見る既存テストがあるが、`wctx` の往復を見ていない | 新しく書く。wtrealm、wreply、wfresh、割り当てを 1 つずつ崩してトークンが出ないことと、割り当て済みの利用者へ署名済みの RSTR と同じ wctx が返ることを見る |
| 002-02 | 再認証への誘導は見ているが、トークンが出ないことを見ていない | 新しく書く。トークンが出ないことを見て、新しい認証なら同じ `wfresh` で発行される対照を置く |
| 002-03、003-01 | wtrealm、wreply、wauth の拒否が別々のテストに散り、イベントかトークン不在の片方しか見ていない。割り当ての拒否は無い | 4 つの次元を 1 本で回す拒否テストを書く。003-01 の 3 次元は 002-03 の 4 次元の部分集合なので、同じテストが両方を名指す |
| 004-01 | 成功経路と一部の拒否はあるが、必須要素を「すべて」検証することを 1 本で見ていない | 新しく書く。成功経路の RSTR を読み、必須要素を 1 つずつ崩してトークンが出ないことを見る |
| 004-02 | リプレイの拒否は見ているが、`WsTrustTokenRejected` を見ていない | 新しく書く |
| 004-03 | 401 とトークン不在は見ている | 具体例は `AccessDeniedError` と言い、実装は 401 の平文を返す。台帳に残す（下の「直さない食い違い」） |
| 005-01 | To、Action、リプレイは見ているが、MessageID、AppliesTo、RequestType、KeyType の崩しとイベントの観測が揃っていない | 新しく書く。6 要素を 1 つずつ崩し、400 または 401、`WsTrustTokenRejected`、トークン不在の 3 つを見る |

001-02 の具体例は拒否を `AccessDeniedError` と呼ぶが、スコープ不足で製品が返すのは 403 の `insufficient_scope` である。契約は 403 の本文として `InsufficientScopeError` と `AccessDeniedError` の双方を宣言している。SAML の同型の具体例 EX-SAML-005-02 と同じ読み方をとり、403 で拒否されることを具体例の観測とし、型は `insufficient_scope` として固定する。

### 直す欠陥

004-01 の成功経路を RP と同じ手順で読むと、能動 STS が返す RSTR の assertion は IdP の証明書で検証できなかった（`crypto/rsa: verification error`）。`requests_wstrust.BuildRSTR` が直列化の前に文書全体へ `doc.Indent(2)` をかけていて、包んだ署名済み assertion の内部（`SignedInfo` を含む）にも空白のテキストノードが入り、正規化した形が署名時と変わるためである。パッシブ側の `responses_wsfederation.BuildRSTR` は整形しないので、同じ assertion が検証できる。

修正は `Indent` を外すことだけであり、RSTR の要素も契約も変わらない。設計上の判断をやり直さないので、Scope の「難しくない修正」として本項目で直す。単体境界では `requests_wstrust` に、`BuildRSTR` が包んだ SAML 1.1 と 2.0 の assertion の署名を検証するテストを置く。既存の単体テストは RSTR の外形だけを見ていたので、この破損を検出できなかった。

### 直さない食い違い

- **001-03。** トークンのテナントとリクエスト先のテナントが一致しないとき、具体例は `AccessDeniedError` と言う。SAML の EX-SAML-005-03 と同じ組み立てであり、実測すると実装は 401 `invalid_token` を返した。[[wi-558-name-the-refusal-a-cross-tenant-api-token-actually-gets]] の判断を待つ行として台帳に残す。
- **004-03。** UsernameToken の資格情報が不正なとき、具体例は `AccessDeniedError` を返すと言う。実装は 401 の平文 `invalid credentials` を返し、契約 `WsTrustIssue` はこの 401 を `WsTrustIssueError401` として宣言している。どの経路も 403 `AccessDeniedError` を返さない。資格情報の誤りは認証の失敗であり、ポリシーを満たさないことを表す `AccessDeniedError` とは意味が違う。写像の欠落ではなく、具体例と実装のどちらを正とするかの判断なので、別の work item へ切り出して台帳に残す。

## Plan

1. 台帳から 11 件を外し、`mise run check-spec` が 11 件を名指しで落とすことを観測する。残す件は最後に理由を付けて戻す。
2. 管理 API の 3 件を実測する。001-03 の応答を測り、台帳に残すかを決める。
3. 能動 STS の 4 件、パッシブサインインの 4 件の順にテストを書き、1 件ずつ台帳から外す。
4. 004-03 の判断を引き取る work item を起票し、台帳の行へ `blocked_by` と `finding` を書く。
5. 変更したテストの所属パッケージを変異させ、生き残りを読む。

## Tasks

- [x] T001 [Acceptance] `mise run check-spec` が WsFederation の 11 件を名指しで落とすことを、消化前に観測する。
- [x] T002 [Adapters] 管理 API の粒度スコープの 001-01、001-02 を消化し、001-03 を実測する。
- [x] T003 [Adapters] 能動 STS の 004-01、004-02、005-01 を消化し、004-03 を実測する。
- [x] T004 [Adapters] パッシブサインインの 002-01、002-02、002-03、003-01 を消化する。
- [x] T005 [Docs] 残す 2 件に `blocked_by` と `finding` を書き、004-03 の work item を起票する。
- [x] T006 [Verify] 変異の読み取りと `mise run verify`。

検証の手順は次のとおりとする。RED、GREEN、故障注入は `mise run test-go-test -- ./backend/wsfederation/handlers_http <Test>` で 1 本ずつ回す。入口ごとに GREEN になったら `mise run test-go-package -- ./backend/wsfederation/handlers_http` と `mise run lint-go` を回す。

## 作業の進め方

[[wi-565-make-backing-declared-examples-cheap]] が、この作業の 1 件あたりの費用を下げる道具を用意した。使う。

- 読む順と観測点は `mise run spec-route -- <id>` が出す。規則、当の具体例の本文、契約が宣言する候補 operation とそのメソッド・パス・スコープ、同じ規則の隣の id を名指している既存テストが返る。**出力は読む順を決める材料であり、台帳から外す根拠にはしない。**
- 新しく書くテストは `backend/shared/http/testing_stack` の上に載せる。`Register` と同じ配線が option の合成で建ち、保存先は型付きの field から読み直せる。既存 fixture の全面移行はしない。触る必要が出た範囲だけ移す。
- 変更した Go の変異は `mise run test-go-mutation -- <package-directory>` で読む。手で書く故障注入は、変異器が表現できない「配線を外す」「既定の分岐を差し替える」種類だけに残す。
- 実装が具体例と食い違って消化できない件は、台帳の当該行へ `blocked_by`（先に決着すべき work item）と `finding`（実装が実際に何を返すか）を書いて残す。散文にだけ書くと、次の読み手が同じ測定をやり直す。

## Verification

- `mise run check-spec` が、`docs/domain/ws-federation/scenarios.feature.md` の 11 件を `tools/check/example-coverage-debt.json` から外した状態で通る。台帳から外す前に同じ検査が当の id を名指しで落とすことを、消化ごとに観測する。
- 消化したテストの所属パッケージに対する `mise run test-go-package -- <package>`。
- `mise run verify`

## Risk Notes

- **注記だけを足して終わる。** `checkNormativeCoverage` は文字列の一致しか見ないので、読まずに id を貼れば件数は速く減る。減った件数は何も意味しない。注記に「この具体例の何を固定しているか」を書かせることで区別する。親項目の測定では、16 件のうち注記だけで済んだのは 4 件だけだった。
- **`nearby` に分類された件を、テストがある証拠として読む。** 分類が言っているのは「近傍に他の拒否テストがある」だけである。親項目の測定では、`claim-mapping` の 3 件はすべて `nearby` でありながら 2 件はテストが無かった。分類は読む順の材料にとどめる。
- **拒否である具体例のテストが、応答の字面だけを見て書かれる。** 新しく書く拒否テストには [[wi-392-refusal-tests-assert-the-absent-effect]] の規範が効く。注記へ「効果の不在を何で観測したか」を書き、後から区別できるようにする。
- **具体例の `Then` が複数あるのに、観測が 1 つで済まされる。** 親項目の測定では、`EX-WORKLOADIDENTITY-008-01` のようにイベントと実際の効果の双方を言う具体例が複数あった。`Then` の数だけ観測が要る。

## Completion

- **Completed At**: 2026-09-23
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返す。規範は 1 行も動いていない。
  変わったのは、`docs/domain/ws-federation/scenarios.feature.md` が宣言する 11 件のうち 9 件が、
  その id を名指しするテストから到達されるようになったことと、
  能動 STS `/trust/usernamemixed` が返す RSTR の assertion が IdP の証明書で検証できるようになったことである。
  修正前は `requests_wstrust.BuildRSTR` の整形が署名済み assertion の内部に空白を足し、署名を壊していた。
  台帳は 57 件から 48 件へ減った。残した 2 件には `blocked_by` と `finding` を書いた。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`（`checkNormativeCoverage`）
  - **Requirement**: REQ-WSFEDERATION-001
  - **Observed Failure**: 同じ検査が REQ-WSFEDERATION-001 から REQ-WSFEDERATION-005 までの具体例を同時に名指しした。
    WsFederation の 11 件を `tools/check/example-coverage-debt.json` から外した状態で exit 1。
    11 行それぞれが `EX-WSFEDERATION-NNN-MM is declared, but no test names it.` の形で id を名指しした。
  - **Detection Reason**: この検査は宣言された id と、テストの `//spec:covers` が名指した id の集合を比べる。
    テストを書かずに台帳から外せば必ず落ちるので、「消化した」と「台帳から消した」を取り違えられない。
- **Unit RED Evidence**:
  - **Test**: `TestBuildRSTR_CarriesTheSignedAssertionVerifiably`
    (`backend/wsfederation/requests_wstrust/rstr_test.go`)
  - **Requirement**: REQ-WSFEDERATION-004
  - **Observed Failure**: `SAML version 1: the assertion carried in the RSTR does not validate: crypto/rsa: verification error`。
    入口でも `TestWsTrustIssue_ValidatesEveryRequiredElementAndAnswersAValidRST` が
    `assertion signature did not validate against the IdP certificate: crypto/rsa: verification error` で落ちた。
  - **Detection Reason**: RP と同じ手順で RSTR から assertion を取り出し、署名まで検証する。
    既存の単体テストは RSTR の外形（要素の有無と AppliesTo）だけを見ていたので、署名を壊す整形を通していた。
    `doc.Indent(2)` を外すと両テストとも GREEN になる。
- **Change-Resistance Results**:
  リスクは low だが、消化したテストが実際に何を検出するかを、変異器が表せない配線の故障 6 件で確かめた。6 件とも検出された。
  1. `SignInService.Issue` の割り当てゲートの判定を無視する →
     `TestWsFedPassiveSignIn_FailsClosedOnEveryUntrustedDimension` と
     `TestWsFedPassiveSignIn_ValidatesTheRequestAndReturnsASignedRSTRForm` が落ちる。
  2. RST の解析失敗で `WsTrustTokenRejected` を発行しない → `TestWsTrustIssue_RejectsAMalformedEnvelope` が落ちる。
  3. MessageID のリプレイ記録の結果を無視する → `TestWsTrustIssue_RejectsAReusedMessageIDWithinTheAssertionLifetime` が落ちる。
  4. 自動 POST フォームへ `wctx` を渡さない → `TestWsFedPassiveSignIn_ValidatesTheRequestAndReturnsASignedRSTRForm` が落ちる。
  5. `wfresh` の判定を無視する → `TestWsFedPassiveSignIn_StaleAuthenticationIsSentToReauthenticateWithoutAToken` と
     `TestWsFedPassiveSignIn_ValidatesTheRequestAndReturnsASignedRSTRForm` が落ちる。
  6. `requireAdminApiTokenScope` を素通しにする → `TestWsFedAdminOperationsFollowTheGranularScopes` と
     `TestWsFedReadScopeCannotChangeTrustSettings` が落ちる。

  `mise run test-go-mutation -- backend/wsfederation/requests_wstrust` は 33 個の変異を作り、25 個を殺し 5 個が生き残った。
  変異器はそのパッケージ自身のテストしか回さない。生き残りのうち 108 行（RequestType）、111 行（KeyType）、
  175 行（RelatesTo）の 3 個は、手で同じ変異を入れると入口のテスト
  `TestWsTrustIssue_RejectsAMalformedEnvelope` と `TestWsTrustIssue_ValidatesEveryRequiredElementAndAnswersAValidRST` が殺した。
  残る 2 個は今回変えていない行にある。103 行は Timestamp の Created に許す時計のずれ（5 分）で、
  どのテストも 5 分以内の未来の Created を送らないので固定されていない。
  190 行は TokenType が空のときに要素を省く分岐で、入口は常に TokenType を渡すため経路に無い。
  修正した行（`Indent` の削除）は変異器の演算子では表せず、上の Unit RED がその故障そのものである。
- **Verification Results**:
  - `mise run verify` - passed
  - `mise run check-spec` - passed（本項目の完了時点の残りは `EX-WSFEDERATION-001-03` と `EX-WSFEDERATION-004-03` の 2 件。後者は wi-14033 が消化した）
  - `mise run test-ui-e2e` - 実行していない。変えたのは能動 STS の SOAP 応答の直列化とテストだけで、
    製品のフロントエンドはこの入口を呼ばない。

### 消化した具体例とテストの対応

| 具体例 | テスト |
| --- | --- |
| EX-WSFEDERATION-001-01 | `TestWsFedAdminOperationsFollowTheGranularScopes` |
| EX-WSFEDERATION-001-02 | `TestWsFedReadScopeCannotChangeTrustSettings` |
| EX-WSFEDERATION-002-01 | `TestWsFedPassiveSignIn_ValidatesTheRequestAndReturnsASignedRSTRForm` |
| EX-WSFEDERATION-002-02 | `TestWsFedPassiveSignIn_StaleAuthenticationIsSentToReauthenticateWithoutAToken` |
| EX-WSFEDERATION-002-03 | `TestWsFedPassiveSignIn_FailsClosedOnEveryUntrustedDimension` |
| EX-WSFEDERATION-003-01 | `TestWsFedPassiveSignIn_FailsClosedOnEveryUntrustedDimension` |
| EX-WSFEDERATION-004-01 | `TestWsTrustIssue_ValidatesEveryRequiredElementAndAnswersAValidRST` |
| EX-WSFEDERATION-004-02 | `TestWsTrustIssue_RejectsAReusedMessageIDWithinTheAssertionLifetime` |
| EX-WSFEDERATION-005-01 | `TestWsTrustIssue_RejectsAMalformedEnvelope` |

### 台帳に残した 2 件

- `EX-WSFEDERATION-001-03` は残した。`acme` レルムで発行した `wsfed:read` と `wsfed:write` のトークンを
  `default` レルムの一覧、登録、削除、Entra 構成へ提示すると、8 通りすべてが 401 `invalid_token` で、保存先の RP は変わらない。
  拒否は効いていて、食い違っているのは型だけである。EX-SAML-005-03 と同じ経路なので、
  [[wi-558-name-the-refusal-a-cross-tenant-api-token-actually-gets]] の判断を待つ。
- `EX-WSFEDERATION-004-03` は残した。誤ったパスワードと未知の username はどちらも 401 の平文 `invalid credentials` と
  `WsTrustTokenRejected` で拒否され、トークンは出ない。具体例は `AccessDeniedError` と言うが、
  資格情報の誤りは認証の失敗であり、写像の欠落ではなく規範と実装のどちらを正とするかの判断になる。
  [[wi-14033-name-the-refusal-a-wrong-ws-trust-credential-gets]] を起票して引き渡した。
  同じ変更のなかで wi-14033 が具体例を 401 に合わせ、テストを付けて台帳から外した。

### 親項目の測定への追加

親項目 [[wi-496-burn-down-the-example-coverage-debt]] は「注記だけで済むのは少数」と測った。WsFederation では 0 件だった。
既存テストのうち 4 件（002-01、002-02、004-02、005-01）は観測の一部を持っていたが、
`wctx` の往復、トークン不在、拒否イベントのどれかが欠けていた。9 件すべてに新しいテストを書いた。
新しく書いた 004-01 の成功経路の観測が、能動 STS の RSTR の署名破損という既存の欠陥を見つけた。
