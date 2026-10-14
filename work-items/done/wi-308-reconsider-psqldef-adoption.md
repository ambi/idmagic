---
status: completed
archived_from: e43e8fec330f631fe4e470b7b567465d524a751c
---

# psqldefの採用継続を再検討する

- psqldef の採用を継続する決定を記録した。当初「pgschema の方が構造的に有望では」という再検討を行ったが、
実際には upstream (sqldef/sqldef) が活発かつ応答が速く(発見した不具合のうち2件は報告から1週間程度で
修正リリース済み)、乗り換えを急ぐ根拠は乏しいと判断した。ADR-071 の再検討(ツール入れ替え)は選択肢として
残すが実行しない。
- なぜ今回まとめて4件もの不具合が見つかったかを調査し、`sqldef` が2025年11月頃から続く「generic parser」
統合、および2026年5月にDDL順序ロジックを固定バケツ順からトポロジカルソートへ全面書き換えした
(sqldef/sqldef#1209)ばかりであることを特定した。古い技術的
負債ではなく、進行中の刷新による比較的新しいエンバグである可能性が高い。
- `sqldef/psqldef:3.11` → `3.11.18` へピン留めを更新し、不具合#2(無名複合UNIQUE制約の命名不一致)と
#4(CHECK IN句のアルファベット順diff)が実際にこのバージョンで修正済みであることを空DBに対する実地検証で
確認した。不具合#3(CHECK制約の無言DROP)は3.11.18でも再現することを同様に実地確認した(既存の回避策は
全て維持)。
- `infra/schema/check-convergence.sh`(新規)・`just check-schema`・CI `schema-convergence` ジョブを追加し、
空のPostgreSQLに対する apply→dry-run→apply→dry-run の収束を機械的に検証する恒久的な安全網を実装した。
不具合#3の再現パターンに対して実際に失敗を検知することを確認済み。
- `infra/schema/README.md` と `ARCHITECTURE.md` を、upstream 修正状況とCIによる自動検証の追加を反映して
更新した。

全文は `git show e43e8fec330f631fe4e470b7b567465d524a751c:<このファイルのパス>` で読む。
