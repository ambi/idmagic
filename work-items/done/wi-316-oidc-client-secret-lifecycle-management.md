---
status: completed
archived_from: 1b59925b11832db895e261b669b455583a34d7a6
---

# OIDCクライアントシークレットを複数保持・有効期限管理できるようにする

OIDC client secret を有効期限付きで追加発行し、最大2件を同時利用しながら credential 単位で
明示失効できる lifecycle を、SCL から domain/use case、memory/PostgreSQL、Application HTTP、
token authentication、audit、UI まで実装した。既存の期限なし credential は維持し、table が
空の legacy client だけを最初の発行 transaction 内で backfill する。ADR-152 で ADR-125 の
自動 sunset UI 判断だけを部分的に置き換え、旧 rotate API は互換経路として残した。
UI はクライアント ID を含む通常設定 Card と別のトップレベル Card にし、設定 `<form>` の外へ
完全に分離した。専用 Card には見出し・説明・credential 一覧・発行・失効だけを置き、両 Card は
`gap-6` の兄弟要素である。単なる下方向への移動や内側 section 化ではないことを DOM test で
固定した。完了後の実機フィードバックで判明した状態バッジの内容幅依存も解消し、「有効」だけの
場合も十分な楕円幅を保ち、「失効済み」は折り返さない表示へ修正した。

全文は `git show 1b59925b11832db895e261b669b455583a34d7a6:<このファイルのパス>` で読む。
