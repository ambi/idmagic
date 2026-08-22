---
depends_on: [wi-390-security-control-test-standard-and-gate]
status: completed
authors: [tn]
risk: medium
created_at: 2026-08-22
priority: p1
change_kind: tooling
initial_context:
  specification:
    - spec/contexts/signing-keys/scenarios.md
    - spec/contexts/authorization/scenarios.md
    - spec/contexts/identity-governance/scenarios.md
    - spec/contexts/sharedsignals/scenarios.md
    - spec/contexts/workloadidentity/scenarios.md
    - spec/contexts/authentication/scenarios.md
    - spec/contexts/oauth2/scenarios.md
  typespec:
    - IdMagic.SigningKeys.Operations.RotateTenantSigningKey
    - IdMagic.SigningKeys.Operations.DisableTenantKey
  source:
    - tools/check/src/security-controls.ts
    - tools/check/src/check-security-controls.ts
    - tools/check/security-refusal-debt.json
    - backend/shared/http/support_http/auth.go
  tests:
    - tools/check/src/security-controls.test.ts
    - backend/authorization/handlers_http/routes_test.go
    - backend/shared/http/support_http/admin_scope_test.go
  stop_before_reading:
    - frontend
    - spec/generated
spec_impact: { kind: none, reason: "検査の追加と、既存シナリオへの拒否の宣言の補完である。補完によって規範は増えるが、それは現在の実装が既に行っている拒否を書き起こすものであり、製品の振る舞いを変えない。振る舞いを変える拒否が見つかった場合は個別の work item を切る。" }
---

# 状態を変える操作が返す拒否をシナリオに必ず宣言させ、検出漏れしていた拒否を洗い直す

## Motivation

[[wi-390-security-control-test-standard-and-gate]] の R3 が問うのは「**宣言された**拒否にテストがあるか」だけである。**何を宣言すべきかは問うていない。** 拒否を 1 つも書かないコンテキストは検査を素通りする。

素通りが実際に何を見逃すかは、契約と規範を突き合わせると分かる。TypeSpec は操作ごとに「この操作は 403 でこのエラーを返す」と宣言している。403 は「呼び出す資格が無いので断った」という意味であり、状態を変える操作に付いている 403 は、**誰がその操作を実行できないか**を契約が明言しているということである。ところが、その拒否がシナリオに 1 行も書かれていないコンテキストがある (2026-08-23 実測)。

| コンテキスト | 宣言の無い 403 のエラー型 | 対象の変更操作 |
|---|---|---|
| authorization | AccessDeniedError | PutAuthorizationModel ほか 4 件 |
| identity-governance | AccessDeniedError | CreateLifecycleWorkflow ほか 7 件 |
| sharedsignals | AccessDeniedError | RegisterSsfTransmitterStream ほか 6 件 |
| workloadidentity | AccessDeniedError | RegisterWorkloadTrustBundle ほか 10 件 |
| authentication | MfaEnrollmentNotAllowedError | StartBrowserMfaEnrollment ほか 2 件 |
| oauth2 | OAuthAccessDeniedError | Token |

実装はいずれも拒否している。しかし**書かれていない拒否は、実装が拒否をやめても何とも矛盾しない**。認可モデルを誰でも差し替えられるようになっても、ライフサイクルワークフローを一般利用者が作れるようになっても、仕様の側からは退行と言えない。

もう 1 つ、洗い直しが要る。wi-390 の検出器は当初シナリオの `ALT` 行だけを見ており、`REQ-SIGNINGKEYS-009` (拒否がシナリオ自身の帰結として `THEN` に書かれている) と `REQ-IDGOVERNANCE-012` (テナント境界が結果として言い換えられている) を丸ごと取りこぼしていた。wi-390 の作業中に直した結果、宣言は 78 件から 175 件、テストに引用されていないものは 58 件から 137 件になった。**増えた 79 件は新しい負債ではなく、それまで見えていなかった負債である。** 見えるようになっただけで、1 件ずつの中身はまだ誰も確かめていない。

## Scope

- **宣言の最低要件 (R4)**。状態を変える操作 (`@post` / `@put` / `@patch` / `@delete`) が 403 で返すと契約が言っているエラー型ごとに、そのコンテキストのシナリオがその拒否を宣言していることを要求する。
- **欠けている拒否の補完**。R4 が落とすコンテキストについて、実装が実際に行っている拒否をシナリオへ書き起こし、テストから参照する。
- **署名鍵の拒否の補完**。`signing-keys` は R4 を通るが、通す根拠は参照系の操作に対する拒否 (`REQ-SIGNINGKEYS-009`) だけで、鍵の回転と無効化を誰が要求できないかはどこにも書かれていない。Motivation の出発点そのものなので、R4 の判定とは別に補う。
- **137 件の洗い直し**。許可リストの各項目を「テストは実在するが注記が無い」と「テスト自体が無い」に分ける手順を決め、分類を助ける報告タスクを用意し、着手した分の分類結果を記録する。

## Out of Scope

- 拒否テストが副作用の不在を assert していない 84 件の修正。[[wi-392-refusal-tests-assert-the-absent-effect]] が持つ。ただし本 work item が新しく参照するテストは、その規範を満たす形で書く。
- R1 / R2 の拡張。ユースケース層への拡大は [[wi-393-guard-rules-reach-the-usecase-layer]] が持つ。
- フロントエンド。`spec/authorization.md` により強制はサーバー側にあり、UI が唯一の強制点になっている箇所は無い。
- 新しい防護の追加。既に実装されている拒否を書き起こすことに限る。実装が拒否していない箇所や、契約と実装が食い違う箇所が見つかった場合は、本 work item では直さず欠陥として切り出す。

## Design

### 実測 (2026-08-23)

着手時点の数値を測り直した。着想の時点で書いていた「`signing-keys` の宣言は 0 件」という表は、wi-390 が検出器を直す前の数値であり、もう成り立たない。

| 測ったもの | 値 |
|---|---|
| 状態を変える操作 | 199 |
| うち 403 を宣言する操作 | 177 |
| そのうちシナリオが操作名に言及しているもの | 37 |
| 変更操作を持ちながら拒否を 1 つも宣言しないコンテキスト | 0 |
| 403 のエラー型のうちシナリオが宣言していないもの | 6 コンテキスト 6 型 |

### 1. 最低要件の粒度 — エラー型ごと (案 C) を採る

| 案 | 要求 | 今日の落ち方 | 判断 |
|---|---|---|---|
| A | 変更操作を持つコンテキストは拒否を 1 つ以上宣言する | 0 件 | 採らない |
| B | 変更操作ごとに、それを拒否するシナリオがある | 140 件 | 見送る |
| C | 変更操作が 403 で返すエラー型ごとに、それを宣言するシナリオがある | 6 件 | **採る** |

案 A は今日すでにどこも落ちない。入力検証の拒否が 1 つあれば満たせるので、「誰が呼べないか」を 1 行も書かないコンテキストが素通りする。門として名目だけになる。

案 B が本来欲しい保証だが、操作とシナリオを機械的に結ぶ手段が無い。TypeSpec の操作名をシナリオに書かせれば結べるものの、177 件のうち操作名に言及しているシナリオを持つのは 37 件しかなく、残り 140 件は許可リストに固定してから 1 件ずつ剥がすことになる。1 件あたりシナリオとテストの両方が要るので、この work item では引き受けられない。シナリオの書式に操作名の参照を持ち込むかどうかという、別に決めるべき論点も抱えている。

案 C は、契約が既に持っている情報だけで結べる。TypeSpec は操作ごとに 403 の本文の型を宣言しており、シナリオ側のエラー型名は wi-390 が拒否の signal として既に読んでいる。両者を突き合わせるだけなので、シナリオの書式を変えずに済む。「この操作は 403 を返しうると契約が言っているのに、いつ返るのかを規範が言っていない」という状態を落とす。

**案 C の限界を記録しておく。** 判定はコンテキスト単位なので、参照系の操作に対する拒否が 1 つあれば、同じエラー型を返す変更操作は無宣言でも通る。`signing-keys` がまさにその形で、R4 は通るのに鍵の回転を誰が要求できないかは書かれていない。この穴を塞ぐには操作単位の対応 (案 B) が要る。案 C は暫定であり、案 B へ進む余地を残す。

**許可リストは作らない。** 落ちるのは 6 件だけなので、この work item で全件を補える。ratchet を持たない検査は、載せた例外を剥がす作業も、例外が居座る危険も持たない。

### 2. 補完する拒否の出どころ — 実装と契約の双方から書き、食い違いは欠陥として切り出す

シナリオに書く拒否は、(1) TypeSpec が宣言する 403 のエラー型と、(2) 実装が実際に返すものの双方を読んで書く。実装だけを読むと現状追認になり、契約だけを読むと実装と食い違ったまま規範が増える。

補完した 6 件のうち 5 件は両者が一致していた。1 件は食い違いを露呈させた。

- `oauth2` の `Token` は契約が 403 `OAuthAccessDeniedError` を宣言しているが、実装の `writeOAuthError` は `invalid_client` を 401、`server_error` を 500 に写すだけで、`access_denied` は 400 で返す (RFC 6749 §5.2 に従う形)。`Token` が 403 を返す経路は存在しない。**シナリオには実装と一致する「`OAuthAccessDeniedError` で拒否される」を書き、状態コードの食い違いは契約の欠陥として [[wi-397-token-endpoint-declares-a-403-it-never-returns]] へ切り出した。**
- `REQ-OAUTH2-042` は同じ拒否を `AccessDeniedError` と書いていた。これは管理 API が使う Problem Details の型で、OAuth のエラー本文を返す `Token` の拒否ではない。R4 が突き合わせたことで見つかったので、契約に合わせて直した。

### 3. 137 件の分類の進め方 — 機械的に三分し、確認は 1 件ずつ

全件を人手で読むと着手できず、全件を機械に判定させると誤分類を固定する。**確実に言えることだけを機械に言わせ、残りを人が確認する**形にする。

`mise run report-refusal-debt` を足した。許可リストの各 id について、シナリオの見出しと拒否の段、その拒否を所有する Go パッケージ、そして拒否を assert しているテスト関数の候補を出し、3 つに分ける。

| 分類 | 意味 | 次の一手 |
|---|---|---|
| named | 同じエラーを名指しして拒否を assert するテストがある | 注記だけが欠けている見込み。1 件読んで注記を付ける |
| nearby | 拒否を assert するテストはあるが、当のエラーは名指ししていない | 1 件読んで判断する |
| none | そのパッケージに拒否を assert するテストが 1 つも無い | テスト自体が無い見込み |

着手時点の内訳は named 65・nearby 68・none 3 だった (`system` は Go パッケージを持たないため none)。

テストは「同名パッケージの配下にあるもの」だけでなく「そのパッケージを import しているもの」も候補に含める。署名鍵の管理 API のテストは `backend/oauth2/handlers_http` にあり、パッケージの配下だけを見ると候補なしになってしまう。ただし import 経由の候補は当の拒否と無関係なことが多いので、パッケージ内を先に並べ `(via import)` と印を付ける。

この報告は分類そのものではなく、**1 件ずつ確認する順番を決めるための道具**である。報告タスクにしたのは、リポジトリに置いた台帳がテストを 1 つ足すたびに古くなるからで、wi-390 が `report-security-test-gaps` で採ったのと同じ判断である。

### 4. R4 が実際に見つけたもの — 検査が要求した assert が現存の欠陥を暴いた

R4 は宣言を要求するだけで、実装は見ない。しかし宣言にはテストが要る (R3) ため、[DEVELOPMENT.md](../DEVELOPMENT.md) の「拒否のテスト」に従って**拒否が触らなかったもの**を assert したところ、`authorization` の非管理者テストが落ちた。403 を返しながら認可モデルの版が作られていた。

原因は `WriteAdminAccessError` が `WriteProblem` の結果 (成功時は `nil`) を返していたことである。ハンドラーが直接返す限りは正しく止まるが、`requireAuthorizationAdmin` と `requireWorkflowAdmin` は**その戻り値を返すヘルパー**であり、`if err := d.requireXxx(c); err != nil` が素通りしていた。wi-390 が塞いだ CSRF の欠陥と同じ形が、writer を 1 段挟んだところに残っていた。

`WriteAdminAccessError` に `ErrAdminAccessRefused` (`ErrResponseWritten` を包む) を返させて塞いだ。R1 がこの形を捕まえられないこと自体は別の欠落なので、[[wi-398-guard-rules-follow-one-level-of-indirection]] へ切り出した。

## Plan

- R4 を先に入れ、落ちるコンテキストを検査に言わせてから補完に入る。
- 補完は 1 件ずつ、シナリオ → テスト → 検査の順で進める。テストは [DEVELOPMENT.md](../DEVELOPMENT.md) の「拒否のテスト」に従い、返った状態と、拒否が触らなかったものの双方を assert する。
- 137 件の分類は、報告タスクで順番を決めてから確認する。分類の基準が定まらないうちに自動化すると、誤分類を固定する。

## Tasks

- [x] T001 [Design] 最低要件の粒度、補完する拒否の出どころ、137 件の分類の進め方を確定し `## Design` に記録した。
- [x] T002 [Tooling] R4 を `check-security-controls` に足した。許可リストは持たない。導入時点で 6 コンテキストが落ちた。
- [x] T003 [Guardrail] `security-controls.test.ts` に R4 の違反・充足と、契約側 (参照系・403 以外) の読み取りを固定するテストを置いた。
- [x] T004 [Spec] `signing-keys` に鍵の回転と無効化の拒否 (REQ-SIGNINGKEYS-011) を書き起こし、`TestAdminKeysRotateRejectsNonAdmin` (現在の鍵が変わらず SigningKeyRotated が出ないことを assert) と `TestAdminApiTokenScopeEnforcement` の `signing-keys:read` 行から参照した。
- [x] T005 [Spec] R4 が落とした 6 件を補完し、それぞれをテストから参照した。
  - REQ-AUTHORIZATION-010 / `TestAuthorizationAdminRoutes` "a non-administrator is rejected on every endpoint" (モデルの版とタプルの版を読み直す)。**この assert が現存の欠陥を暴いた** (Design 4)。
  - REQ-IDGOVERNANCE-014 / `TestAdminLifecycleWorkflowRejectsNonAdmin` (作成・有効化・一覧の 3 経路と、403 の本文に一覧が続かないこと)。
  - REQ-SHAREDSIGNALS-011 / `TestRegisterTransmitterStreamRejectsNonAdmin` (ストリームが増えていないこと)。
  - REQ-WORKLOADIDENTITY-010 / `TestRegisterTrustBundleRejectsNonAdmin` (信頼設定が増えていないこと)。
  - REQ-AUTHENTICATION-018 (MfaEnrollmentNotAllowedError を明示) / `TestBrowserAuthorizationFlowRejectsUnregisteredUserWithoutEnrollmentApproval` (登録 API が拒否し、シークレットを返さないこと)。許可リストから外れた。
  - REQ-OAUTH2-042 (AccessDeniedError → OAuthAccessDeniedError) / `TestApprovalExchangeRejectsDeniedRequest` (トークンを発行せず Denied のままであること)。
- [x] T006 [Tooling] `report-refusal-debt` を足し、131 件を named / nearby / none に三分した。
- [x] T007 [Survey] 三分の結果を `## Design` に記録し、`signing-keys` の 5 件を 1 件ずつ確認した。5 件とも「テストは実在し注記だけが無い」だったので注記を付け、許可リストを 137 → 131 に縮めた。
- [x] T008 [Verify] `sharedsignals` から拒否の宣言を外すと R4 が落ち、戻すと通ることを確認した。`mise run verify` を通した。

## Verification

- `mise run check`
- `mise run verify`
- 手動: 変更操作が 403 で返すエラー型の宣言をシナリオから外し、R4 が落ちることを確認する。
- 手動: `report-refusal-debt` の出力が許可リストの件数と一致することを確認する。

## Risk Notes

リスクは medium。**現状追認になる危険がもっとも大きい。** 実装を読んで拒否を書き起こすと、実装が拒否し忘れている箇所はそのまま「拒否しない仕様」として固定される。書き起こしは実装と TypeSpec の双方から行い、食い違いは欠陥として切り出す。実際に `Token` の 403 で 1 件出た。

コンテキスト単位の判定は、参照系の拒否 1 つで変更操作の宣言義務が満たせてしまう。`signing-keys` がその実例で、R4 の判定とは別に補った。この抜け道は案 B でしか塞げないので、暫定であることを Design に残す。

137 件の分類は退屈で、機械化したくなる。基準が定まらないうちに自動化すると誤分類を固定するので、機械には「確実に言えること」だけを言わせる。

## Completion

- **Completed At**: 2026-08-23
- **Summary**:
  「宣言された拒否にテストがあるか」しか問えなかった検査に、**何を宣言すべきかを問う規則 (R4)** を足した。契約が状態を変える操作へ 403 を宣言しているなら、その拒否がいつ起きるのかをシナリオが言わなければならない。突き合わせの鍵は両側にすでに書かれているエラー型なので、シナリオの書式は変えていない。

  意味の差は 4 つある。

  第一に、**誰がその操作を実行できないかが、6 コンテキストで初めて規範になった**。R4 は導入時点で `authorization`・`identity-governance`・`sharedsignals`・`workloadidentity` の `AccessDeniedError`、`authentication` の `MfaEnrollmentNotAllowedError`、`oauth2` の `OAuthAccessDeniedError` を落とした。いずれも実装は拒否していたが、規範のどこにも書かれていなかった。5 件を新しいシナリオとして書き起こし、1 件 (REQ-AUTHENTICATION-018) は既存のシナリオが「登録 API も利用できない」とだけ書いていたのでエラー型を明示した。落ちる件数が 6 件で収まったので、**許可リストは作っていない**。R4 は例外を持たない。

  第二に、**その要求が現存の欠陥を 1 件暴いた**。宣言にはテストが要り、拒否のテストは「触らなかったもの」を assert する。そう書いた `authorization` のテストが落ち、非管理者による `PutAuthorizationModel` が 403 を返しながらモデルの版を作っていることが分かった。原因は `WriteAdminAccessError` が応答を書いた結果 (`nil`) を返していたことで、その戻り値を返す `requireAuthorizationAdmin` / `requireWorkflowAdmin` の 2 つのヘルパーが拒否を呼び出し元へ伝えていなかった。`identity-governance` では 403 の本文に続けてワークフロー一覧がそのまま書き出されていた。`ErrAdminAccessRefused` (`ErrResponseWritten` を包む) を返させて塞いだ。**wi-390 が塞いだ形が、writer を 1 段挟んだところに残っていた。**

  第三に、**最低要件の粒度を測ったうえで選んだ**。コンテキスト単位で拒否を 1 つ要求する案は、今日どこも落ちないので採らなかった。操作単位で要求する案は本来欲しい保証だが、403 を宣言する変更操作 177 件のうちシナリオが操作名に言及しているのは 37 件で、残り 140 件を許可リストに固定してから 1 件ずつ剥がすことになる。エラー型単位を採り、参照系の拒否で変更操作の義務が満たせてしまう限界を Design に残した。`signing-keys` がその実例なので、R4 の判定とは別に鍵の回転と無効化の拒否 (REQ-SIGNINGKEYS-011) を補った。

  第四に、**137 件の負債が「読む順番」を持つようになった**。`mise run report-refusal-debt` が、拒否を assert するテストの候補から named / nearby / none に三分する。分類そのものは機械に決めさせない。着手時点は named 65・nearby 68・none 3 で、`signing-keys` の 5 件を確認したところ全て「テストは実在し注記だけが無い」だったので注記を付け、許可リストは 137 → 131 に縮んだ。

- **Verification Results**:
  - `mise run verify` - passed
  - `mise run check` - passed (`check-security-controls` は宣言 180 件、状態変更の 403 が約束する拒否 18 種、テスト待ち 131 件)
  - `mise run test-tools` - passed (170 件)
  - `mise run spec-diff` - 追加 5 件 (REQ-AUTHORIZATION-010 / REQ-IDGOVERNANCE-014 / REQ-SHAREDSIGNALS-011 / REQ-SIGNINGKEYS-011 / REQ-WORKLOADIDENTITY-010)、変更 2 件 (REQ-AUTHENTICATION-018 / REQ-OAUTH2-042)
  - 手動: `sharedsignals` から拒否の宣言を外すと R4 が落ち、戻すと通る。
  - 手動 (RED): `WriteAdminAccessError` を修正前の形へ戻すと、`TestAuthorizationAdminRoutes` が「拒否したはずのモデルの版が作られている」で落ち、`TestAdminLifecycleWorkflowRejectsNonAdmin` が「403 の本文にワークフロー一覧が続いている」で落ちる。**欠陥をそのまま再現して捕まえられることを確認した。**

- **Left Undone**:
  - **操作単位の最低要件 (案 B)**。403 を宣言する変更操作 177 件のうち、シナリオが操作名に言及しているのは 37 件にすぎない。エラー型単位の R4 は、参照系の拒否 1 つで同じ型を返す変更操作の義務を満たせてしまう。塞ぐにはシナリオと操作を結ぶ参照が要り、書式の変更を伴うので別の work item が要る。
  - **残る 131 件の分類**。`report-refusal-debt` が順番を付けただけで、確認したのは `signing-keys` の 5 件である。named 63・nearby 65・none 3 が残る。
  - **注記を付けたテストのアサーション**。今回注記を付けた 5 件は「拒否が触らなかったもの」を assert していないものを含む。[[wi-392-refusal-tests-assert-the-absent-effect]] が扱う。
  - **R1 が 1 段の間接を越えないこと**。今回の欠陥を R1 は捕まえられなかった。[[wi-398-guard-rules-follow-one-level-of-indirection]] へ切り出した。
  - **`Token` が返さない 403**。契約と実装の食い違いは [[wi-397-token-endpoint-declares-a-403-it-never-returns]] へ切り出した。シナリオ側は実装に合わせてある。
