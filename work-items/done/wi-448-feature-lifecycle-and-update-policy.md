---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# 機能の成熟度、依存、有効化、更新方針を一つの正本で管理する

`mise run spec-diff` は `normative specification change against main` を返し、`REQ-SYSTEM-016` と `REQ-SYSTEM-017` を変更した。TypeSpec には `FeatureMaturity`、`FeatureRuntimeMetadata`、`FeatureUpdatePolicy`、`RuntimeFeatureMetadata` を追加した。製品ビルドが所有する静的な機能 registry、既定値・明示指定・依存閉包を決定的に解決する純粋計算、起動前の集約拒否、起動警告、設定リファレンス、形式版付き `/health` メタデータを同じ解決結果へ接続した。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
