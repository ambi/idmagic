---
depends_on: []
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-06
priority: p2
change_kind: maintenance
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 標準の行にも製品の振る舞いにも変更が無く、増えたのはテストと注記だけなので、リリースの読み手に見えるものが無い。
  references: []
spec_impact: { kind: none, reason: "宣言済みの標準行に、その id を名指しするテストを対応付ける作業である。standards.md の行そのものも製品の振る舞いも変えない。テストが書けない行が見つかった場合、それは製品が宣言した採用を満たしていないということなので、欠陥として個別の work item に切り出す。" }
initial_context:
  specification:
    - docs/contexts/authentication/standards.md
    - docs/contexts/authentication/internals.md
  typespec:
    - IdMagic.Contract.LoginSession
  source:
    - backend/authentication/password/usecases/password_policy.go
    - backend/authentication/password/usecases/change_password.go
    - backend/authentication/password/domain/password_policy_resolver.go
    - backend/shared/security/passwords_argon2id/argon2id_password_hasher.go
    - backend/shared/security/testing_passwords/hasher.go
    - backend/authentication/webauthn/usecases/webauthn.go
    - backend/authentication/webauthn/usecases/account_webauthn.go
    - backend/authentication/webauthn/usecases/verify_webauthn_factor.go
    - backend/authentication/totp/usecases/totp.go
    - backend/authentication/federation/protocol_oidc/client.go
    - backend/authentication/federation/usecases/flow.go
    - backend/authentication/federation/usecases/broker.go
    - backend/authentication/session/domain/login_session.go
    - backend/authentication/session/usecases/session_manager.go
    - backend/authentication/usecases/acr_vocabulary.go
    - tools/check/src/normative-coverage.ts
    - tools/check/standards-coverage-debt.json
  tests:
    - backend/authentication/password/usecases/password_policy_test.go
    - backend/authentication/password/usecases/change_password_test.go
    - backend/authentication/webauthn/usecases/webauthn_test.go
    - backend/authentication/totp/usecases/totp_test.go
    - backend/authentication/federation/protocol_oidc/client_test.go
    - backend/authentication/federation/usecases/flow_test.go
    - backend/authentication/federation/usecases/broker_test.go
  stop_before_reading:
    - docs/contexts/oauth2/standards.md
    - backend/oauth2
    - frontend
---

# Authentication が宣言する標準 9 行にテストを対応付け、標準の被覆台帳から外す

## Motivation

[[wi-495-burn-down-the-standards-coverage-debt]] は標準の被覆台帳へ受入集合を入れ、消化の単位を所有文書と決めた。本項目はそのうち `docs/contexts/authentication/standards.md` の 9 行を引き取る。この文書は 10 行のうち 9 行が名指しを持たない。

9 行が扱うのはパスワードの規則と保管、WebAuthn の登録と認証、TOTP、認証方式の申告（`amr`）、そして認可要求の CSRF 防護である。いずれも認証そのものの強度に直結し、外部の規範が具体的な形を指定している領域である。

## Scope

- 次の 9 行を消化する。

| ID | Adoption |
|---|---|
| `NIST63B4-NO-COMPOSITION` | required |
| `NIST63B4-PASSWORD-MINIMUM` | excluded |
| `NIST63B4-PASSWORD-STORAGE` | required |
| `OIDC-CORE-CSRF` | required |
| `OIDC-DISCOVERY-ISSUER` | required |
| `RFC6238-TOTP` | optional |
| `RFC8176-AMR-VOCABULARY` | required |
| `WEBAUTHN3-AUTHENTICATION` | required |
| `WEBAUTHN3-REGISTRATION` | required |

- 1 行ごとに、その行の `Statement` を区別できる入力と観測を持つテストを対応付け、`// <ID>: <この行の何を固定しているか>` の注記を足す。
- 観測の形は行の `Adoption` に従う。型は [[wi-495-burn-down-the-standards-coverage-debt]] の Design が定める。
- 消化した id を `tools/check/standards-coverage-debt.json` から外す。
- 宣言した採用を満たしていない行が見つかったら、欠陥の work item を切り出す。その行は台帳に残し、理由を「投入時からある」から見つけた内容へ書き換える。

## Out of Scope

- 他の文書が宣言する標準行。文書ごとに別の work item が持つ。OAuth2 が持つ `OIDC-CORE-*` と `OIDC-DISCOVERY-*` の他の行は [[wi-499-back-oauth2-standards-rows-with-tests]] が持つ。
- `standards.md` の行の追加、削除、`Adoption` および `Strength` の変更。
- 見つかった実装の欠陥の修正。欠陥として切り出した先で扱う。**T006 で 1 件見つかったので、[[wi-508-amr-vocabulary-declaration-and-implementation-disagree]] へ切り出した。**
- 受入集合の導入と、台帳ファイルそのものの削除。親の [[wi-495-burn-down-the-standards-coverage-debt]] が持つ。
- 具体例の被覆負債。[[wi-496-burn-down-the-example-coverage-debt]] が持つ。
- 認証要素の追加や、パスワード規則そのものの変更。
- id を名指しする文字列があるだけを根拠にした台帳からの削除。

## Design

`NIST63B4-PASSWORD-MINIMUM` が `excluded` であり、`NIST63B4-NO-COMPOSITION` が `required` である。この 2 行は対になっている。組成規則（大文字・記号の強制）を課さないことを宣言し、長さの下限だけを別の形で扱うという構えである。したがって観測も対にする。組成規則を満たさないが長さは足りるパスワードが受理されること、および `excluded` の側が禁じている入力の扱いが、行の書きぶりと一致していることを併せて確かめる。片方だけを観測すると、両方を課している実装と区別できない。

`NIST63B4-PASSWORD-STORAGE` は保管の形を宣言する行である。観測は「保存されたものが平文でも可逆でもないこと」であり、ハッシュ関数を呼んでいることではない。保存先を読み直して確かめる。

`WEBAUTHN3-REGISTRATION` と `WEBAUTHN3-AUTHENTICATION` は、登録と認証で別の検証を持つ。1 つのテストで両方を名指しても、片方の検証が無い実装を区別できない。行ごとに別のテストを対応付ける。

`RFC6238-TOTP` は `optional` である。提供しているならその振る舞いを観測する。提供していないなら行の `Adoption` が誤っているので、規範の変更として切り出す。

`OIDC-CORE-CSRF` の観測は、`state` の照合が無い実装で落ちることである。値が返ってくることではなく、違う値では認可が成立しないことを観測する。

### `excluded` の観測の型（本文書で決めた）

[[wi-495-burn-down-the-standards-coverage-debt]] は `excluded` の型を各子 work item がその文書の 1 件目で決めるとした。本文書の `excluded` は `NIST63B4-PASSWORD-MINIMUM` の 1 行だけであり、次の型を採った。

**`excluded` の行の観測は、その標準を採用した実装なら拒否するはずの入力が、製品では受理されることである。** 受理の一点だけでは「下限そのものを持たない実装」と区別が付かないので、行が代わりに何を課しているかを併せて観測する。`NIST63B4-PASSWORD-MINIMUM` では、NIST の 15 文字下限に届かない 14 文字が受理されること（採用していないこと）と、デフォルトの下限 12 が実在すること、テナントがそれを引き上げられることの 3 つになる。

`oauth2` の `excluded` 15 行の多くは「Implicit Grant を提供する」のように標準側の機能を書いているので、そちらでは「要求が届いたときに拒否し、拒否が防いだ効果を観測する」型になるはずである。本文書の型がそのまま移るとは限らない。

### 対の入力を、互いの変数で汚さない

`NIST63B4-NO-COMPOSITION` と `NIST63B4-PASSWORD-MINIMUM` は同じパスワード規則の裏表なので、入力を選び損ねると片方のテストがもう片方を観測してしまう。**各行の入力から、相手の行の変数を消す。**

- `NIST63B4-NO-COMPOSITION` の入力は単一文字種 20 文字。長さは下限を明らかに超えているので、拒否されたなら理由は構成規則しかない。
- `NIST63B4-PASSWORD-MINIMUM` の入力は 4 文字種を混ぜた 14 文字。構成規則をどう課しても満たすので、拒否されたなら理由は長さしかない。

### `RFC8176-AMR-VOCABULARY` は消化できない（T006 の結論）

この行だけは、行を満たすテストが書けない。`standards.md` は `LoginSession.amr` の語彙を 7 語（`pwd` / `otp` / `webauthn` / `hwk` / `swk` / `rc` / `tdev`）と宣言しているのに、同じ context の `scenarios.feature.md:17` と `internals.md:13` は連合ログインが `federated` を持つセッションを発行すると規範として書いており、実装（`backend/authentication/federation/usecases/broker.go:106`）は後者に従っている。`federated` は RFC 8176 の登録値でも、行が挙げる非 IANA 拡張値でもない。

2 つの正典文書が同じフィールドについて両立しないことを言っているので、どちらへ寄せるかは規範の変更であり、本項目の Out of Scope である。行の語彙を強制する場所が実装のどこにも無いこと（`LoginSession` の zog スキーマは `AMR` に `Min(1)` しか課していない）と、`rc` が `acr` を引き上げるかどうかについて TypeSpec の doc と `mfaAMRValues` が食い違っていることも同じ場所で見つかった。3 件まとめて [[wi-508-amr-vocabulary-declaration-and-implementation-disagree]] へ切り出した。

台帳からは外さず、理由を書き換えて残す。**「テストが無い」ことの理由が「まだ書いていない」から「行と製品が食い違っているので書けない」へ変わったことが、この行について本項目が生んだ差分である。** 理由の欄が読める状態に保たれるのは、[[wi-495-burn-down-the-standards-coverage-debt]] が台帳の各エントリーへ理由を要求している目的そのものである。

## Plan

1. ~~`NIST63B4-NO-COMPOSITION` と `NIST63B4-PASSWORD-MINIMUM` を対で消化し、`excluded` の観測の型をここで決める。~~ 完了。型は Design の「`excluded` の観測の型」節。
2. ~~`NIST63B4-PASSWORD-STORAGE` を、保存先を読み直す形で消化する。~~ 完了。
3. ~~`WEBAUTHN3-REGISTRATION` と `WEBAUTHN3-AUTHENTICATION` を、行ごとに別のテストで消化する。~~ 完了。
4. ~~`OIDC-CORE-CSRF` と `OIDC-DISCOVERY-ISSUER` を消化する。~~ 完了。
5. ~~`RFC8176-AMR-VOCABULARY` を、実際に発行されたトークンの `amr` を読む形で消化する。~~ 消化できないと判断した。Design の「消化できない」節と [[wi-508-amr-vocabulary-declaration-and-implementation-disagree]]。
6. ~~`RFC6238-TOTP` の実装の有無を確かめ、観測するか切り出すかを決める。~~ 実装あり。観測した。
7. ~~解決した id を台帳から外す。~~ 完了。8 件を外し、`RFC8176-AMR-VOCABULARY` は理由を書き換えて残した。

## Tasks

- [x] T001 [Acceptance] `NIST63B4-NO-COMPOSITION` と `NIST63B4-PASSWORD-MINIMUM` を対で消化し、`excluded` の型を決める。
  `backend/authentication/password/usecases/password_policy_standards_test.go` に
  `TestPasswordPolicyImposesNoCompositionRule` と `TestPasswordPolicyExcludesTheFifteenCharacterMinimum` を置いた。
  どちらも `ChangePassword` という製品の入口を通す。型は Design の「`excluded` の観測の型」節。
  recipe: `mise run test-go-package -- ./backend/authentication/password/usecases`
- [x] T002 [Acceptance] `NIST63B4-PASSWORD-STORAGE` を、保存先を読み直して消化する。
  同ファイルの `TestPasswordStorageKeepsNeitherPlaintextNorAReversibleForm`。user と password history の
  2 つの保存先を読み直し、平文を含まないこと、コストパラメーターを持つこと、同じ平文が別の値になること
  （salt）を観測する。ハッシュ関数の呼び出しは観測しない。
  recipe: `mise run test-go-package -- ./backend/authentication/password/usecases`
- [x] T003 [Acceptance] `WEBAUTHN3-REGISTRATION` と `WEBAUTHN3-AUTHENTICATION` を、行ごとに別のテストで消化する。
  `backend/authentication/webauthn/usecases/webauthn_ceremony_test.go` に ES256 鍵を持つソフトウェア認証器を
  置き、`TestWebAuthnRegistrationVerifiesTheCeremonyAndStoresTheCOSEKey` と
  `TestWebAuthnAuthenticationVerifiesTheOriginAndRelyingPartyScopedCredential` を分けた。
  recipe: `mise run test-go-package -- ./backend/authentication/webauthn/usecases`
- [x] T004 [Acceptance] `OIDC-CORE-CSRF` と `OIDC-DISCOVERY-ISSUER` を消化する。
  `flow_test.go` の `TestCompleteLoginRejectsAStateThatIsNotTheOneItIssued` と
  `client_test.go` の `TestRefreshDiscoveryRequiresAnExactIssuerAndHTTPSAuthorities`。
  recipe: `mise run test-go-package -- ./backend/authentication/federation/usecases` と
  `mise run test-go-package -- ./backend/authentication/federation/protocol_oidc`
- [x] T005 [Acceptance] `RFC8176-AMR-VOCABULARY` を、発行されたトークンの `amr` から消化する。
  **消化できなかった。** 行と規範シナリオが `federated` について食い違っている。Design の
  「消化できない」節。[[wi-508-amr-vocabulary-declaration-and-implementation-disagree]] へ切り出した。
- [x] T006 [Inventory] `RFC6238-TOTP` の実装の有無を確かめ、観測するか切り出すかを決める。
  `backend/authentication/totp/usecases/totp.go` に実装がある。生成側は RFC 6238 Appendix B の
  テストベクターで既に固定されていたので注記のみ。検証側は窓しか観測していなかったので、別の共有
  シークレットの OTP が拒否されることを足した。
  recipe: `mise run test-go-package -- ./backend/authentication/totp/usecases`
- [x] T007 [Ledger] 解決した id を `standards-coverage-debt.json` から外す。
  8 件を外し（134 → 130 → 122）、`RFC8176-AMR-VOCABULARY` は理由を書き換えて残した。
  recipe: `mise run check-spec`
- [x] T008 [Verify] `mise run verify`。

## Verification

- 本項目が持つ 9 件のうち 8 件が `tools/check/standards-coverage-debt.json` から消えている。残る 1 件
  （`RFC8176-AMR-VOCABULARY`）は理由の欄が「投入時からある」から、行と製品の食い違いと切り出し先を
  名指す文へ書き換わっている。
- 注記を足した各テストについて、対応する production の判断を崩すとそのテストが落ちる。
- `mise run check-spec`
- `mise run verify`

## Risk Notes

- **`excluded` と `required` の対を片方だけ観測する。** `NIST63B4-PASSWORD-MINIMUM` と `NIST63B4-NO-COMPOSITION` は同じパスワード規則の裏表なので、片方のテストがもう片方を名指してしまいやすい。行ごとに別の入力と別の観測を与える。
- **保管の観測がハッシュ関数の呼び出しになる。** 呼んでいることは保管の形ではない。保存先を読み直し、平文でも可逆でもないことを観測する。
- **台帳の同時編集。** 自分が持つ id のエントリーだけを削除し、コメント配列と他の id には触れない。
- **WebAuthn の検証をテスト側の認証器で骨抜きにする。** 応答を組み立てる側がライブラリの内部と同じ計算を持つと、検証の有無を区別できないテストになる。本項目のソフトウェア認証器は署名鍵を実際に持ち、詐称する要素（challenge / RP ID / origin / 署名鍵）を 1 つだけ差し替えた応答を作る。

## Completion

- **Completed At**: 2026-09-06
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返す。規範の変更は無い。
  差分は `docs/contexts/authentication/standards.md` の 10 行のうち、名指しを持たなかった 9 行に対する
  被覆の状態である。8 行がその行の `Statement` を区別できる入力と観測を持つテストを得て
  `tools/check/standards-coverage-debt.json` から消え、台帳は 130 件から 122 件になった。
  残る `RFC8176-AMR-VOCABULARY` は、行と `scenarios.feature.md` が `federated` について両立しないため
  行を満たすテストが書けない。台帳には残したうえで、理由の欄を「投入時からある」から食い違いと
  切り出し先を名指す文へ書き換えた。製品コードは 1 行も変わっていない。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`（9 件を台帳から外した状態で）
  - **Requirement**: N/A: 標準の被覆はテストの有無についての性質であり、製品の規範要求ではない。
  - **Observed Failure**: exit 1。`docs/contexts/authentication/standards.md` の 9 行それぞれについて
    `<ID> is declared, but no test names it. Cite the id from the test that exercises it, or list it in
    tools/check/standards-coverage-debt.json with a reason.`
  - **Detection Reason**: この検査は、宣言された id・テストが名指す id・台帳の 3 つを突き合わせる。
    台帳から外した id は、名指すテストが実在しない限り必ず報告される。つまり「台帳を縮めた」という
    主張は、テストを書かずには通せない。8 件を消化した後の同じコマンドは
    `ok normative coverage (156 standard(s), 310 rule(s), 742 example(s), 161 id(s) named by a test)` を返す。
- **Unit RED Evidence**:
  - **Test**: 各行に対応付けたテストへの故障注入（Change-Resistance Results を参照）
  - **Requirement**: N/A: 上と同じで、行ごとの観測は標準の Statement に対応し、REQ 番号には対応しない。
  - **Observed Failure**: 注記を足した 8 行それぞれについて、対応する production の判断を崩すと
    そのテストが落ちることを観測した。内訳は Change-Resistance Results の表。
  - **Detection Reason**: `checkNormativeCoverage` は文字列の一致しか見ないので、id を書くだけで検査は
    通ってしまう。名指しが実在の検証に付いていることの担保は、production を崩したときにその
    テストが落ちるという観測しかない。したがって本項目ではこれを Unit RED の代わりに置いた。
  - **見つけた不足**: 最初に書いた `TestRefreshDiscoveryRequiresAnExactIssuerAndHTTPSAuthorities` は、
    故障を 2 通り注入してもどちらも生き残った。fixture の JWKS URL が引けず、issuer と endpoint の
    検証に到達する前にすべての事例が落ちていたからである。**拒否だけを並べたテストは、別の理由で
    落ちていても通ってしまう。** 無傷の document が受理されて connection を書き換えることを対照として
    足し、fixture の endpoint をすべて引けるようにしたところ、両方の故障を検出するようになった。
- **Change-Resistance Results**:
  8 行すべてについて、行が言っていることを production 側で崩し、対応するテストが落ちることを観測した。

  | 行 | 注入した故障 | 落ちたテストと観測 |
  |---|---|---|
  | `NIST63B4-NO-COMPOSITION` | `passwordSchemaFor` に「数字を 1 文字以上含むこと」を追加 | `TestPasswordPolicyImposesNoCompositionRule`: 単一文字種 20 文字が `too_short` で拒否 |
  | `NIST63B4-PASSWORD-MINIMUM` | `PasswordPolicyMinLength` を 12 → 15 | `TestPasswordPolicyExcludesTheFifteenCharacterMinimum`: 14 文字が `too_short` で拒否 |
  | 〃 | `ResolvePasswordPolicy` から `MinLength` の override 適用を削除 | 同テスト: 解決後の下限が 20 ではなく 12 |
  | `NIST63B4-PASSWORD-STORAGE` | `ChangePassword` が平文をそのまま保存 | `TestPasswordStorageKeepsNeitherPlaintextNorAReversibleForm`: `password_history` が平文を保持 |
  | 〃 | Argon2id の salt を固定 | 同テスト: 同じ平文の 2 回の保存が同じ値になる |
  | `WEBAUTHN3-REGISTRATION` | `CreateCredential` の error を無視して credential を保存 | `TestWebAuthnRegistrationVerifiesTheCeremonyAndStoresTheCOSEKey`: 別 RP ID の attestation が受理される |
  | 〃 | `fromWebAuthnCredential` が COSE 公開鍵と sign count を捨てる | 同テスト: 保存された公開鍵が authenticator の COSE 鍵と異なる |
  | `WEBAUTHN3-AUTHENTICATION` | `ValidateLogin` の error を無視して assertion を成立させる | `TestWebAuthnAuthenticationVerifiesTheOriginAndRelyingPartyScopedCredential`: 別 origin の assertion が受理される |
  | 〃 | 検証失敗時に sign count を先に書き換えてから拒否 | 同テスト: 拒否したのに sign_count が 99 へ進む |
  | `OIDC-CORE-CSRF` | `Attempts.Consume` が失敗したらその場で attempt を作って続行 | `TestCompleteLoginRejectsAStateThatIsNotTheOneItIssued`: 発行していない state でログインが成立 |
  | `OIDC-DISCOVERY-ISSUER` | issuer の比較を完全一致から `strings.HasPrefix` へ | `TestRefreshDiscoveryRequiresAnExactIssuerAndHTTPSAuthorities`: 末尾スラッシュ違いが受理される |
  | 〃 | endpoint への `validateRemoteURL` を削除 | 同テスト: `http://` の authorization endpoint が受理される |
  | `RFC6238-TOTP` | HMAC の鍵を共有シークレットではなく固定値に | `TestVerifyTOTPWindow`: 別シークレットの OTP が受理される |
  | 〃 | 時間ステップを 30 秒から 60 秒へ | `TestGenerateTOTPRFC6238Vectors`: RFC 6238 のテストベクターと一致しない |

  等価な変異として残るもの: `NIST63B4-PASSWORD-MINIMUM` について、下限を 12 から 11 以下へ下げる変異は
  どのテストも殺さない。行が課しているのは「15 を課さないこと」と「デフォルトは 12」であり、後者は
  `ValidatePassword(11 文字)` の拒否で固定してあるが、10 文字以下のさらに緩い下限を区別する観測は
  置いていない。行がそこまで言っていないからである。
- **Verification Results**:
  - `mise run check-spec` - passed（`ok normative coverage (156 standard(s), ..., 161 id(s) named by a test)`）
  - `mise run check-work-items` - passed
  - `mise run check-ids` - passed
  - `mise run test-go-changed` - passed（205 packages）
  - `mise run verify` - passed
  - `mise run test-ui-e2e` - `N/A: 変更は Go のテストと tools の JSON だけで、ブラウザーへ到達する経路が無い。`
