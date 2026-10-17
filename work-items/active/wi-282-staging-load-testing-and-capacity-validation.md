---
status: pending  # pending | in_progress | completed | cancelled
authors: [tn]
risk: medium      # low | medium | high | critical
created_at: 2026-07-25  # YYYY-MM-DD
priority: p1
change_kind: operations
affected_spec:
  - { path: docs/modules/system/features/operations.md, requirement: REQ-SYSTEM-001 }
depends_on: []
---

# staging 実負荷で性能・容量の判断を実測し確定する（負荷テスト基盤）

## Motivation

idmagic には specification の非機能 objective（login 応答成功率・latency、introspection の p99 等）と、
実測で確定すべき暫定の実装判断が複数ある。特に wi-278（揮発性状態の PostgreSQL 統合、
`docs/design/data/database.md`）は、高 churn テーブルの **UNLOGGED / LOGGED 選択**と **GC（ephemeral sweep）間隔**を
「staging 実測で確定する」暫定のまま実装を完了しており、この検証だけが未了で残っている。

これらは机上では決められない。write 増幅・autovacuum 負荷・dead tuple 肥大・p99 latency は
実ワークロードでしか測れず、PostgreSQL が唯一のステートフル基盤になった今、その振る舞いを
本番前に把握しておくことは運用上の要である。本 WI は wi-278 固有の検証にとどめず、**再利用可能な
負荷テスト基盤（k6 シナリオ＋測定手順）を整備し、OAuth2/OIDC クリティカルパスの性能ベースラインを
取り、暫定判断を確定する**ことを目的とする。以後の性能回帰検知・容量計画の土台にもする。

## Scope

機能変更ではなく検証・測定であり、specification の feature は変更しない（既存 objective に対する
実測検証）。整備・記録する対象:

- 負荷テストシナリオ（k6 等）: `/authorize`→code→token 交換、introspection 高 RPS、
  device flow、PAR、WebAuthn 登録/認証、ログイン失敗連打（throttle）。
- 測定手順とダッシュボード用クエリ（`pg_stat_user_tables` / `pg_stat_statements` /
  `pg_stat_wal` / `pgstattuple` / アプリの p99 metric）。
- 結果の記録先: 本 WI の `## Completion` の `Evidence` と `the work item Completion`。
- 確定に伴う調整があれば: `infra/schema/postgres.sql` の storage param（UNLOGGED/LOGGED、
  fillfactor、per-table autovacuum 設定）、ephemeral sweep 既定間隔
  （`EPHEMERAL_SWEEP_INTERVAL` / batch）。
- [キャパシティ設計](../../docs/design/performance/capacity.md#想定ワークロード)の全想定ワークロードと、[品質要件](../../docs/requirements/quality.md#キャパシティ受入れ)の全 `CAP-*` を測定対象へ対応付ける。認証とトークンだけでなく、管理、SCIM、監査、worker のレーン、batch、データと保持量、混合負荷での持続能力を対象にする。
- 採用した実測値を、測定条件と証拠への参照付きでキャパシティ設計へ反映し、未測定の仮定値を区別する。測定基盤を整備しただけで、未実測のリスクを解消したことにしない。

## Out of Scope

- specification objective（SLO 値そのもの）の変更。閾値を見直す必要が判明したら別 WI で specification を更新する。
- 恒常的な性能回帰 CI ゲートの構築（本 WI はシナリオと手順の整備＋一度の実測確定まで。CI 常設は将来 WI）。
- 機能追加・アダプタ実装（wi-278 で完了済み）。

## Plan

**方針**: staging を本番相当構成（`PERSISTENCE=postgres`、REGIONAL/HA DB）で起動し、
再現可能な負荷シナリオを流し、DB 内部統計とアプリ metric を突き合わせて暫定判断を確定する。
確定は「暫定どおり（変更なし・実測値を Evidence に記録）」か「調整（storage param / GC 間隔を変更）」の
いずれか。

**測定の主眼**（wi-278 Risk Notes 由来）:
- UNLOGGED 群（auth_request/code/par/device/replay/webauthn/saml replay）: crash/failover で
  TRUNCATE される前提が許容か（in-flight フロー放棄→再開で回復）を実際に検証。
- LOGGED 群（access_token_denylist / login_throttle_counters）: failover 越しに残ること、
  write 増幅・vacuum 負荷が許容範囲か。
- 高 churn テーブルの dead tuple / autovacuum / bloat 推移と、GC 間隔・batch の追随性。
- ホット読取 `AccessTokenDenylist.IsRevoked`（introspection 毎）と CAS 経路
  （Redeem/Consume/Exchange、throttle RecordFailure）の p99。

**未決定**: UNLOGGED/LOGGED の最終値・per-table autovacuum 設定・GC 間隔は実測で確定する。

### キャパシティの仮定値と実測の対応

2026-10-13 の確認では、[システム全体のリスク](../../docs/design/architecture/risks.md)にあるキャパシティの仮定値は未実測である。
本項目はその測定の担当として、既存の認証負荷試験に次の対応と完了条件を加える。

| 対象 | 観測と反映 |
| --- | --- |
| `CAP-TOKEN-THROUGHPUT`、`CAP-AUTHORIZE-THROUGHPUT`、`CAP-INTROSPECT-THROUGHPUT` | 品質要件が定めるウォームアップ、測定時間、同時負荷、既存 SLO の条件で測る |
| テナント、利用者、セッション、各記録と保持量 | 宣言した分布のデータを用意し、実際の行、索引、WAL の量と保持期間の前提を記録する |
| 管理、SCIM、監査を含む API の種別 | 同時に与える混合負荷と種別ごとの到達率、持続能力、p95/p99、エラーを測る。未定義の SLO は新設せず、観測値として記録する |
| worker の実行レーンと batch | 到着率、処理時間、必要枠、滞留と期限切れデータの掃引が追従するかを測る |
| レプリカ、DB 接続とストレージの算出 | 実測した持続能力、データ量、処理時間を算出式へ代入し、想定構成が必要な余裕を持つかを確認する |

すべての仮定値を、測定で置き換えられる値、利用規模の設計入力、測定不能な前提に分類する。
利用規模の設計入力そのものを測定値と呼ばず、性能と容量の判断に使う値は実測する。
測定不能な前提は理由と再検討の条件を記録し、実測が必要な対象の未実施を対象外として完了しない。
測定条件には、日付、Git revision、実行環境、データ分布、負荷構成、試験時間、結果の保存先を含める。
品質目標を測定結果に合わせて暗黙に下げない。
不足が分かった場合は、その是正項目と必要な再測定を記録し、受け入れ目標を満たしていない状態を成功扱いしない。

## Tasks

- [ ] T001 [Infra] staging を `PERSISTENCE=postgres`・本番相当 DB 構成で起動し、負荷投入元
  （k6 runner 等）と観測（pg 統計・p99 metric）を用意する。
- [ ] T002 [Verify] 負荷シナリオを整備・実行する（authorize→token / 高 RPS introspection /
  device / PAR / WebAuthn / ログイン失敗連打）。
- [ ] T003 [Verify] dead tuple / autovacuum / bloat / WAL / p99 を測定し、UNLOGGED/LOGGED と
  GC 間隔（`EPHEMERAL_SWEEP_INTERVAL`/batch）の判断を確定する（wi-278 の暫定を解消）。
- [ ] T004 [Verify] crash/failover 試験で UNLOGGED 群の TRUNCATE 後も in-flight フローが
  再開で回復し、LOGGED 群（denylist/throttle）が残ることを確認する。
- [ ] T005 [DB] 確定結果で必要なら `infra/schema/postgres.sql` の storage param / autovacuum、
  sweep 既定値を調整しコミットする（不要なら「暫定どおり」を Evidence に記録）。
- [ ] T006 [Verify] OAuth2/OIDC クリティカルパスの性能ベースラインと specification objective 適合を
  Evidence に記録する。
- [ ] T007 [Inventory] 全 CAP と想定ワークロードの入力、仮定値、測定対象、測定手順を対応表にする。
- [ ] T008 [Measure] 宣言したデータ分布と混合負荷で API の全種別、worker のレーン、batch、接続とストレージを測り、条件と実測値を記録する。
- [ ] T009 [Docs] 採用した実測値とサイジングをキャパシティ設計へ反映し、リスクを解消した範囲と残る前提を明示する。

## Verification

- 実測メトリクス（dead tuple / autovacuum / bloat / WAL / p99）を Evidence に記録。
- failover 後の回復・残存を実挙動で確認。
- specification objective（login 応答成功率・latency、introspection p99）への適合を数値で確認。
- 調整コミットがあれば `mise run verify` green。
- 全 CAP の達成条件を品質要件の宣言元と照合し、全測定対象の証拠と、仮定値から採用した実測値への対応を記録する。
- ローカル Kubernetes の起動証拠だけを容量と SLO の達成証拠にはしない。測定環境の代表性と差分を明記する。

## Risk Notes

- **測定環境の代表性**: staging が本番と乖離すると判断が外れる。DB tier・レプリカ構成・データ量を
  本番相当に寄せる。乖離があれば Evidence に明記する。
- **破壊試験の副作用**: crash/failover 試験は staging 限定で行い、durable データに影響しない手順とする。
- **確定の先送り防止**: 「暫定どおり」で終える場合も、根拠となる実測値を必ず Evidence に残す
  （測らずに追認しない）。
