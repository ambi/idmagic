# PostgreSQL スキーマの作業手順

`postgres.sql` は PostgreSQL アダプターの現在状態を宣言するスキーマである。sqldef ツール群の PostgreSQL 用コマンドである `psqldef` で適用する。

## psqldef のインストール

macOS の場合:

```bash
brew install sqldef/sqldef/psqldef
```

Linux では、sqldef のリリースページからビルド済みの `psqldef` バイナリをダウンロードするか、配備ジョブで sqldef の Docker イメージを使う。CI/CD ジョブでは無指定の最新版を使わず、バージョンを固定する。開発用 Compose の `schema` サービスは `sqldef/psqldef:3.11.18` に固定しているため、CI または Compose の結果を正確に再現するときはローカルでも同じバージョンを使う。

インストールしたコマンドを確認する:

```bash
psqldef --version
```

## 接続変数

ローカル開発用 Compose に相当する接続設定は次のとおり:

```bash
export PGHOST=localhost
export PGPORT=5432
export PGUSER=idmagic
export PGPASSWORD=idmagic
export PGDATABASE=idmagic
```

`psqldef` は `psql` 形式の接続オプションを使う。本番の配備ジョブでは、この手順を実行する前に `DATABASE_URL` シークレットを `PGHOST`、`PGPORT`、`PGUSER`、`PGPASSWORD`、`PGDATABASE` へ対応付ける。

## 変更手順

1. `infra/schema/postgres.sql` を望ましい現在のスキーマへ編集する。
2. 変更にデータ移動が必要な場合は、バックフィルまたは値変換のための明示的なランブックか専用の SQL スクリプトを追加する。宣言的スキーマファイルにデータ移動を隠さない。
3. 適用せずに予定される DDL を生成する:

```bash
psqldef -U "$PGUSER" -h "$PGHOST" -p "$PGPORT" "$PGDATABASE" \
  --dry-run < infra/schema/postgres.sql \
  | tee /tmp/idmagic-schema-plan.sql
```

4. `/tmp/idmagic-schema-plan.sql` をレビューする。
   - 出力が空なら、データベースはすでに現在のスキーマと一致している。
   - `DROP` 操作には人による明示的なレビューが必要であり、自動化ではデフォルトで有効にしない。
   - 長時間ロックする操作、型の変更、データのあるテーブルへの `NOT NULL` 追加には、別の展開計画が必要である。
5. レビュー済みのスキーマ変更を適用する:

```bash
psqldef -U "$PGUSER" -h "$PGHOST" -p "$PGPORT" "$PGDATABASE" \
  --apply < infra/schema/postgres.sql
```

6. もう一度プレビューを実行する。出力が空になることを期待する:

```bash
psqldef -U "$PGUSER" -h "$PGHOST" -p "$PGPORT" "$PGDATABASE" \
  --dry-run < infra/schema/postgres.sql
```

7. 生成した計画と、最後のプレビューが空だったことを作業項目の完了記録またはリリース証跡に残す。

## Docker を使ったローカル開発

開発用 Compose ファイルには、1 回だけ実行する `schema` サービスがある:

```bash
just dev-compose
```

`schema` は PostgreSQL を待ち、`psqldef --apply --file /schema/postgres.sql` を実行して終了する。その後に `idp` が起動する。適用処理は冪等であり、データベースが `postgres.sql` と一致した後に Compose を再実行しても、追加の DDL は生成されない。

スキーマだけを変更し、スタックがすでに動いている場合は、スタック全体を作り直さずに適用する:

```bash
just schema-compose
```

適用前に開発用データベースを確認するには、`psqldef` をインストールしたホストからプレビューを実行する:

```bash
psqldef -U idmagic -h localhost -p 5432 idmagic \
  --dry-run < infra/schema/postgres.sql
```

## CI での収束検査

CI はプッシュとプルリクエストのたびに `just check-schema` を実行する。破棄可能な空の PostgreSQL データベース (開発スタックではなく、隔離した Compose プロジェクト) へ `postgres.sql` を適用し、「適用 → プレビュー (何もしないこと) → 再適用 → プレビュー (引き続き何もしないこと)」の順で収束を検証する。これは `work-items/done/wi-308-reconsider-psqldef-adoption.md` に記録した psqldef の不具合群に対する、恒久的かつ機械検査可能な防御である。特に、対応する `ADD` を伴わない暗黙の `DROP CONSTRAINT` (下の Rules を参照) は、1 回のプレビューを人が確認するだけでは見落としやすい。ローカルでは次のコマンドで実行する:

```bash
just check-schema
```

## 本番環境への配備

アプリケーションは起動時にスキーマ変更を適用しない。本番では、新しいアプリケーションバージョンを起動する前に、明示的な配備手順として適用する。

空のデータベースへの初回配備:

```bash
psqldef -U "$PGUSER" -h "$PGHOST" -p "$PGPORT" "$PGDATABASE" \
  --dry-run < infra/schema/postgres.sql
psqldef -U "$PGUSER" -h "$PGHOST" -p "$PGPORT" "$PGDATABASE" \
  --apply < infra/schema/postgres.sql
psqldef -U "$PGUSER" -h "$PGHOST" -p "$PGPORT" "$PGDATABASE" \
  --check < infra/schema/postgres.sql
```

2 回目以降の配備も同じ順序を使う。`--dry-run` は、既存のデータベースを新しい望ましいスキーマへ移行するために必要な DDL を示す。レビュー後に `--apply` で適用し、新しいアプリケーションバージョンを昇格する前に、`--check` が保留中の DDL なしを返さなければならない。

プレビューに破壊的な変更や長時間ロックする変更が含まれる場合は停止し、別の展開計画を作成する。そのリリースについて明示的な承認を得ずに、自動化された本番ジョブへ `--enable-drop` を追加してはならない。

## 空のデータベースの初期化

新しい PostgreSQL データベースには、同じ `--apply` コマンドで `postgres.sql` を直接適用する。参照データはこのファイルに含めない。デフォルトテナントなどの必須行は、アプリケーションが起動時に収束させる。

## 規則

- 構造的なスキーマは `postgres.sql` に置く。
- データ移行、バックフィル、危険性の高い破壊的変更は、`infra/schema/` の外に明示的な SQL スクリプトまたはランブックとして置く。
- このファイルに参照データを置かない。デフォルトテナントなどの必須参照データは、アプリケーションが起動時に収束させる。
- アプリケーション起動時の移行ランナーを再導入しない。スキーマ変更は配備時の操作である。
- レビュー済みの移行計画なしに、自動化で `--enable-drop` を使わない。
- `postgres.sql` に SQL コメント (`--`) を書かない。理由は独立して 2 つある:
  - 設計上の根拠は DDL ファイルではなく、責務を持つ `spec/**/SPECIFICATION.md` に置く。「なぜ」を言い直すコメントは、意思決定を複製した場合と同じように正本の設計記録からずれる。列型の規則と `tenant_id` の保持区分は `spec/SPECIFICATION.md` の `## Cross-cutting Concerns` > Database design policy を参照し、ここでは繰り返さない。
  - `postgres.sql` にコメントを含めない。`psqldef` の依存順序の解決へコメントが影響しないようにするためである。
- 複数列を対象とする制約には、PostgreSQL が無名の単一列制約へ付けるデフォルト名と衝突しない明示名を付ける。`UNIQUE` の `<table>_<column>_key` と `CHECK` の `<table>_<column>_check` に相当する名前を避け、たとえば整合性制約には `..._consistency` を使う。衝突すると `psqldef` が制約を自動生成扱いし、再適用時に対応する追加なしで削除する可能性があるためである。`just check-schema` はこの問題を検出する安全網であり、命名規則の代替ではない。
- すべての `CHECK (col IN (...))` の値一覧をアルファベット順に書く。意味のない制約の再作成を避け、`psqldef` のプレビューを実際の変更を示す信頼できる信号に保つためである。
- 次の規約は設計ではなく SQL の書き方に関するため、このファイルで維持する。これを超える内容は `spec/SPECIFICATION.md` を参照する:
  - テーブル自身の識別子は `id` とする。別のテーブルから `User` を参照する列は `user_id` とし、所有者の参照は `owner_user_id` とする。
  - すべてのテーブルが `created_at` を持つ。作成後に行を更新できるテーブルは `updated_at` も持つが、挿入専用または削除専用の行は持たない。Domain のタイムスタンプ (`issued_at`、`granted_at`、`occurred_at`、`expires_at`、`revoked_at`、`first_seen`、`last_seen`) はそれぞれの意味を維持し、`created_at` の代わりにはしない。
  - 秒精度への丸めは外部プロトコル境界 (SCIM、SAML、WS-Fed の書式化) でのみ行い、スキーマでは行わない。
  - Go では UUID 列を文字列として保持し、`base.go` が UUID の OID にテキストコーデックを登録する。
  - 外部キーではない `tenant_id` 列 (`audit_events`、`authentication_event_buckets`) は `TEXT` のままとし、UUID を文字列で保持する。`audit_events` はテナントなしを表す番兵値 `''` も保持する。
  - `users.lifecycle` は JSONB 正規化の候補として印を付けている。
