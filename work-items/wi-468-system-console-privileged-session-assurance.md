---
status: pending
authors: [tn]
risk: high
reversibility: irreversible
created_at: 2026-09-03
change_kind: feature
priority: p1
depends_on:
  - wi-460-cross-tenant-health-control-plane-membership
  - wi-461-control-plane-credential-boundary
  - wi-462-control-plane-console-single-entry
affected_spec:
  - { path: docs/contexts/authentication/scenarios.md, requirement: REQ-AUTHENTICATION-029 }
  - { path: spec/contexts/authentication/main.tsp, symbol: IdMagic.Authentication.Operations.CompleteStepUpAuthentication }
  - { path: spec/contexts/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.SetTenantEndpointStyle }
  - { path: spec/contexts/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.DisableTenant }
  - { path: spec/contexts/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.EnableTenant }
  - { path: spec/contexts/jobs/main.tsp, symbol: IdMagic.Jobs.Operations.CancelJob }
---

# システムコンソールに特権セッション保証と操作理由を要求する

## Motivation

制御面主体の判定は、制御面テナントの有効な User と `system_admin` ロールを要求する。

しかし現在のシステムコンソールは、その User が MFA を成立させたか、認証またはステップアップが十分に新しいかを確認しない。

長時間残ったセッションを取得した者は、全テナントの監査と鍵状態を読み、テナントを停止または再公開し、正規ロケーションを変更し、別テナントのジョブを取り消せる。

[[wi-461-control-plane-credential-boundary]] は一部操作を対話セッションへ限定するが、セッションの認証強度と新しさを変更しない。

[[wi-152-just-in-time-privileged-role-activation]] もテナント横断特権を対象外としているため、システムコンソール固有の保証が別に必要になる。

## Scope

- システムコンソールとシステム API の読出しに、MFA を成立させた対話セッションを要求する。
- `SetTenantEndpointStyle`、`DisableTenant`、`EnableTenant`、システム API からの `CancelJob` に、MFA による直近のステップアップを要求する。
- ステップアップ結果に時刻だけでなく成立した認証方式を保持し、パスワードだけの再認証を制御面の MFA ステップアップとして扱わない。
- 対象テナントの停止、再開、正規ロケーション切替、別テナントのジョブ取消しに、空でない操作理由を要求する。
- 画面は対象の `tenant_id`、realm、操作、影響、入力した理由を確認画面へ表示する。
- 監査イベントへ操作者、対象テナント、操作理由、ステップアップの認証方式を記録する。
- MFA 未成立、ステップアップ期限切れ、パスワードだけのステップアップ、理由欠落による拒否で対象状態が変わらないことを確認する。
- Authentication、System、Tenancy、Jobs に特権セッション保証の規範シナリオと決定を追加する。

## Out of Scope

- `system_admin` の割当て規則。
  [[wi-463-reserve-system-admin-role-to-control-plane]] が扱う。
- 管理者ロール全般の JIT 有効化。
  [[wi-152-just-in-time-privileged-role-activation]] が扱う。
- API アクセストークンから許可する操作の分類。
  [[wi-461-control-plane-credential-boundary]] が扱う。
- 外部 PAM またはチケットシステムとの連携。
- break-glass 資格情報の自動発行。
  本作業項目の展開前に、既存の明示的な環境 seed と復旧手順で制御面 User と MFA を回復できることを検証し、回復できなければ別の運用作業項目を起票する。
- テナント内の通常管理操作へ同じ保証を広げること。

## Design

システムコンソールの保証を二段に分ける。

| 段階 | 対象 | 必要条件 |
| --- | --- | --- |
| `privileged_read` | システムコンソールへの入場、テナント横断の読出し | 制御面 User、`system_admin`、対話セッション、MFA 成立 |
| `privileged_change` | テナント停止、再開、正規ロケーション切替、別テナントのジョブ取消し | `privileged_read`、有効期間内の MFA ステップアップ、操作理由、対象確認 |

通常ログインの `auth_time` が新しくても、AMR が単一要素なら `privileged_change` を満たさない。

既存の `StepUpAt` だけでは成立方式を判別できないため、セッションに直近ステップアップの AMR または同等の保証レベルを保存し、共通の `RequirePrivilegedSession` で評価する。

画面のルートガードは早い案内のために同じ条件を確認するが、認可の正は各システム API のサーバー側ゲートに置く。

操作理由を監査イベントだけへ後付けする案は採らない。

要求の時点で理由を必須入力にしなければ、誰がどの判断で影響の大きい変更を行ったかを監査から再構成できないためである。

システムコンソールを開いた時点で一度だけステップアップし、その後は無期限に信用する案も採らない。

セッション取得後の時間を制限できず、再認証の目的を満たさないためである。

## Plan

1. 特権読出しと特権変更の保証条件、有効期間、MFA とみなす AMR を仕様で確定する。
2. MFA 未成立の `system_admin` と期限切れセッションが現在はシステム API を利用できることを受け入れ境界で観測する。
3. ステップアップ結果へ認証方式を保存し、制御面用の共通保証判定を実装する。
4. システム API の読出しと対象変更操作へ段階別のゲートを接続する。
5. 理由入力と対象確認を TypeSpec、ハンドラー、監査イベント、日英 UI へ通す。
6. 拒否時に対象テナントまたはジョブの状態が変わらないことを確認する。
7. MFA を失った場合の復旧手順を実機に近い構成で検証する。

## Tasks

- [ ] T001 [Spec] 特権セッション保証、有効期間、許可する MFA AMR、理由必須操作を規範化する。
- [ ] T002 [Acceptance] MFA 未成立または期限切れのセッションが現在は制御面へ到達できることを観測し、RED を確認する。
- [ ] T003 [Domain] ステップアップの成立方式をセッションへ保持し、`RequirePrivilegedSession` の Unit RED を確認してから実装する。
- [ ] T004 [App] システム API の読出しへ `privileged_read`、対象変更へ `privileged_change` を適用する。
- [ ] T005 [App] 操作理由を要求モデル、ユースケース、監査イベントへ伝搬する。
- [ ] T006 [UI] MFA 案内、ステップアップ、対象と影響の確認、理由入力を日英で実装する。
- [ ] T007 [Acceptance] 各拒否で対象状態が変わらず、正しい MFA ステップアップでは成功することを確認する。
- [ ] T008 [Operations] 制御面 User と MFA の復旧手順を検証し、手順が成立しなければ別の作業項目を起票する。
- [ ] T009 [Verify] 仕様生成物を再生成し、セキュリティ検査を通す。

## Verification

- `mise run test-go-race`
- `mise run test-ui-unit`
- `mise run test-ui-e2e`
- `mise run check-security-controls`
- `mise run report-security-test-gaps`
- `mise run check-api-compat`
- `mise run check-spec`
- `mise run check-work-items`
- `mise run verify`

## Risk Notes

保証判定を画面だけへ置くと直接の API 呼出しが迂回路になるため、サーバー側の拒否を受け入れ証拠にする。

`StepUpAt` だけを確認すると、パスワードだけで更新された時刻も MFA と誤認する。

認証方式を保存して判定するテストを必須にする。

MFA を失った唯一のシステム運用者を締め出すと復旧不能になる。

展開前の復旧検証を完了条件とし、成立しない場合は本変更を完了扱いにしない。

新しい規範 ID、セッション保証の意味、公開要求の理由フィールドを割り当てるため、`reversibility` は irreversible とする。
