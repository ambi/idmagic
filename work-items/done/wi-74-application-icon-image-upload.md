---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# アプリケーションアイコンに画像アップロード機能を追加する

Application アイコンを URL 自由入力から tenant-scoped な画像アップロードへ置き換えた。
ADR-073 で PostgreSQL blob 保存、PNG/JPEG/WebP/GIF の magic byte 検証、256 KiB 上限、
SVG 非対応、nosniff 配信を決定し、SCL に icon_object_key、upload/delete/serve interface、
安全配信と tenant isolation の invariant を追加した。
Go 実装では ApplicationIconStore port、memory/PostgreSQL adapter、schema、upload/delete/serve
handler、ApplicationIconUpdated event を追加した。管理 UI は Application 編集画面の
アイコン URL 入力をファイル選択 + プレビュー + 削除に置き換え、作成 API から icon_url 入力を
外した。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
