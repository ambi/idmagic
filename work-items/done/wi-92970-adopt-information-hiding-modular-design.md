---
status: completed
archived_from: d0267d4be1e827899e42d71fd755e184fbede85c
---

# 境界の判断を、観測できる事実と検査される宣言に基づくモジュール設計へ置き換える

`mise run spec-diff` は main に対して規範仕様の差分がないことを示した。
変わったのは設計方針、設計文書の用語、境界の検査、開発用のレポートである。
境界の判断を、機械抽出する事実、検査される宣言、証拠を比較する判断の三つに分け、境界を選ぶ判断手順（D1〜D8）と相談の条件を設計ガイドラインへ置いた。
概念を定義する文書の Bounded Context と Context をモジュールへ改名し、Context Map と Subdomain の区分とその検査を撤去した。
境界検査は責務表からモジュール、公開方式、公開パッケージ、組み立て地点を読み、`private-import`、`module-cycle`、`shared-dependency`、`composition-import`、`domain-effect`、`table-write` を判定する。
`table-write` は `sqlc parse` の構文木から書き込み先を読み、データベース設計の所有モジュールと照合する。
負債の増加検査は基準 revision のソースにも同じ規則を当て、旧形式の文書を互換処理で読む。
`mise run report-change-coupling` と、Go の `internal/` の拒否を確かめる `mise run check-go-internal-visibility` を追加した。
台帳の移行は次のとおりである（着手時の revision `8b12057ef` の 510 ID から 211 ID）。
| 旧分類 | 旧 ID 数 | 新分類 | 新 ID 数 | 区分 |
| --- | --- | --- | --- | --- |
| `private-import` | 111 | `private-import` | 111 | ID は同一 |
| `undeclared-context-edge` | 240 | なし | 0 | 規則の撤去 |
| `context-cycle` | 66 | `module-cycle` | 57 | 57 は改名。9 は `shared` を経由した到達だけで成立していた循環で、規則の撤去（原因の import は `shared-dependency` として数える） |
| `shared-detour` | 73 | `shared-dependency` | 16 | 分類変更。原因の `shared` パッケージとモジュールの組ごとに数え直した |
| `domain-effect` | 20 | `domain-effect` | 20 | ID は同一 |
| なし | 0 | `table-write` | 7 | 新規。データベース設計に記録済みの 4 種類と一致し、理由に原子性と参照を書いた |
| なし | 0 | `composition-import` | 0 | 新規。既存の違反はなかった |
実装の変更で解消した ID は 0 件である。
後続として、残りの文書の改名（wi-26986）とモジュール一つの `internal/` 移行（wi-50428）を起票し、テナントの公開契約（wi-39119）と負債の棚卸し（wi-33994）を新しい分類と D1〜D8 へ合わせた。

全文は `git show d0267d4be1e827899e42d71fd755e184fbede85c:<このファイルのパス>` で読む。
