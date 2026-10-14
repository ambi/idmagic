---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# 開発サーバ再起動後の管理画面で再ログイン復旧導線を出す

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

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
