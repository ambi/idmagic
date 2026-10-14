---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# 管理画面に「設定」ページを追加してテナント単位の設定を集約する

テナント内 admin と system_admin が自テナントの表示名・パスワードポリシー上書きを
閲覧・更新できる `/api/admin/settings` (GET / PATCH) と `/admin/settings` ページを
追加した。対象は actor.tenant_id に固定し cross-tenant 操作は構造的に発生しない。
password_policy_override が objectives.PasswordPolicy の標準値より弱い値を含む
場合は use case 側で reject する。既存 `/admin/tenants` (system_admin) の挙動は
不変だが、password_policy_override が UpdateTenant に到達しなかった既存バグも
併せて修正した。
続いて UX 整理のフォローアップで、`AdminSettingsResponse` に
`password_policy_defaults` を埋め込み、UI に「標準 最小長 12 / 最大長 128 /
履歴件数 5」のサマリと具体数字の hint・placeholder を表示するようにした。
「global default」というユーザ向け文言を「標準値」に統一し、weaker policy の
エラー文言にも具体数値 (`min_length≥12 / max_length≤128 / history_depth≥5`) を
含めた。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
