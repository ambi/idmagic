---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# ログイン履歴を実運用相当に拡張する (成功・連携・失敗・セッション、攻撃時の爆発を抑える)

価値優先の 3 スライスで、wi-21 の activity ステージを最短で解きつつ
攻撃時の監査爆発も止めた。全スライスは新たな永続テーブルを足さず、
既存の監査イベントストアと in-memory アダプタの上で動かした:
- スライス 1: end-user サインイン履歴。既存 `UserAuthenticated` を
発生時刻の降順で射影する `GET /api/account/signin_activity` と
admin の `/api/admin/users/{sub}/signin_activity` (UI は wi-21)。
- スライス 2: 本人セッションの一覧・個別失効・他端末ログアウト
(`/api/account/sessions` *). `SessionEnded` (reason `self_revoke`) と
`SessionEndReason` enum、SessionStore.ListBySub を追加 (UI は wi-21)。
- スライス 3: 失敗ログインの bucket 集約。ADR-029 の throttle が key を
lock したら個別 `AuthenticationFailed` を止め、(tenantId, kind, keyHash,
5 分窓) の bucket に集約して `AuthenticationEventAggregated` を窓ごと
1 件だけ emit する。bucket と throttle は同じ keyHash を共有し、admin は
`GET /api/admin/authentication_event_buckets` で観測する。
永続ストア・属性拡張・retention・admin 検索 UI・federation / MFA challenge /
impersonation の語彙といったインフラ層の残件は
[[wi-44-authentication-event-store-and-search]] に集約し、admin / OIDC logout
側のセッション完成は [[wi-28-session-management-and-oidc-logout-completion]]
に委ねた。当初の ADR-041/045/046 は wi-44 で正式起草する。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
