---
status: completed
---

# CSV の取り込みが Hard Quota で止まったときも QuotaExceeded を記録する

規範仕様の差分はない（`mise run spec-diff`）。
User と Group の CSV の取り込みの適用で、`users` または `groups` の Hard Quota で拒否した行ごとに、`ApplyUserImport` と `ApplyGroupImport` が `QuotaExceeded` を一件発行するようにして、REQ-TENANCY-013 に実装を合わせた。
発行は committer が戻った後に use case が行い、PostgreSQL とメモリーの実装で同じ結果になる。
再評価ジョブの `active_jobs` の拒否は Jobs の投入器が発行済みなので、資源で絞って重ねない。
取り込み一件で一度にまとめる案は、イベントの形を変えるので採らなかった。

着手時に、メモリーの Group の committer が `groups` の使用量を加算も検査も解放もしていないと分かったので、加算、検査、解放を加えた。
メモリーの構成の組み立ては `WithMemoryUserImport` から `WithMemoryImports` に改めた。
メモリーの Group の committer が Group の監査の記録を書かない差は残っており、この項目では扱っていない。
