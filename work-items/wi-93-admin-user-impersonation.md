---
depends_on: [wi-43-account-portal-step-up-auth, wi-44-authentication-event-store-and-search]
status: pending
authors: ["tn"]
risk: high
created_at: 2026-07-03
---

# 管理者による代理ログイン (impersonation) を監査付きで導入する

## Motivation
サポート業務では「ユーザとして画面を見て操作する」代理ログインが、再現困難な
不具合や設定問題の調査に有効。代表的な IdP も提供する:

- Keycloak: user impersonation。
- Okta / Entra: 管理者による代理アクセス。

一方 impersonation は権限昇格・なりすまし・監査欠落のリスクが大きい。本 WI は、
専用権限・対象と時間の境界・厳格な監査 (actor chain、ADR-049 と整合)・
impersonation 中の可視バナー・機微操作の禁止を備えた形で導入する。

## Scope
- **decision**:
  - 新規 ADR: impersonation の許可条件 (専用権限 / 対象制限 / 時間制限)、監査 (開始・終了・操作を impersonator を含む actor chain で記録、 ADR-049 と整合)、可視バナーによる不可視ななりすまし防止、機微操作 (パスワード / MFA 変更 / 削除) の禁止範囲、テナント設定での有効化を記録する。
- **scl**:
  - §3.3 interfaces: StartImpersonation / EndImpersonation を追加する。
  - §3.7 permissions: impersonation は専用権限に固定し fail-closed とする。
  - §3.4 states/events: ImpersonationStarted / ImpersonationEnded を追加し、 session に impersonator を保持する。
  - §3.5 invariants: 全操作が impersonator を含めて監査され、cross-tenant impersonation を禁止することを明示する。
- **go**:
  - impersonation usecase を追加し、session に actor + impersonator を持たせ、 機微操作をブロックし、全操作を監査する。
- **http**:
  - admin からの開始 / 終了と、impersonation バナー用 context を追加する。
- **ui**:
  - AdminUsers 詳細に「代理ログイン」、全画面に impersonation 中バナーを 追加する。
- **documentation**:
  - README に impersonation の権限・制約・監査を追記する。

## Out of Scope
- 恒久的な delegated access ([[wi-94-delegated-administration]] とは別)。
- cross-tenant impersonation。
- impersonation 中のパスワード / MFA 変更・アカウント削除。

## Plan
- 既に authentication SCL/events に `ImpersonationStarted/Ended` があるため語彙を再利用し、通常admin sessionとは別の `ImpersonationSession` を発行する。元actor sessionを保持し、subject sessionへroleをコピーしない。
- 開始は専用permission、step-up、ticket/reason、短いTTLを必須にし、self、system_admin、同等以上privilege、別tenant、disabled/deleted userを拒否する。対象userのeffective roleをそのまま上限とする。
- portal OIDC token/sessionに `act`/impersonator claimとsession typeを載せ、全backend authorization/auditがactorとsubjectを分離して受け取る。token exchangeのdelegation/impersonationとは操作目的が異なるため混同しない。
- frontendは全画面固定banner、対象/actor、残り時間、終了操作を表示し、危険操作（role変更、impersonation開始、secret/key管理等）はimpersonated sessionから禁止する。
- start/end/expiryと実行操作のauditはADR-046に従い短縮・PII masking対象外の必須証跡とし、本人へのsecurity notification（wi-90）を接続する。

## Tasks
- [ ] T001 [SCL] 既存eventsを核にsession model、Start/End interfaces、permission/prohibited-actions/invariants/scenariosを追加して再生成する。
- [ ] T002 [Domain/Persistence] ImpersonationSession、actor/subject/original session、TTL/end reasonとrepositoryを実装する。
- [ ] T003 [Usecases] step-up/target privilege/reasonを検証するstart、idempotent end/expire、notification/auditを実装する。
- [ ] T004 [OIDC/Authz] portal session/tokenのactor claim解決と、shared policy contextのactor/subject分離、禁止action gateを実装する。
- [ ] T005 [Admin UI] user detailの開始dialog、全route固定banner/timeout、元sessionへ戻る終了導線を追加する。
- [ ] T006 [Verify] self/system-admin/cross-tenant拒否、role変化中session、token refresh、expiry/end replay、危険操作、audit/notificationをE2E検証する。

## Verification
- `just test-go`
- `just lint-go`
- `just build-go`
- `just typecheck-ui`
- `just lint-ui`
- `just build-ui`
- 手動: 専用権限を持つ admin が代理ログイン → バナーが出て対象ユーザ画面を閲覧 できる。機微操作が禁止され、開始 / 終了 / 操作が監査に impersonator 付きで 残ることを確認する。
- 手動: 権限が無い admin / cross-tenant では impersonation が拒否されることを確認する。

## Risk Notes
impersonation は本質的に権限昇格・なりすましであり、監査欠落や機微操作の許可は
重大インシデントに直結する。専用権限・fail-closed・全操作監査・機微操作禁止・
cross-tenant 禁止・可視バナーを必須ガードとしてテストで担保する。
