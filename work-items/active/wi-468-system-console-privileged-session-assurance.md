---
status: pending
authors: [tn]
risk: high
reversibility: irreversible
created_at: 2026-09-03
change_kind: feature
priority: p1
depends_on:
  - wi-61037-bind-bearer-step-up-to-the-login-session
affected_spec:
  - { path: docs/modules/authentication/features/account-portal.md, requirement: REQ-AUTHENTICATION-039 }
  - { path: docs/modules/system/features/api-boundary.md, requirement: REQ-SYSTEM-020 }
  - { path: docs/modules/tenancy/features/resolution.md, requirement: REQ-TENANCY-011 }
  - { path: docs/modules/tenancy/features/lifecycle.md, requirement: REQ-TENANCY-027 }
  - { path: docs/modules/jobs/features/admin.md, requirement: REQ-JOBS-015 }
  - { path: docs/modules/data-keys/features/lifecycle.md, requirement: REQ-DATAKEYS-003 }
  - { path: docs/modules/data-keys/features/lifecycle.md, requirement: REQ-DATAKEYS-008 }
  - { path: spec/modules/authentication/main.tsp, symbol: IdMagic.Authentication.Operations.CompleteStepUpAuthentication }
  - { path: spec/modules/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.CreateTenant }
  - { path: spec/modules/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.UpdateTenant }
  - { path: spec/modules/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.UpdateTenantQuota }
  - { path: spec/modules/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.SetTenantEndpointStyle }
  - { path: spec/modules/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.DisableTenant }
  - { path: spec/modules/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.EnableTenant }
  - { path: spec/modules/jobs/main.tsp, symbol: IdMagic.Jobs.Operations.CancelSystemJob }
  - { path: spec/modules/data-keys/main.tsp, symbol: IdMagic.DataKeys.Operations.RotateTenantDataKey }
  - { path: spec/modules/data-keys/main.tsp, symbol: IdMagic.DataKeys.Operations.DisableTenantDataKey }
---

# システムコンソールに特権セッション保証と操作理由を要求する

## Motivation

制御面主体の判定（`RequireControlPlaneUser`）は、制御面テナントの有効な User と `system_admin` ロールを要求する。

しかし現在のシステムコンソールは、その User が MFA を成立させたか、認証またはステップアップが十分に新しいかを確認しない。

長時間残ったセッションを取得した者は、全テナントの監査と鍵状態を読み、テナントを停止または再公開し、正規ロケーションを変更し、別テナントのジョブを取り消せる。

既存のステップアップ判定は `max(auth_time, step_up_at)` の時刻だけを見て、成立した方式を記録しない。
パスワードだけの再認証も、MFA による再認証と区別できない。

[[wi-152-just-in-time-privileged-role-activation]] はテナント横断特権を対象外としているため、システムコンソール固有の保証が別に必要になる。

## Scope

- システムコンソールとシステム API へログインセッション（Bearer の場合は [[wi-61037-bind-bearer-step-up-to-the-login-session]] が結び付けた発行元のセッション）で到達した要求に、MFA を成立させたセッションを要求する（`privileged_read`）。
- 制御面のすべての変更操作に、MFA による直近のステップアップを要求する（`privileged_change`）。対象は `CreateTenant`、`UpdateTenant`、`UpdateTenantQuota`、`SetTenantEndpointStyle`、`DisableTenant`、`EnableTenant`、`CancelSystemJob`、`RotateTenantDataKey`、`DisableTenantDataKey` である。
- ステップアップの記録に時刻だけでなく成立した方式を保持し、パスワードだけの再認証を制御面の MFA ステップアップとして扱わない。
- 影響の大きい操作（`SetTenantEndpointStyle`、`DisableTenant`、`EnableTenant`、`CancelSystemJob`、`RotateTenantDataKey`、`DisableTenantDataKey`）に、空でない操作理由を要求する。
- 画面は対象の `tenant_id`、realm、操作、影響、入力した理由を確認画面へ表示する。
- 監査イベントへ操作者、対象テナント、操作理由、ステップアップの方式を記録する。
- `SetTenantEndpointStyle`、`DisableTenant`、`EnableTenant` の `x-api-token-scopes` を `interactive_session` に直す。テナントの制御面の経路は実行時すでに API アクセストークンを受けておらず（`newAuthenticator(d, nil)`）、宣言だけが `tenants:write` のまま残っている。旧 wi-461 の範囲であり、本項目へ統合した。
- MFA 未成立、ステップアップ期限切れ、パスワードだけのステップアップ、理由欠落による拒否で対象状態が変わらないことを確認する。

## Out of Scope

- API アクセストークンで届く制御面の要求の扱い。
  到達の可否は各操作の `x-api-token-scopes` の宣言に任せ、`privileged_read` と `privileged_change` のセッション条件を課さない（2026-11-14 に利用者と決定）。
  理由必須の操作は、API アクセストークンからでも理由を要求する。
- `CreateTenant`、`UpdateTenant`、`UpdateTenantQuota` の `x-api-token-scopes` の変更。
  宣言は `tenants:write` のままだが、実行時は API アクセストークンを受けていない。宣言と実行時の不一致の解消は、自動化の要否と合わせて別に判断する。
- `system_admin` の割当て規則。
- 管理者ロール全般の JIT 有効化。
  [[wi-152-just-in-time-privileged-role-activation]] が扱う。
- テナント管理者の操作（`DisableTenantKey` など）へ同じ保証を広げること。
- 外部 PAM またはチケットシステムとの連携。
- break-glass 資格情報の自動発行。
  本作業項目の完了前に、既存の明示的な環境 seed と復旧手順で制御面 User と MFA を回復できることを検証し、回復できなければ別の運用作業項目を起票する。

## Design

### 二段の保証

| 段階 | 対象 | 必要条件 |
| --- | --- | --- |
| `privileged_read` | システムコンソールへの入場、制御面の読出し（テナント一覧と取得、鍵と DEK のヘルス、システムの監査とジョブ、監査エクスポート） | 制御面 User、`system_admin`、ログインセッションの `amr` に MFA の方式を含む |
| `privileged_change` | Scope に挙げた制御面の変更操作 | `privileged_read`、有効期間内の MFA ステップアップ。理由必須の操作では操作理由 |

MFA とみなす方式は `otp`、`webauthn`、`hwk`、`swk`、`rc` とする（2026-11-14 に利用者と決定）。
`tdev`（信頼済みデバイス）、`pwd`、`federated`、`wia` は含めない。
復旧コードを含めるのは、端末を失った唯一の運用者が復旧できなくなるのを避けるためである。
既存の `IsMfaAMR` は `tdev` を含むので、制御面には `SecondFactorAuthenticated` と同じく `tdev` を除いた集合を使う。

### ステップアップの方式の記録

既存の `StepUpAt` だけでは成立方式を判別できない。
セッションに直近ステップアップの方式（`step_up_amr`）を保存し、`CompleteStepUp` の各方式から AMR の値（`pwd`、`otp`、`webauthn`、`rc`）へ写して記録する。

`privileged_change` の有効期間は既存の `StepUpRecencySeconds`（5 分）を使う。
直近の根拠は次のどちらかとする。

- `auth_time` が 5 分以内で、ログインの `amr` が MFA の方式を含む。
- `step_up_at` が 5 分以内で、`step_up_amr` が MFA の方式を含む。

`auth_time` が新しくても、`amr` が単一要素なら満たさない。

判定は Authentication が持つ純粋な関数にし、共有の HTTP 支援の制御面ゲートから呼ぶ。

```go
type PrivilegedAssurance string // "privileged_read" | "privileged_change"

func EvaluatePrivilegedSession(authn *AuthenticationContext, level PrivilegedAssurance, now time.Time) error
// ErrPrivilegedMFARequired、ErrStepUpRequired（MFA ステップアップが必要）を返す
```

### 拒否の応答

- `privileged_read` の不足は `403 mfa_required` とする。画面は MFA の登録またはステップアップへ案内する。
- `privileged_change` の不足は既存と同じ `403 step_up_required` とし、応答に MFA の方式だけを許すことを示す。`StepUpDialog` はパスワードを選択肢から外す。
- 理由の欠落は `400 invalid_request` とする。

### 認可の正

画面のルートガードは早い案内のために同じ条件を確認するが、認可の正は各システム API のサーバー側ゲートに置く。

操作理由を監査イベントだけへ後付けする案は採らない。
要求の時点で理由を必須入力にしなければ、誰がどの判断で影響の大きい変更を行ったかを監査から再構成できないためである。

システムコンソールを開いた時点で一度だけステップアップし、その後は無期限に信用する案も採らない。
セッション取得後の時間を制限できず、再認証の目的を満たさないためである。

### 起票後に見直した点（2026-11-14）

- 依存していた wi-460、wi-462 は完了した。wi-461 は本項目へ統合し、中止した。
- ステップアップの規範は REQ-AUTHENTICATION-039 が持つ。起票時の REQ-AUTHENTICATION-029 は信頼済みデバイスの規範だったので差し替えた。
- システム経路のジョブ取消しは `CancelSystemJob` になった。
- 起票後に増えた制御面の変更操作（`CreateTenant`、`UpdateTenant`、`UpdateTenantQuota`、`RotateTenantDataKey`、`DisableTenantDataKey`）を `privileged_change` に含めた。
- ポータルは Bearer で API を呼ぶため、セッションの記録が届かない。その前提を [[wi-61037-bind-bearer-step-up-to-the-login-session]] に切り出した。

## Plan

1. 特権読出しと特権変更の保証条件、有効期間、MFA とみなす AMR、理由必須操作を仕様で確定する。
2. MFA 未成立の `system_admin` と期限切れのセッションが、現在は制御面を利用できることを受け入れ境界で観測する。
3. ステップアップの方式を保存し、制御面の判定を実装する。
4. システム API の読出しと変更へ段階別のゲートを接続する。
5. 理由入力と対象確認を TypeSpec、ハンドラー、監査イベント、日英 UI へ通す。
6. 拒否時に対象テナント、ジョブ、DEK の状態が変わらないことを確認する。
7. MFA を失った場合の復旧手順を実機に近い構成で検証する。

## Tasks

- [ ] T001 [Spec] 特権セッション保証、有効期間、許可する MFA AMR、理由必須操作、3 操作の `interactive_session` を規範化する。
- [ ] T002 [Acceptance] MFA 未成立または期限切れのセッションが現在は制御面へ到達できることを観測し、RED を確認する。
- [ ] T003 [Domain] ステップアップの方式をセッションへ保持し、`EvaluatePrivilegedSession` の Unit RED を確認してから実装する。
- [ ] T004 [App] システム API の読出しへ `privileged_read`、変更へ `privileged_change` を適用する。
- [ ] T005 [App] 操作理由を要求モデル、ユースケース、監査イベントへ伝搬する。
- [ ] T006 [UI] MFA 案内、MFA に限るステップアップ、対象と影響の確認、理由入力を日英で実装する。
- [ ] T007 [Acceptance] 各拒否で対象状態が変わらず、正しい MFA ステップアップでは成功することを確認する。
- [ ] T008 [Operations] 制御面 User と MFA の復旧手順を検証し、手順が成立しなければ別の作業項目を起票する。
- [ ] T009 [Verify] 仕様生成物を再生成し、セキュリティ検査を通す。

## Verification

- 制御面の各操作について、HTTP 境界の受け入れテストで拒否と成功を確かめる。
- `mise run check-api-compat`：理由フィールドの追加と `interactive_session` への変更が公開契約に与える影響を確かめる。
- `mise run check-security-controls`、`mise run report-security-test-gaps`
- `mise run test-ui-e2e`：システムコンソールのステップアップと理由入力。
- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>` とする。

## Risk Notes

保証判定を画面だけへ置くと直接の API 呼出しが迂回路になるため、サーバー側の拒否を受け入れ証拠にする。

`StepUpAt` だけを確認すると、パスワードだけで更新された時刻も MFA と誤認する。
認証方式を保存して判定するテストを必須にする。

MFA を失った唯一のシステム運用者を締め出すと復旧不能になる。
展開前の復旧検証を完了条件とし、成立しない場合は本変更を完了扱いにしない。

新しい規範 ID、セッション保証の意味、公開要求の理由フィールドを割り当てるため、`reversibility` は irreversible とする。
