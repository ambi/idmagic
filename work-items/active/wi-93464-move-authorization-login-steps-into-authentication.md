---
status: in_progress
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-10-10
priority: p2
depends_on: [wi-87746-publish-authentication-operations-other-modules-use, wi-28791-publish-application-sign-in-policy-evaluation, wi-52235-require-the-tenant-default-sign-in-policy-for-idmagics-own-consoles]
change_kind: refactor
spec_impact: { kind: none, reason: "認可エンドポイントの途中で行うログインと追加の認証の手順の実装を、OAuth2 の handler から Authentication へ移すだけである。手順の順序、各手順の判定、HTTP の応答、セッションの状態、ドメインイベントは変えない。" }
---

# 認可エンドポイントの対話の手順を Authentication へ移す

## 動機

`tools/check/boundary-debt.json` の `private-import` のうち 9 件は、`backend/oauth2/handlers_http` から Authentication の非公開パッケージへの依存である。
OAuth2 の handler は、認可要求の途中のログイン、パスワードの期限、MFA の登録、TOTP、WebAuthn、回復コード、信頼済みデバイスの各手順を、Authentication の九つの usecases と `webauthn/handlers_http` を直接呼んで編成している。

各手順の規則と状態は Authentication が持つので、Authentication の変更が OAuth2 の handler へ波及する。
Authentication と OAuth2 の共変更は 4 回で、共変更の組の上位にある。

## 対象範囲

- 認可要求の途中で行う認証の手順の実装を Authentication へ移す。
- OAuth2 には、認可要求を保持し、認証が完了したら要求を再開する責務だけを残す。
- 解消した違反 ID を台帳から消す。

## 対象外

- 手順の順序と各手順の判定の変更。
- サインイン方針の評価（Application への依存）。[Application のサインイン方針の公開](../done/wi-28791-publish-application-sign-in-policy-evaluation.md)で扱う。

## 設計

[境界の負債の順位付け](../done/wi-33994-reinventory-the-remaining-boundary-debt.md)の判断を引き継ぐ。

- D1：各手順の規則と状態（チャレンジ、登録の途中の状態、試行の回数）の所有者は Authentication である。
- D8：OAuth2 と Authentication を独立して変えたい変更シナリオは、新しい認証方式の追加と、認可要求の形式の変更である。現状はどちらも `oauth2/handlers_http` を変える。

| 案 | 判断 |
| --- | --- |
| 手順の handler を Authentication へ移し、OAuth2 は認可要求の再開だけを受け持つ | 第一候補。新しい認証方式の追加が Authentication で閉じる |
| Authentication が手順ごとの公開操作を出し、OAuth2 が編成を続ける | 比較する。依存は公開パッケージへ移るが、手順の追加は引き続き OAuth2 を変える |

利用者との確認で第一候補を採った。
第二の案は、パスワードの照合や第二要素の検証の画面の処理を OAuth2 に残すので、本人の確認という Authentication の責務の置き場所を直さない。

### 境界の判断の記録

入力は、台帳の `private-import:OAuth2->Authentication` の 8 件（`mfa`、`password`、`recovery`、`session`、`totp`、`trusteddevice`、`webauthn` の各 `usecases` と `webauthn/handlers_http`）である。
TypeSpec は `/api/auth/login`、`/api/auth/totp`、`/api/auth/webauthn/challenge`、`/api/auth/webauthn`、`/api/auth/recovery-code`、`/api/auth/mfa/enrollment/totp/{start,confirm}` を Authentication の操作として定めており、実装の置き場所だけが仕様の担当とずれている。
`/api/auth/transaction` は OAuth2 の操作である。

対話のログインには続きが三つある。
OAuth の認可トランザクションの再開、管理コンソールへの直接のログイン（`return_to=/admin`）、WS-Federation（`return_to=/wsfed`）である。
途中のサインインポリシーの評価は続きによって変わり、first-party のクライアントと `return_to` ではテナントのデフォルト（[REQ-APPLICATION-016](../../docs/modules/application/sign-in-policy/README.md#req-application-016-idmagic-自身の画面へのサインインにはテナントのデフォルトのポリシーを適用する)）、それ以外のクライアントでは Application のアクセスの判定（REQ-APPLICATION-009、010）を使う。

| 置き場所 | 置くもの |
| --- | --- |
| Authentication の `signin/handlers_http`（新設） | 上の 7 操作の handler、ログインの抑制、信頼済みデバイスの発行と評価、第二要素の要求、MFA の登録の開始、テナントのデフォルトの適用、最終ログインと必須操作の記録、`UserAuthenticated` と `AuthenticationFailed` の発行、`return_to` の検証と戻り |
| OAuth2 の `handlers_http` | 認可要求の保持、続きの実装、同意、コードの発行、`/authorize`、`/authorize/resume`、`/api/auth/transaction`、`/end_session` |

継ぎ目は三つで、どれも要求する側がインターフェースを定め、組み立て地点で結ぶ（D4）。
Authentication と OAuth2 は互いの非公開パッケージを import しない。

| 継ぎ目 | 定める側 | 実装する側 | 内容 |
| --- | --- | --- | --- |
| 認証の後の続き | Authentication | OAuth2 | 要求に結び付いた認可要求を返し、その続きが求めるサインインの条件を答え、完了した認証で認可を再開する |
| 既存のセッションでの追加の認証 | OAuth2 | Authentication | テナントのデフォルトの適用、第二要素の要求、保留中の画面の経路、第二要素の選択肢、サインインの完了の記録 |
| ログアウトでのセッションの終了 | OAuth2 | Authentication | Cookie が指すセッションの解決、主体の照合、終了、Cookie の消去 |

```go
// Authentication の公開パッケージ signin/ports が定め、OAuth2 が実装する。
type PendingAuthorizations interface {
  // Find はリクエストの Cookie が指す認可要求を返す。なければ理由を表すエラーを返し、
  // ログインは return_to の経路を試す。
  Find(c *echo.Context) (Authorization, error)
}
type Authorization interface {
  ClientID() string
  // Requirement は、この続きが求めるサインインの条件を返す。応答を書いて止めるときは
  // support_http.ErrResponseWritten を包むエラーを返す。
  Requirement(c *echo.Context, authn *authdomain.AuthenticationContext) (SignInRequirement, error)
  EnrollmentPolicy(c *echo.Context, applicationID string) (*appdomain.MfaEnrollmentPolicy, error)
  DeniedRedirect(c *echo.Context, code, description string) string
  // 二つの Continue は完了した認証で認可を続け、同意かコードの発行への応答を書く。
  ContinueAfterPassword(c *echo.Context, authn *authdomain.AuthenticationContext) error
  ContinueAfterSecondFactor(c *echo.Context, completed *authdomain.AuthenticationContext) error
}
type SignInRequirement struct {
  TenantDefault        bool // テナントのデフォルトを Authentication が評価する
  Allowed              bool
  StepUpRequired       bool
  TrustedDeviceAllowed bool
  ApplicationID        string
}
```

OAuth2 の続きの実装は、今の `handleLoginAPI` と `finishSecondFactor` の OAuth の分岐（クライアントの解決、Application のアクセスの判定、`AttachAuthentication`、`completeAfterAuthn`）をそのまま持つ。
コードの発行の直前に Application の判定をやり直す関門と、保留中のセッションを第二要素の画面へ戻す `completeAfterAuthn` の分岐は、OAuth2 に残す。

| 案 | 判断 |
| --- | --- |
| 続きを Authentication が定めるポートにし、OAuth2 が実装する | 採る。Authentication は OAuth2 を import しない |
| Authentication が OAuth2 の認可要求の保存先を直接読む | 採らない。認可要求の状態遷移と照合を Authentication が書き直すことになり、D2 に反する |
| 既存のセッションでの追加の認証も Authentication の公開操作にする | 採らない。作用（Cookie の発行、セッションの状態の変更）を伴い、`legacy` の公開判定では `domain` か `ports` に置けない。要求する OAuth2 がポートを定める |

### 証拠

公開状態は `unpublished` である。
触れる保証の要件は、[REQ-APPLICATION-016](../done/wi-52235-require-the-tenant-default-sign-in-policy-for-idmagics-own-consoles.md) で IdMagic 自身の画面の経路まで `//spec:covers` のテストでそろえた。
ほかの要件（REQ-APPLICATION-009、010、REQ-AUTHENTICATION-007、015、017〜020、024、026、027、036、040）も、`//spec:covers` のテストを持つ。
テストを変えずに移すことを振る舞いを保つ主張の検査にし、移した後にテナントのデフォルトの評価を呼ぶ箇所と、Application の判定を呼ぶ箇所を一つずつ無効にして、要件を引くテストが落ちることを確かめる。

## タスク

- [x] T001 [Design] 認可要求の保持と再開の境界を決め、案を比較する。
- [x] T002 [Requirement] 移す範囲の保証を要件のテストが押さえているかを確かめる。
  欠けていた IdMagic 自身の画面の経路は、wi-52235 で REQ-APPLICATION-016 とテストを足した。
- [x] T003 [App] ログアウトのセッションの終了を、OAuth2 が定めるポートの経由にする。
  OAuth2 の `LoginSessions` を Authentication の `LogoutSessions` が満たす。テストを変えずに `oauth2`、`shared/http`、`authentication` が成功し、組み立て地点の結線を外すと `/end_session` のテスト 7 件が落ちた。
- [x] T004 [App] 対話のログインの handler を Authentication へ移し、二つの継ぎ目で結ぶ。
  `authentication/signin/handlers_http` を新設し、7 操作の handler とログインの抑制、信頼済みデバイス、MFA の登録を移した。OAuth2 は `SignInContinuation` で続きを実装し、`SignInSteps` で追加の認証を委ねる。
  移した関数を直接呼ぶ内部のテスト 4 ファイル（ログインの抑制、認証の失敗の属性、失敗の集約、第二要素の種類）は、本文を変えずに同じパッケージへ移した。ほかのテストは変えていない。
- [x] T005 [Tooling] 解消した違反 ID を台帳から消す。
  `private-import:OAuth2->Authentication` の 8 件を消した。OAuth2 から Audit への唯一の import はログインの抑制が使う塩の保存先だったので、`module-cycle:OAuth2->Audit` も消えた。`check-boundary-debt-ratchet` は `private-import` を 17 件から 9 件、`module-cycle` を 57 件から 56 件と報告した。
- [x] T006 [Verify] 変更を検証する。
  テナントのデフォルトの評価を呼ぶ 4 か所と、Application の step-up の判定を一つずつ無効にすると、それぞれ要件を引くテストが落ちた。
  ログインでの Application の拒否の判定と、第二要素の後の `AttachAuthentication` は、無効にしてもテストが落ちない。拒否はコードの発行の直前の判定が同じ応答で返し、認可要求への認証の結び付けは同意とコードの発行で同じ値をもう一度結ぶので、移す前から重複している。
  `test-go-mutation` は `oauth2/handlers_http` の 13 件のうち 11 件を殺した。生存した `/end_session` の主体の照合の無効化は、ログアウトの口を通る HTTP の境界のテストがなかったので `TestEndSessionRejectsIDTokenHintForAnotherSubject`（EX-OAUTH2-024-06）を足して殺した。セッションの Cookie の `MaxAge` の符号の反転は、1 秒で消える Cookie になるだけなので等価として扱った。`signin/handlers_http` の変異はサーバーを通すテストからだけ到達するので、上の手での無効化で代えた。
  `verify` の静的な契約の検査が、続きの実装のメソッド名 `complete` を連合のログインの handler と取り違えたので、名前を変えた。

## 検証

- `mise run check-boundaries`
- `mise run check-boundary-debt-ratchet`
- `mise run verify`
- `mise run test-ui-e2e`

## リスク

- 手順の移動で、追加の認証を経ずに認可要求が再開される経路が生じると、認証の保証が破れる。
  移す前に要件のテストをそろえ、移した後に評価を呼ぶ箇所を一つずつ無効にして、テストが落ちることを確かめる。

## 要約

`mise run spec-diff` は規範の差分なしを報告する。
`/api/auth/login`、`/api/auth/totp`、`/api/auth/webauthn/challenge`、`/api/auth/webauthn`、`/api/auth/recovery-code`、`/api/auth/mfa/enrollment/totp/{start,confirm}` の実装を、TypeSpec の担当どおり Authentication の `signin/handlers_http` へ移した。パスワードの照合、ログインの抑制、第二要素、信頼済みデバイス、MFA の登録、テナントのデフォルトのポリシーの適用、`return_to` の戻りが Authentication に閉じる。
OAuth2 との継ぎ目は三つで、どれも要求する側が定めて組み立て地点で結ぶ。Authentication が定める `signin/ports.PendingAuthorizations` を OAuth2 が実装し、認可要求が求めるサインインの条件を答えて認可を続ける。OAuth2 が定める `SignInSteps` と `LoginSessions` を Authentication が実装し、既存のセッションへの追加の認証と `/end_session` のセッションの終了を受け持つ。
境界の負債の台帳から `private-import` の 8 件と `module-cycle:OAuth2->Audit` を消した。
`/end_session` の主体の照合を HTTP の境界で確かめるテストを足した。
手順の順序、各判定、HTTP の応答、セッションの状態、ドメインイベントは変えていない。
