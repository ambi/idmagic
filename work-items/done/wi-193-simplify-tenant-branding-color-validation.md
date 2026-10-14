---
status: completed
archived_from: a5e9afc073b261d086555467cc1c384f751393a6
---

# テナントブランディングの色コントラスト制約を廃止する

primary_color / accent_color は `#rrggbb` 形式だけを検証し、低コントラストの色を保存・取得できるようにした。
UI と HTTP のエラー文言を形式検証に合わせ、ADR-097 でテナントが可読性を判断する方針を記録した。

全文は `git show a5e9afc073b261d086555467cc1c384f751393a6:<このファイルのパス>` で読む。
