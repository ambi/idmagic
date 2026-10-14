---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# 管理コンソールに専用の詳細画面を設け、右ペインは「スクロール不要の簡易ビュー」に役割分担する

管理コンソールの 4 エンティティ (ユーザー / グループ / アプリケーション
(client) / ロール) に専用の詳細画面を新設し、右ペインとの役割分担を整えた。
詳細画面は対象の全情報を縦スクロール前提で網羅表示する。ユーザー詳細は
基本プロフィール・ライフサイクル (status / last_login_at /
password_changed_at / 強制アクション)・全 attributes (組み込み OIDC 標準 /
SCIM 組織 / tenant custom を区分し label + key 併記)・ロール・所属グループ
を見せる。右ペインは要点と主要操作を残した簡易ビューとし、上部に
「詳細を開く」(primary) → 「編集」(outline) → その他操作 (⋮) の順で
そろえた共有アクション行 `AdminPaneActions` を全画面に適用して、
一覧画面ごとにバラバラだったボタン配置を統一した。ルーティングは
`resolvePageData` に path param ルート (`/admin/users/:sub` ほか) と新 kind
を追加し、deep link でも単独描画できる。バックエンドは既存 admin GET の
再利用のみで新規エンドポイント・SCL・権限境界の変更は無い。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
