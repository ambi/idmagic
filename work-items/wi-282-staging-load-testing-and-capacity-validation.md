---
status: pending  # pending | in_progress | completed | cancelled
authors: [tn]
risk: medium      # low | medium | high | critical
created_at: 2026-07-25  # YYYY-MM-DD
priority: p1
change_kind: operations
affected_spec:
  - { path: docs/contexts/system/scenarios.md, requirement: REQ-SYSTEM-001 }
depends_on: []
---

# staging 実負荷で性能・容量の判断を実測し確定する（負荷テスト基盤）

## Motivation

idmagic には specification の非機能 objective（login 応答成功率・latency、introspection の p99 等）と、
実測で確定すべき暫定の実装判断が複数ある。特に wi-278（揮発性状態の PostgreSQL 統合、
`docs/persistence.md`）は、高 churn テーブルの **UNLOGGED / LOGGED 選択**と **GC（ephemeral sweep）間隔**を
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

## Verification

- 実測メトリクス（dead tuple / autovacuum / bloat / WAL / p99）を Evidence に記録。
- failover 後の回復・残存を実挙動で確認。
- specification objective（login 応答成功率・latency、introspection p99）への適合を数値で確認。
- 調整コミットがあれば `mise run verify` green。

## Risk Notes

- **測定環境の代表性**: staging が本番と乖離すると判断が外れる。DB tier・レプリカ構成・データ量を
  本番相当に寄せる。乖離があれば Evidence に明記する。
- **破壊試験の副作用**: crash/failover 試験は staging 限定で行い、durable データに影響しない手順とする。
- **確定の先送り防止**: 「暫定どおり」で終える場合も、根拠となる実測値を必ず Evidence に残す
  （測らずに追認しない）。
