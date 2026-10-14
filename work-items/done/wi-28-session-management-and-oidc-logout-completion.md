---
status: completed
archived_from: e6f80d088eca5545f490b9992e1d48243ea8d371
---

# セッション管理と OIDC logout を実運用相当に完成させる

session/token のローカル失効を実運用相当まで完成させた。(T004) OIDC の `sid`
(Authentication の `LoginSession.id` と同値) を `AuthorizationRequest` →
`AuthorizationCodeRecord` → `RefreshTokenRecord` → ID Token へ一貫して伝播させ、
session revoke (self-service 個別/全revoke、admin 個別/全revoke、
RP-Initiated Logout) のいずれからも `RevokeTokensBySid` で同じ sid を共有する
refresh token を family/client 横断で失効させるようにした。(T005) `/end_session`
に `id_token_hint` の署名・iss・aud・sub・sid 検証を追加し (fail-closed、
client_id 不一致は拒否、hint 無しは既存 cookie 解決へ後方互換フォールバック)、
local logout を use case へ抽出した。(T007) admin 向け session 一覧・個別/全revoke
(`ListSessions`/`RevokeSession`/`RevokeUserSessions`, ADR-127 決定9) を Go/UI
両方に実装し、self-service と同じ oauth2 cascade を admin 経路にも適用した。(T008)
上記の検証観点 (複数端末、期限切れ hint、redirect 攻撃、DB永続性) を既存の
回帰テストと実サーバーでの手動確認で満たした。
front/back-channel logout の通知配送 (client metadata、`ClientSession`、
`LogoutNotification`、logout token signer、Jobs 経由配送、`CheckSessionIframe`) は
独立した実装単位として [[wi-257-oidc-front-back-channel-logout-notifications]] へ
切り出し、本 WI のスコープからは意図的に除外した (ADR-127 の決定5/6/8 がその WI の
設計を規定する)。

全文は `git show e6f80d088eca5545f490b9992e1d48243ea8d371:<このファイルのパス>` で読む。
