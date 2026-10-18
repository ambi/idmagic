# IdManagement のデータ

この文書は、IdManagement の Aggregate の永続化、保持と削除を扱う。
カラムと索引の定義はスキーマファイル（`infra/schema/postgres.sql`）が、データの区分ごとの消え方はシステムの[データライフサイクル設計](../../design/data/lifecycle.md)が定める。

## 集約

Aggregate root は `User`、`Group`、`Agent` の三つである。

| Aggregate | 境界の内側にあるもの | 他の Aggregate との関係 |
| --- | --- | --- |
| `User` | 属性（`attributes`）、必須操作 | `Group` のメンバーになる。`Agent` の所有者になる |
| `Group` | `GroupMembership`、`DynamicGroupRule` | メンバーの `User` を ID で参照する |
| `Agent` | `AgentCredentialBinding` | 所有者の `User` または `Group` と、バインドする `OAuth2Client` を ID で参照する |

境界の内側にあるものは、その Aggregate root を経由せずに参照しない。
`User` の実効ロールは、直接付与したロールと、所属する `Group` のロールの和集合である。

### 値オブジェクト

| 値 | 使う項目 | 正規化 | 比較 | 一意性の範囲 |
| --- | --- | --- | --- | --- |
| 名前 | `User` の `preferred_username`、`Group` の `name`、`Agent` の `name` | 前後の空白を除き、残りを入力された表記のまま保存し、表示する | Unicode の case folding をした値どうしで比較し、大文字と小文字を区別しない | 同じテナントの、削除されていない同じ種類の Aggregate |
| メールアドレス | `User` の `email` | 前後の空白を除き、残りを表記のまま保存する。空になる値は設定しない | 名前と同じく、case folding をした値どうしで比較する | 同じテナントの削除されていない `User` |

- **判断**：名前を大文字と小文字を区別せずに比較するのは、主要な IdP（Okta、Entra ID、Google Workspace、AWS IAM）と SCIM の `userName`（`caseExact: false`）に合わせるためである。表記を保つのは、Okta と Entra ID と同じく、利用者が入力した表記を表示するためである。
- **判断**：比較には小文字への変換ではなく case folding（Unicode のデフォルトの caseless matching）を使う。小文字への変換では、`ß` と `SS`、語末の `ς` と `Σ` のように、大文字と小文字だけが異なる表記が一致しない。
- **判断**：メールアドレスをテナントの中で一意にするのは、Google Workspace、Keycloak、Auth0 と同じく、メールアドレスで User を一人に決められるようにするためである。

## 永続化

| Aggregate | 永続化の単位 |
| --- | --- |
| `User` | 一つの Aggregate を一つのレコードとして保存する。属性は疎な対応表として保存し、値を設定したキーだけが領域を使う |
| `Group` | `Group` を経由して保存し、メンバーシップと規則を単独で変えない |
| `Agent` | `Agent` を経由して保存する |

User の状態の一次情報は、`lifecycle.status` と `status_changed_at` の組だけである。
遷移の時刻の記録は、`UserDisabled` や `UserDeleted` などのイベントに残す。

CSV の行の確定は、IdManagement の行、同じ Tx の Tenancy の使用量、Audit の監査、Authentication のパスワードの履歴をまとめて保存する。
他モジュールのテーブルへ直接書かず、所有者の `RecordWriterInTx` と `PasswordHistoryWriterInTx` を組み立て地点が渡す。
PostgreSQL の保存先は `NewPostgresModule` が作り、行の確定は必須の所有者の操作を受け取る `WithPostgresImports` で組み立てる。
監査の検索対象は、User の行では対象 User、Group とメンバーシップの行では行為者とし、payload の意味と区別して明示する。

## 保持と削除

| データ | 保持と削除 |
| --- | --- |
| `User` | 完全削除は物理削除ではなく、Tombstone へ置き換える。`sub` は保持し、再利用しない。ユーザー名の一意性の索引は削除済みの User を除外するので、Tombstone の値は将来の User と衝突せず、解放した名前は再び使える |
| CSV の成果物 | テナント単位の不変な成果物ストアに保存し、上限付きのまとまりで書く。成果物と同じ大きさの単一のデータベース値を作らない。作成から 30 日を過ぎた成果物は、Batch の保持期限の削除が分割片ごと消す（REQ-IDMANAGEMENT-080） |
| データエクスポートのファイル | 状態 `expired` へ遷移すると、ファイル本体を消し、メタデータと監査記録だけを残す |

## 排他と一意性

IdManagement は、Aggregate の保存に楽観的な排他も行のロックも使わない。
リポジトリの保存は、Aggregate 全体を ID で上書きする（`ON CONFLICT (id) DO UPDATE`）。
同時に重なってはならない結果は、次の仕組みで守る。

| 守る結果 | 仕組み |
| --- | --- |
| 名前の一意性 | User、Group、Agent の表に、[名前](#値オブジェクト)の比較キー（Go の一つの関数で作る case folding の値）を保存する列を置き、テナントとその列の組に一意索引を張る。User の索引は削除されていない行に限る。アプリケーションの判定と制約が同じキーを使うので、データベースのロケールに左右されない |
| メールアドレスの一意性 | 制約を置かず、作成と更新のユースケースが比較キーの列で検索して判定する。SCIM の取り込みは重複を書けるためである |
| 同じメンバーシップを二つ作らない | Group と User の組の一意性の制約と `ON CONFLICT DO NOTHING` |
| テナントのリソース上限 | 使用量の加算を、上限の条件付きの一つの更新文で行う。上限を超える加算は行を更新せず、上限の超過として返る |
| 同じ再評価のジョブを重ねない | ジョブの `dedup_key` と、`queued` と `running` のジョブに限った部分一意索引 |
| CSV の適用と同時の変更 | 適用は現在の状態に対して計画をやり直す（[CSV の転送の実現方式](features/csv-transfer.md#実現方式)） |

それ以外の更新は後勝ちである。
二人の管理者が同じ User を同時に読んで更新すると、後に保存した更新が先の更新の項目を上書きする。

## テナントの属性スキーマ

`TenantUserAttributeSchema` は `Tenant` に埋め込まず、`tenant_id` をキーとする独立した Aggregate にする。
スキーマはテナントの設定より頻繁に変わり、テナントを削除するときに明示的なカスケードが要るからである。
