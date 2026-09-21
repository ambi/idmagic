---
depends_on: []
status: completed
authors: ["tn"]
risk: high
created_at: 2026-07-09
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

- SCL sections: `authentication.models`, `authentication.interfaces`, `authentication.scenarios`,
  `authentication.flows`, `application.models`, `application.interfaces`, `application.scenarios`。
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

- [x] T001 [SCL] MFA enrollment policy / required action / temporary bypass / events / UX を仕様化する。
- [x] T002 [Domain] MFA 登録状態、登録猶予、期限切れ、バイパスの判定関数を追加する。
- [x] T003 [UseCase] パスワード認証後、MFA 必須かつ未登録時に enrollment-required flow へ進む処理を追加する。
- [x] T004 [UseCase] ADR-110 の初期 factor である TOTP の登録完了後、同一 LoginSession を MFA 済みに昇格させる。
- [x] T005 [UseCase] enrollment 不許可・期限切れ・factor 登録不能時は fail closed で拒否する。
- [x] T006 [Admin] サインインポリシー画面に未登録ユーザー数、強制開始日、猶予設定、警告を追加する。
- [x] T007 [Admin] 一時バイパス発行 / 取消 API と UI を追加する。
- [x] T008 [Account/UI] ログイン途中の MFA 登録画面を追加し、通常マイページとは区別する。
- [x] T009 [Audit] 登録要求、登録完了、バイパス発行・消費・失効を監査イベントに出す。
- [x] T010 [Verify] E2E で、未登録者のオンボーディング、期限切れ拒否、既登録者の MFA 必須、自由登録不可を固定する。

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

## Completion

- **Completed At**: 2026-07-15
- **Summary**:
  - MFA 強制開始日、登録猶予、管理者承認バイパスをサインインポリシーへ追加し、未登録者数と
    ロックアウト影響を管理画面から確認できるようにした。
  - 未登録ユーザーは、期限内かつ管理者が発行した一回限りのバイパスがある場合だけ、パスワード
    検証後に登録専用 pending session へ進む。TOTP 登録完了後は同じ LoginSession と OAuth2
    transaction を昇格・継続し、それ以外は fail closed とした。
  - バイパスの発行・取消・原子的消費・期限切れを memory/PostgreSQL adapter、管理 API/UI、監査
    イベントまで実装し、認証途中の TOTP 登録画面と管理者復旧導線を追加した。
  - 登録オンボーディングの trust anchor と初期 factor の判断を ADR-110 に記録し、SCL 派生物を
    再生成した。
- **Verification Results**:
  - `just yaml-check-scl` — passed
  - `just scl-render` — passed
  - `just test-go` — passed
  - `just verify-ui` — passed (59 test files / 348 tests, format, lint, typecheck, build)
  - `just test-ui-e2e` — passed
  - `just verify` — passed (YAML, tools 208 tests, Go lint/test/race, UI 348 tests, build)
  - `git diff --check` — passed
- **Affected Guarantees State**:
  - guarantee: MFA 必須かつ登録済みのユーザーは、引き続き第二要素の検証なしに認証完了しない。
  - state: passed
  - guarantee: MFA 未登録ユーザーは、有効な管理者承認バイパスなしに factor を自由登録できない。
  - state: passed
  - guarantee: 登録専用 session は通常ログイン完了として扱われず、TOTP 登録完了後だけ同じ
    authorization transaction を継続する。
  - state: passed
  - guarantee: バイパスは tenant/user に束縛され、一回限りで、取消・期限切れ時は fail closed になる。
  - state: passed
- **Evidence**:
  - procedure: SCL-first で enrollment policy、pending purpose、管理者バイパス、監査イベント、画面遷移を
    定義し、domain、use case、persistence、HTTP/OAuth2、UI の順に実装した。単体・PostgreSQL・HTTP E2E・
    UI テストで登録済み要求、承認済みオンボーディング、未承認拒否、期限切れ拒否を固定した。
  - commands: `just yaml-check-scl`, `just scl-render`, `just test-go`, `just verify-ui`,
    `just test-ui-e2e`, `just format-go`, `just verify`, `git diff --check`
  - environment: macOS arm64 workspace
  - actor: Codex (implement-work-item skill)
  - source: pre-commit working tree based on `da249148`
  - result: passed
  - artifacts: `spec/contexts/authentication.yaml`, `spec/contexts/application.yaml`,
    `decisions/ADR-110-admin-authorized-mfa-enrollment-bypass.md`, `spec/idmagic.html`,
    `spec/idmagic.models.schema.json`, `spec/idmagic.openapi.json`
