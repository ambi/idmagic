---
status: in_progress
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-18
priority: p2
depends_on: []
change_kind: refactor
spec_impact: { kind: none, reason: "Application の作成と同じトランザクションで行うプロトコル設定の結び付けを、テーブルの所有モジュールの操作へ移す。結び付く行、テナントの範囲、原子性、HTTP の応答とイベントは変えない。" }
---

# プロトコル設定とアプリケーションの結び付けをテーブルの所有者へ移す

## 動機

wi-88544 のうち Application から OAuth2、Saml、WsFederation の表へ書く3件を独立して解消する。
CSV の書き込みとは別の原子性の境界であり、再評価ジョブのクォータと発行の欠陥を前提としない。

## 対象範囲

- application_id の条件付き更新を、所有する三つのモジュールの sqlc クエリへ移す。
- 各所有者が呼び出し側の pgx.Tx に参加する操作をルートで公開し、組み立て地点が Application の保存先へ結ぶ。
- 解消した3件の台帳を除き、現在の設計を同期する。

## 対象外

- CSV の履歴、監査、ジョブの書き込み。wi-88544 と、その前提の wi-55044 が扱う。
- プロトコル設定の作成、参照の結合、外部キー、削除の cascade の変更。

## 設計

入力は REQ-APPLICATION-007、データベース設計の「所有と書き込みの境界」、ApplicationRepository.Create と三つの Link クエリである。
Application がトランザクションを開始し、Application の保存、未結合の設定の application_id 更新、結び付いたキーの再読を同じトランザクションで行ってから確定する。
失敗すればすべて巻き戻し、再試行、通知、イベントの追加はしない。

| 案 | D1、D2、D3、D4、D7 と変更シナリオ | 判断 |
| --- | --- | --- |
| Application が他の表の SQL を保持する | 原子性は保てるが、プロトコル側の保存規則の変更が Application のクエリへ波及し、D2 に反する | 不採用 |
| 公開の Repository で別に保存する | 原子性を失うか、設定全体の型と更新を Application に漏らす | 不採用 |
| 所有者の条件付き更新を、呼び出し側の Tx で実行する | 所有者が SQL を隠し、Application は結び付けと検証だけを編成する。公開パッケージとモジュール間の import を増やさない | 採用 |

三つのルートは `LinkApplicationInTx(ctx context.Context, tx pgx.Tx, tenantID, applicationID, protocolKey string) error` を公開する。
Application のアダプターは同じシグネチャのコールバック三つを持つ ProtocolLinkers を受け取り、プロトコルに応じて一つを呼ぶ。
`NewPostgresModule(pool, linkers)` で必須として渡し、未構成のコールバックを本番の経路へ出さない。
クエリは今と同じテナントとプロトコルのキーで絞り、application_id が空の場合だけ更新する。
結び付けができなかった場合の再読と拒否は Application に残す。
隠す判断は各プロトコルの保存表現であり、共有ライブラリや新しいモジュールは作らない。

## 計画

1. 作成に失敗したときに Application を残さない既存の原子性を固定する。
2. SQL と更新操作を所有者へ移し、正式な組み立てを結ぶ。
3. 三つのプロトコルの正常系と、更新後に失敗する故障によるロールバックを検証する。

## タスク

- [x] T001 [App] 既存の失敗時の原子性を確認する。
- [x] T002 [App] 所有者の更新と配線へ移す。
- [x] T003 [Tooling] 台帳と現在の設計を同期する。
- [x] T004 [Verify] 既存の保存、HTTP の経路、原子性を検証する。

## 検証

- `mise run test-go-package -- ./backend/application/internal/db_postgres`：正常な結び付け、失敗した作成と所有者の書き込みのロールバック。
- `mise run sqlc-generate` と `mise run check-boundaries`、`mise run check-boundary-debt-ratchet`：所有者、生成物、台帳。
- `mise run lint-go` と `mise run test-go-race`：三つの所有者と正式な組み立ての呼び出し元。
- `mise run test-go-mutation -- backend/application/internal/db_postgres 2 off /tmp/idmagic-wi55774-mutation.json main` の変更行と、所有者の更新後に失敗する手動注入：原子性と配線の故障。所有者の Tx 操作は Application の統合テストから実行し、各所有者パッケージだけの mutation でこの配線の検出能力を主張しない。
- `mise run check`、`mise run render-docs`：現在の設計と仕様影響。

## リスク

Tx の代わりに接続プールを渡すと、未確定の Application を参照できず、原子性も失う。
所有者の更新の成功後にエラーを返し、Application とプロトコルの結び付けの両方が残らないことを確かめる。

## 要約

規範仕様の差分はない。
OAuth2、Saml、WsFederation が application_id の条件付き更新を所有し、起動の組み立てが同じ Tx へ参加する操作を Application に渡す。
元のテナントと未結合の条件、Application 側の再読、確定と巻き戻しの範囲を保ち、台帳の3件を除いた。
三つのプロトコルの正常系、参照先がない作成、所有者の更新直後の故障による両側のロールバックを PostgreSQL で検証する。
Application の変更行の5変異はすべて検出し、生存した変異はない。
CSV の残り4件は、クォータと発行を省く再評価の経路をそのまま公開しないため wi-55044 を前提として wi-88544 に残した。
