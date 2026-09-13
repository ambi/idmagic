---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-13
priority: p3
depends_on: []
change_kind: bugfix
spec_impact: { kind: none, reason: "宣言済みの具体例が要求する警告の表示を、アカウント API も UI も持っていない。規範は動かさず実装を合わせる。" }
affected_spec:
  - { path: docs/contexts/authentication/scenarios.feature.md, requirement: REQ-AUTHENTICATION-019 }
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

## Verification

- `mise run check-spec`
- `mise run test-go-package -- ./backend/authentication/handlers_http`
- `mise run test-ui-unit-file -- src/features/account/AccountSecurityPage.test.tsx`

## Risk Notes

- **すべての利用者へ日時を返す。** 既に認証要素を登録している利用者にとって強制開始は予告にならない。未登録のときだけ載せる。
- **管理者向けの生のポリシーをそのまま返す。** アカウント API が返すのは実効サインインポリシーから導いた 1 つの日時であって、ルールの配列ではない。ルールごと返すと、利用者向けの応答が管理者の設定を露出する。
- **UI の文言テストが翻訳済みの文字列を直書きする。** 辞書の値を参照する ([[feedback-tests-reference-dictionary-values]])。
