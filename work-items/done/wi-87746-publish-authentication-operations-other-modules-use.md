---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-10
priority: p2
depends_on: []
change_kind: refactor
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  reason: 判定と操作を置くパッケージだけを変えるので、認証の判定、HTTP の応答、発行するイベントは変わらず、リリースの読者へ知らせることがない。
  references: []
initial_context:
  specification: []
  typespec: []
  source:
    - backend/authentication/domain/amr.go
    - backend/authentication/domain/acr.go
    - backend/authentication/mfa/usecases/step_up.go
    - backend/authentication/password/usecases/password_policy.go
    - backend/authentication/password/usecases/password_expiry.go
    - backend/authentication/session/usecases/session_manager.go
    - backend/authentication/session/usecases/sessions.go
    - backend/authentication/trusteddevice/usecases/trusted_devices.go
    - backend/application/usecases/sign_in_policy.go
    - backend/idmanagement/user/usecases/admin_users.go
    - backend/idmanagement/user/handlers_http/account_handler.go
    - backend/idmanagement/deps_http/deps.go
    - backend/saml/handlers_http/routes.go
    - backend/wsfederation/handlers_http/routes.go
    - backend/tenancy/handlers_http/admin_tenant_handler.go
    - backend/shared/http/server_http/routes.go
    - tools/check/boundary-debt.json
  tests: []
  stop_before_reading: [frontend, spec, backend/oauth2/handlers_http]
spec_impact: { kind: none, reason: "ほかのモジュールが使うセッション、パスワードの方針、step-up、信頼済みデバイス、ACR の判定を、Authentication の公開パッケージの経由へ変えるだけである。認証の判定、HTTP の応答、セッションの状態、ドメインイベントは変えない。" }
---

# ほかのモジュールが使う Authentication の判定と操作を公開する

## 動機

`tools/check/boundary-debt.json` の `private-import` のうち 10 件は、Application、IdManagement、Saml、WsFederation、Tenancy から Authentication の非公開パッケージへの依存である。

| 利用側 | 使っているもの |
| --- | --- |
| Saml、WsFederation、IdManagement の handler | セッションの解決（`SessionManager`、`SessionCookie`、`ErrSessionNotFound`） |
| IdManagement の usecases と handler、Tenancy の handler | パスワードの方針（`ValidatePasswordWith`、`ResolveTenantPolicy`、`PasswordPolicyError`、方針の上下限） |
| IdManagement の handler | step-up の判定（`StepUpSatisfied`、`ErrStepUpRequired`） |
| IdManagement の usecases、Application の usecases | 信頼済みデバイスの失効（`RevokeAllForUser`）と AMR の値 |
| Application の usecases | ACR と AMR の判定（`ACRSatisfies`、`ACRMFA`、`IsMfaAMR`） |

Authentication と IdManagement の共変更は 4 回で、共変更の組の上位にある。

## 対象範囲

- 上の表の判定と操作を、Authentication の公開パッケージとして公開し、利用側を書き換える。
- 解消した違反 ID を台帳から消す。

## 対象外

- 認可エンドポイントの対話の手順（`oauth2/handlers_http` からの依存）。[認可の対話の手順の移動](../active/wi-93464-move-authorization-login-steps-into-authentication.md)で扱う。
- 認証の判定の変更と、Authentication の `internal/` への移動。

## 設計

[境界の負債の順位付け](wi-33994-reinventory-the-remaining-boundary-debt.md)の判断を引き継ぐ。

- D1：セッション、パスワードの方針、step-up、信頼済みデバイス、ACR の規則の所有者は Authentication である。
- D3：利用側が使うのは判定の結果と少数の操作だけである。保存先、セッションの内部状態、方針の解決の途中の値は公開しない。

| 案 | 判断 |
| --- | --- |
| 判定と操作を公開パッケージにする | 採る |
| 方針の上下限を Tenancy へ写す | 採らない。同じ規則を二か所で保つことになる |
| `SessionManager` をそのまま公開する | 採らない。セッションの発行と失効まで利用側が呼べるようになる。利用側が必要とするのは解決と cookie の名前だけである |

Tenancy の handler からの依存は、Tenancy と Authentication の循環の辺でもある。
公開パッケージへの依存に変えても循環は残るので、辺の向きは[依存の向きの決定](../active/wi-21934-decide-module-dependency-order-and-break-cycles.md)で決める。

### 置き場所の決定

Authentication の各機能には `domain` と `ports` があり、`legacy` の公開方式では公開パッケージになる。
利用側が使うものの多くは、入力だけで結果が決まる判定と、定数、型、センチネルエラーである。

| 利用側が使うもの | 移した後 |
| --- | --- |
| `IsMfaAMR`、`AMRTrustedDevice`（Application） | すでに `authentication/domain` にあるので、そちらを参照する |
| `ACRSatisfies`、`ACRMFA`、`ACRPassword`、`DeriveACR`（Application ほか） | `authentication/domain` へ移す |
| `StepUpSatisfied`、`ErrStepUpRequired`（IdManagement） | `mfa/domain` へ移す |
| `ErrSessionNotFound`、`SessionCookie`（IdManagement、SAML、WS-Federation） | `session/domain` へ移す |
| `ValidatePasswordWith`、`PasswordPolicyError`、方針の上下限の定数、既定の方針（IdManagement、Tenancy） | `password/domain` へ移す。テナントからの方針の解決 `ResolvePolicyForTenant` も `password/domain` の `PolicyForTenant` にし、IdManagement は文脈のテナントを渡す |
| `SessionManager.Revoke`（SAML、WS-Federation） | 利用側がセッションの失効だけを求めるポート `SessionRevoker` を定義し、組み立て地点が `SessionManager` を渡す（D4）。`SessionManager` 全体は公開しない |
| `RevokeAllForUser`（IdManagement の無効化） | `trusteddevice/ports` に「利用者の信頼済みデバイスをすべて失効し、発行すべき `TrustedDeviceRevoked` を返す」契約 `UserDeviceRevoker` を置き、`trusteddevice/usecases` が実装し、組み立て地点が渡す。イベントを返すのは、IdManagement が無効化と同じトランザクションに結びついた発行先で発行するためである。イベントの組み立ては `trusteddevice/domain` に置き、Authentication の内部の失効と共有する |

定数、型、センチネルエラーは、移した先を定義とし、`usecases` には同じ名前の別名を残す。
`trusteddevice/usecases` の `AMRTrustedDevice` と `password/usecases` の `PasswordPolicySnapshot` が先例である。
Authentication の内部の参照を書き換えずに済み、センチネルエラーは同じ値なので `errors.Is` の判定も変わらない。
関数は移した先だけに置き、転送するだけの関数は残さない。

### 証拠

テストの参照を書き換えるので、振る舞いを保つ変更には当たらない。

| 証拠 | 内容 |
| --- | --- |
| 受け入れ RED | N/A: 製品の振る舞いを変えない配置の変更であり、対応する REQ がない。代替として、台帳から本項目が扱う 10 件を先に消し、`mise run check-boundaries` が未記録の違反として失敗することを確かめる |
| 単体 RED | N/A: 振る舞いを変えないので、実装前に失敗する単体テストはない。新しく置く `UserDeviceRevoker` の実装は、失効した端末ごとのイベントを返すことを確かめるテストを置く |
| 変更耐性 | 移した判定の各パッケージと `trusteddevice/usecases` に `mise run test-go-mutation` を走らせ、生き残りを読む |

## タスク

- [x] T001 [Design] 公開する判定と操作の型を決める。
  - 設計の「置き場所の決定」に記録した
- [x] T002 [App] 利用側を書き換える。
  - ACR の語彙と判定を `authentication/domain`、step-up の判定を `mfa/domain`、セッションの cookie の名前とセンチネルエラーを `session/domain`、パスワードの方針の判定と既定値を `password/domain` へ移した。`usecases` には定数、型、センチネルエラーの別名だけを残した
  - step-up の判定は、時刻がゼロなら現在時刻で補う分岐を消した。呼び出し元はすべて現在時刻を渡している
  - SAML と WS-Federation は `SessionRevoker` のポートでセッションを失効させる。nil の `*SessionManager` をインターフェースへ入れると nil の確認をすり抜けるので、組み立て地点は `SessionManager` があるときだけ渡す
  - 信頼済みデバイスの一括失効は `trusteddevice/ports.UserDeviceRevoker` で行い、Authentication のルートの `Module.UserDeviceRevoker()` が保存先のあるときだけ実装を返す。HTTP の組み立て地点と worker の組み立ての両方がこれを使う
  - 移した判定を確かめるテスト（`acr_test.go`、`password_policy_test.go`、`TestStepUpSatisfiedRecencyWindow`）を、判定と同じ `domain` のパッケージへ移した。`TestUserDeviceRevokerReturnsOneRevokedEventPerDevice` を足した
  - 検査: `mise run lint-go`、`mise run test-go-changed`
- [x] T003 [Tooling] 解消した違反 ID を台帳から消す。
  - 本項目の 10 件に加え、ACR の判定を移したことで OAuth2 の handler から `authentication/usecases` への 1 件も解消したので消した。台帳は 117 件から 106 件になった
  - 検査: `mise run check-boundaries`、`mise run check-boundary-debt-ratchet -- main`
- [x] T004 [Verify] 変更を検証する。

## 検証

- `mise run check-boundaries`
- `mise run check-boundary-debt-ratchet`
- `mise run verify`

## リスク

- step-up とセッションの解決は認証の保証に関わる。
  書き換える前に、利用側の handler の拒否の振る舞いが `//spec:covers` のテストで固定されているかを確かめ、固定されていなければ特性化テストを先に置く。

## 完了

- **完了日**: 2026-10-11
- **要約**:
  `mise run spec-diff` は規範の差分なしを報告する。
  ほかのモジュールが使う Authentication の判定を、入力だけで決まる計算として各機能の `domain` へ移した（ACR の語彙と判定、step-up の判定、セッションの cookie の名前とセンチネルエラー、パスワードの方針の判定と既定値）。`usecases` には定数、型、センチネルエラーの別名だけを残し、Authentication の内部の参照と `errors.Is` の判定は変えていない。
  SAML と WS-Federation は、自分で定義した `SessionRevoker` のポートでログアウトのセッションを失効させる。
  IdManagement は、`trusteddevice/ports.UserDeviceRevoker` で無効化に伴う信頼済みデバイスの失効を行い、返されたイベントを自分の発行先で発行する。
  境界の負債の台帳から、本項目の 10 件と、ACR の判定の移動で解消した OAuth2 の 1 件を消し、台帳は 117 件から 106 件になった。
  認証の判定、HTTP の応答、セッションの状態、発行するイベントは変えていない。
- **受け入れ RED の証拠**:
  - **テスト**: `mise run check-boundaries`
  - **要件**: N/A: 規範上の振る舞いを変えない配置の変更であり、対応する REQ がない。
  - **観測した失敗**: 台帳から本項目の 5 項目を先に消すと、`mise run check-boundaries` が 10 件を `violation(s) are absent from boundary-debt.json` として失敗した。移した後は通り、OAuth2 の 1 件は `no observed violation has this id` として消すよう求められた。
  - **検出できる理由**: 配置を変える作業なので、製品の振る舞いの RED は存在しない。代わりに、リファクタリングを必要にした構造ゲートが変更の前後の違いを観測した。
- **単体 RED の証拠**:
  - **テスト**: `TestUserDeviceRevokerReturnsOneRevokedEventPerDevice`、移したテスト（`acr_test.go`、`password_policy_test.go`、`TestStepUpSatisfiedRecencyWindow`）
  - **要件**: N/A: 規範上の振る舞いを変えない配置の変更であり、対応する REQ がない。
  - **観測した失敗**: 該当なし。振る舞いを変えないので、実装前に失敗する単体テストはない。移したテストは表明を変えずに通り、無効化で端末を失効させるテスト（`TestSetUserDisabledRevokesTrustedDevices`）は、組み立てに失効の契約を渡す前に「無効化の後も端末が有効」で失敗し、渡した後に通った。
  - **検出できる理由**: リスクの節が求めた拒否の振る舞いは、step-up の拒否を引く `//spec:covers` のテスト（`REQ-AUTHENTICATION-003`、`REQ-AUTHENTICATION-029`、`REQ-OAUTH2-043` など）が固定しており、変更後もすべて通った。利用側の変更は修飾子と依存の型だけで、判定の中身は変えていない（step-up の判定から消した時刻の補完は、呼び出し元がすべて現在時刻を渡すので本番から届かない）。
- **変更耐性の結果**:
  `mise run test-go-mutation` の結果は、`authentication/domain` が 6 件中 5 件、`mfa/domain` が 14 件中 12 件、`password/domain` が 25 件中 23 件、`trusteddevice/usecases` が 29 件中 27 件を検出した。
  移したものの生き残りは、step-up の判定の `recent <= 0` を `< 0` にする 1 件だけで、`recent` が 0 のときどちらも「満たさない」になる結果の変わらない変異である。ほかの生き残りは今回変えていない行（`DeviceLabel`、`ResolvePasswordPolicy`、`Evaluate`、`RevokeOne`、MFA の登録の免除）にあり、分類しない。
  判定のテストを `usecases` に残したままでは、移した `StepUpSatisfied` と `ValidatePasswordWith` への変異が `NOT COVERED` だったので、テストを判定と同じパッケージへ移した。
- **検証結果**:
  - `mise run check-boundaries` - 成功
  - `mise run check-boundary-debt-ratchet -- main` - 成功（private-import 37 → 26）
  - `mise run lint-go` - 成功
  - `mise run test-go-changed` - 成功
  - `mise run verify` - 成功
  - `mise run test-ui-e2e` - 成功（39 件）
