---
status: in_progress
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-10
priority: p2
depends_on:
  - wi-55774-move-protocol-application-links-to-table-owners
  - wi-55044-preserve-jobs-rules-in-transactional-csv-reconciliation
change_kind: refactor
spec_impact: { kind: none, reason: "ほかのモジュールのテーブルへの書き込みを、所有者が公開する、呼び出し側のトランザクションに参加する操作の経由へ変えるだけである。書き込む行、原子性、HTTP の応答、ドメインイベントは変えない。" }
---

# テーブルの所有者が、トランザクションに参加する書き込みを公開する

## 動機

`tools/check/boundary-debt.json` の `table-write` の 7 件は、所有者でないモジュールの sqlc のクエリが、ほかのモジュールのテーブルへ書き込むものである。

| 書き込む側 | 所有者 | 書き込み | 同じトランザクションにする理由 |
| --- | --- | --- | --- |
| Application | OAuth2、Saml、WsFederation | 作るアプリケーションとプロトコル設定の結び付け（`application_id`） | 結び付いていないプロトコル設定を残さない |
| IdManagement | Audit | CSV 取り込みの監査の記録 | 確定した取り込みに監査の記録が欠けない |
| IdManagement | Authentication | 取り込んだパスワードの履歴 | 再利用の検査が取り込んだパスワードを見落とさない |
| IdManagement | Jobs | グループの取り込みでの再評価のジョブ | 確定した取り込みに再評価が欠けない |

どれも原子性の理由がある。
公開操作を別のトランザクションで呼ぶ形へ変えるだけでは原子性を保てないので、解消にならない。

## 対象範囲

- 各所有者が、呼び出し側のトランザクションに参加する書き込みの操作を公開する。
- 書き込む側の sqlc のクエリを消し、公開された操作を呼ぶ。組み立て地点で所有者の実装を結ぶ。
- 解消した違反 ID を台帳から消す。

## 対象外

- 書き込む行と原子性の範囲の変更。

## 設計

### 負債の棚卸しでの着手条件

wi-55774 と wi-55044 が完了したので、残る Audit の二つのクエリと Authentication の一つのクエリへ着手する。
Group のクエリはメンバーシップの CSV からも使われるため、その監査の記録も同時に移す。

2026-10-18 に書き込みと公開操作を比較した。
Application の3件は独立した原子性の境界なので wi-55774 へ分ける。
CSV の再評価ジョブは active_jobs の確認と JobEnqueued の発行を省いており、既存のクエリを公開するだけでは REQ-JOBS-002 と REQ-TENANCY-013 に反する操作が残る。
この経路を維持する配置変更は今は行わず、wi-55044 で保証を満たす投入操作を先に設計する。
既存の AuditEventRepository.Append も、CSV の監査と違って自分でトランザクションを開始し、user_id を payload の userId から取るため、そのまま置き換えない。
監査の検索に使う主体と payload の意味、パスワード履歴、ジョブを含めた一行の原子性を保つ範囲は、前提の解消後にまとめて実装する。

[境界の負債の順位付け](../done/wi-33994-reinventory-the-remaining-boundary-debt.md)の判断を引き継ぐ。

- D2：各テーブルの不変条件の検証と更新は所有者に残す。
- D7：原子性の要件は[データベース設計](../../docs/design/data/database.md#所有と書き込みの境界)が定める。参加する更新、トランザクションの所有者（書き込む側）、コミット後の通知を変えない。
- [テナントの公開契約](../done/wi-39119-publish-the-resolved-tenant-as-tenancy-public-language.md)で使った形（所有者が `pgx.Tx` を受け取る構築関数を公開し、書き込む側は構築関数の必須引数として受け取る）を使う。

| 案 | 判断 |
| --- | --- |
| 所有者がトランザクションに参加する書き込みを公開する | 採る |
| 所有者の公開操作を別のトランザクションで呼ぶ | 採らない。片方だけが確定する窓を作る |
| 台帳に残し、検査の例外にする | 採らない。書き込む側が所有者の不変条件を別々に保つ状態が続く |

### 残る三つの書き込みの設計

入力は User と Group の CSV の行の不可分性、パスワード履歴の保持、監査の検索の現行の意味である。
D1、D2、D4、D7 を適用し、履歴の保存と監査の保存の判断を所有者へ残す。
Audit の通常の Append は payload.userId を検索の主体にするが、CSV は User の行では対象 User を、Group と所属の行では行為者を検索の主体にする。
この意味を変更せず、Audit が `RecordWriterInTx(tx pgx.Tx) auditports.AuditEventWriter` を公開する。
`AuditEventWriter.Append(ctx, record *AuditEventRecord, indexedUserID string) error` は検索の主体を明示し、同じ内部の監査の保存処理を使う。
検索属性も Audit が保存し、トランザクションを確定しない。
IdManagement はセル値と行番号を含めない従来の payload を作り、検索の主体を従来どおり渡す。

Authentication は `PasswordHistoryWriterInTx(tx pgx.Tx) passwordports.PasswordHistoryWriter` を公開し、既存の履歴の Add を同じ Tx に結ぶ。
IdManagement は履歴 ID と SQL の列を知らず、対象 User、エンコード済みの値と時刻だけを渡す。
履歴の ID は既存の所有者の Add と同じ DB の UUID 生成を使う。
以前も無作為な UUID であり、呼び出し元は ID を読まないため、保存する履歴と順序の契約は変わらない。

IdManagement の `NewPostgresModule(pool)` は所有する保存先を作り、`WithPostgresImports(pool, PostgresImportDeps)` が行の確定を組み立てる。
PostgresImportDeps は同じ Tx のクォータ、監査、履歴を返す三つの生成関数を必須とし、nil は構築時に拒否する。
既存のメモリの WithMemoryUserImport と同じく、保存先の組み立てと他モジュールを含む確定の組み立てを分ける。
本番の bootstrap で所有者の公開関数を結び、行の確定処理は所有者のルートを import しない。
CSV の依存なしではハンドラーを登録できず、必須の監査だけを省く経路は作らない。

通常の Append を別の Tx で呼ぶ案は原子性と検索の主体の両方を変えるため採らない。
CSV 用の生の INSERT の公開は、不変条件を二か所に残すため採らない。
所有者の共通の保存処理を Tx へ結ぶ案は、保存表現の変更を所有者に閉じ、利用側へ操作と失敗だけを公開できるため採る。
メモリの CSV の既存の不足、イベントの種類と payload、通常の監査の可用性の設計は対象外とする。

## 計画

まず従来の実装で、監査の失敗と履歴の保存の失敗が先行する行、履歴、使用量を戻すことを実 PostgreSQL で固定する。
既存の特性化テストは、依存の組み立てだけを変え、操作列と期待値は変えない。
所有者の Tx 用の操作を先に構造だけのコミットへ分け、次に依存の配線と越境 SQL の削除を行う。
成功の配線には wi-55044 の共通のハンドラーのテストも使い、同じ Tx と監査の検索の主体の維持を確かめる。
原子性の故障には、所有者の書き込み後に失敗させる実 Tx のラッパーと、別 Tx の書き込みの手動注入を使う。
公開 HTTP の変更はなく、構成入口への影響は Go 全体の race 検査、静的検査と文書生成で確かめる。

## タスク

- [x] T001 [Characterize] 各書き込みの原子性を確認した。TestUserImportRollsBackWhenHistoryOrAuditFails が履歴と監査の保存失敗の無変更を、TestMembershipImportRollsBackWhenAuditFails が監査失敗後の所属の無変更を固定する。変更前の user パッケージ39件、group パッケージ23件が実 PostgreSQL で成功した。Group 本体のコミット失敗時の巻き戻しは wi-55044 のテストが固定する。
- [ ] T002 [App] 所有者ごとにトランザクションに参加する書き込みを公開し、書き込む側を書き換える。
- [ ] T003 [Tooling] 解消した違反 ID を台帳から消す。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `mise run check-boundaries`
- `mise run check-boundary-debt-ratchet`
- `mise run verify`

## リスク

- 書き込みを呼び出し側のトランザクションの外で行う誤りは、通常の経路のテストでは見えない。
  別のトランザクションで確定させる変異を手で入れ、特性化テストが検出することを確かめる。
