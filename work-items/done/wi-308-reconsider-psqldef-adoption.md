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

### 同種ツール比較（2026-07-30 調査）

T001 の判断材料として、psqldef を含む宣言的／命令的スキーマ管理ツールを比較した。

| ツール | 方式 | 対象DB | ライセンス/課金 | 規模・活発さ | idmagic 適用上の懸念 |
|---|---|---|---|---|---|
| **psqldef**（現行） | 宣言的（diff適用） | Postgres/MySQL/SQLite/SQL Server | OSS（寛容ライセンス） | ★3.1k、Issue管理は GitHub Discussions に移行済み | 本 WI で発見した不具合のうち「無名複合UNIQUE制約の命名不整合」は、他ユーザーからも同時期（2026-07-23）に同一問題が
  [Discussion #1290](https://github.com/sqldef/sqldef/discussions/1290) として報告済みだが、メンテナー未対応（返信0件）。 |
| **Atlas** | 宣言的＋バージョン管理両対応 | Postgres/MySQL/SQLite/SQL Server 他 | OSS（Apache-2.0）＋ Pro（$9/席/月） | ★8.6k、Ariga社の商用OSS | `atlas schema apply`（宣言的モード）自体は無料版で利用できるが、**view/function/trigger/sequence/role/権限は Pro 限定**（`atlas login` 要求）。idmagic のスキーマは
  `create_default_saml_identity_provider_profile` という FUNCTION＋TRIGGER を実際に使用しており、これは現時点で Pro 機能の対象。
  ADR-071 の却下理由は「checkpoint が Pro 限定」だったが、実際の障壁は checkpoint ではなく view/function/trigger の gating にある。 |
| **pgschema** | 宣言的（Terraform風 plan/apply） | PostgreSQL 14–18 のみ | OSS（Apache-2.0）、Bytebase がスポンサーで「課金予定なし」表明 | ★989、2025-06 発足の新しいツール | `plan`/`apply` のたびに `fergusstrange/embedded-postgres` で**実物の PostgreSQL サーバープロセスを起動**し、desired state SQL を実際に適用してから
  `pg_catalog` を読んで比較する設計（独自パーサーを廃止しこの方式に切り替えた経緯が pgschema 自身の `CLAUDE.md` に明記）。
  [Issue #450](https://github.com/pgplex/pgschema/issues/450) に添付された実測ログでは、embedded instance の起動〜接続確立だけで約2.8秒を要しており、
  ローカルで繰り返し `plan` を回す用途には重い。同 issue は「一時インスタンスにはアプリ側ロールが存在しないため、ロールへの GRANT を含む
  desired state で `plan` が失敗する」という embedded 方式起因の既知の制約も示す。
  なお idmagic 自体は unit test・`idmagic-dev-infra`（`backend/shared/storage/testing_postgres/pgtest.go` 他）で既に
  `embedded-postgres`（go.mod: v1.34.0）を使っており「新規の重い依存を持ち込む」わけではないが、CI のスキーマ収束チェックに使うと
  テストとは別に embedded postgres をもう一系統起動することになり、二重運用でコストが積み増しされる点は考慮が必要。 |
| **pistachio** | 宣言的（pg_query_go 使用） | PostgreSQL のみ | OSS（MIT） | ★19、2026-02 発足、更新は活発 | pgschema の重さを理由に開発された対抗馬（[作者の解説記事](https://zenn.dev/kanmu_dev/articles/16789ef1f4283a)）。
  `--check` で drift 検出を exit code に反映でき CI 収束チェックと相性は良いが、実運用実績・コミュニティが最も薄く裏付けが乏しい。 |
| **golang-migrate/migrate** | 命令的（up/down SQLファイル） | 多数 | OSS（MIT） | ★18.8k、最大手 | ADR-071 が脱却した「旧 migration ランナー方式」への回帰そのもの。現在形 schema を履歴の総和から読む問題は解決しない。 |
| **dbmate** | 命令的（up/down SQLファイル） | Postgres/MySQL/SQLite | OSS（MIT） | ★7k | 同上。 |

**所見**：命令的2ツールは ADR-071 の課題意識と逆行するため候補から外れる。pgschema・pistachio は PostgreSQL 専業ゆえ設計はシンプルだが、
どちらも実績が浅い（pgschema は加えて実行コストの懸念あり）。Atlas は無料版でも宣言的 apply 自体は使えるが、idmagic が実際に使っている
FUNCTION／TRIGGER が Pro 限定機能に該当するため、現状では乗り換え候補にならない。ツール入れ替えは現時点で決め手を欠くため、
まず「CI に apply→dry-run 収束チェックを追加する」「upstream（#1290 等）を追跡する」を優先し、ADR-071 の再検討はそれらの状況を見てから
判断するのが妥当と考えられる。

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
