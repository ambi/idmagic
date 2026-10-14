---
status: completed
archived_from: 0fca46fb861aad86ad8b000c8feebf5172954e84
---

# API アクセストークン設定で接続先と scope の意味を理解できるようにする

API アクセストークンタブの主見出しを他の設定タブと同じ構造で維持し、
重複していた一覧側を「発行済みトークン」と明確に命名した。
管理 API、SCIM 2.0 API、発行者本人のアカウント API の Base URL を
用途説明・コピー操作付きで提示した。
scope 選択は管理 API、SCIM 2.0 API、アカウント API の折りたたみグループへ分割し、
リソース名、参照/変更の意味、正準 scope 値、選択数を表示するようにした。
2 列表示では左を参照、右を変更に固定し、write-only scope も右列へ揃えた。

全文は `git show 0fca46fb861aad86ad8b000c8feebf5172954e84:<このファイルのパス>` で読む。
