---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-13
priority: p3
depends_on: []
change_kind: bugfix
evidence_policy: risk-based-v3
documentation_impact:
  level: release_note
  reason: アカウントのセキュリティ設定の応答へ任意フィールドが増え、利用者の画面に強制開始の予告が現れるため、リリースの読者へ知らせる。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-572-the-mfa-enforcement-date-never-reaches-the-user.md }
initial_context:
  specification: [docs/domain/authentication/scenarios.feature.md#REQ-AUTHENTICATION-019]
  typespec: [IdMagic.Contract.AccountSecurityResponse]
  source:
    - backend/authentication/handlers_http/account_security_handler.go
    - backend/authentication/deps_http/deps.go
    - backend/application/usecases/sign_in_policy.go
    - backend/oauth2/handlers_http/authorize_enrollment.go
    - backend/shared/http/server_http/routes.go
    - frontend/src/features/account/AccountSecurityPage.tsx
    - frontend/src/features/account/AccountSecurityPage.i18n.ts
    - frontend/src/types.ts
  tests:
    - backend/application/usecases/sign_in_policy_test.go
    - backend/authentication/handlers_http/account_handlers_test.go
    - frontend/src/features/account/AccountSecurityPage.test.tsx
  stop_before_reading: [backend/authentication/mfa, frontend/tests/e2e]
primary_use_cases:
  - id: announce-upcoming-mfa-enforcement
    requirement: REQ-AUTHENTICATION-019
    observable_result: 強制開始前かつ認証要素が未登録の利用者に、アカウントのセキュリティ設定の応答が強制開始日時を返し、画面がその日時と事前登録を促す警告を表示する。
    unit_test: { path: backend/application/usecases/sign_in_policy_test.go, name: TestUpcomingMfaEnforcementStartReturnsOnlyAFutureStart, task: test-go-race }
    e2e_test: { path: backend/authentication/handlers_http/account_handlers_test.go, name: TestHandleGetAccountSecurityAnnouncesUpcomingMfaEnforcement, task: test-go-race }
    unit_fault_model: 強制開始を過ぎた日時、または MFA を要求しないルールの日時を予告として返す。
    e2e_fault_model: ルーティングがテナントデフォルトポリシーのリポジトリをアカウント API へ渡さない、またはハンドラーが登録済みの利用者にも日時を返す。
affected_spec:
  - { path: docs/domain/authentication/scenarios.feature.md, requirement: REQ-AUTHENTICATION-019 }
  - { path: spec/contexts/authentication/models.tsp, symbol: IdMagic.Contract.AccountSecurityResponse }
---

# MFA の強制開始日時が利用者へ届かない

## Motivation

`EX-AUTHENTICATION-019-01` は、MFA の強制開始前について 3 つを要求する。

1. 強制開始前なので、パスワードだけのセッションが成立する
2. UI は強制開始日時と事前登録を促す警告を表示する
3. 利用者は通常のステップアップ認証を経たアカウントのセキュリティ設定画面から認証要素を事前登録できる

1 と 3 は成立している。1 は `EvaluateSignInPolicy` が `MfaEnrollment.EnforcementStartAt` を読んで判定しており、`backend/application/usecases/sign_in_policy_test.go` の `TestEvaluateSignInPolicyDoesNotRequireMfaBeforeEnforcementStart` が固定した。3 は `EX-AUTHENTICATION-011-01` が持つ。

**2 が無い。** `GET /api/account/v1/security` の応答は登録済みの認証要素と復旧コードの残数だけを返し、強制開始日時を運ばない。`enforcement_start_at` を読んでいる画面は管理者向けのサインインポリシー編集画面だけで、アカウント側の画面には現れない。日時が API を出ないので、UI は表示しようがない。

帰結として、利用者は強制開始の当日にログインできなくなるまで予告を受け取らない。事前登録という逃げ道は用意されているのに、それがあることを知る経路が無い。

[[wi-539-back-authentication-examples-with-tests]] がこれを測り、`EX-AUTHENTICATION-019-01` を台帳へ残した。

## Scope

- アカウントのセキュリティ設定の応答へ、実効サインインポリシーが持つ MFA の強制開始日時を載せる。未設定または既に強制開始済みなら載せない。
- アカウントのセキュリティ画面が、強制開始前かつ未登録の利用者へ、日時と事前登録を促す警告を表示する。
- `tools/check/example-coverage-debt.json` から `EX-AUTHENTICATION-019-01` を外す。

## Out of Scope

- 強制開始が近いことのメール通知。届ける経路を増やすのは別の判断である。
- 管理者向けのサインインポリシー編集画面。日時の入力側は既にある。
- 猶予期間と登録バイパスの規則そのもの。

## Design

アカウントポータルへのサインインにはアプリ個別のポリシーがなく、テナントデフォルトポリシーが実効ポリシーになる。予告する日時はそこから導く。

判断は `backend/application/usecases` の純粋関数に置き、時刻は引数で受け取る。

```go
// 実効ルールのうち MFA を要求する最初の有効なルールが、now より後の強制開始日時を持つときその日時を返す。
func UpcomingMfaEnforcementStart(rules []domain.SignInRule, now time.Time) *time.Time
```

`handleGetAccountSecurity` は認証要素 (TOTP などの MFA 要素と WebAuthn 資格情報) を数えた後、どちらもなければ `Deps.DefaultSignInPolicyRepo` からテナントデフォルトポリシーを読み、`time.Now().UTC()` でこの関数を呼ぶ。結果を `mfa_enforcement_start_at` に載せる。リポジトリが未配線なら載せない。永続化の作用はこの読み取り 1 つだけである。

`authentication/deps_http.Deps` に `DefaultSignInPolicyRepo appports.DefaultSignInPolicyRepository` を加え、`server_http/routes.go` が `d.Application.DefaultSignInPolicyRepo` を渡す。

UI は応答に日時があり、画面上で TOTP もパスキーも登録されていない間だけ、警告を表示する。画面内で登録を終えると警告は消える。

採用しない案は次のとおりである。

| 案 | 採用しない理由 |
| --- | --- |
| サインイン直後の応答で予告する | 予告はポータルのセキュリティ画面に置くと規範が定め、登録手段も同じ画面にある |
| 実効ルールの配列を返して UI が判定する | 利用者向けの応答が管理者の設定を露出する |
| 復旧コードも登録済みに数える | 復旧コードは第二要素の代替であり、それだけでは強制開始後の MFA を満たせない |

## Tasks

- [x] T001 [Spec] `AccountSecurityResponse` へ任意の `mfa_enforcement_start_at` を加え、`check-spec` と `check-api-compat` を通す。
- [x] T002 [Acceptance] `TestHandleGetAccountSecurityAnnouncesUpcomingMfaEnforcement` の RED を `mise run test-go-test -- ./backend/authentication/handlers_http TestHandleGetAccountSecurityAnnouncesUpcomingMfaEnforcement` で確認する (`EX-AUTHENTICATION-019-01`)。
- [x] T003 [Domain] `TestUpcomingMfaEnforcementStartReturnsOnlyAFutureStart` の RED を `mise run test-go-test -- ./backend/application/usecases TestUpcomingMfaEnforcementStartReturnsOnlyAFutureStart` で確認し、GREEN にする。
- [x] T004 [Adapters] Deps とルーティングを配線し、受け入れテストを GREEN にする。
- [x] T005 [UI] `mise run test-ui-unit-file -- src/features/account/AccountSecurityPage.test.tsx` で警告の RED を確認し、GREEN にする。
- [x] T006 [Verify] 台帳から `EX-AUTHENTICATION-019-01` を外し、フォールト注入、変異テスト、`mise run verify`、`mise run test-ui-e2e` を通す。

## Verification

- `mise run check-spec`
- `mise run test-go-package -- ./backend/authentication/handlers_http`
- `mise run test-ui-unit-file -- src/features/account/AccountSecurityPage.test.tsx`

## Risk Notes

- **すべての利用者へ日時を返す。** 既に認証要素を登録している利用者にとって強制開始は予告にならない。未登録のときだけ載せる。
- **管理者向けの生のポリシーをそのまま返す。** アカウント API が返すのは実効サインインポリシーから導いた 1 つの日時であって、ルールの配列ではない。ルールごと返すと、利用者向けの応答が管理者の設定を露出する。
- **UI の文言テストが翻訳済みの文字列を直書きする。** 辞書の値を参照する ([[feedback-tests-reference-dictionary-values]])。

## Completion

- **Completed At**: 2026-09-23
- **Summary**:
  `mise run spec-diff` は main に対する規範仕様の差分なしを報告した。規範は動かさず、`EX-AUTHENTICATION-019-01` が要求していた警告の表示を実装へ揃えた。
  公開契約では `IdMagic.Contract.AccountSecurityResponse` に任意の `mfa_enforcement_start_at` が加わり、`check-api-compat` は破壊的変更なしと判定した。
  `GET /api/account/v1/security` は、テナントデフォルトポリシーの MFA 強制開始がまだ来ておらず、利用者が MFA 要素もパスキーも持たないときだけ、その日時を返す。
  アカウントのセキュリティ画面はその日時と事前登録を促す警告を表示し、画面上で認証アプリかパスキーを登録すると警告を消す。
  `tools/check/example-coverage-debt.json` から `EX-AUTHENTICATION-019-01` を外した。
- **Primary Use Case Evidence**:
  - id: announce-upcoming-mfa-enforcement
    unit_red: 関数が常に nil を返す段階で、TestUpcomingMfaEnforcementStartReturnsOnlyAFutureStart が強制開始前の事例で nil を返して失敗した。
    e2e_red: 応答が日時を運ばないため、TestHandleGetAccountSecurityAnnouncesUpcomingMfaEnforcement が「未登録の利用者への強制開始日時 = <nil>」で失敗した。
    unit_fault_injection: 境界の `!now.Before(start)` を `now.After(start)` に替えると、TestUpcomingMfaEnforcementStartReturnsOnlyAFutureStart が「開始時刻ちょうど」の事例で失敗した。
    e2e_fault_injection: routes.go から DefaultSignInPolicyRepo の配線を外すと未登録の利用者の事例で、登録済みの判定を外すと TOTP 登録済みの事例で、WebAuthn 資格情報の判定だけを外すとパスキー登録済みの事例で、それぞれ TestHandleGetAccountSecurityAnnouncesUpcomingMfaEnforcement が失敗した。
- **Acceptance RED Evidence**:
  - **Test**: `frontend/src/features/account/AccountSecurityPage.test.tsx` の `warns an unenrolled user of the enforcement date and asks them to enroll`
  - **Requirement**: REQ-AUTHENTICATION-019
  - **Observed Failure**: 辞書に文言を加えた後、警告の見出しが描画されず `getByText(t.mfaEnforcementTitle)` が失敗した。
  - **Detection Reason**: 辞書の値から組み立てた日時入りの文を完全一致で探すため、日時を落とす実装や警告を出さない実装と区別できる。画面内で TOTP を登録した後に警告が残る実装は、`removes the warning once the user enrolls an authenticator app` が `!enrolled` の条件を外すフォールトで失敗することで区別した。
- **Change-Resistance Results**:
  `mise run test-go-mutation -- backend/application/usecases` は 171 件中 157 件を殺した。追加した `UpcomingMfaEnforcementStart` と、それが使う `MfaEnrollmentPolicyFromRules` の変異 (337 行、348 行) はすべて殺された。生き残った 14 件は、この記録で変更していない既存の関数の行にある。
  変異器が表現できない境界の置き換え、配線の除去、判定の除去は上の fault injection で手作業により確かめた。
- **Verification Results**:
  - `mise run check-spec` - 成功
  - `mise run check-api-compat` - 成功
  - `mise run check-contract-drift` - 成功
  - `mise run lint-go` - 成功
  - `mise run test-go-changed` - 成功
  - `mise run test-ui-unit-file -- src/features/account/AccountSecurityPage.test.tsx` - 成功 (27 件)
  - `mise run verify` - 成功 (初回は UI の整形で失敗し、`mise run format-ui` の後に成功)
  - `mise run test-ui-e2e` - 成功 (38 件)
