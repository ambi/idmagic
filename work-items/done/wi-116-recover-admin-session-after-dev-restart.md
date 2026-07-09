---
status: completed
authors: [tn]
risk: medium
created_at: 2026-07-04
---

# 開発サーバ再起動後の管理画面で再ログイン復旧導線を出す

## Motivation
`./dev.sh` を memory persistence で再起動すると、ブラウザ側には古い cookie / token / OIDC state が残る一方、サーバ側の login session や認可 transaction は失われる。
その状態で管理画面をリロードしたとき、認証を続行できないこと自体は正しいが、現在は「認証を続行できません」という行き止まりの画面になり、利用者が再ログインして元の管理画面へ戻れない。
開発時に頻繁に起きる再起動で管理 UI が復旧不能に見えると、実装確認の流れが切れ、実際の認証失効時にも不親切な失敗体験になる。

## Scope
- `spec/contexts/authentication.yaml`
  - 失効または欠落した login session / auth transaction / OIDC callback state を検出したとき、ログイン開始導線と元 URL へ戻る `return_to` を提供する挙動を明記する。
  - `return_to` は同一 origin / 相対パスに限定し、open redirect を起こさないことを明記する。
- `spec/scl.yaml`
  - Authentication と OAuth2/OIDC、および first-party Admin UI の復旧フローに関わる保証を必要に応じて更新する。
- Go / HTTP
  - 失効 cookie、存在しない session、存在しない auth transaction、無効な callback state を区別し、再ログイン可能な応答または安全な `/login?return_to=...` 誘導を返す。
  - 失効状態では古い cookie / client-side token を破棄できるよう、必要な cookie clearing / logout 相当の処理を行う。
- UI
  - 「認証を続行できません」画面を行き止まりにせず、再ログインボタンと元の管理画面へ戻る導線を表示する。
  - 管理画面リロード時に API が 401 / invalid session を返した場合、保持している access token / refresh token / callback state を破棄して再認可を開始する。
  - 再ログイン後は、直前に開いていた管理画面のパスへ復帰する。
- Tests
  - memory persistence を想定し、ログイン済み SPA 状態からサーバ側 session / transaction が失われたケースのハンドラテストまたは e2e テストを追加する。

## Out of Scope
- memory persistence でセッションをサーバ再起動後も保持すること。
- PostgreSQL / Valkey persistence での HA セッション永続化の再設計。
- 長期ログイン、remember me、refresh token lifetime の変更。

## Verification
- `just yaml-check-scl`
- `just verify-go`
- `just verify-ui`
- 手動: `./dev.sh` 起動 → 管理画面へログイン → 任意の管理画面を開く → `./dev.sh` 停止 → 再起動 → その画面をリロード → 再ログイン導線が表示されること。
- 手動: 再ログイン後、再起動前に開いていた管理画面パスへ戻れること。
- 手動: 外部 URL を `return_to` に指定してもリダイレクトされず、安全な既定画面へ戻ること。

## Risk Notes
認証失効時の復旧導線は open redirect や stale token 再利用と隣り合わせになる。
復旧時は古いクライアント状態を明示的に捨て、`return_to` を相対パスに制限し、認証が完了するまでは管理 API を再試行し続けないようにする。

## Completion
- **Completed At**: 2026-07-04
- **Summary**:
  first-party 管理コンソール / アカウントポータル (ADR-061 の OIDC RP) が、保持する
  access token を提示した `/api/auth/account` 呼び出しで 401 / 失効セッションを受けたとき、
  行き止まりの「認証を続行できません」画面に落ちず復旧するようにした。
  - SCL: `spec/contexts/system.yaml` に UX 要件 `UX-PORTAL-SESSION-RECOVERY`
    (保持トークン / OIDC callback state の破棄・同一オリジン相対 `return_to` での 1 回だけの
    再認可・認証完了まで管理 API を再試行しない・open redirect 拒否) を追加し、
    `AdminDashboard` 画面に `reauthenticating` state を足した。
    `spec/contexts/authentication.yaml` に失効セッションからの復旧シナリオ
    (return_to 安全性と再試行抑止の extension を含む) を追加した。
  - UI: `ui/src/api/oidc.ts` に `recoverPortalSession` (stale トークン + refresh token +
    進行中 OIDC callback state を破棄し、returnTo を保って 1 回だけ再認可。直近再認可からの
    ループは `ra_oidc_reauth_*` マーカーで抑止)、`markPortalAuthenticated`、
    `restartPortalLogin` を追加。`ui/src/routes/-guards.ts` は `/api/auth/account` の
    `UnauthenticatedError` を捕捉して復旧を起動する。`ui/src/router.tsx` の `ErrorScreen` は
    portal パスでは行き止まりにせず、現在のパスへ戻る「再ログイン」導線を表示する。
  - Go: 復旧が依拠する識別可能なシグナルを回帰固定する handler test を追加
    (`TestAccountContextRejectsStaleBearerToken`: 失効 Bearer → 401 authentication_required)。
    サーバの既存挙動 (transaction 欠落 + `return_to` での login 応答、失効セッションの
    401、`return_to` の相対 / 同一オリジン検証) を再利用したため production Go コードは変更なし。
  - E2E: `ui/tests/e2e/admin-session-recovery.spec.ts` を追加。/admin/users にログイン後、
    sessionStorage の access token を検証不能な値へ差し替えてリロードし、行き止まりにならず
    再認可して /admin/users へ戻り、stale トークンが破棄されることを検証する。
- **Verification Results**:
  - `just yaml-check-scl`
    - result: ok (全 12 ファイル OK)
  - UI: `bun run format:check` / `lint` / `typecheck` / `build`
    - result: ok
  - Go: `go test -race ./...`
    - result: ok (FAIL / panic なし)
  - Go lint: `golangci-lint run ./internal/shared/adapters/http/server/...`
    - result: 0 issues
  - E2E: `bun test tests/e2e/admin-session-recovery.spec.ts`
    - result: 1 pass
- **Affected Guarantees State**: 既存の認証・認可・return_to 検証の保証は不変。
  first-party portal の失効時復旧という UX 保証を新設した。サーバ側の認証判定・
  return_to 検証は既存挙動を再利用しており実体に変更はない。
