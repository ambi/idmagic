---
status: pending
authors: [tn]
risk: high
reversibility: irreversible
created_at: 2026-09-03
change_kind: feature
priority: p1
depends_on:
  - wi-52890-step-up-bearer-requests-with-rfc-9470
  - wi-23765-enforce-requested-mfa-on-reauthorization
  - wi-93893-verify-control-plane-mfa-recovery
affected_spec:
  - { path: docs/modules/system/features/api-boundary.md, requirement: REQ-SYSTEM-020 }
  - { path: docs/modules/tenancy/features/resolution.md, requirement: REQ-TENANCY-011 }
  - { path: docs/modules/tenancy/features/lifecycle.md, requirement: REQ-TENANCY-027 }
  - { path: spec/modules/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.SetTenantEndpointStyle }
  - { path: spec/modules/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.DisableTenant }
  - { path: spec/modules/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.EnableTenant }
---

# システムコンソールとシステム API に MFA の認証と直近の MFA 再認証を求める

## Motivation

制御面主体の判定（`RequireControlPlaneUser`）は、制御面テナントの有効な User と `system_admin` ロールを要求する。
しかし、その User が MFA で認証したか、認証が十分に新しいかを確認しない。
長時間残ったセッションやトークンを取得した者は、全テナントの監査と鍵状態を読み、テナントを停止または再公開し、正規ロケーションを変更し、別テナントのジョブを取り消せる。

[[wi-152-just-in-time-privileged-role-activation]] はテナント横断特権を対象外としているため、システムコンソール固有の保証が別に必要になる。

## Scope

- システムコンソールとシステム API へ利用者の認証で到達した要求（ポータルの Bearer のアクセストークン、またはセッションの Cookie）に、MFA で成立した認証を要求する（`privileged_read`）。
- 制御面のすべての変更操作に、5 分以内の MFA の認証を要求する（`privileged_change`）。対象は `CreateTenant`、`UpdateTenant`、`UpdateTenantQuota`、`SetTenantEndpointStyle`、`DisableTenant`、`EnableTenant`、`CancelSystemJob`、`RotateTenantDataKey`、`DisableTenantDataKey` である。
- 判定に認証の時刻だけでなく方式（`amr`）を使い、パスワードだけの再認証と記憶済みの端末を制御面の MFA として扱わない。
- 不足は RFC 9470 の応答で返し、システムコンソールは MFA の水準を求める再認可で解消する。第二要素を持たない運用者には MFA の登録を案内する。
- `SetTenantEndpointStyle`、`DisableTenant`、`EnableTenant` の `x-api-token-scopes` を `interactive_session` に直す。テナントの制御面の経路は実行時すでに API アクセストークンを受けておらず（`newAuthenticator(d, nil)`）、宣言だけが `tenants:write` のまま残っている。旧 wi-461 の範囲であり、本項目へ統合した。
- MFA 未成立、期限切れ、パスワードだけ、記憶済みの端末だけの認証による拒否で、対象の状態が変わらないことを確認する。

## Out of Scope

- 影響の大きい操作の理由、確認画面、監査イベントの理由と方式。[[wi-47234-require-reasons-for-control-plane-changes]] が扱う（2026-11-14 に利用者と決定して切り出した）。
- MFA を失った運用者の復旧の検証。[[wi-93893-verify-control-plane-mfa-recovery]] が扱い、本項目はその完了に依存する。
- API アクセストークンで届く制御面の要求の扱い。
  到達の可否は各操作の `x-api-token-scopes` の宣言に任せ、`privileged_read` と `privileged_change` の条件を課さない（2026-11-14 に利用者と決定）。
- `CreateTenant`、`UpdateTenant`、`UpdateTenantQuota` の `x-api-token-scopes` の変更。
  宣言は `tenants:write` のままだが、実行時は API アクセストークンを受けていない。宣言と実行時の不一致の解消は、自動化の要否と合わせて別に判断する。
- `system_admin` の割当て規則。
- 管理者ロール全般の JIT 有効化。[[wi-152-just-in-time-privileged-role-activation]] が扱う。
- テナント管理者の操作（`DisableTenantKey` など）へ同じ保証を広げること。
- 外部 PAM、break-glass の資格情報の自動発行。

## Design

### 二段の保証

| 段階 | 対象 | 必要条件 |
| --- | --- | --- |
| `privileged_read` | システムコンソールへの入場、制御面の読出し（テナント一覧と取得、鍵と DEK のヘルス、システムの監査とジョブ、監査エクスポート） | 制御面 User、`system_admin`、認証の `amr` に MFA の方式を含む |
| `privileged_change` | Scope に挙げた制御面の変更操作 | `privileged_read` に加え、認証の `auth_time` が 5 分以内 |

MFA とみなす方式は `otp`、`webauthn`、`hwk`、`swk`、`rc` とする（2026-11-14 に利用者と決定）。
`tdev`（信頼済みデバイス）、`pwd`、`federated`、`wia` は含めない。
復旧コードを含めるのは、端末を失った唯一の運用者が復旧できなくなるのを避けるためである。
既存の `IsMfaAMR` は `tdev` を含むので、制御面には `SecondFactorAuthenticated` と同じく `tdev` を除いた集合を使う。

### 判定の根拠

Bearer の要求はアクセストークンの `auth_time`、`amr`、`acr` を認証の結果として持ち（[[wi-52890-step-up-bearer-requests-with-rfc-9470]]）、リフレッシュ後も元の認証の値を保つ（[[wi-61037-carry-the-original-authentication-into-refreshed-tokens]]）。
Cookie の要求は、ログインセッションの `auth_time` と `amr` を同じ規則で読む。
`privileged_change` の新しさは `auth_time` だけで判定し、セッションの `step_up_at` は使わない。`step_up_at` は方式を記録しないので、パスワードだけの再認証と区別できない。

判定は Authentication が持つ純粋な関数にし、共有の HTTP 支援の制御面ゲートから呼ぶ。

```go
type PrivilegedAssurance string // "privileged_read" | "privileged_change"

func EvaluatePrivilegedAuthentication(authn *AuthenticationContext, level PrivilegedAssurance, now time.Time) error
// ErrPrivilegedMFARequired、ErrPrivilegedReauthenticationRequired を返す
```

### 拒否の応答

Bearer の要求には、RFC 9470 の `401` と `WWW-Authenticate: Bearer error="insufficient_user_authentication"` を返す。

- `privileged_read` の不足には `acr_values="urn:idmagic:acr:mfa"` を付ける。
- `privileged_change` の不足には `acr_values="urn:idmagic:acr:mfa"` と `max_age="300"` を付ける。

Cookie の要求には、[[wi-52890-step-up-bearer-requests-with-rfc-9470]] の Cookie の経路と同じく `403 step_up_required` を返す。
ポータルは管理 API を Bearer で呼ぶので、Cookie で制御面へ届くのは緊急のログイン経路だけである。

### 再認可の流れ

既存のポップアップの再認可が、応答の `acr_values` と `max_age` をそのまま `/authorize` へ渡す。
`/authorize` は MFA の水準を満たすまで認可コードを発行せず、再認証の要求では記憶済みの端末を照合しない（[[wi-23765-enforce-requested-mfa-on-reauthorization]]）。
第二要素を一つも登録していない運用者には `unmet_authentication_requirements` が返るので、システムコンソールはこれを MFA の登録の案内に変える。
画面の読み込みで `privileged_read` を求められた場合は、利用者の操作がないのでポップアップを開けず、画面全体で再認可する。

### 解決した問い（2026-11-14 に利用者と決定）

- 再認可で要求する `acr_values`：既存の `urn:idmagic:acr:mfa` のまま要求する。新しい `acr` の値は定義しない。制御面の判定は `amr` から `tdev` を除く規則を、二重の防御として残す。
- 将来、特権の変更操作をフィッシング耐性のある方式に限る段階は、[[wi-63330-define-built-in-assurance-levels]] の組み込みの水準で足せる。今回は TOTP、WebAuthn 系、復旧コードを認める決定を変えない。

### 認可の正

画面のルートガードは早い案内のために同じ条件を確認するが、認可の正は各システム API のサーバー側ゲートに置く。

システムコンソールを開いた時点で一度だけステップアップし、その後は無期限に信用する案は採らない。
セッション取得後の時間を制限できず、再認証の目的を満たさないためである。

### 起票後に見直した点（2026-11-14）

- 依存していた wi-460、wi-462 は完了した。wi-461 は本項目へ統合し、中止した。
- システム経路のジョブ取消しは `CancelSystemJob` になった。
- 起票後に増えた制御面の変更操作（`CreateTenant`、`UpdateTenant`、`UpdateTenantQuota`、`RotateTenantDataKey`、`DisableTenantDataKey`）を `privileged_change` に含めた。
- ポータルは Bearer で API を呼ぶため、判定の根拠を RFC 9470 とトークンのクレームに置き、前提を wi-61037、wi-52890、wi-23765 に切り出した。セッションへステップアップの方式を記録する当初の設計はやめた。
- 操作理由と監査を wi-47234 に、復旧の検証を wi-93893 に切り出した。

## Plan

1. 特権読出しと特権変更の条件、有効期間、MFA とみなす AMR を規範に書き、制御面の操作の 401 を TypeSpec に宣言する。
2. MFA 未成立や古い認証の `system_admin` が、現在は制御面を利用できることを受け入れ境界で観測する。
3. 判定の関数を単体テストから実装し、システム API のゲートへつなぐ。
4. システムコンソールの再認可と MFA の登録の案内を実装する。

## Tasks

- [ ] T001 [Spec] 特権読出しと特権変更の条件、有効期間、MFA とみなす AMR、3 操作の `interactive_session` を規範化し、制御面の操作の 401 を宣言する。
- [ ] T002 [Acceptance] MFA 未成立、古い認証、パスワードだけ、記憶済みの端末だけの認証が、現在は制御面へ到達できることを観測し、RED にする。
- [ ] T003 [Domain] `EvaluatePrivilegedAuthentication` の Unit RED を確認してから実装する。
- [ ] T004 [App] システム API の読出しへ `privileged_read`、変更へ `privileged_change` を適用する。
- [ ] T005 [UI] システムコンソールで、MFA を求める再認可と MFA の登録の案内を日英で実装する。
- [ ] T006 [Acceptance] 各拒否で対象の状態が変わらず、MFA の認証では成功することを確かめる。
- [ ] T007 [Verify] 仕様生成物を再生成し、セキュリティ検査を通す。

## Verification

- 制御面の各操作について、HTTP 境界の受け入れテストで拒否と成功を確かめる。
- `mise run check-published-api-compat`：401 の宣言と `interactive_session` への変更の影響を確かめる。
- `mise run check-security-controls`、`mise run report-security-test-gaps`
- `mise run test-ui-e2e-file`：システムコンソールの再認可。
- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>` とする。

## Risk Notes

保証判定を画面だけへ置くと直接の API 呼出しが迂回路になるため、サーバー側の拒否を受け入れ証拠にする。

認証の時刻だけを確認すると、パスワードだけの再認証も MFA と誤認する。
`amr` で判定するテストを必須にする。

MFA を失った唯一のシステム運用者を締め出すと復旧不能になる。
[[wi-93893-verify-control-plane-mfa-recovery]] の完了を本項目の前提にした。

新しい規範 ID と、制御面の操作の応答の意味を変えるため、`reversibility` は irreversible とする。
