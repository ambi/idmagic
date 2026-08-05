---
status: accepted
authors: [tn]
created_at: 2026-07-04
---

# ADR-042: エンドユーザー account portal の self-service スコープ

## コンテキスト

end-user が自分自身に対して操作できる UI は `/account/password` (パスワード変更) と
`/account/profile` (表示名・属性編集、wi-19) しか無く、admin に頼らず完結できる
self-service の中核と、その権限境界が言語化されていなかった。Keycloak Account Console /
Okta End-User Dashboard / Google アカウント相当の "マイページ" を持ち込むにあたり、
**self が変更してよいもの**と**admin だけが変更できるもの**を曖昧にしたまま API/UI を
増やすと、権限昇格や誤編集の温床になる。

## 決定

account portal の trust boundary を `self` と `admin` で分ける。API は全て `/api/account/`
プレフィックスで認証済みセッションの `actor.sub` のみを操作対象にし (`requireAuthenticatedSub`)、
URL/body/query の sub・tenant_id は信用しない。admin shell 用の `/api/auth/account` (roles を
含む) とは別契約とし、portal の概要は roles を含まない `AccountSummary` で返す — 誤って admin
権限情報を self 経路へ露出しないため。self が変更できる範囲は表示名・`editable_by_user=true` の
属性・パスワードに限り、`roles`/`status`/組織属性/`editable_by_user=false` 属性・
`required_actions` の付与解除は admin 専用のまま残す。portal shell は admin shell と分離し、
admin ロールを持つ user が開いても管理コンソールへの導線を出さない。高 sensitivity 操作の
step-up とデータエクスポート形式は対象機能を実装する後続ステージで別 ADR (ADR-043) として定める。

現在の設計は [`backend/authentication/ARCHITECTURE.md`](../backend/authentication/ARCHITECTURE.md)
の Account portal trust boundary and step-up セクションにある。

## 影響

- self mutation は全て `actor.sub == target.sub` を最低要件とし、admin RBAC とは独立した
  境界になる。SCL では self interface (`GetAccountSummary` / `GetUserProfile` /
  `UpdateUserProfile`) を admin interface と分けて表現する。
- `AccountSummary` は roles を含まないため、portal が誤って admin 権限情報を露出しない。
- 後続ステージの API/UI は本 ADR の self/admin 分類表に従って追加し、admin 専用項目を
  self 経路に出さないことをテストで担保する。

## 参照

- [[wi-21-end-user-account-portal]] — 本 ADR を導く WI。
- [[ADR-040]] — `editable_by_user` を含む属性ポリシー。
- [[wi-19-rich-user-attributes]] — self プロフィール編集と required actions の基盤。
- [[wi-18-unauthenticated-admin-redirect]] — 未認証リダイレクトの pattern。
