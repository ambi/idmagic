---
depends_on: [wi-146-extract-audit-bounded-context, wi-184-transactional-event-log-foundation]
status: pending
authors: ["tn"]
risk: medium
created_at: 2026-07-11
---

# event log から再構築可能な監査検索 read model と保持運用を整備する

## Motivation

監査イベントの原本を event log に統合すると、管理画面の検索・集計に最適化された
`audit_events` を同期書込みする必要はなくなる。一方、原本を JSON payload のみで直接
検索すると、管理画面のフィルタ、期間検索、テナント境界、長期保持を高負荷にし得る。

event log を唯一の再生可能な原本とし、`audit_events` を非同期で更新・再構築できる read model
にすることで、書込み経路を短く保ちながら調査用 UX と保持要件を両立させる。

## Scope

- **dependency**:
  - [[wi-184-transactional-event-log-foundation]] の event log / delivery state を前提とする。
  - audit API / port の ownership は [[wi-146-extract-audit-bounded-context]] に従う。
- **scl**:
  - audit context の models、interfaces、scenarios、objectives に、event log からの
    audit read model 投影、投影遅延、再構築、保持期間を追加する。
  - System context に、監査原本と read model の整合性・復旧手順・保管ポリシーを追加する。
- **persistence / worker**:
  - `audit_events` と search attributes を event ID を一意キーとする idempotent read model にする。
  - projection worker の checkpoint、失敗記録、replay / rebuild 手順を実装する。Kafka 配送状態と
    audit projection 状態を混同しない。
  - event log を `occurred_at` による PostgreSQL range partition の候補として評価し、実測した
    容量・検索パターンが閾値を満たす場合のみ月次 partition を導入する。
  - 長期保持は PostgreSQL 正本を起点に、必要時に immutable object storage へ検証可能な
    export を追加する。時系列 DB / 検索エンジンは正本ではなく非同期投影としてのみ採用する。
- **operations / ui**:
  - read model の最新投影時刻・遅延・失敗を観測可能にし、管理画面または運用 API で再構築を
    実行できるようにする。
  - 既存 admin audit API の URL・認可・レスポンス互換性を維持する。

## Out of Scope

- event log と業務状態の atomic commit（[[wi-184]]）。
- Kafka / 外部 SaaS への配送仕様。
- PostgreSQL から時系列 DB・検索エンジンへの早期移行。導入は実測した容量・分析要件がある場合のみ。
- 法的 WORM 保管を要する本番クラウド事業者・リージョン・鍵管理方式の選定。

## Plan

1. event log を原本、audit_events を projection と定義し、投影が遅延・停止しても原本と
   業務 mutation の可用性を損なわないようにする。
2. event ID による冪等 upsert と checkpoint を用意し、指定期間・全期間の rebuild を可能にする。
   projection worker は Kafka relay とは独立した consumer とし、片方の成功をもう片方の成功と
   見なさない。
3. 検索アクセスは read model の明示的な列と必要最小限の index へ寄せる。PostgreSQL の通常 index は
   同期的に維持されるため、「非同期 index」ではなく非同期 projection を採る。
4. partition / archive / TimescaleDB・ClickHouse・OpenSearch 等の導入は、イベント件数、保持年数、
   p95 検索遅延、ストレージ成長の計測結果で判断する。PostgreSQL は当面の正本として維持する。

## Tasks

- [ ] T001 [SCL] audit / System SCL に projection、rebuild、retention、observability の仕様と
  acceptance scenarios を追加し、生成物を同期する。
- [ ] T002 [Persistence] event ID 一意の audit read model、search attributes、checkpoint / failure
  state の schema と adapter を実装する。
- [ ] T003 [Worker] idempotent projection worker、期間指定 replay / rebuild、遅延・失敗の計測を
  実装する。
- [ ] T004 [API/UI] 既存 admin audit API を read model へ接続し、投影時刻・遅延と再構築の運用導線を
  追加する。既存の閲覧認可と response 互換性を維持する。
- [ ] T005 [Operations] 容量・検索負荷を測定し、partition / archive 導入の閾値と手順を README / ADR に
  記録する。閾値を満たす場合のみ partition を実装する。
- [ ] T006 [Verify] replay、重複、worker 障害、長期期間検索、テナント隔離の回帰を検証する。

## Verification

- `just yaml-check-scl`
- `just scl-render`
- `just yaml-check-work-items`
- `just test-go`
- `just verify-go`
- read model に同じ event ID を複数回投影しても 1 レコードになることを確認する。
- worker 停止中に event log が蓄積し、再開・rebuild 後に audit API の結果が原本と一致することを
  PostgreSQL 結合テストで確認する。
- テナント、event type、期間、検索属性による admin audit 検索と export の既存互換性を確認する。

## Risk Notes

medium。read model を非同期化すると、直後の管理画面には監査イベントが見えない場合がある。
原本 event log から再構築できること、投影遅延を可視化すること、security investigation が必要な
場合は原本照会または投影の catch-up を使えることを保証する。partition や外部 archive を推測で
導入すると運用複雑性だけが増えるため、測定値に基づく後段判断とする。
