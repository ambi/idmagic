---
depends_on: [wi-565-make-backing-declared-examples-cheap]
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-12
priority: p3
change_kind: maintenance
spec_impact: { kind: none, reason: "宣言済みの具体例に、その id を名指しするテストを対応付ける作業である。シナリオも製品の振る舞いも変えない。テストが書けない具体例が見つかった場合、それは実装が具体例のとおりに振る舞っていないということなので、欠陥として個別の work item に切り出す。" }
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 宣言済みの具体例に検証を与える作業であり、公開契約も運用手順も変わらない。
  references: []
initial_context:
  specification:
    - docs/contexts/authentication/scenarios.feature.md
  typespec: []
  source:
    - backend/shared/http/testing_stack/stack.go
  tests:
    - backend/authentication/handlers_http/refusal_effects_test.go
    - backend/authentication/handlers_http/admin_refusal_effects_test.go
    - backend/authentication/federation/usecases/broker_test.go
    - backend/authentication/session/usecases/sessions_test.go
    - backend/authentication/trusteddevice/usecases/trusted_devices_test.go
    - backend/authentication/securitynotification/usecases/dispatch_test.go
    - backend/shared/http/server_http/routes_e2e_test.go
    - backend/shared/http/server_http/trusted_device_e2e_test.go
  stop_before_reading:
    - backend/authentication/db_postgres
    - frontend
    - infra
---

# Authentication が宣言する具体例 71 件にテストを対応付け、被覆台帳から外す

## Motivation

[[wi-496-burn-down-the-example-coverage-debt]] は具体例の被覆台帳の消化単位を Context と決め、測定のうえで残りを Context ごとの子 work item へ割った。本項目はそのうち `docs/contexts/authentication/scenarios.feature.md` が宣言する 71 件を引き取る。

親項目が `claim-mapping` の 3 件と `workloadidentity` の 13 件で測った結果は、**16 件のうち注記だけで済んだのは 4 件だけ**だというものである。3 件はテストが 1 つも無く、9 件は既存テストへ新しい観測を足す必要があった。件数は作業量の目安にならない。

**71 件のうち拒否は 8 件しかなく、63 件は拒否以外である。** 拒否の具体例は観測の型が決まっているが、拒否以外は「何が起きたか」を Context ごとに決める必要がある。本 Context はその比率が最も偏っているので、拒否以外の観測の型をここで固める。

## Scope

- 71 件を 1 件ずつ確認し、次のいずれかに解決して `tools/check/example-coverage-debt.json` から外す。
  - 当の具体例を検証しているテストが実在する → そのテストに `//spec:covers EX-AUTHENTICATION-NNN-MM: <この具体例の何を固定しているか>` のディレクティブを足す。
  - 当の具体例を検証しているテストが無い → 書く。具体例が拒否なら、[[wi-392-refusal-tests-assert-the-absent-effect]] が定める形（拒否応答と、拒否が防いだ効果の双方を観測する）で書く。
  - 具体例が宣言されなくなっている → 台帳から外す（検査が落ちて教える）。
- ディレクティブは「何を固定しているか」を書く。id だけの注記は禁止する。
- 実装が具体例のとおりに振る舞っていないことが分かった場合は、**本 work item では直さず欠陥として切り出す**。

## Out of Scope

- 他の Context が宣言する具体例。Context ごとに別の work item が持つ。
- `tools/check/standards-coverage-debt.json`。[[wi-495-burn-down-the-standards-coverage-debt]] が消化済みである。
- 既に台帳に載っていない拒否テストが、防いだ効果を観測していない件。[[wi-392-refusal-tests-assert-the-absent-effect]] が持つ。本項目が新しく書く拒否テストは、その規範を満たす形で書く。既存テストに注記を足すだけの件で、そのテストが効果の不在を見ていない場合は、注記を足したうえで wi-392 の対象として残す。
- シナリオと具体例の追加、削除、書き換え。具体例の記述が実装と食い違うことが判明した場合は規範の変更なので、別の work item が扱う。
- 見つかった実装の欠陥の修正。切り出した先で扱う。
- 行カバレッジ率の目標または閾値。
- `report-coverage-debt` の分類（`named`、`nearby`、`none`）を根拠にした台帳からの削除。分類は読む順を決める材料であり、台帳から外す根拠にはならない。

## 作業の進め方

[[wi-565-make-backing-declared-examples-cheap]] が、この作業の 1 件あたりの費用を下げる道具を用意した。使う。

- 読む順と観測点は `mise run spec-route -- <id>` が出す。規則、当の具体例の本文、契約が宣言する候補 operation とそのメソッド・パス・スコープ、同じ規則の隣の id を名指している既存テストが返る。**出力は読む順を決める材料であり、台帳から外す根拠にはしない。**
- 新しく書くテストは `backend/shared/http/testing_stack` の上に載せる。`Register` と同じ配線が option の合成で建ち、保存先は型付きの field から読み直せる。既存 fixture の全面移行はしない。触る必要が出た範囲だけ移す。
- 変更した Go の変異は `mise run test-go-mutation -- <package-directory>` で読む。手で書く故障注入は、変異器が表現できない「配線を外す」「既定の分岐を差し替える」種類だけに残す。
- 実装が具体例と食い違って消化できない件は、台帳の当該行へ `blocked_by`（先に決着すべき work item）と `finding`（実装が実際に何を返すか）を書いて残す。散文にだけ書くと、次の読み手が同じ測定をやり直す。

## Design

### 拒否以外の観測の型

本 Context は 71 件のうち 63 件が拒否以外である。親項目が「拒否以外は Context ごとに観測を決める必要がある」と書いた部分で、実際に決まったのは次の 4 つだった。どれも「応答は正しいが効果が無い実装」を落とすためにある。

| 具体例の `Then` の形 | 観測 | これが無いと通る実装 |
| --- | --- | --- |
| 「〜が発行される」 | `Deps.Emit` の記録を読む | 状態は変えるが記録を残さない |
| 「〜が作成される / 変更される」 | 保存層から読み直す | 戻り値だけを組み立てる |
| 「〜画面へ進む」 | 次画面の名前に加え、その状態でしか通らない API を叩く | 画面だけ進めて保留を解かない |
| 「〜が省略される / 省略できない」 | 省略の可否と、そのセッションの `amr` と `acr` | 画面は飛ばすが認可の判定には効かない |

4 つ目が本 Context に固有である。信頼済みデバイスの具体例 (026、027) は「第二要素の画面を挟まない」と書いているが、認可の判定が読むのは `amr` と `acr` だけなので、画面の名前しか見ないテストは「飛ばしはするが MFA としては通用しない」実装を通す。`TestTrustedDeviceSkipsTheSecondFactorOnTheNextLogin` にこの観測を足した。

### イベントの配線が 4 つの fixture に無かった

wi-559 が `testing_stack` へ `EventLog` を常設した理由と同じことが、本 Context の既存 fixture でも起きていた。`routes_e2e_test.go`、`admin_authenticator_reset_e2e_test.go`、`password_expiry_e2e_test.go`、`trusted_device_e2e_test.go` の 4 つが `Deps.Emit` を配線しておらず、通常経路の `Then` に並ぶ発行を 1 つも読めなかった。

**wi-559 の教訓どおり、具体例を消化する前に基盤を見た。** 各 fixture へ `newXxxWithEvents` を足し、既存の構築関数はそれへ委譲させた。呼び出し側 21 個のうち書き換えたのは 5 個だけで済んでいる。

### 1 件あたりの費用がどこにあったか

wi-559 が測った「経路探索は速くなり、`Then` との突き合わせは変わらない」は本 Context でも同じだった。変わったのは配分である。

| 段階 | 本項目での実測 |
| --- | --- |
| 観測点を決める | 規則 1 つあたり 1〜2 回。`rg -n "^func Test"` でパッケージのテスト名を一覧するのが `spec-route` より速かった |
| 候補テストを読み、`Then` と突き合わせる | 1 件あたり 1 ファイル読み。ここが費用の大半 |
| 既存テストへ観測を足す | 71 件中 40 件 |
| テストを新しく書く | 71 件中 29 件 |
| 欠陥として切り出す | 71 件中 2 件 |

**注記だけで済んだ件は 0 件である。** 親項目の測定 (16 件中 4 件) より悪い。71 件のうち 40 件は既存テストが具体例の `Then` を部分的にしか見ておらず、残り 29 件はテストが無かった。

`spec-route` の候補 operation は本 Context でも役に立たなかった。理由は wi-559 と同じで、規則の本文がエンドポイントをバッククォート付きで書いていないためほぼ全 operation が同点で並ぶ。役に立ったのは「隣の id を名指す既存テスト」の一覧である。

### 道具の側へ足したもの

`testing_stack` へ 4 つ足した。どれも「その具体例を観測するには基盤が足りない」と分かった時点で足している。

- `WithLoginThrottle`: アカウント単位と IP 単位の失敗回数 (007-03)。閾値は呼び出し側が決める。具体例の 10 回をそのまま使うと 1 件の観測に 10 往復かかる。
- `WithEndpointRateLimitReached`: ポリシーと鍵の組を上限到達にする (007-04)。**鍵まで見るのは変異が教えた。** ポリシーだけで拒否すると、送信元 IP が要求から読めていない実装でも同じ拒否が起きる。
- `SignInAttempt` / `SignInWithBadCSRF` / `SessionCookie`: 成立しないログインと、発行された Cookie を読む入口 (007-01、007-02)。
- `WithWsFederation` と `Stack.SessionStore`: サインアウトの 2 つのプロトコル入口と、サーバー側の失効を読む先 (035)。

### 消化の状況

引き取った 71 件のうち **69 件を消化し、2 件を欠陥として切り出した。**

| 具体例 | 状態 |
| --- | --- |
| 69 件 | 消化。台帳から外し、`//spec:covers` を持つテストへ対応付けた |
| EX-AUTHENTICATION-001-03 | 台帳に残す。`state` 不一致では `FederatedLoginRejected` が出ない。[[wi-571-a-state-mismatch-leaves-no-federation-record]] |
| EX-AUTHENTICATION-019-01 | 台帳に残す。強制開始日時が利用者向けの API にも UI にも現れない。[[wi-572-the-mfa-enforcement-date-never-reaches-the-user]] |

どちらも、具体例の `Then` のうち 1 つだけが満たされていない形である。**部分的に消化して台帳から外す誘惑があった。** 実際に 001-03 は一度そうしかけている。`Then` の数だけ観測が要るという本項目自身の Risk Note がそれを止めた。残った側は台帳へ `blocked_by` と `finding` を書いて残し、テストは書いたうえでディレクティブには規則 id だけを置いた。

### 規則 id はディレクティブに残す

`//spec:covers REQ-AUTHENTICATION-030` を `EX-AUTHENTICATION-030-01` へ置き換えたところ、`mise run check-work-items` が落ちた。完了済みの [[wi-456-retroactive-primary-use-case-evidence]] が、そのテストファイルが当の要件 id を名指していることを主要ユースケースの証拠として参照していたためである。

**具体例 id は規則 id を置き換えない。** 列挙の先頭に規則 id を置く形が規約で認められているので、本項目が書き換えた 74 本のディレクティブはすべて `REQ-<CONTEXT>-NNN, EX-<CONTEXT>-NNN-MM: <何を固定しているか>` の形にした。

## Tasks

- [x] T001 [Acceptance] `mise run check-spec` が対象 71 件を名指しで落とすことを、消化前に観測する。
- [x] T002 [Tooling] 既存 fixture 4 つへイベントの記録を足す。通常経路の `Then` はほぼ発行で書かれているのに、どれも `Deps.Emit` を配線していなかった。
- [x] T003 REQ-AUTHENTICATION-001 から 006 を消化する。`mise run test-go-package -- ./backend/authentication/federation/...`、`./backend/authentication/handlers_http`。
- [x] T004 [Tooling] `testing_stack` へ失敗回数・流量制限・成立しないログインの駆動部を足し、REQ-AUTHENTICATION-007 と 008 を消化する。
- [x] T005 REQ-AUTHENTICATION-010 から 018 を消化する。
- [x] T006 REQ-AUTHENTICATION-021 から 025 を消化する。
- [x] T007 REQ-AUTHENTICATION-026 から 029 を消化する。資格情報が変わる 6 つの入口のうち、テストが無かった 4 つを書く。
- [x] T008 [Tooling] `testing_stack` へ WS-Federation とセッションの保管先を足し、REQ-AUTHENTICATION-030 から 035 を消化する。
- [x] T009 [Decision] `EX-AUTHENTICATION-001-03` と `EX-AUTHENTICATION-019-01` を欠陥として切り出し、台帳へ `blocked_by` と `finding` を書く。
- [x] T010 [Change-Resistance] `mise run test-go-mutation -- ./backend/shared/http/testing_stack` を読み、生き残りを直す。
- [x] T011 [Verify] `mise run verify`。

## Verification

- `mise run check-spec` が、`docs/contexts/authentication/scenarios.feature.md` の 71 件を `tools/check/example-coverage-debt.json` から外した状態で通る。台帳から外す前に同じ検査が当の id を名指しで落とすことを、消化ごとに観測する。
- 消化したテストの所属パッケージに対する `mise run test-go-package -- <package>`。
- `mise run verify`

## Risk Notes

- **注記だけを足して終わる。** `checkNormativeCoverage` は文字列の一致しか見ないので、読まずに id を貼れば件数は速く減る。減った件数は何も意味しない。注記に「この具体例の何を固定しているか」を書かせることで区別する。親項目の測定では、16 件のうち注記だけで済んだのは 4 件だけだった。
- **`nearby` に分類された件を、テストがある証拠として読む。** 分類が言っているのは「近傍に他の拒否テストがある」だけである。親項目の測定では、`claim-mapping` の 3 件はすべて `nearby` でありながら 2 件はテストが無かった。分類は読む順の材料にとどめる。
- **拒否である具体例のテストが、応答の字面だけを見て書かれる。** 新しく書く拒否テストには [[wi-392-refusal-tests-assert-the-absent-effect]] の規範が効く。注記へ「効果の不在を何で観測したか」を書き、後から区別できるようにする。
- **具体例の `Then` が複数あるのに、観測が 1 つで済まされる。** 親項目の測定では、`EX-WORKLOADIDENTITY-008-01` のようにイベントと実際の効果の双方を言う具体例が複数あった。`Then` の数だけ観測が要る。

## Completion

- **Completed At**: 2026-09-13
- **Summary**:
  `mise run spec-diff -- 9b574186` は「no normative specification change」を返す。規範は動いていない。
  動いたのは対応付けで、Authentication が宣言する具体例のうち本項目が引き取った 71 件から 69 件を
  `tools/check/example-coverage-debt.json` から外し、`//spec:covers` を持つテストへ結び付けた。
  台帳の Authentication 行は 71 件から 2 件へ減り、残る 2 件はどちらも `blocked_by` と `finding` を持つ。
  併せて、既存の e2e fixture 4 つがイベントを記録するようになり、`testing_stack` が
  失敗回数・エンドポイント流量制限・成立しないログイン・WS-Federation・セッションの保管先を配るようになった。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`
  - **Requirement**: REQ-AUTHENTICATION-007
  - **Observed Failure**: 対象 71 件を台帳から外した状態で、71 件すべてを名指しで落とした。例:
    `docs/contexts/authentication/scenarios.feature.md:173: EX-AUTHENTICATION-007-01 is declared, but no test names it.`
  - **Detection Reason**: 検査は「その id を名指したテストが存在するか」だけを見る。
    台帳から外したうえで落ちることを先に観測しているので、通ったことは注記が実在することを意味する。
    注記の中身が空でないことは検査では読めないため、そこは各テストで `Then` の数だけ観測を置いた。
- **Unit RED Evidence**:
  - **Test**: `TestCredentialChangingEntryPointsRevokeEveryTrustedDevice/メールのリセットリンクでパスワードを再設定する`
    (`backend/authentication/handlers_http`)
  - **Requirement**: REQ-AUTHENTICATION-028
  - **Observed Failure**: `password_reset_handler.go` の `RevokeTrustedDevices` 呼び出しを
    `if false` で括って配線を外すと `refusal_effects_test.go:1440: 記憶済みの端末が 1 台残っている` で落ちる。
    元に戻すと通る。
  - **Detection Reason**: この経路の応答は成功時 200 で、端末を残したかどうかを運ばない。
    応答だけを読むテストは配線の外れた実装をそのまま通す。保存層を読み直す観測がその差を出す。
    資格情報が変わる 6 つの入口のうち 4 つには、この観測を持つテストが 1 つも無かった。
- **Change-Resistance Results**:
  本項目が触れた製品コードは `backend/shared/http/testing_stack` だけである (ほかはテスト)。
  `mise run test-go-mutation -- ./backend/shared/http/testing_stack` を 3 回読んだ。
  1 回目は Killed 53 / Lived 4 / Not covered 4。生き残りのうち 2 件が本物だった。
  1 つは `blockingRateLimiter` がポリシーだけを見ていたことで、**送信元 IP が要求から読めていない
  実装でも同じ拒否が起きる**。鍵まで照合する形へ直し、`X-Forwarded-For` の連なりを
  組み立てる駆動部を `SignInAttempt` へ入れた (信頼するホップが 1 つなので連なりは 2 つ要る)。
  もう 1 つは `WithWsFederation` の署名者の既定値で、`WithSaml` と合成したときしか踏まれていなかった。
  2 回目は Killed 57 / Lived 6 / Not covered 3 で、新しい駆動部 (`SignInAttempt`、
  `SignInWithBadCSRF`、`SessionCookie`、`WithWsFederation`) が自 package のテストから
  1 度も実行されていないことが残った。呼び出し側が落ちれば気づけるが、そのとき壊れているのが
  製品か駆動部か分からない。`TestBrowserDrivesBothSidesOfSignIn` と
  `TestWsFederationComposesWithSaml` を足して 3 回目が Killed 62 / Lived 3 / Not covered 1 になった。
  残る 3 件は `len(raw) > 0` を `>= 0` にする変異で、どれも空の本文を `json.Unmarshal` へ渡すだけになり、
  その戻り値は捨てているので振る舞いが変わらない。等価変異として残す ([[wi-559-back-oauth2-remaining-examples-with-tests]]
  が同じ 3 件を同じ理由で残している)。残る 1 件は `AuthorizationQuery` の上書き削除で、本項目が
  触れていない既存の経路である。
  手書きの故障注入は 2 つ行った。1 つは上記の Unit RED で、変異器が表現できない「配線を外す」種類である。
  もう 1 つは `SignInAttempt` が `X-Forwarded-For` を組み立てない形への差し替えで、
  `TestBrowserLoginRefusesTheCorrectPasswordOnceTheEndpointLimitIsReached` が
  `status=200 ... 期待は 429` で落ちることを観測した。
- **Verification Results**:
  - `mise run verify` - passed
  - `mise run test-ui-e2e` - 実行していない。本項目が変更した frontend のファイルは
    `AccountActivityPage.test.tsx` の 1 本だけで、製品の UI コードは 1 行も動いていない。
    ブラウザーへ届く変更が無いため、後退の起きようがない。
