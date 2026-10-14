---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# Vite を Bun のフロントエンド開発サーバーとバンドラーへ置き換えられるか実証し、成立する場合だけ移行する

`mise run spec-diff` は `no normative specification change against main` を返す。移行可否を実証した結果は移行不可であり、既定の開発サーバーと本番バンドラーは Vite 8.2.2 のまま変えていない。製品の観測可能な振る舞い、公開契約、設定名、CSP、配信境界に差分は無い。リポジトリに残るのは本 Work Item の測定値と阻害条件の記録だけで、試作のコード、スクリプト、依存、`bunfig.toml` の追記、E2E フィクスチャの切替は作業ツリーから完全に撤去した。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
