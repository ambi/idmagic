---
status: completed
archived_from: 9de43662a232afc9cbf5f895c8e310b6a2db7c35
---

# 通知メールをローカライズ済みテンプレートカタログ化し、テナントでカスタマイズ可能にする

通知メールを「組込み既定カタログ + テナント上書き」の 2 段解決に寄せ
([[ADR-142-notification-template-catalog-and-locale-resolution]])、usecase 内にハードコード
されていた英語プレーンテキスト 3 箇所を全て置換した。`backend/shared/notification/template`
が ja / en の組込み既定 (`embed` した locale 別 YAML)、`{{name}}` 記法のレンダラ
(許可集合検査・HTML エスケープ・text/html 同時生成・文書外枠)、locale 解決、Notifier を
所有し、テナント上書きの永続化と管理 API は Tenancy が持つ。lifecycle dispatcher の
`TemplateKey` 直挿しも解消した。管理コンソールには一覧・編集・placeholder 表示・
プレビュー・既定へのリセット・自分宛テスト送信を持つ「メール」タブが入った
(これまで「近日公開」の無効タブ)。

全文は `git show 9de43662a232afc9cbf5f895c8e310b6a2db7c35:<このファイルのパス>` で読む。
