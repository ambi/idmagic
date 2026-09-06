---
depends_on: []
status: completed
authors: [tn]
risk: high
reversibility: irreversible
created_at: 2026-09-06
priority: p2
change_kind: bugfix
evidence_policy: risk-based-v3
documentation_impact:
  level: release_note
  reason: 復旧コードで成立した第二要素が MFA の要求を満たすようになる。これまで満たさなかったので、MFA 必須のアプリケーションでは要素を失った利用者が復旧コードを使っても先へ進めなかった。振る舞いの変更としてリリースの読み手に見える。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-508.md }
affected_spec:
  - { path: docs/contexts/authentication/standards.md, requirement: RFC8176-AMR-VOCABULARY }
  - { path: docs/contexts/authentication/scenarios.feature.md, requirement: REQ-AUTHENTICATION-036 }
  - { path: spec/contexts/authentication/models.tsp, symbol: IdMagic.Contract.LoginSession }
initial_context:
  specification:
    - docs/contexts/authentication/standards.md#RFC8176-AMR-VOCABULARY
    - docs/contexts/authentication/scenarios.feature.md#REQ-AUTHENTICATION-001
    - docs/contexts/authentication/scenarios.feature.md#REQ-AUTHENTICATION-036
    - docs/contexts/authentication/internals.md
  typespec:
    - IdMagic.Contract.LoginSession
  source:
    - backend/authentication/domain/authentication_context.go
    - backend/authentication/usecases/acr_vocabulary.go
    - backend/authentication/session/domain/login_session.go
    - backend/authentication/session/usecases/session_manager.go
    - backend/authentication/federation/usecases/broker.go
    - backend/application/usecases/sign_in_policy.go
    - backend/oauth2/handlers_http/authorize_second_factor.go
    - backend/oauth2/handlers_http/authorize_handler.go
    - backend/shared/security/tokens_jose/jwt_signer.go
    - backend/authentication/trusteddevice/usecases/trusted_devices.go
  tests:
    - backend/authentication/session/domain/login_session_test.go
    - backend/authentication/session/usecases/sessions_test.go
    - backend/authentication/federation/handlers_http/federated_login_e2e_test.go
    - backend/oauth2/handlers_http/authorize_handler_test.go
    - backend/oauth2/handlers_http/authorize_enrollment_test.go
  stop_before_reading:
    - frontend
    - backend/saml
    - backend/wsfederation
primary_use_cases:
  - id: recovery-code-satisfies-mfa
    requirement: REQ-AUTHENTICATION-036
    observable_result: 復旧コードで第二要素を通した利用者の次の認可要求が、第二要素の画面へ戻されずに認可コードの発行まで進む。
    unit_test: { path: backend/authentication/usecases/acr_vocabulary_test.go, name: TestDeriveACRTreatsRecoveryCodeAsASecondFactor, task: test-go-race }
    e2e_test: { path: backend/oauth2/handlers_http/recovery_code_mfa_e2e_test.go, name: TestRecoveryCodeSecondFactorSatisfiesMfaPolicy_REQ_AUTHENTICATION_036, task: test-go-race }
    unit_fault_model: DeriveACR が rc を第二要素として扱わず、acr が urn:idmagic:acr:pwd のままになる。
    e2e_fault_model: 復旧コードのハンドラーが CompleteFactor へ rc を渡さず、セッションの amr が変わらない。
  - id: federated-login-amr-accepted
    requirement: RFC8176-AMR-VOCABULARY
    observable_result: 連合ログインの callback が federated を amr に持つ LoginSession を保存し、認証解決がそれを返す。語彙の外の値なら保存されない。
    unit_test: { path: backend/authentication/session/domain/login_session_test.go, name: TestLoginSessionClosesTheAMRVocabulary, task: test-go-race }
    e2e_test: { path: backend/authentication/federation/handlers_http/federated_login_e2e_test.go, name: TestFederatedLoginPrimaryUseCase_REQ_AUTHENTICATION_001, task: test-go-race }
    unit_fault_model: 語彙の検証が値を素通しし、語彙の外の値を持つ LoginSession が有効と判定される。
    e2e_fault_model: 語彙から federated が落ち、連合ログインのセッション保存が拒否されて callback が失敗する。
---

# `amr` の語彙について、standards.md と scenarios.feature.md が食い違っている

## Motivation

[[wi-501-back-authentication-standards-rows-with-tests]] が `RFC8176-AMR-VOCABULARY` にテストを対応付けようとして見つけた。**この行はいま製品と一致していないので、行を満たすテストが書けない。**

`docs/contexts/authentication/standards.md` の `RFC8176-AMR-VOCABULARY` は、`LoginSession.amr` に許される語彙を `pwd` / `otp` / `webauthn` / `hwk` / `swk` / `rc` / `tdev` の 7 語と宣言している。一方 `docs/contexts/authentication/scenarios.feature.md:17` は「AMR に `federated` を持つ LoginSession を発行する」を規範として書いており、`docs/contexts/authentication/internals.md:13` も同じことを書いている。実装 (`backend/authentication/federation/usecases/broker.go:106`) は後者に従っている。`federated` は RFC 8176 の登録値でもなければ、標準の行が挙げる非 IANA 拡張値でもない。

同じ context の 2 つの正典文書が、同じフィールドについて両立しないことを言っている。どちらが正しいかを決めるのは規範の変更であり、テストの追加ではない。

第 2 の食い違いが同じ場所にある。`spec/contexts/authentication/models.tsp` の `LoginSession.acr` の doc と `internals.md:77` は、復旧コード (`rc`) による第二要素の成立で `acr` が `urn:idmagic:acr:mfa` へ上がると書いている。実装の `mfaAMRValues` (`backend/authentication/usecases/acr_vocabulary.go:17`) は `otp` / `webauthn` / `hwk` / `swk` / `tdev` の 5 語で、`rc` を含まない。

**これは文書上の不一致では済まない。** `amr = ["pwd","rc"]` だと `DeriveACR` は `urn:idmagic:acr:pwd` を返し、`ACRSatisfies("...:pwd", "...:mfa")` は false になるので、`sign_in_policy.go` の `mfaSatisfied` が false を返す。`authorize_handler.go:130` が次の `/authorize` でこれを評価するため、**復旧コードで第二要素を通したセッションは、MFA 必須のアプリケーションに対して何度でも第二要素を求められる。** 要素を失った利用者のための唯一の経路が、まさにその場面で通らない。加えて、その回に発行される ID トークンの `acr` は「パスワードのみ」と申告するので、リライングパーティーに対しても事実と違うことを言う。

第 3 に、語彙そのものを強制する場所がどこにも無い。`LoginSession` の zog スキーマは `AMR` について `Min(1)` しか課しておらず、`LoginSession.Validate()` は production のどこからも呼ばれていない (呼んでいるのは自身のテストだけである)。行が「語彙のみを許可する」と書いている以上、許可されない値が拒否されることの観測が要る。

## Scope

- 標準の行の語彙へ `federated` を加え、語彙の外の値を持つ LoginSession を保存しないことを行に書く。
- `rc` を MFA 充足の `amr` に加え、TypeSpec の `acr` の doc を実装と一致させる。復旧コードによる第二要素の成立を `REQ-AUTHENTICATION-036` として規範に起こす。
- 語彙を Authentication context の domain に 1 か所だけ持ち、`LoginSession` の検証がそれを使う。セッションマネージャーは `amr` を書く 2 か所 (作成と第二要素の成立) で保存前に検証を通す。
- `RFC8176-AMR-VOCABULARY` を `tools/check/standards-coverage-debt.json` から外す。

## Out of Scope

- `docs/contexts/authentication/standards.md` の他の行。[[wi-501-back-authentication-standards-rows-with-tests]] が済ませた。
- 認証要素の追加や削除。
- `acr` の URN そのものの変更。
- `federated` を MFA 充足として扱うこと。上流の IdP が何を検証したかはブローカーに分からないので、別の判断であり別の work item が要る。
- 上流の IdP の `amr` を透過して局所の `amr` へ写すこと。同上。
- 保存済みの行に対する移行。製品は未リリースであり、語彙の強制は書き込みの側にしか置かない。

## Design

### `federated` は語彙へ入れる。実装ではなく行を直す

3 つの理由による。

**第一に、`amr` は取り消しにくい。** `backend/shared/security/tokens_jose/jwt_signer.go:102,141` が ID トークンとアクセストークンの両方に `amr` クレームとして載せるので、`federated` はリライングパーティーまで届く wire 上の値である。値を変えると、外部が既に読んでいるものを壊す。`reversibility: irreversible` はこの点を指している。

**第二に、行の構えと一貫する。** 行はすでに `rc` と `tdev` という非 IANA 拡張値を 2 つ認めている。RFC 8176 は未登録値の使用そのものを禁じていない。`federated` を 3 つ目として認めるのは、行がもともと採っている構えの延長である。

**第三に、実装側を寄せる案には行き先が無い。** RFC 8176 に連合を表す登録値は無く、上流の IdP が何で認証したかはブローカーには分からない。`pwd` へ潰すのは、検証していないことを検証したと申告することになる。

### `rc` は MFA 充足に入れる。実装を直す

いまの状態は「復旧コードは MFA として弱い」という設計判断が表現されているのではなく、`acr` が事実と違うだけである。第二要素を実際に提示したのに、そう申告されていない。まず事実に合わせる。

「復旧コードでは MFA 要求を満たさせたくない」という判断が別途あるなら、その表現の前例は同じ関数の中にある。`tdev` は `mfaAMRValues` に含まれて `acr` を上げるが、`SignInRule.allow_trusted_device=false` のルールは充足として認めない (`mfaSatisfied` の後半がそれである)。`rc` も同じ形で後から足せる。**いま必要なのは表現の追加ではなく、`acr` を事実に合わせることである。**

`internals.md:77` が「復旧コードを `mfa_enrolled` に数えない」と書いているのは登録の話であって、提示の話ではない。復旧コードだけを唯一の MFA 手段として登録させない、という規則はこの変更で緩まない。

### 語彙は Authentication context の domain に置く

`amr` の語彙を必要とするのは `backend/authentication/session/domain` (`LoginSession` の検証) と `backend/authentication/usecases` (`DeriveACR`) の 2 つで、後者が前者へ依存すると use case → domain の向きが逆になる。両方が既に依存している `backend/authentication/domain` (context ルートの共有 domain) に置く。

型と操作は次のとおり。効果は持たない純粋な計算である。

```go
// backend/authentication/domain
type AMRValue = string
func AMRVocabulary() []AMRValue      // 宣言された語彙。順序は宣言順
func UnknownAMRValues(amr []AMRValue) []AMRValue  // 語彙の外の値だけを返す
func MfaAMRValues() []AMRValue       // acr を mfa へ上げる部分集合
```

`UnknownAMRValues` が bool ではなく値の並びを返すのは、拒否のときに何が語彙の外だったかを言えるようにするためである。「不正な amr」とだけ言う error は、要素を足した人にとって何の手がかりにもならない。

### 強制は書き込みの側に置く

`LoginSession` の zog スキーマへ語彙の検査を足し、`SessionManager` が `amr` を書く 2 か所 — `CreateWithPending` と `CompleteFactor` — で保存前に `Validate()` を通す。読み出しの側には課さない。書き込みを閉じれば語彙の外の行は生まれないので、読み出しの検証は二重になる。製品は未リリースなので、保存済みの行に対する移行は要らない。

`Validate()` は今まで production のどこからも呼ばれていなかった。語彙の列挙をスキーマへ足すだけでは何も強制されないので、**呼び出しの配線までが 1 組の変更である。**

### 時刻・乱数・永続化の境界

この変更は新しい効果を持ち込まない。語彙の検査は入力だけから決まる計算であり、`SessionManager` が持つ時刻 (`now`)、識別子生成 (`spec.NewUUIDv4`)、永続化 (`ports.SessionStore`) の境界は変わらない。検査は永続化の直前、すでに効果が集まっている場所に入る。

### リスクを `high` へ上げた

起票時は `medium` としていた。実際の帰結は認証強度そのものであり、間違えたときの損害が [docs/development/specification-first-workflow.md](../../docs/development/specification-first-workflow.md) の強い行の記述に当たる。語彙を締めすぎればセッションが 1 つも作れなくなり (fail-closed で全利用者がサインインできない)、`rc` の扱いを間違えれば MFA の関門が緩む。どちらも変更した純粋なロジックが小さいので、体系的な変異は現実的な費用で払える。

## Plan

1. 標準の行、`REQ-AUTHENTICATION-036`、TypeSpec の `amr` / `acr` の doc、`internals.md` の機構を先に直す。`RFC8176-AMR-VOCABULARY` を台帳から外し、Acceptance RED を観測する。
2. `backend/authentication/domain` に語彙を置く。Unit RED → GREEN。
3. `LoginSession` の検証へ語彙を足し、`SessionManager` の 2 か所へ配線する。
4. `mfaAMRValues` へ `rc` を足す。
5. `REQ-AUTHENTICATION-036` の E2E を、復旧コードの正式入口から `/authorize` の継続まで通しで書く。
6. 連合ログインの既存 E2E へ `RFC8176-AMR-VOCABULARY` の名指しと `amr` の観測を足す。
7. 変更した純粋ロジックを体系的に変異させ、殺せなかったものを記録する。
8. リリースノートを書き、`mise run verify`。

## Tasks

- [x] T001 [Spec] 標準の行、`REQ-AUTHENTICATION-036`、TypeSpec の doc、`internals.md` を直す。
- [x] T002 [Acceptance] 台帳から id を外し、`mise run check-spec` の Acceptance RED を観測する。
- [x] T003 [Domain] `backend/authentication/domain` へ `amr` の語彙を置く。
  recipe: `mise run test-go-package -- ./backend/authentication/domain`
- [x] T004 [Domain] `LoginSession` の検証へ語彙を足す。
  recipe: `mise run test-go-package -- ./backend/authentication/session/domain`
- [x] T005 [Use Cases] `SessionManager` の 2 か所へ検証を配線し、`mfaAMRValues` へ `rc` を足す。
  recipe: `mise run test-go-package -- ./backend/authentication/session/usecases` と
  `mise run test-go-package -- ./backend/authentication/usecases`
- [x] T006 [Adapters] `REQ-AUTHENTICATION-036` の E2E を復旧コードの正式入口から書く。
  recipe: `mise run test-go-package -- ./backend/oauth2/handlers_http`
- [x] T007 [Adapters] 連合ログインの E2E へ `RFC8176-AMR-VOCABULARY` の名指しと `amr` の観測を足す。
  recipe: `mise run test-go-package -- ./backend/authentication/federation/handlers_http`
- [x] T008 [Evidence] 変更した純粋ロジックを体系的に変異させ、生き残りを記録する。
  recipe: `mise run test-go-mutation`
- [x] T009 [Docs] `docs/releases/changes/wi-508.md` を書く。
- [x] T010 [Verify] `mise run verify`。

## Verification

- `RFC8176-AMR-VOCABULARY` が `tools/check/standards-coverage-debt.json` から消えている。
- 語彙の外の値を持つ `LoginSession` が保存されないことを観測するテストがある。
- 復旧コードで第二要素を通したセッションが、MFA 必須のアプリケーションで第二要素を再び求められないことを、正式入口からの E2E が観測する。
- `federated` と `rc` について、`standards.md`、`scenarios.feature.md`、`internals.md`、TypeSpec の doc、実装が同じことを言っている。
- `mise run check-spec`
- `mise run verify`

## Risk Notes

- **語彙を締めすぎるとサインインが止まる。** 検査は fail-closed なので、語彙から実在の値を落とすとその経路のセッションが 1 つも作れなくなる。`federated` がまさにその例であり、E2E で連合ログインが通ることを対照として置く。
- **`rc` を MFA 充足に含めるとサインインポリシーの強度が変わる。** 復旧コードだけで「毎回 MFA」を満たせるようになる。`internals.md:77` が `mfa_enrolled` に復旧コードを数えない理由と併せて読む必要がある。登録の規則は緩めない。
- **`amr` はリライングパーティーまで届く。** 値の追加は取り消しにくい。`reversibility: irreversible` はこれを指す。
- **検証の配線を片方だけ入れる。** `amr` を書くのは作成と第二要素の成立の 2 か所であり、片方だけに入れるともう片方から語彙の外の値が入る。両方に入れたことを、それぞれ別の観測で確かめる。

## Completion

- **Completed At**: 2026-09-06
- **Summary**:
  `mise run spec-diff` は `REQ-AUTHENTICATION-036` の追加と `RFC8176-AMR-VOCABULARY` の変更を返す。
  復旧コードで成立した第二要素が MFA の要求を満たすようになった。`amr` に `rc` が加わると `acr` が
  `urn:idmagic:acr:mfa` へ上がり、以後の認可要求は第二要素を再び求めない。`LoginSession.amr` の語彙は
  `federated` を加えた 8 語となり、語彙の外の値を持つセッションは保存されなくなった。語彙の宣言は
  `backend/authentication/domain` の 1 か所にあり、`LoginSession.Validate()` がそれを使う。この検証は
  それまで production のどこからも呼ばれていなかったので、`SessionManager` が `amr` を書く 2 か所へ
  配線するところまでが 1 組の変更である。
- **Primary Use Case Evidence**:
  - id: recovery-code-satisfies-mfa
    unit_red: "TestDeriveACRTreatsRecoveryCodeAsASecondFactor は acr=urn:idmagic:acr:pwd を得て、urn:idmagic:acr:mfa を期待して落ちた。"
    e2e_red: "TestRecoveryCodeSecondFactorSatisfiesMfaPolicy_REQ_AUTHENTICATION_036 は保存された acr が urn:idmagic:acr:pwd のままで落ちた。"
    unit_fault_injection: "mfaAMRValues から rc を外すと、同テストが同じ観測で落ちた。"
    e2e_fault_injection: "mfaSatisfied が rc を充足として読まないようにすると、認可が /realms/default/totp へ戻されて落ちた。これが Motivation で読み解いた締め出しの、実際に観測された姿である。"
  - id: federated-login-amr-accepted
    unit_red: "TestLoginSessionClosesTheAMRVocabulary は語彙の外の値 mfa が受理されて落ちた。"
    e2e_red: "mise run check-spec が RFC8176-AMR-VOCABULARY is declared, but no test names it を返した。TestFederatedLoginPrimaryUseCase_REQ_AUTHENTICATION_001 はこの行をまだ名指していなかった。"
    unit_fault_injection: "語彙の検査が入力を見ないようにすると、同テストが同じ観測で落ちた。"
    e2e_fault_injection: "語彙から federated を外すと callback が 401 federation_failed になり同テストが落ちた。語彙を締めすぎると正規の経路が止まることの観測でもある。"
- **Change-Resistance Results**:
  `risk: high` なので、変更した純粋ロジックを体系的に変異させた。`mise run test-go-mutation` を
  変更した 3 パッケージへ掛けた結果は次のとおり。

  | パッケージ | 変更した箇所の変異 | 結果 |
  |---|---|---|
  | `backend/authentication/session/domain` | `login_session.go:97` の `len(unknown) > 0` | CONDITIONALS_NEGATION と CONDITIONALS_BOUNDARY の 2 件とも KILLED |
  | `backend/authentication/usecases` | `acr_vocabulary.go:29` の `ACRSatisfies` | 3 件とも KILLED |
  | `backend/authentication/domain` | `amr.go` | **変異が 1 件も生成されなかった** |

  **方法の限界を 2 つ記録する。**

  第一に、`amr.go` に gremlins は変異を作れない。この道具が書き換えるのは条件式、算術、増減だけで
  あり、`amr.go` は slice リテラルと `slices.Contains` しか持たないからである。**変異が 0 件であることは
  テストが強いことの証拠ではなく、道具がこの形のコードについて何も言えないということである。**
  この部分は手で入れた故障が受け持つ: 語彙から `federated` を外す、`mfaAMRValues` から `rc` を外す、
  検査が入力を見ないようにする、の 3 通りで、いずれも上の Primary Use Case Evidence が観測している。

  第二に、`acr_vocabulary.go` の 3 件は最初の実行では NOT COVERED だった。`ACRSatisfies` を通す
  テストがそのパッケージに無かったためである。この関数は「`pwd` が `mfa` の要求を満たすか」を
  決める述語であり、**本項目が直した締め出しはここが false を返したことで起きていた。** 変異が
  それを名指したので `TestACRSatisfiesTreatsMfaAsStrongerThanPassword` を足し、再実行で 3 件とも
  KILLED になった。

  変更していない箇所の生き残り (`signin_activity.go` の 3 件、`login_session.go:75` の Touch の
  境界、`user_lifecycle_helpers.go` の 3 件) は本項目の変更範囲外なので触っていない。
  `login_session.go:75` の CONDITIONALS_BOUNDARY は、touch 間隔ちょうどの 1 点でしか違わない
  等価に近い変異である。
- **Verification Results**:
  - `mise run check-spec` - passed (`ok normative coverage (156 standard(s), 311 rule(s), 744 example(s), 164 id(s) named by a test)`)
  - `mise run check-work-items` - passed
  - `mise run check-ids` - passed
  - `mise run test-go-changed` - passed
  - `mise run test-go-mutation` - 上表のとおり
  - `mise run verify` - passed
  - `mise run test-ui-e2e` - N/A: 変更は Go と正典文書だけで、ブラウザーの画面に到達する変更が無い。第二要素の画面へ戻らないことは Go の handler テストが正式入口から観測している。
