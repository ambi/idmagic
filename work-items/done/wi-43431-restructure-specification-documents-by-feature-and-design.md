---
status: completed
archived_from: ff7f8040afa55d0448ee81de50c3444bcaf080bb
---

# 仕様文書を、機能仕様と内部設計を軸にした多層の構造へ改める

`mise run spec-diff` は、規範仕様の差分を報告しない。
仕様文書の構造を、機能仕様と内部設計を軸にした形式へ改める規約と道具を整えた。
新しい形式の Context は `design/README.md` の有無で判定し、旧形式の Context は `tools/check/legacy-spec-layout.json` に列挙した（現時点は全 21 件）。
検査は、機能仕様の `#### REQ-…` 見出しを規則の宣言として読み、付録 `examples.feature.md` の例と照合し、担保手段の欄を必須とし、理由の欄を判断の欄へ改め、機能仕様の節の順序と `design/decisions.md` の判断の骨格を確かめる。
spec-diff、work item の参照の解決、brief、spec-route、セキュリティ統制の検査、render-docs も新しい形式を読む。
render-docs は、機能仕様に規則一覧と未決事項を、Context の README に機能地図を生成して差し込む。
完了した work item の旧パスは `tools/check/relocated-spec-paths.json` で読み替える。現時点では空である。
セキュリティ統制の検査が固定のパスで読む `api-tokens/scenarios.feature.md` の扱いは、wi-26063 の対象範囲に加えた。

全文は `git show ff7f8040afa55d0448ee81de50c3444bcaf080bb:<このファイルのパス>` で読む。
