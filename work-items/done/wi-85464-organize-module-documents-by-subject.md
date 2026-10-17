---
status: completed
---

# モジュール設計文書を対象ごとのファイルに分け、`design/` と機能群の段を廃止する

`mise run spec-diff` は main に対して規範仕様の変更なしと報告する。
モジュール設計文書を対象ごとのファイルに分けた：設計領域の文書と記録をモジュールの直下へ、機能仕様と例と章を `features/<feature>.md`、`<feature>.feature.md`、`<feature>.<chapter>.md` へ移し、機能群の段と任意名の横断的概念の文書を廃止した。
仕様と設計はファイルではなく文の性質で区別すると文書ガイドに定め、モジュールの `README.md` の最後の節を「文書」にした。
責務表を区分ごとにモジュール設計文書の入口へ移し、読み始める順序と、import から生成して古さを検査する依存の図を置いた。
仕様差分と境界の負債の比較は、移す前の配置の revision も読む。移す前の revision と比べる必要がなくなれば、この読み替えは外せる。

全文は `git log -1 -- work-items/active/wi-85464-organize-module-documents-by-subject.md` が示すコミットの親、またはそのコミットで読む。
