---
status: completed
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

## Decision

**psqldef の採用は継続する。安全網(CI 収束チェック)を追加しつつ、将来的な別ツールへの移行は可能性として残す。**
ADR-071 の再検討(Atlas/pgschema 等への乗り換え)は今回は行わない。

### なぜ乗り換えないか

着手時、「pgschema の方がアーキテクチャ上この種のバグを踏みにくいのでは」という再検討を行った。

- pgschema は OSS(Apache-2.0)で function/procedure/trigger/view/RLS/sequence まで無制限にサポートしており、
  Atlas のような Pro gating はない。idmagic が使う `create_default_saml_identity_provider_profile`
  (FUNCTION+TRIGGER)は問題にならない。wi-308 の当初所見が挙げた懸念のうち、Issue #450(GRANT 文を含む
  スキーマで一時インスタンスにロールが無くて `plan` が失敗する)は、`infra/schema/postgres.sql` に
  `GRANT`/`CREATE ROLE` 文が一つもないため無関係。embedded postgres 起動コスト(~2.8秒)も、CI 専用の
  収束チェック用途では無視できる。つまり pgschema を退けた当初の理由は、再検証すると根拠が薄かった。

しかし、upstream (sqldef/sqldef) の実際の状況を調べ直すと、乗り換えを急ぐ理由も同様に薄いことが分かった。

- Discussion #1290(不具合#2: 無名複合UNIQUE制約の命名不一致)は [PR #1292](https://github.com/sqldef/sqldef/pull/1292)
  で報告から**7日**で修正され、**v3.11.17(2026-07-27)**に含まれている。
- Issue #1295(不具合#4: CHECK制約IN句のアルファベット順正規化diff)も**2026-07-30に修正マージ済み**、
  **v3.11.18** に含まれている。
- 2026-07-21〜07-30の9日間で 3.11.16 → 3.11.17 → 3.11.18 と3パッチリリースがあり、2026年1〜7月通算でも
  月3〜4回のペースでリリースが続く、活発に保守されているプロジェクトである。

さらに「なぜ今になってこれだけの不具合が集中したのか」を調べたところ、**古い技術的負債ではなく、進行中の
大規模リファクタリングによる比較的新しいエンバグ**である可能性が高いと分かった。

- 不具合#1(コメント内容依存の順序バグ)が関わる依存関係グラフによる並び替えロジックは、
  [PR #1209](https://github.com/sqldef/sqldef/pull/1209)(**2026-05-16マージ**)で、従来の
  「固定バケツ順」を「トポロジカルソート」へ全面書き換えしたばかりのコード。本 WI の調査(2026-07末)は
  この書き換えのわずか2.5ヶ月後にあたる。
- さらに背景として、sqldef は2025年11月頃(v3.5.0)から、mysqldef/psqldef/sqlite3def 等が個別に持っていた
  パーサーを共通の "generic parser" へ統合する移行を継続中で、2026年半ば(v3.13.0)時点でも
  「generic parser adoption」が進行中と changelog に明記されている。CHECK制約の比較ロジックも
  2025年10月に「文字列比較 → AST比較」へ書き換えられており、同じ刷新の一部とみられる。

この「活発だが今まさに中核ロジックを刷新中」という状態は両刃の剣である。修正は速い(#2・#4 は報告から
1週間程度)反面、刷新が続く限り新しい退行が今後も出る可能性がある。したがって「一度確認すれば終わり」の
人的検証ではなく、**upstream が刷新を続ける間ずっと機能する機械的な安全網(CI 収束チェック)を今回追加し、
psqldef の採用は当面継続する**。ADR-071 の再検討(pgschema 等への乗り換え)は選択肢として残すが、今回は
実行しない — 乗り換える緊急性より、まず安全網を張ることを優先する。

### 決めたこと

1. psqldef の採用を継続する。ADR-071 の再検討は今回は行わない(将来、upstream の刷新が停滞する、または
   安全網をすり抜ける新しい不具合が続くようなら再検討の材料にする)。
2. ピン留めバージョンを `sqldef/psqldef:3.11` → `sqldef/psqldef:3.11.18` に更新する。
3. CI に、空の PostgreSQL に対する apply→dry-run→apply→dry-run の収束チェックを追加する
   (upstream の刷新が続く間、新しい退行を継続的に検知するための恒久的な安全網)。
4. 不具合#1・#3 は upstream 未報告と見られる。issue 報告は本 WI の実装としては行わず、
   Tasks に推奨事項として残す(第三者リポジトリへの公開投稿はユーザーの明示的な承認を要するため)。

## Plan

1. `infra/docker/docker-compose.dev.yaml` の `schema` サービスのイメージを `sqldef/psqldef:3.11.18` に更新する。
2. 不具合#2・#4 の回避策(明示的な制約名、IN句のアルファベット順)を一時的に外し、3.11.18 に対して空DBから
   apply→dry-run を行い実際に収束することを確認してから戻す(upstream 修正の裏取り)。回避策自体は安価で
   無害なため、修正が確認できても `postgres.sql` からは外さない。
3. `infra/schema/check-convergence.sh` を新規作成し、専用の compose project で分離した使い捨て環境に対して
   apply→dry-run(空であること)→apply→dry-run(空であること)を検証する。
4. `justfile` に `check-schema` レシピを追加する(`check:` の集約レシピには含めない)。
5. `.github/workflows/idmagic-ci.yaml` に軽量な新規ジョブを追加し `just check-schema` を実行する。
6. `infra/schema/README.md` に、CIによる自動収束チェックの追加と、不具合#2・#4のupstream修正状況を記録する。
7. `ARCHITECTURE.md` の Persistence 節に、CIがこの収束を強制する旨を1文加える。

## Tasks

- [x] T001 [Decision] CI に apply→dry-run 収束チェックを追加するか、psqldef バージョン追従方針にするか、
      ADR-071 を再検討するかを判断する。→ 上記「Decision」参照: psqldef 継続 + 安全網追加、乗り換えは
      選択肢として保留。
- [x] T002 [Infrastructure] 判断した方針を実装する(バージョン更新・収束チェックスクリプト・justfile・CI・
      README・ARCHITECTURE.md)。
- [ ] T003 [Track] 不具合#1(順序バグ)・#3(CHECK制約無言DROP)を upstream (sqldef/sqldef) にまだ報告して
      いない。#2・#4 の修正ターンアラウンド(報告から約1週間)を踏まえると、報告すれば近く直る見込みがある。
      報告するかどうか・実施時期はユーザーの判断を仰ぐ(第三者リポジトリへの公開投稿のため)。

## Verification

実装時に空DBに対する実地検証を行い、以下を確認した(いずれも `sqldef/psqldef:3.11.18`、専用の使い捨て compose
project `idmagic-schema-check` 上):

- **不具合#2(無名複合UNIQUE制約の命名不一致)は修正済み**: `dynamic_group_rules_tenant_id_group_id_key` の
  明示名を一時的に外し、既に収束済みのDBに対して dry-run したところ `-- Nothing is modified --`。psqldef が
  PostgreSQL 自身の自動命名規則と一致する名前を正しく導出できることを確認。
- **不具合#4(CHECK IN句のアルファベット順diff)は修正済み**: `notification_templates.template_key` のIN句を
  一時的にアルファベット順から崩し、既に収束済みのDBに対して dry-run したところ同じく空。
- **不具合#3(CHECK制約の無言DROP)は依然再現する**: `lifecycle_workflows_enabled_revision_consistency` を
  衝突パターン `..._enabled_revision_check` に戻し、空DBから1回目 `--apply` で正しく作成されることを
  `pg_constraint` で確認した直後、同じ desired state に対する `--dry-run` が
  `DROP CONSTRAINT "lifecycle_workflows_enabled_revision_check"`(ADDなし)を提案し、実際に `--apply` すると
  `pg_constraint` から消えることを再確認した。3.11.18 でも未修正。
- **不具合#1(コメント起因の順序バグ)は今回は再現できず**: `CREATE TABLE jobs` の直前に代表的なコメントを
  1本追加して空DBに apply したが、今回のコメント内容ではエラーは再現しなかった。元の bisection で使われた
  正確なトリガー文字列は既に `postgres.sql` から削除済みで残っておらず、短時間の再試行では再現条件を
  再発見できなかった。`postgres.sql` は引き続き無コメント方針を維持するため、実運用上のリスクは低いままだが、
  upstream での真の修正状況は未確認のまま。
- **`just check-schema` は実際に機能する**: 現状の(全ワークアラウンド適用済みの)`postgres.sql` に対しては
  成功(`schema convergence check passed`)。不具合#3の衝突パターンを一時的に再現させた状態で実行すると、
  期待通り非ゼロ終了で dry-run の diff を表示して失敗することを確認した。
- `just check-compose` は新しい `infra/docker/docker-compose.schema-check.yaml` オーバーレイ追加後も
  引き続き成功。
- `just check`(SCL/YAML/work-item/architecture 検証一式)は成功。SCL変更は伴わない。

## Risk Notes

- 3番の不具合は、今回たまたま気づいたが、レビューなしで自動適用される環境（`docker compose up` での
  ローカル開発）では気づけない形で安全制約が消える。同種の命名パターンを踏む新しい制約が将来追加された場合、
  本 WI の対応（`infra/schema/README.md` の注意書き）だけでは人的レビューに依存し、機械的な防止策ではない。
  → 本 WI の実装(`just check-schema`、CI `schema-convergence` ジョブ)で機械的な防止策を追加したため、この
  リスクは軽減された。ただし CI がその PR/push の対象範囲でしか走らない点、および upstream が現在進行形で
  中核ロジックを刷新中で新しい退行が今後も出うる点(Decision 参照)は残るリスクとして認識しておく。
- 不具合#1・#3は upstream 未報告のままである。psqldef の実際の修正ターンアラウンド(#2・#4 は報告から
  1週間程度)を踏まえると、報告すれば近く直る可能性があるが、報告は本WIでは実行していない(T003参照)。

## Completion

- **Completed At**: 2026-07-31
- **Summary**:
  - psqldef の採用を継続する決定を記録した。当初「pgschema の方が構造的に有望では」という再検討を行ったが、
    実際には upstream (sqldef/sqldef) が活発かつ応答が速く(発見した不具合のうち2件は報告から1週間程度で
    修正リリース済み)、乗り換えを急ぐ根拠は乏しいと判断した。ADR-071 の再検討(ツール入れ替え)は選択肢として
    残すが実行しない。
  - なぜ今回まとめて4件もの不具合が見つかったかを調査し、`sqldef` が2025年11月頃から続く「generic parser」
    統合、および2026年5月にDDL順序ロジックを固定バケツ順からトポロジカルソートへ全面書き換えした
    ([sqldef/sqldef#1209](https://github.com/sqldef/sqldef/pull/1209))ばかりであることを特定した。古い技術的
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
- **開示(未対応・保留事項)**:
  - 不具合#1(コメント起因の順序バグ)は今回の再検証では再現できなかった(元のトリガー文字列は既に削除済みで
    再発見に至らず)。upstreamでの真の修正状況は未確認のまま。`postgres.sql` の無コメント方針が主な防御線。
  - 不具合#1・#3のupstream issue報告は実行していない(T003、第三者リポジトリへの公開投稿のためユーザーの
    判断を仰ぐ)。
  - ADR-071の再検討(pgschema等への乗り換え)は選択肢として保留のままであり、今回は判断・実行していない。
  - Scope通り、SCL変更・psqldef自体の置き換えは対象外。
