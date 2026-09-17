# PostgreSQL スキーマの作業手順

`postgres.sql` は PostgreSQL アダプターの現在状態を宣言するスキーマである。sqldef ツール群の PostgreSQL 用コマンドである `psqldef` で適用する。

## psqldef のインストール

macOS の場合:

```bash
brew install sqldef/sqldef/psqldef
```

Linux では、sqldef のリリースページからビルド済みの `psqldef` バイナリをダウンロードするか、配備ジョブで sqldef の Docker イメージを使う。CI/CD ジョブでは無指定の最新版を使わず、バージョンを固定する。開発用 Compose の `schema` サービスは `sqldef/psqldef:3.11.20` に固定しているため、CI または Compose の結果を正確に再現するときはローカルでも同じバージョンを使う。

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
2. 変更にデータ移動が必要な場合は、バックフィルまたは値変換のための SQL を `data-migrations/` に追加する。宣言的スキーマファイルにデータ移動を隠さない。
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
mise run dev-compose
```

`schema` は PostgreSQL を待ち、`psqldef --apply --file /schema/postgres.sql` を実行して終了する。その後に `api` が起動する。適用処理は冪等であり、データベースが `postgres.sql` と一致した後に Compose を再実行しても、追加の DDL は生成されない。

スキーマだけを変更し、スタックがすでに動いている場合は、スタック全体を作り直さずに適用する:

```bash
mise run schema-compose
```

適用前に開発用データベースを確認するには、`psqldef` をインストールしたホストからプレビューを実行する:

```bash
psqldef -U idmagic -h localhost -p 5432 idmagic \
  --dry-run < infra/schema/postgres.sql
```

## CI での収束検査

CI はプッシュとプルリクエストのたびに `mise run check-schema` を実行する。破棄可能な空の PostgreSQL データベース (開発スタックではなく、隔離した Compose プロジェクト) へ `postgres.sql` を適用し、「適用 → プレビュー (何もしないこと) → 再適用 → プレビュー (引き続き何もしないこと)」の順で収束を検証する。これは `work-items/done/wi-308-reconsider-psqldef-adoption.md` に記録した psqldef の不具合群に対する、恒久的かつ機械検査可能な防御である。特に、対応する `ADD` を伴わない暗黙の `DROP CONSTRAINT` (下の Rules を参照) は、1 回のプレビューを人が確認するだけでは見落としやすい。ローカルでは次のコマンドで実行する:

```bash
mise run check-schema
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

何を `postgres.sql` に置くか、構造の変更をどう段階に分けるか、`psqldef` の性質から来る規則（制約の命名、`CHECK` の値の順序、`--enable-drop` の扱いなど）とその理由は、[スキーマ管理](../../docs/design/data/schema-management.md)が定める。
ここでは手順に直接かかわる規則だけを置く。

- データ移行、バックフィル、改名は `data-migrations/` に `YYYY-MM-DD-<変更内容>.sql` として置き、スキーマ適用の前後どちらで実行するかと、後退できるかをファイル冒頭に書く。
- `postgres.sql` に SQL コメント (`--`) を書かない。設計上の根拠は `docs/design/data/` に置き、DDL の中で言い直さない。`psqldef` の依存順序の解決へコメントが影響しないようにする目的もある。
- テーブルを追加、削除するとき、テーブル種別（`LOGGED` / `UNLOGGED`）や `tenant_id` 列の区分を変えるときは、`docs/design/data/database.md` の ER 図とテーブル一覧を同じ変更で更新する。`mise run check-schema-tables` が食い違いを検出する。
- 次の規約は設計ではなく SQL の書き方に関するため、このファイルで維持する。これを超える内容は `docs/design/data/database.md` を参照する:
  - テーブル自身の識別子は `id` とする。別のテーブルから `User` を参照する列は `user_id` とし、所有者の参照は `owner_user_id` とする。
  - すべてのテーブルが `created_at` を持つ。作成後に行を更新できるテーブルは `updated_at` も持つが、挿入専用または削除専用の行は持たない。Domain のタイムスタンプ (`issued_at`、`granted_at`、`occurred_at`、`expires_at`、`revoked_at`、`first_seen`、`last_seen`) はそれぞれの意味を維持し、`created_at` の代わりにはしない。
  - 秒精度への丸めは外部プロトコル境界 (SCIM、SAML、WS-Fed の書式化) でのみ行い、スキーマでは行わない。
  - Go では UUID 列を文字列として保持し、`base.go` が UUID の OID にテキストコーデックを登録する。
  - 外部キーではない `tenant_id` 列 (`audit_events`、`authentication_event_buckets`) は `TEXT` のままとし、UUID を文字列で保持する。`audit_events` はテナントなしを表す番兵値 `''` も保持する。
  - `users.lifecycle` は JSONB 正規化の候補として印を付けている。
