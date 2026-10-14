---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# ユーザー属性を実運用相当 (Keycloak / Okta / Google IAM 同等) まで拡張する

ユーザー属性を実運用相当まで拡張した。中核は ADR-039 (thin core +
sparse attribute bag) / ADR-040 で、User は識別・認証・表示名・RBAC・
ライフサイクルを型付きで持ち、OIDC §5.1 optional claim と SCIM 組織属性は
組み込みカタログ駆動の単一 attribute bag (JSONB) に格納する。admin/self の
属性編集 API・UI、tenant custom attribute schema、scope 別の claim 露出
(UserInfo / ID Token) は先行 PR で実装済み。本セッションで残りの
ライフサイクル配線を完了した: (1) Required Actions 一式 (admin の
付与/解除 API・usecase・イベント、パスワード変更での update_password 自動
解除、未充足時のログイン強制誘導フック、admin UI トグル)、(2) ログイン時の
last_login_at とパスワード変更時の password_changed_at の記録。あわせて
当初設計のうち amendments(a)〜(c) (多値連絡先 / VerifiedClaim / 属性ごとの
粒度イベント) を thin-core 再設計に伴い正式に descope した。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
