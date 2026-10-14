---
status: completed
archived_from: 8c887e58d159c4c4be68a715c8bf565db319d988
---

# テナントロゴをアップロード後に hosted UI で確実に表示する

realm ごとの tenant branding asset URL を Caddy / Vite が backend へ転送するようにし、
管理プレビュー、login、consent、account portal が同一 origin の検証済みロゴだけを表示するようにした。
併せて、E2E fixture の backend 起動先、UUID 化済み demo client、画面遷移・API filter 名を現行実装へ同期し、
ポートを共有する E2E spec をプロセス単位で直列化した。

全文は `git show 8c887e58d159c4c4be68a715c8bf565db319d988:<このファイルのパス>` で読む。
