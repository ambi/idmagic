---
id: idp-wi-116-recover-admin-session-after-dev-restart
title: "開発サーバ再起動後の管理画面で再ログイン復旧導線を出す"
created_at: 2026-07-04
authors: [tn]
status: pending
risk: medium
---

# Motivation
`./dev.sh` を memory persistence で再起動すると、ブラウザ側には古い cookie / token / OIDC state が残る一方、サーバ側の login session や認可 transaction は失われる。
その状態で管理画面をリロードしたとき、認証を続行できないこと自体は正しいが、現在は「認証を続行できません」という行き止まりの画面になり、利用者が再ログインして元の管理画面へ戻れない。
開発時に頻繁に起きる再起動で管理 UI が復旧不能に見えると、実装確認の流れが切れ、実際の認証失効時にも不親切な失敗体験になる。

# Scope
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

# Out of Scope
- memory persistence でセッションをサーバ再起動後も保持すること。
- PostgreSQL / Valkey persistence での HA セッション永続化の再設計。
- 長期ログイン、remember me、refresh token lifetime の変更。

# Verification
- `just yaml-check-scl`
- `just verify-go`
- `just verify-ui`
- 手動: `./dev.sh` 起動 → 管理画面へログイン → 任意の管理画面を開く → `./dev.sh` 停止 → 再起動 → その画面をリロード → 再ログイン導線が表示されること。
- 手動: 再ログイン後、再起動前に開いていた管理画面パスへ戻れること。
- 手動: 外部 URL を `return_to` に指定してもリダイレクトされず、安全な既定画面へ戻ること。

# Risk Notes
認証失効時の復旧導線は open redirect や stale token 再利用と隣り合わせになる。
復旧時は古いクライアント状態を明示的に捨て、`return_to` を相対パスに制限し、認証が完了するまでは管理 API を再試行し続けないようにする。
