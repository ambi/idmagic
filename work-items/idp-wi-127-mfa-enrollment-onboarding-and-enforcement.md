---
id: idp-wi-127-mfa-enrollment-onboarding-and-enforcement
title: "MFA 必須化時の登録オンボーディングと強制を分離する"
created_at: 2026-07-09
authors: ["codex"]
status: pending
risk: high
---

# MFA 必須化時の登録オンボーディングと強制を分離する

## Motivation

MFA 必須ポリシーは、登録済み factor を持つユーザーにはサインイン時の第二要素を要求し、
未登録ユーザーには安全な登録導線を提示する必要がある。未登録ユーザーを即座に完全拒否
すると新規ユーザー追加や段階導入が運用不能になり、一方でログイン途中の無条件登録を許すと
パスワードを知る攻撃者が自分の factor を登録でき、実質 1 要素認証になる。

主要 IdP はこの問題を「MFA enforcement」と「MFA enrollment / required action / grace
period」を分けて扱う。Okta はグローバルセッションポリシーで MFA required と prompting
frequency を持ち、認証ポリシーの factor 要件と組み合わせる。Keycloak は `Configure OTP`
などの Required Action / Conditional OTP を認証フローに組み込む。Microsoft Entra ID は
MFA / SSPR の combined registration で登録割り込みを扱う。Google Workspace は 2-Step
Verification の enforcement start date と、未登録ユーザーの把握を管理者に求める。

idmagic でも、MFA 強制を「未登録なら永久ロック」または「未登録なら自由登録」の二択にせず、
明示的な登録オンボーディング状態と期限、管理者の可視性、復旧操作を仕様化する。

## Scope

- `spec/contexts/authentication.yaml` の `LoginSession`, MFA enrollment state, browser auth flow,
  account security enrollment, authentication events, user experience。
- `spec/contexts/application.yaml` の tenant default / application sign-in policy 表現と管理 UI。
- Authentication use cases: MFA enrollment required action, enrollment challenge, grace / deadline
  evaluation, temporary bypass / recovery path。
- OAuth2 browser login handlers: MFA 必須だが未登録のユーザーを enrollment-required flow へ誘導し、
  登録完了後に同一 auth transaction を継続する。
- Account / Admin UI: 未登録者数、強制日、登録猶予、未登録時のログイン画面、管理者復旧導線。
- Persistence adapters: 必要に応じて enrollment deadline / bypass token / required action を保存する。
- Audit events: enrollment required, enrollment completed, bypass issued / consumed / expired。

## Out of Scope

- SMS / voice など新しい factor 種別の追加。
- 外部 IdP の MFA claim を信頼する federation assurance mapping。
- デバイス信頼や remember-MFA の本格実装。
- ヘルプデスク本人確認ワークフローの自動化。初期実装は admin 操作としての一時バイパスに留める。

## Plan

- 方針:
  - MFA policy は `enforcement` と `enrollment` を分ける。
  - `MFA required` かつ factor 未登録のユーザーは、ポリシーで許可された登録オンボーディング条件を
    満たす場合だけ、パスワード認証後に MFA 登録専用の pending flow へ進める。
  - 登録オンボーディング flow は通常の認証完了セッションとして扱わない。登録完了後に同じ
    LoginSession / authorization transaction を MFA 済みに昇格させる。
  - 登録オンボーディングが許可されていない、期限切れ、または対象外の場合は fail closed で拒否し、
    管理者復旧を案内する。
  - 管理者 UI は MFA 必須化前に未登録ユーザー数と影響を表示し、強制日または登録猶予を設定できる。
- 参考にする外部パターン:
  - Okta: MFA required と prompt timing を policy rule として分離。
  - Keycloak: required action / conditional OTP によるログイン時登録。
  - Microsoft Entra ID: combined registration による登録割り込み。
  - Google Workspace: enforcement start date と未登録ユーザー確認。
- 却下する代替案:
  - 未登録なら常に自由登録: パスワードだけで factor 登録できるため不可。
  - 未登録なら常に拒否: 新規ユーザー追加・段階導入・復旧が運用不能。
  - MFA 必須化時に既存ユーザー全員へ自動バイパス付与: 強制の意味が曖昧になり監査しづらい。

## Tasks

- [ ] T001 [SCL] MFA enrollment policy / required action / temporary bypass / events / UX を仕様化する。
- [ ] T002 [Domain] MFA 登録状態、登録猶予、期限切れ、バイパスの判定関数を追加する。
- [ ] T003 [UseCase] パスワード認証後、MFA 必須かつ未登録時に enrollment-required flow へ進む処理を追加する。
- [ ] T004 [UseCase] enrollment flow で TOTP / WebAuthn 登録完了後、同一 LoginSession を MFA 済みに昇格させる。
- [ ] T005 [UseCase] enrollment 不許可・期限切れ・factor 登録不能時は fail closed で拒否する。
- [ ] T006 [Admin] サインインポリシー画面に未登録ユーザー数、強制開始日、猶予設定、警告を追加する。
- [ ] T007 [Admin] 一時バイパス発行 / 取消 API と UI を追加する。
- [ ] T008 [Account/UI] ログイン途中の MFA 登録画面を追加し、通常マイページとは区別する。
- [ ] T009 [Audit] 登録要求、登録完了、バイパス発行・消費・失効を監査イベントに出す。
- [ ] T010 [Verify] E2E で、未登録者のオンボーディング、期限切れ拒否、既登録者の MFA 必須、自由登録不可を固定する。

## Verification

- `just yaml-check`
- `just test-go`
- `just verify-ui`
- `just test-ui-e2e`
- 手動確認:
  - MFA 必須 + 登録済みユーザーは第二要素を要求される。
  - MFA 必須 + 未登録ユーザー + enrollment allowed は登録専用 flow に進み、登録完了後だけログイン完了する。
  - MFA 必須 + 未登録ユーザー + enrollment disabled / expired はログイン完了できない。
  - 登録 flow 中のセッションはマイページやアプリへアクセスできない。
  - 管理者は未登録者数とロックアウト影響を確認できる。

## Risk Notes

リスクは高い。MFA 必須化は認証強度を保証する境界であり、登録 flow の扱いを誤ると
「パスワードだけでログイン完了」「攻撃者による factor 乗っ取り登録」「全ユーザーロックアウト」
のいずれかになる。初期実装では登録オンボーディングを明示的に有効化されたポリシー下に限定し、
通常セッションとは別の pending 状態として扱い、監査イベントを必須にする。
