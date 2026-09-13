---
depends_on: [wi-565-make-backing-declared-examples-cheap]
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-12
priority: p3
change_kind: maintenance
spec_impact: { kind: none, reason: "宣言済みの具体例に、その id を名指しするテストを対応付ける作業である。シナリオは変えない。併せて直す欠陥も、既に別経路にある写像や発行の欠落を埋めるものに限る。公開契約は動かない。" }
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 宣言済みの具体例に検証を与える作業であり、公開契約も運用手順も変わらない。
  references: []
initial_context:
  specification:
    - docs/contexts/oauth2/scenarios.feature.md#REQ-OAUTH2-001
    - docs/contexts/oauth2/scenarios.feature.md#REQ-OAUTH2-005
  typespec: []
  source: []
  tests:
    - backend/oauth2/token/usecases/exchange_code_test.go
    - backend/oauth2/handlers_http/authorize_handler_test.go
    - backend/oauth2/handlers_http/validation_test.go
    - backend/shared/http/server_http/authorization_request_standards_test.go
    - backend/shared/http/server_http/routes_e2e_test.go
  stop_before_reading:
    - backend/oauth2/db_postgres
    - frontend
    - infra
---

# OAuth2 が宣言する残り 99 件の具体例にテストを対応付け、被覆台帳から外す

## Motivation

[[wi-538-back-oauth2-examples-with-tests]] は `docs/contexts/oauth2/scenarios.feature.md` が宣言する 105 件を引き取り、REQ-OAUTH2-002 から REQ-OAUTH2-004 までの 6 件を消化して 1 件あたりの所要を測った。**6 件のうち注記だけで済んだものは 1 件も無かった。** 親項目の測定（16 件中 2 件）より悪い。

その所要を根拠に、wi-538 は残る 99 件を 5 つの子 work item へ割った。

**その分割は取り消した。** [[wi-565-make-backing-declared-examples-cheap]] が測り直したところ、所要が高い原因は具体例の側ではなく道具の側にあった。94 個のテストファイルが 44 フィールドの `Deps{}` を手で組み立てていること、具体例 id から観測点を引く索引が無いこと、台帳が読んだ結果を保持しないことである。wi-565 がこの 3 つを直す。原因が消えるなら分割の根拠も消えるので、5 記録を本記録 1 つへ畳んだ。5 つに割ることは readiness と Design と Completion の固定費を 5 回払わせるだけだった。

本項目は残る 99 件すべてを引き取る。

## Scope

- 99 件を 1 件ずつ確認し、次のいずれかに解決して `tools/check/example-coverage-debt.json` から外す。
  - 当の具体例を検証しているテストが実在する → そのテストに `//spec:covers EX-OAUTH2-NNN-MM: <この具体例の何を固定しているか>` のディレクティブを足す。
  - 当の具体例を検証しているテストが無い → 書く。具体例が拒否なら、[[wi-392-refusal-tests-assert-the-absent-effect]] が定める形（拒否応答と、拒否が防いだ効果の双方を観測する）で書く。
  - 具体例が宣言されなくなっている → 台帳から外す（検査が落ちて教える）。
- 読む順は `mise run spec-route -- <id>` で決める。出力は読む順の材料であり、台帳から外す根拠にはしない。
- 新しく書くテストは `backend/shared/http/testing_stack` の上に載せる。既存 fixture を触る必要が出た範囲だけ、同じ基盤へ移す。全面移行はしない。
- 注記は「何を固定しているか」を書く。id だけの注記は禁止する。
- **実装が具体例のとおりに振る舞っていない件のうち、修正が局所で済むものは本項目で直す。** 判定の基準は、直す対象が既存の写像や規則の欠落であり、設計上の判断をやり直さずに済むことである。同じ規則が別の経路に既にあって、そこから写せば済むなら局所である。port の署名や、遷移を誰が行うかを決め直す必要があるなら局所ではない。
- 基準に当てはまらないものは欠陥として切り出し、台帳の当該行へ `blocked_by` と `finding` を書いて残す。
- 変更した Go の変異は `mise run test-go-mutation -- <package>` で読む。手書きの故障注入は、変異器が表現できない「配線を外す」種類だけに残す。

## Out of Scope

- 他の Context が宣言する具体例。Context ごとに別の work item が持つ。
- `EX-OAUTH2-003-04`。[[wi-564-name-the-refusal-a-cross-tenant-oauth-admin-token-gets]] が持つ。
- `tools/check/standards-coverage-debt.json`。[[wi-495-burn-down-the-standards-coverage-debt]] が消化済みである。
- 既に台帳に載っていない拒否テストが、防いだ効果を観測していない件。[[wi-392-refusal-tests-assert-the-absent-effect]] が持つ。本項目が新しく書く拒否テストは、その規範を満たす形で書く。
- シナリオと具体例の追加、削除、書き換え。具体例の記述が実装と食い違うことが判明した場合は規範の変更なので、別の work item が扱う。
- 局所で済まない欠陥の修正。切り出した先で扱う。
- `backend/shared/http/testing_stack` への既存 94 fixture の全面移行。wi-565 の Out of Scope でもある。
- 行カバレッジ率の目標または閾値。
- `report-coverage-debt` の分類（`named`、`nearby`、`none`）を根拠にした台帳からの削除。分類は読む順を決める材料であり、台帳から外す根拠にはならない。

## Design

### 新しい道具で 1 件あたりがどう変わったか

最初の規則群 (REQ-OAUTH2-005 の 8 件) で測った。比較対象は [[wi-538-back-oauth2-examples-with-tests]] が同じ Context の 6 件で払った所要である。

| 段階 | wi-538 | 本項目 |
| --- | --- | --- |
| 観測点を決める | 具体例 1 件あたり 4〜6 回のツール呼び出し (routes.go、operations_gen.go、ハンドラ) | 規則 1 つあたり 5 回。うち 1 回が `mise run spec-route` で、残りはテスト関数名の列挙 |
| 候補テストを読み、`Then` と突き合わせる | 1 件あたり 1 ファイル読み | 変わらない。ここは道具が肩代わりできない |
| 観測を足す、またはテストを書く | 6 件中 5 件 | 3 件中 3 件 |

**経路探索は約 6 倍速くなり、観測の突き合わせは変わっていない。** 後者が残りの費用の大半である。

`spec-route` の候補 operation は、この規則では役に立たなかった。REQ-OAUTH2-005 の本文は `/authorize` も `/token` もバッククォート付きで書いておらず、`InvalidRequestError` だけでは Context のほぼ全 operation が同点で並ぶ。**役に立ったのは「隣の id を名指す既存テスト」の一覧**で、3 ファイルを 1 回で示した。

### 消化の状況

引き取った 93 件のうち **89 件を消化し、4 件を欠陥として切り出した。**

| 具体例 | 状態 |
| --- | --- |
| 89 件 | 消化。台帳から外し、`//spec:covers` を持つテストへ対応付けた |
| EX-OAUTH2-001-01 | 台帳に残す。`aud` がレルムの IdMagic API にならず、account リソースサーバーも audience を読まない。[[wi-570-account-scoped-tokens-name-no-audience]] |
| EX-OAUTH2-006-02 | 台帳に残す。`TokenRevoked` が出ない。原因は 005-07 と同じ `RevokeFamily` の署名。[[wi-566-authorization-code-replay-and-expiry-leave-no-record]] |
| EX-OAUTH2-026-01、035-02 | 台帳に残す。拒否は成立しているが、境界とエラー種別が宣言と違う。[[wi-569-declared-refusals-that-live-at-a-different-boundary]] |

### 費用がどこにあったか

道具の側で効いたのは 2 つで、どちらも**具体例ではなく基盤の欠落**だった。

1 つは **`Deps.Emit` の未配線**である。通常経路の `Then` はほぼイベント発行で書かれているのに、既存 fixture はイベントを 1 つも受けていなかった。私はこれに気づく前に、4 つの fixture (`nigFixture`、`tiFixture`、`tpFixture`、承認の handler) へ同じ配線を手で足している。**その 4 回が無駄だった。** `testing_stack` へ `EventLog` を常設したのはそのあとで、順序が逆だった。次の Context では基盤を先に見る。

もう 1 つは **ブラウザー経由の認可を通す seed** である。wi-538 が `EX-OAUTH2-001-01` を子項目へ回した理由がこれで、「ログインできる利用者」と「リダイレクト先を登録したクライアント」が揃った fixture がどこにも無かった。`WithBrowserFlow` を 1 つ足すと、そこから 7 件が出た。

**残りの費用は候補テストを読んで `Then` と突き合わせる判断で、ここは下がっていない。** コマンドは律速ではない (`check-spec` 1〜2 秒、`lint-go` 5 秒、対象 package のテスト 9 秒)。

### 局所と判定して直した欠陥

認可コードの再提示で発行ファミリーを失効させる 2 箇所 (`exchange_code.go` の使用済み/期限切れ判定と、並行交換の検出) が、`RefreshTokenReuseDetected` を発行していなかった。`refresh_tokens.go` は refresh トークン再利用の検出でまったく同じ組 (`RevokeFamily` の直後に `emit`) を 2 箇所に持っている。既存の写像がそこにあり、認可コード側だけが欠けていたので局所と判定し、`revokeReplayedFamily` へ寄せた。

監査から見ると、この欠落は「refresh トークンの再利用は記録に残り、認可コードの再利用は残らない」という差になっていた。

`TestEndSessionAcceptsExpiredIDTokenHint` は「期限切れのヒントを受け入れる」と名乗りながら、実際には未来の `exp` を持つトークンを渡していた。テスト自身のコメントが「実装が `exp` を参照しないことでも担保される」と書いていて、観測ではなく実装の読みに寄りかかっていた。`exp` を検証する実装を入れても通ってしまうので、`SignPS256` で `iat` と `exp` だけを過去へ動かした本物の期限切れヒントに差し替えた。

### `//spec:covers` は 1 行に収める

gofumpt はこの形をディレクティブとして扱い、doc コメントの**末尾へ動かす**。説明を次の行へ continuation として続けると、その続きだけが上に取り残されて文が裂ける。リポジトリの既存の注記にも同じ裂け方をしたものがある。本項目が足した注記は、説明まで含めて 1 行に収めた。散文が要るときはディレクティブの上に、それだけで完結する文として置く。

## Tasks

- [x] T001 [Acceptance] `mise run check-spec` が対象 9 件を名指しで落とすことを、消化前に観測する。
- [x] T002 [Use Cases] EX-OAUTH2-005-06 を消化する。`mise run test-go-package -- ./backend/oauth2/token/usecases`。
- [x] T003 [Use Cases] EX-OAUTH2-005-07 の `RefreshTokenReuseDetected` 発行漏れを RED → GREEN で直す。
- [x] T004 [Decision] EX-OAUTH2-005-07 の `TokenRevoked` と EX-OAUTH2-005-08 の `expired` 遷移を局所でないと判定し、[[wi-566-authorization-code-replay-and-expiry-leave-no-record]] へ切り出して台帳へ finding を書く。
- [x] T005 [Tooling] `testing_stack` にイベント記録 (`EventLog`) とブラウザー駆動 (`WithBrowserFlow`、`Browser`) を足す。通常経路の `Then` はほぼイベント発行で書かれているのに、既存 fixture は `Deps.Emit` を配線していなかった。
- [x] T006 EX-OAUTH2-005-01 から 005-05、008、009-01、021-01、022-01 を、その基盤の上で消化する。`mise run test-go-package -- ./backend/oauth2/handlers_http`。
- [x] T007 残る REQ-OAUTH2-006 以降を消化する。既存テストのある行は `//spec:covers` を足し、`Then` の観測が欠けている分を補強する。
- [x] T008 [Decision] 具体例と実装が食い違う 4 件 (001-01、006-02、026-01、035-02) を局所でないと判定し、[[wi-566-authorization-code-replay-and-expiry-leave-no-record]]、[[wi-569-declared-refusals-that-live-at-a-different-boundary]]、[[wi-570-account-scoped-tokens-name-no-audience]] へ切り出して台帳へ finding を書く。
- [x] T009 [Tooling] `testing_stack` の変異を読み、駆動部が自 package のテストから 1 度も実行されていないことを直す。`mise run test-go-mutation -- ./backend/shared/http/testing_stack`。
- [x] T010 [Verify] `mise run verify`。

## Verification

- `mise run check-spec` が、対象の 99 件を `tools/check/example-coverage-debt.json` から外した状態で通る。台帳から外す前に同じ検査が当の id を名指しで落とすことを観測する。
- 消化したテストの所属パッケージに対する `mise run test-go-package -- <package>`。
- `mise run verify`

## Risk Notes

- **注記だけを足して終わる。** `checkNormativeCoverage` は文字列の一致しか見ないので、読まずに id を貼れば件数は速く減る。減った件数は何も意味しない。注記に「この具体例の何を固定しているか」を書かせることで区別する。wi-538 の測定では、6 件のうち注記だけで済んだものは 1 件も無かった。
- **`spec-route` の出力を、テストがある証拠として読む。** 候補 operation は契約から導出した読む順の材料である。`report-coverage-debt` の分類が根拠にならないのと同じ理由で、これも根拠にはならない。
- **具体例の `Then` が複数あるのに、観測が 1 つで済まされる。** `EX-OAUTH2-005-01` は 5 つのイベント発行を並べ、`EX-OAUTH2-005-07` は応答とファミリー失効とイベント 2 種を並べている。`Then` の数だけ観測が要る。
- **並行交換を、逐次の 2 回で代用する。** `EX-OAUTH2-042-05` は「並行に 2 回交換する」と言っている。逐次に 2 回叩くテストは `042-04` の再交換であって、`042-05` ではない。
- **委譲深さの上限を、境界の片側だけで観測する。** `EX-OAUTH2-048-02` は上限以内、`048-03` は上限超えを言う。片側だけでは比較演算子の向きを取り違えた実装を見分けられない。
- **Agent の種別を、既知の値だけで観測する。** `EX-OAUTH2-050-03` は「既知のどの値でもない」場合を言う。列挙を網羅するテストはこの件を通してしまう。
- **99 件を、また量を理由に割る。** 割ってよいのは意味が 2 つあるときだけである。所要が下がらないと分かったら、下がらない原因を測って道具の側を直す。それが wi-565 のやり方である。

## Completion

- **Completed At**: 2026-09-13
- **Summary**:
  `mise run spec-diff -- 00a1aa20` は「no normative specification change」を返す。規範は動いていない。
  動いたのは対応付けで、OAuth2 が宣言する具体例のうち本項目が引き取った 93 件から 89 件を
  `tools/check/example-coverage-debt.json` から外し、`//spec:covers` を持つテストへ結び付けた。
  台帳の OAuth2 行は 96 件から 7 件へ減り、残る 7 件はすべて `blocked_by` と `finding` を持つ。
  併せて、`testing_stack` がイベントの記録とブラウザー経由の認可を配るようになった。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`
  - **Requirement**: REQ-OAUTH2-005
  - **Observed Failure**: 対象 id を台帳から外した状態で 86 件を名指しで落とした。例:
    `docs/contexts/oauth2/scenarios.feature.md:110: EX-OAUTH2-005-01 is declared, but no test names it.`
  - **Detection Reason**: 検査は「その id を名指したテストが存在するか」だけを見る。
    台帳から外したうえで落ちることを先に観測しているので、通ったことは注記が実在することを意味する。
    注記の中身が空でないことは検査では読めないため、そこは各テストで `Then` の数だけ観測を置いた。
- **Unit RED Evidence**:
  - **Test**: `TestRefreshTokenRotatesAndReuseRevokesTheWholeFamily` (`backend/shared/http/server_http`)
  - **Requirement**: REQ-OAUTH2-006
  - **Observed Failure**: `TokenRevoked が発行されていない: [UserAuthenticated ConsentGranted
    AuthorizationCodeIssued AccessTokenIssued AuthorizationCodeRedeemed RefreshTokenIssued
    RefreshTokenRotated AccessTokenIssued RefreshTokenReuseDetected]`
  - **Detection Reason**: 応答と保存層だけを読むテストは、失効はするが通知を出さない実装を通す。
    実際にこの観測が `EX-OAUTH2-006-02` の欠落を出した。原因は `EX-OAUTH2-005-07` と同じ
    `RefreshTokenStore.RevokeFamily` の署名で、局所では直せないため
    [[wi-566-authorization-code-replay-and-expiry-leave-no-record]] へ移した。
- **Change-Resistance Results**:
  本項目が触れた製品コードは `backend/shared/http/testing_stack` だけである (ほかはテスト)。
  `mise run test-go-mutation -- ./backend/shared/http/testing_stack` を 3 回読んだ。
  1 回目は Killed 18 / Lived 0 / Not covered 37 で、**駆動部の全行が自 package のテストから
  1 度も実行されていなかった**。呼び出し側のテストが落ちれば気づけるが、そのとき壊れているのが
  製品か駆動部か分からない。`TestBrowserFlowDrivesAuthorizationThroughToAToken` を足して
  Killed 51 / Lived 3 / Not covered 1 になった。
  生き残った 3 件のうち `WithBrowserFlow` の `if b.stack.Refresh == nil` の否定は本物で、
  反転すると保管先を配線しないまま認可コードだけが出る。保管先を直接読む観測を足して殺した
  (Killed 52 / Lived 2)。
  残る 2 件は `PostToken` と `PushAuthorizationRequest` の `len(raw) > 0` を `>= 0` にする変異で、
  どちらも空の本文を `json.Unmarshal` へ渡すだけになり、その戻り値は捨てているので振る舞いが
  変わらない。等価変異として残す。
  手書きの故障注入は行っていない。本項目が足したのは観測であって配線ではなく、
  「配線を外す」種類の変異を当てる新しい結線が無いためである。
- **Verification Results**:
  - `mise run verify` - passed
