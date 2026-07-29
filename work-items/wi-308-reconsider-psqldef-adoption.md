---
status: pending
authors: [tn]
risk: low
created_at: 2026-07-29
depends_on: []
---

# psqldefの採用継続を再検討する

## Motivation

`just dev-compose` のトラブルシューティング中に、`infra/schema/postgres.sql`（実際のプロダクションスキーマ）に対して
`psqldef`（sqldef/sqldef、ADR-071 で採用）の具体的なバグを複数、実際の PostgreSQL コンテナ上で再現・特定した。
宣言的スキーマ管理ツールの中核的な価値提案は「再適用しても安全に収束すること」であり、そのうち1件は
**実際にデプロイされたデータベースに対して安全制約を無言で消失させる**という、契約違反レベルの不具合だった。
これは奇をてらったエッジケースではなく、ごく普通の複数カラム CHECK 制約の命名でも起きた。

本 WI は「今すぐ psqldef をやめる／使い続ける」を決定するものではない。今回の調査内容を記録し、
CI での機械検証追加、upstream 修正の追跡、または ADR-071 自体の再検討（Atlas 再検討を含む）を
後日判断できるようにするための記録・追跡用 WI である。

## Scope

- 該当なし（本 WI 自体は調査の記録と今後の判断のための track のみ。SCL 変更を伴わない）

## Out of Scope

- psqldef の置き換え自体の実装
- upstream sqldef/sqldef への修正提案・PR

## Design

今回の調査で発見した4件の不具合（sqldef/sqldef、pinned image `sqldef/psqldef:3.11` および `latest` (3.11.17) の両方で確認）。
いずれも本セッション中に `infra/schema/postgres.sql` を修正して回避済み（`infra/schema/README.md` の Rules に
再発防止の注意書きも追加済み）。

1. **CREATE TABLE が自身の CREATE INDEX より後に生成される順序バグ**
   空データベースへの初回 `--apply` で、`CREATE TABLE jobs` が対応する `CREATE INDEX ... ON jobs` より後に
   生成され、`relation "jobs" does not exist` で失敗した。引き金はテーブル直前の SQL コメントの
   **内容（意味ではなく厳密なテキスト）**であることを二分探索で特定したが、正確な発生メカニズムは
   psqldef 側のコードを読んでも完全には特定できなかった。`schema/ddl_ordering.go` の依存関係グラフによる
   並び替え（[sqldef/sqldef#1209](https://github.com/sqldef/sqldef/pull/1209) で導入された比較的新しいロジック）が関与している。
   → 対応: `postgres.sql` から全コメントを削除（設計根拠は ARCHITECTURE.md へ移設）。

2. **無名の複数カラム UNIQUE 制約の命名不整合**
   `UNIQUE (tenant_id, group_id)` のように名前を明示しない制約について、psqldef が生成する名前が
   PostgreSQL 自身の自動命名規則と異なり、かつ**その誤った名前が全テーブルで同一**になるため、
   再適用時に無関係な2テーブルの制約名が衝突し `relation "tenant_id" already exists` で失敗した。
   → 対応: 該当する全ての無名 UNIQUE 制約に明示的な名前を付与。

3. **【重大】CHECK 制約が無言で DROP され、再 ADD されない**
   `lifecycle_workflows_enabled_revision_check` という、複数カラム（`status`, `enabled_revision`）にまたがる
   ごく普通の CHECK 制約が、2回目の `--apply` で `DROP CONSTRAINT` のみが生成され、対応する `ADD CONSTRAINT` が
   一切生成されないことを確認した。`--apply` を実際に実行し、`pg_constraint` から該当制約が消えることを確認済み
   （dry-run の表示上の問題ではない）。
   原因を特定: この制約名が **PostgreSQL 自身が単一カラム `enabled_revision` への無名 CHECK 制約に付ける
   デフォルト名（`<table>_<column>_check`）と偶然完全一致**していた。psqldef はこの命名パターンに一致する
   制約を「暗黙・自動生成された制約」として扱っているらしく、実際には複数カラムにまたがる明示的な制約であるにも
   関わらず、比較ロジックの内部で矛盾を起こしていた。
   → 対応: `lifecycle_workflows_enabled_revision_check` を `lifecycle_workflows_enabled_revision_consistency` に改名。
   ファイル全体を走査し、同様のパターン（`<table>_<実在カラム名>_check` や `_key`）が他にないことを確認済み。

4. **CHECK (col IN (...)) のリスト順序による繰り返しdiff**
   IN リストの記述順がアルファベット順でない場合、再適用のたびに無意味な `DROP CONSTRAINT` / `ADD CONSTRAINT`
   が生成され続ける（実害はないが、dry-run 出力を「本当に変更があったか」の判断材料として使えなくする）。
   PostgreSQL は元の記述順をそのまま保持して返すが、psqldef 側は「desired state」をアルファベット順に
   正規化してから比較するため、ソース側がアルファベット順でないと恒久的に不一致になる。
   これは upstream で既知・未解決の [sqldef/sqldef#1295](https://github.com/sqldef/sqldef/issues/1295)。
   → 対応: `notification_templates.template_key` の IN リストをアルファベット順に並べ替え。

### 検討すべき今後の選択肢

- `just check-compose` 等に「空DBへの `--apply` の後、`--dry-run` が空になること」を機械的に検証する CI ガードを
  追加し、同種の不具合が新しい制約追加時に再発しても気づけるようにする。
- psqldef の upstream リリースを追跡し、#1295 等の修正が入ったバージョンへ計画的に追従する。
- ADR-071 で却下した Atlas（当時は OSS に checkpoint 機能がなく却下）を含め、宣言的スキーマ管理ツールの
  選定自体を再検討する。

## Plan

（本 WI 自体は調査記録であり、実装計画は次にこの WI に着手する際に具体化する）

## Tasks

- [ ] T001 [Decision] CI に apply→dry-run 収束チェックを追加するか、psqldef バージョン追従方針にするか、
      ADR-071 を再検討するかを判断する。
- [ ] T002 [Impl] 判断した方針を実装する。

## Verification

（着手時に定める）

## Risk Notes

- 3番の不具合は、今回たまたま気づいたが、レビューなしで自動適用される環境（`docker compose up` での
  ローカル開発）では気づけない形で安全制約が消える。同種の命名パターンを踏む新しい制約が将来追加された場合、
  本 WI の対応（`infra/schema/README.md` の注意書き）だけでは人的レビューに依存し、機械的な防止策ではない。
