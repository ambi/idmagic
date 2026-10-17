---
status: completed
---

# モジュールの README を概要と道案内に絞り、モジュールの `architecture.md` を廃止する

`mise run spec-diff` は main に対して規範仕様の変更なしと報告する。
モジュールの `README.md` を「責務と境界、主な流れ、設計の要点、機能一覧、文書」の概要と道案内に絞り、モジュールの `architecture.md` を廃止した。
Aggregate は `data.md` の「集約」へ移し、「公開する契約」の節はなくした。
内部の設計は素直な形をデフォルトとして書かず、そうでない構造と理由だけを書くと規則文書に定めた。
設計の索引は `設計領域 | 文書 | 内容` の 3 列にし、生成サイトのモジュールのページから API と型の一覧と機能地図を外した。

全文は `git log -1 -- work-items/active/wi-38847-make-module-readme-an-overview.md` が示すコミットの親、またはそのコミットで読む。
