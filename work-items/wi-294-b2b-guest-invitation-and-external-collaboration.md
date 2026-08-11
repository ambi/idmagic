---
status: pending
authors: [tn]
risk: high
created_at: 2026-07-25
depends_on: [wi-30-inbound-federation-and-identity-broker]
change_kind: feature
initial_context:
  scl:
    IdManagement:
      - models.User
      - models.UserLifecycle
      - interfaces.CreateAdminUser
      - interfaces.ListAdminUsers
      - interfaces.DisableAdminUser
    Application:
      - interfaces.AssignApplication
      - interfaces.ListApplicationAssignments
  decisions:
    - decisions/ADR-032-tenant-as-first-class-aggregate.md
    - decisions/ADR-039-user-profile-shape.md
    - decisions/ADR-072-two-stage-user-deletion-soft-delete-and-restore.md
    - decisions/ADR-141-inbound-identity-sourcing-taxonomy.md
  source:
    - backend/idmanagement/user/domain
    - backend/idmanagement/user/usecases
    - backend/application/usecases
    - backend/authentication/usecases
  tests:
    - backend/idmanagement/user/usecases
    - backend/application/usecases
  stop_before_reading:
    - backend/saml
    - backend/wsfederation
affected_spec:
  - { path: spec/contexts/identity-management/models.tsp, symbol: IdMagic.Contract.User }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.Contract.CreateAdminUser }
  - { path: spec/contexts/application/main.tsp, symbol: IdMagic.Contract.AssignApplication }
---

# 外部協業者を招待するゲストユーザー (B2B collaboration) を導入する

## Motivation

IdMagic の User はすべて「そのテナントが資格情報を管理する内部ユーザー」である。
外部の協業者 (取引先・業務委託・グループ会社の社員) に自社アプリを使わせる手段は、
現状「内部ユーザーとして作り、こちらでパスワードを発行する」しかない。

これは実務上 2 つの問題を起こす:

1. **資格情報を二重管理することになる**。相手企業の社員のパスワードを自社 IdP が持つのは、
   相手企業の退職者管理と切り離されるため危険である。相手企業で退職した人間のアカウントが
   自社側で生き続ける。これは orphan account の典型
   ([[wi-156-orphan-account-discovery-and-reconciliation]] が下流アプリ側で扱う問題の、
   IdP 側での発生源)。
2. **内部ユーザーと外部ユーザーが区別できない**。監査で「社外の人間が何人アクセスできるか」を
   答えられない。既定権限も分けられず、内部向けグループやアプリに紛れ込みうる。

競合はこれを主要機能として持つ:

- **Entra ID**: B2B collaboration。ゲストを招待し、認証はゲストのホームテナントに委ね、
  権限だけを自テナントで管理する。ゲストには既定で制限された権限を与える。
- **Okta**: Okta Orgs / External Identities で B2B シナリオを扱う。
- **Keycloak**: Organizations 機能でメンバーとドメイン紐付け、IdP 連携を提供。

本 WI は「招待 → 受諾 → ゲストとして最小権限で存在する → 期限やアクセス失効で退場する」
というライフサイクルを導入する。認証を外部 IdP に委ねる部分は
[[wi-30-inbound-federation-and-identity-broker]] が提供するので、本 WI はその上に
「ゲストという principal 種別と招待の運用」を載せる。

## Scope

- **decision**:
  - 新規 ADR (ゲストユーザーと招待): ゲストを別 aggregate にせず `User` の種別
    (`user_kind: Member | Guest`) として表す理由、ゲストの認証経路
    (外部 IdP 委任を既定とし、ローカル資格情報を持つゲストを許すか)、ゲストの既定権限
    (グループ / ロール / アプリ割当の既定 deny)、招待の有効期限と再送、
    受諾時の JIT provisioning と [[wi-30-inbound-federation-and-identity-broker]] の
    account linking との関係、ゲストのアクセス期限 (expiration) と失効時の挙動
    (無効化か削除か。[[ADR-072-two-stage-user-deletion-soft-delete-and-restore]] と整合)、
    ゲストに見せない情報 (テナント内ディレクトリの閲覧制限) を記録する。
- **scl**:
  - `IdManagement.models.User` に `user_kind` (Member / Guest) と、ゲスト固有の
    `guest_metadata` (invited_by / invited_at / accepted_at / access_expires_at /
    home_domain) を追加する。
  - `Invitation` model (id / tenant_id / email / user_kind / inviter_id / state /
    expires_at / redeemed_at / target_assignments) と `InvitationState` enum
    (Pending / Redeemed / Expired / Revoked) を追加する。
  - `CreateInvitation` / `ListInvitations` / `GetInvitation` / `ResendInvitation` /
    `RevokeInvitation` / `RedeemInvitation` (未認証で到達する受諾エンドポイント) /
    `SetGuestAccessExpiration` interface を追加する。
  - `states` / events に InvitationCreated / InvitationRedeemed / InvitationExpired /
    InvitationRevoked / GuestAccessExpired を追加する。
  - `authorization` に「ゲストはテナント内ディレクトリ (User / Group 一覧) を参照できない」
    「ゲストは既定でどのアプリにも割当されない (fail-closed)」を明記する。
  - `objectives` は追加せず、既存の Authentication / IdManagement 目標に従う。
  - `scenarios`: 招待メールから受諾して外部 IdP でサインインできる /
    期限切れ招待が受諾できない / 失効済み招待が受諾できない /
    ゲストが割当されていないアプリにアクセスできない / ゲストがユーザー一覧を参照できない /
    アクセス期限到来でゲストが無効化される / 同一メールへの二重招待が既存招待を再利用する。
- **go**:
  - `Invitation` の domain (状態遷移・期限・トークン生成と検証) と repository
    (memory / postgres) を追加する。招待トークンはハッシュで保存し平文を持たない
    (既存の password reset token / email change token の実装に倣う)。
  - 受諾 usecase: トークン検証 → ゲスト User の作成または既存ゲストへの紐付け →
    外部 IdP へのリダイレクト (wi-30 の broker 経由) → 認証成功後に受諾完了。
  - ゲストのアクセス期限を判定する仕組みを追加する。期限到来時の無効化は
    **durable job / batch** で行い、ログインパスでも期限切れを fail-closed で拒否する
    (バッチ遅延で期限切れゲストがログインできる穴を作らない)。
  - ゲストの既定権限を fail-closed にする。ディレクトリ参照 API の認可に `user_kind` を織り込む。
  - 招待メールは [[wi-288-localized-notification-template-catalog-and-tenant-customization]] の
    テンプレートカタログを使う (未完了なら組込み既定テンプレートを追加する)。
- **http**:
  - 招待 CRUD の管理 API と、未認証で到達する受諾エンドポイントを追加する。
    受諾エンドポイントは列挙・総当たりに耐えるようトークンを十分長く取り、
    レート制限の対象にする ([[wi-27-endpoint-rate-limit-and-bot-mitigation]] と整合)。
- **ui**:
  - 管理コンソールに招待画面 (作成 / 一覧 / 再送 / 失効) を追加し、ユーザー一覧で
    Member / Guest を明示的に区別・絞り込みできるようにする。
  - ゲストのアクセス期限の表示・変更を追加する。
  - 受諾画面 (招待リンクから到達し、サインイン方法を提示する) を追加する。
- **documentation**:
  - README に招待の運用、ゲストの既定権限、アクセス期限、外部 IdP 前提を追記する。

## Out of Scope

- テナントを跨いだ相互信頼設定 (Entra の cross-tenant access settings 相当)。
  ゲストの認証は外部 IdP 連携として扱い、IdMagic 同士の特別な連携は作らない。
- セルフサービスのサインアップ。→ [[wi-87-self-service-user-registration]]
- アクセスパッケージ / 承認付きアクセス申請。→
  [[wi-214-self-service-access-request-and-approval]] /
  [[wi-213-access-certification-campaigns]] (ゲストの定期棚卸しはこちらで扱う)
- ゲストへの entitlement / SoD 適用。→ [[wi-154-entitlement-catalog-and-separation-of-duties]]
- 外部 IdP 連携そのもの。→ [[wi-30-inbound-federation-and-identity-broker]] (依存)
- ゲストのプロフィール属性スキーマ拡張。既存の属性スキーマをそのまま使う。

## Plan

- **ゲストを別 aggregate にしない**。`User` の種別として表すことで、既存の
  グループ所属・アプリ割当・監査・SCIM・ライフサイクルワークフローがそのまま効く。
  別 aggregate にすると、それら全てにゲスト用の分岐が生えて維持できなくなる。
  代償として「ゲストに見せない / 与えない」を認可側で fail-closed に書く必要があるので、
  そこを specification の authorization に明記する。
- **wi-30 に依存させる**。ゲストの本質は「認証を相手側に委ねる」ことなので、
  inbound federation が無いとゲストにローカルパスワードを持たせることになり、
  動機で挙げた問題を解決できない。依存を明示し、順序を守る。
- **期限は 2 箇所で効かせる**。バッチによる無効化だけだと、バッチ実行前の期限切れゲストが
  ログインできてしまう。認証パスでも `access_expires_at` を fail-closed で判定する。
  この二重化を scenario で固定する。
- **招待トークンは既存実装に倣う**。password reset token / email change token が
  ハッシュ保存・単回使用・期限付きの形で既にあるので、同じ構造を使い新しい方式を発明しない。
- **二重招待は既存招待の再利用にする**。同一メールへ複数の Pending 招待が並ぶと、
  どれが有効か分からず運用事故になる。同一 (tenant, email) の Pending は 1 件に制約する。
- **ディレクトリ非公開を既定にする**。ゲストがテナント内の全ユーザー一覧を引けると、
  取引先に社員名簿を渡すことになる。認可側で `user_kind == Guest` を deny 条件に入れ、
  ここを最初のテストにする。
- 未決定: ローカル資格情報を持つゲスト (外部 IdP を持たない小規模取引先向け) を許すか。
  第 1 段では**許さない** (外部 IdP 委任のみ) とし、需要があれば ADR を改訂して開く。

## Tasks

- [ ] T001 [Spec] `User` に user_kind / guest_metadata、Invitation / InvitationState、
      interface 7 件、event 5 件、authorization 規則 2 件、scenario 7 件を追加し
      `just check-scl` を通す。
- [ ] T002 [ADR] ゲストユーザーと招待の ADR を起票する (種別方式の理由・認証経路・既定権限・
      期限と失効・ディレクトリ非公開・二重招待の扱い)。
- [ ] T003 [Domain] Invitation の状態遷移、期限、トークン生成 / 検証 (ハッシュ保存・単回使用) を
      実装する。RED: 期限切れ / 失効済み / 再使用が拒否されるテストを先に書く
      (scenario `IdManagement.invitation_expired_rejected`) → GREEN。
- [ ] T004 [Persistence] `invitations` テーブル ((tenant_id, email) の Pending 一意制約) と
      `users.user_kind` / ゲストメタデータ列を `infra/schema/postgres.sql` に追加し、
      `just sqlc-generate` を実行する。RED: 既存ユーザーが Member として読めるテスト → GREEN。
- [ ] T005 [Authz] ゲストの既定 deny を認可に織り込む (ディレクトリ参照不可、
      アプリ割当なしでのアクセス不可)。RED: ゲストがユーザー一覧を引けないテストを
      先に書く → GREEN。
- [ ] T006 [Usecase] 招待作成 / 一覧 / 再送 / 失効と、受諾 (トークン検証 → ゲスト作成 →
      broker へのハンドオフ → 受諾完了) を実装する。RED → GREEN。
- [ ] T007 [Expiration] `access_expires_at` の認証パス判定と、期限到来時の無効化バッチを
      実装する。RED: バッチ未実行でも期限切れゲストがログインできないテスト → GREEN。
- [ ] T008 [HTTP] 招待管理 API と未認証の受諾エンドポイントを追加する。受諾エンドポイントを
      レート制限対象にする。RED: 不正トークンで 404 相当を返す handler テスト → GREEN。
- [ ] T009 [Email] 招待メールをテンプレートカタログ経由で送る
      ([[wi-288-localized-notification-template-catalog-and-tenant-customization]] が
      未完了なら組込み既定テンプレートを追加する)。
- [ ] T010 [UI] 招待画面 (作成 / 一覧 / 再送 / 失効)、ユーザー一覧の Member/Guest 区別と絞り込み、
      アクセス期限の表示・変更、受諾画面を追加する。RED: presentation logic の unit test → GREEN。
- [ ] T011 [Docs] README に招待運用・ゲスト既定権限・期限・外部 IdP 前提を追記する。
- [ ] T012 [Verify] 下記 Verification を緑にする。`just spec-render` を実行する。

## Verification

- `just check` / `just check-scl` / `just check-work-items` / `just check-ids`
- `just test-go` / `just test-go-race` / `just verify-go`
- `just verify-ui` / `just test-ui-unit`
- 手動: `just dev` で (1) 招待を作成しメールのリンクから受諾できること、
  (2) 受諾後のゲストが割当済みアプリだけ使えること、(3) ゲストがユーザー一覧 API を
  叩くと 403 になること、(4) アクセス期限を過去に設定するとログインできなくなること、
  (5) 失効した招待が受諾できないこと、を確認する。

## Risk Notes

**ゲストの既定権限が緩いと、社外の人間に内部情報が渡る**。これが本 WI の最大リスクである。
`user_kind` を認可の deny 条件に入れ、ディレクトリ参照不可とアプリ割当 fail-closed を
最初のテストで固定する。既存の認可経路すべてにゲスト判定が効くことを網羅する。
未認証で到達する受諾エンドポイントは総当たり対象になる。トークン長を十分に取り、
ハッシュ保存・単回使用・レート制限を組み合わせる。存在しないトークンと期限切れトークンで
応答を区別しない (列挙を防ぐ)。
期限切れゲストがバッチ遅延でログインできる穴は権限事故になるため、認証パスでの
fail-closed 判定を必須とする。
`User` に種別を足すため、既存の全ユーザーが Member として解釈されることを移行テストで固定する。
SCIM ([[wi-246-scim-multivalued-core-attributes-and-nested-group-members]] 等) や
Sourcing 経由で作られるユーザーの種別既定も明示する。
