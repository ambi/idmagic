---
depends_on: []
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-08-30
priority: p0
change_kind: feature
affected_spec:
  - { path: docs/contexts/authentication/scenarios.md, requirement: REQ-AUTHENTICATION-016 }
  - { path: docs/contexts/identity-management/scenarios.md, requirement: REQ-IDMANAGEMENT-017 }
---

# パスワード再設定とメール変更に型付きアクショントークンの共通核を導入する

## Motivation

パスワード再設定とメール変更は、乱数トークンの発行、ハッシュだけの保存、有効期限、単回消費、通知、監査という同じ安全性条件を、別々のストアとユースケースで実装している。現在の主要ユースケースは動作しているが、新しい招待や本人確認リンクを追加するときに、目的の束縛、原子的な単回消費、先読み安全性、監査のいずれかを実装し忘れても共通境界が検出しない。

Keycloak のアクショントークンから採用するのは汎用 handler SPI ではなく、用途を閉じた型で識別し、共通の検証を通過した後だけ用途別作用を実行する境界である。用途別 payload と業務作用は所有 Context に残し、第三者コードや実行時登録は受け入れない。

## Scope

- `REQ-AUTHENTICATION-016` と `REQ-IDMANAGEMENT-017` に、目的の一致、期限、単回使用、先読みで作用しないこと、消費と対象変更の失敗時整合性を追加する。
- `ActionTokenPurpose`、`ActionTokenEnvelope`、`ActionTokenDigest` と、発行および検証の決定的な計算を、Authentication と Identity Management が依存できる小さな共有セキュリティ module として定義する。
- `IssueActionToken(purpose, subject, payload, now, ttl, random)` と `VerifyActionToken(raw, expectedPurpose, stored, now)` を共有 module の interface にし、時刻と乱数を明示的な入力として扱う。保存、通知、監査、用途別作用はこの interface に露出させない。
- 各 Context の永続化アダプターは、未使用確認、使用済み化、用途別作用を同じトランザクションで確定する。作用が失敗した場合はトークンを未使用のまま残し、同じトークンで作用が二回成功する状態を作らない。
- パスワード再設定とメール変更を共通核へ移し、既存のエラー秘匿、通知のローカライズ、パスワードポリシー、メール一意性を維持する。
- ブラウザーまたはメールスキャナーによる `HEAD`、リンクプレビュー、同じ URL の先読みがトークンを消費せず、対象の状態も変更しない E2E を追加する。

## Out of Scope

- 任意の JavaScript、Go plugin、handler を実行時に登録する SPI。
- トークンだけを根拠に任意の必須操作、管理権限、セッションを追加すること。
- OAuth2 アクセストークン、API トークン、ログインセッションの統合。
- 招待など、まだ仕様化されていない新しい用途の実装。

## Design

`ActionTokenPurpose` はコードで列挙した閉じた値とし、少なくとも `password_reset` と `email_change` を持つ。`ActionTokenEnvelope` は `id`、`purpose`、`subject`、`payload`、`issued_at`、`expires_at` を持ち、保存時は生の bearer token ではなく `ActionTokenDigest` だけを保持する。用途別 payload は登録された codec で具体型へ復号し、未知の目的、目的と codec の不一致、期限切れ、使用済みをすべて作用前に拒否する。

共有 module の interface は発行と検証の二操作に限定し、Context 固有の repository や transaction callback を受け取らない。Authentication のパスワード変更と Identity Management のメール変更は、それぞれの永続化アダプターが検証済み envelope、token の使用済み化、対象 aggregate の更新、監査イベント記録を同じ transaction にまとめる。memory adapter は同じ意味を一つの lock 内で実装する。HTTP と通知のような外部作用は transaction 内で直接行わず、通知は発行時だけ、監査配送は transaction 後の event log 経由とする。

この seam により、呼び出し側が学ぶ interface は発行と検証だけになり、目的束縛、token digest、有効期限、生 token の非保存という実装を共有 module の内側へ隠せる。用途別 transaction は各 Context に残るため、共通化のために一方の repository を他方へ公開しない。

`GET` または `HEAD` は token の形式と画面表示に必要な非機密情報だけを検査し、`ConsumeAndApply` を呼ばない。作用を生む `POST` は既存の CSRF 境界を通す。無効な目的と無効な token の外部エラーを区別せず、内部監査だけが拒否理由を持つ。

## Plan

1. 既存二用途の仕様へ共通不変条件と先読み安全性を追加し、TypeSpec の操作とエラー契約が変わる場合は先に更新する。
2. 共通の domain 型と決定的な発行および検証を追加し、用途違い、期限切れ、生 token の保存を検出する Unit RED を固定する。
3. 各 Context の memory と PostgreSQL adapter に原子的な消費と作用を実装し、並行消費で一回だけ作用が成功することを検査する。
4. パスワード再設定とメール変更を順に移行し、用途別 payload、業務規則、通知を各 Context に残す。
5. 正式なブラウザー入口から発行、先読み、確定、再利用拒否までを通す E2E を追加する。

## Tasks

- [ ] T001 [Spec] `REQ-AUTHENTICATION-016` と `REQ-IDMANAGEMENT-017` に目的束縛、単回使用、先読み安全性、作用との整合性を定める。
- [ ] T002 [Domain] `ActionTokenPurpose`、envelope、digest、決定的な発行と検証を Unit RED から実装する。
- [ ] T003 [Persistence] 各 Context の memory と PostgreSQL adapter に原子的な消費と作用を実装し、並行消費と rollback を検証する。
- [ ] T004 [Authentication] パスワード再設定を共通核へ移行し、既存の秘匿とパスワード規則を維持する。
- [ ] T005 [Identity Management] メール変更を共通核へ移行し、メール一意性と required action の処理を維持する。
- [ ] T006 [E2E] 二用途の正式入口で発行、先読み、確定、再利用拒否、異なる用途での拒否を検査する。
- [ ] T007 [Verify] 主要ユースケースの Unit/E2E RED と代表的な故障注入を記録し、標準検証を通す。

## Verification

- `mise run check-spec`
- `mise run test-go-package ./backend/authentication/password/...`
- `mise run test-go-package ./backend/idmanagement/user/...`
- `mise run test-ui-e2e`
- `mise run verify`
- 並行する二つの確定要求のうち一つだけが作用と監査を記録し、もう一つが状態を変えず拒否される。
- `HEAD`、期限切れ、用途違い、再利用、用途別作用の失敗で、token と対象状態が仕様どおり維持または拒否される。

## Risk Notes

リスクは high。トークンの目的確認、トランザクション、エラー秘匿を誤ると、アカウント乗っ取り、正当な回復手段の喪失、token の再利用につながる。既存二用途を一度に置き換えず、共通核の検証後に一用途ずつ移し、旧 store の削除は両用途の E2E が通った後に行う。
