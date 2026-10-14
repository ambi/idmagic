---
status: completed
archived_from: a8e03c729bc0cb0163e004b8abe40c9d40181425
---

# リポジトリ入口の文書を置く——脆弱性の報告経路、貢献の作法、変更の告知、対象外の宣言

リポジトリの入口に、これまで存在しなかった 3 つの経路を置いた。`SECURITY.md` は脆弱性の報告を GitHub Security Advisories の非公開経路へ導き、扱う境界を `docs/authorization.md` のテナント境界・フェイルクローズ・鍵素材に紐づけ、開発用の既定値を本番で使った場合を対象外として切り分けた。応答の日数は約束していない。`CONTRIBUTING.md` は Pull Request に求める 5 点（仕様先行、1 変更 1 work item、生成物の再生成、拒否の二重の assert、英語のコミット）を述べ、必須の検査の一覧は複製せず CI 定義を正本として指す。`README.md` には対象外の表 3 行を足し、対象範囲の宣言は `docs/README.md` の Context Map 索引に委ねた。あわせて文書表に `CONTRIBUTING.md`、`SECURITY.md`、`DOCUMENTATION_GUIDE.md`、`work-items/`、`tools/README.md` の 5 行を足した。`CHANGELOG.md` は作っていない。`deprecated_since` と `@deprecated` が `spec/**/*.tsp` に 1 件も無く、告知すべき非推奨が存在しないためである。規範的な仕様は変えていない（`mise run spec-diff` が `no normative specification change`）。

全文は `git show a8e03c729bc0cb0163e004b8abe40c9d40181425:<このファイルのパス>` で読む。
