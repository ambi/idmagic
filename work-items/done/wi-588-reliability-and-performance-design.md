---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-16
priority: p2
depends_on: [wi-583-normalize-design-document-terminology]
change_kind: docs
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: "既存の設計文書の記述を厚くするだけで、利用者が観測する振る舞い、設定項目、移行手順はどれも変わらない。"
  references: []
spec_impact:
  kind: none
  reason: "可用性、復旧、キャパシティ、スケーリングの設計記述を厚くするだけで、目標値、構成ファイル、実装は変えない。"
initial_context:
  specification:
    - docs/design/reliability/availability.md
    - docs/design/reliability/recovery.md
    - docs/design/reliability/README.md
    - docs/design/performance/capacity.md
    - docs/design/performance/scaling.md
    - docs/design/performance/README.md
    - docs/architecture/deployment.md
    - docs/design/infrastructure/platform.md
    - docs/design/infrastructure/network.md
    - docs/design/data/lifecycle.md
    - docs/contexts/system/internals.md
    - docs/runbooks/backup-restore-dr.md
    - docs/requirements/quality.md
    - DOCUMENTATION_GUIDE.md
  typespec: []
  source:
    - infra/k8s/base
    - infra/k8s/overlays/prod
    - infra/k8s/monitoring/prometheus-rule.yaml
    - infra/backup/README.md
    - backend/shared/resilience/circuitbreaker.go
    - backend/shared/storage/db_postgres/base.go
    - backend/shared/http/support_http/admission.go
    - backend/shared/http/server_http/health_handler.go
    - backend/cmd/internal/bootstrap/apiconfig.go
    - backend/cmd/internal/bootstrap/sharedconfig.go
    - backend/jobs/domain/job.go
    - backend/jobs/usecases/runner.go
    - backend/sharedsignals/usecases/deliver.go
    - tools/check/src/terminology.ts
  tests: []
  stop_before_reading:
    - spec
    - infra/schema/postgres.sql
    - frontend/src
---

# 可用性・復旧・キャパシティ・スケーリングの設計を読める水準まで書く

## Motivation

信頼性と性能の四文書は、いずれも判断の結論だけを持ち、結論に至る設計を持たない。

[可用性設計](../../docs/design/reliability/availability.md)は 26 行である。障害単位を列挙し、PostgreSQL の自動フェイルオーバーとゾーン配置が未実装だと述べ、具体化は [[wi-165-high-availability-and-failover-resilience-topology]] に委ねている。障害単位ごとに何が検知し、何が影響を受け、どう切り替わるかは書かれていない。

[復旧設計](../../docs/design/reliability/recovery.md)は 21 行である。`infra/backup/` が何を実装しているかを述べるが、データの種類ごとの取得方式、RPO、RTO、復元試験の頻度という `DOCUMENTATION_GUIDE.md` §9.3 が求める表を持たない。

[キャパシティ設計](../../docs/design/performance/capacity.md)は 119 行あり、数値も算出式も揃っている。問題は語彙で、「参照運用プロファイル」「構成算出規則」「縮退順序」がそれぞれ workload profile、sizing、load shedding を指していることが題名から読めない。語彙は [[wi-583-normalize-design-document-terminology]] が直すが、算出の前提と、実測が前提を外れたときに何を見直すかの記述はこの work item が足す。

[スケーリングと過負荷の設計](../../docs/design/performance/scaling.md)は 43 行である。「入場制御」という節名が何の機構かを説明せず、拒否の順序は書かれているが、どの層が何を測って判断するか、オートスケールの条件、バックプレッシャーの伝わり方、下流が弱ったときに呼び出しを止める仕組みは書かれていない。

## Scope

- [可用性設計](../../docs/design/reliability/availability.md)に、障害単位ごとの検知、影響、切り替え、確認方法を書く。
- [復旧設計](../../docs/design/reliability/recovery.md)に、データの種類ごとの取得方式、RPO、RTO、復元試験の頻度、復元順序を書く。
- [キャパシティ設計](../../docs/design/performance/capacity.md)に、算出の前提と、実測が前提を外れたときの見直し手順を書く。
- [スケーリングと過負荷の設計](../../docs/design/performance/scaling.md)の題名を「スケーリング・負荷設計」にし、アドミッションコントロールの機構、オートスケールの条件、再試行とバックプレッシャーを書く。
- 両ディレクトリの索引を追従させる。

## Out of Scope

- 目標値の追加と変更。SLO、RPO、RTO、`CAP-*` は [品質要求](../../docs/requirements/quality.md)が正本である。
- 高可用性構成の実装と障害試験。[[wi-165-high-availability-and-failover-resilience-topology]] が扱う。
- 負荷試験の実施と実測値の取得。[[wi-282-staging-load-testing-and-capacity-validation]] が扱う。
- データ層の拡張。[[wi-164-data-tier-scalability-partitioning-read-replica-pooling]] が扱う。
- 実作業の手順。[runbook](../../docs/runbooks/) が持つ。
- 監視とアラートの設計。[[wi-589-observability-design]] が扱う。

## Design

### 可用性

障害単位の一覧は既にある。足りないのは、単位ごとに次の四つが決まっていることである。

| 列 | 内容 |
| --- | --- |
| 検知 | 何がその障害に気付くか（プローブ、メトリクス、外形監視） |
| 影響 | 利用者から何が見えなくなるか。どの経路が生き残るか |
| 対処 | 自動で切り替わるか、人が判断するか。切り替えに要する時間の設計上の想定 |
| 現状 | 実装済みか、参照構成にあるだけか、未実装か |

現状の列を持たせるのは、いま `availability.md` が「Kubernetes 構成ファイルは複数レプリカと PDB と HPA を持つが、PostgreSQL の自動フェイルオーバーは実装しない」と散文で述べていることを、行ごとの事実にするためである。

PostgreSQL の可用性は、選択肢（マネージドサービスの高可用性構成、ストリーミングレプリケーションと手動昇格、単一インスタンス）と、それぞれが何を提供し何を提供しないかを書く。どれを採るかはデプロイ先で決まるため、[[wi-584-ground-deployment-design-in-reference-profiles]] が定めるデプロイプロファイルごとに現状を書く。

### 復旧

`DOCUMENTATION_GUIDE.md` §9.3 が求める形に合わせる。

| 列 | 内容 |
| --- | --- |
| データの種類 | 業務データ、短命な認証状態、監査イベント、鍵素材、資産、構成 |
| 取得方式 | 論理バックアップ、PITR、外部サービスの機能、再生成 |
| RPO / RTO | 目標は [品質要求](../../docs/requirements/quality.md)を参照。未確定は未確定と書く |
| 復元試験 | 頻度と、何を確かめるか |

保護を三段（論理削除、バックアップと復元、早期検知）で書く既存の構成は維持し、段ごとに何の失われ方を受け持つかを明示する。
早期検知の段が現在ほぼ空であることを、欠落として書く。破損に気付くのが保持期限の後になると戻す先が残らない。

鍵素材の復旧を独立して扱う。マスター鍵を失うと暗号化データは復旧できず、これは `DestroyTenantDataKey` が意図して実現する暗号学的消去と同じ結果が事故として起きることを意味する。[シークレットと鍵の設計](../../docs/design/security/secrets.md)が既に述べているこの事実を、復旧設計の側からも到達できるようにする。

### キャパシティ

数値と式は現状のまま維持する。足すのは次の二つである。

- **算出の前提**。各 Planning assumption がどこから来たか（全ユーザーの何%、同種プロダクトの一般値、上限としての仮置き）。前提の出どころが無い数は、実測で置き換えるときに何と比べればよいかが決まらない。
- **見直しの契機**。実測が前提を上回った、あるいは下回ったときに何を再計算するか。特にレプリカ上限を増やす前に PostgreSQL の接続予算を再計算するという既存の規則を、見直し手順として書く。

### スケーリング・負荷

題名を「スケーリング・負荷設計」にする。
「過負荷」だけを名指すと、正常時のスケールの設計がこの文書に属さないように読める。

アドミッションコントロールの節に、機構として次を書く。

- どこで測るか（実行中リクエスト数）、何と比べるか（構成スキーマが持つ上限）、何を返すか（503 と `Retry-After`）。
- 優先度クラス（`management_bulk`、`management`、`interactive_auth`）の割り当てが `ROUTE_PRIORITY.md` から生成されること。
- ハンドラーへ到達する前に拒否する理由。過負荷時に認証、認可、テナント境界、再送防止を部分実行しないこと。
- [キャパシティ設計](../../docs/design/performance/capacity.md)のロードシェディング順序との対応。段一と段二が Worker の実行レーン、段三以降が API のアドミッションコントロールで実現されること。

さらに次を足す。

- **オートスケールの条件**。HPA が何の指標で増減するか、その指標が飽和を表しているか、下限と上限を何で決めるか。
- **バックプレッシャー**。受け付けを絞ったことが上流（ゲートウェイ、クライアント）へどう伝わるか。
- **下流が弱ったときの遮断**。`backend/shared/resilience` が持つ再試行の規則と、遮断の仕組みを持つかどうかの現状。
- **再試行の所有**。どの層が再試行してよいかを一か所で決めるという既存の規則を、層の一覧として書く。

## Plan

1. 障害単位ごとの検知・影響・対処・現状を、構成ファイルと実装から読み取って埋める。
2. データの種類ごとの復旧表を作り、未確定の欄を未確定として残す。
3. キャパシティ設計へ算出の前提と見直しの契機を足す。
4. スケーリング・負荷設計を書き直し、アドミッションコントロールの機構とオートスケールの条件を足す。
5. 索引と、この四文書を参照している記述を追従させる。

## Tasks

- [x] T001 [Design] 障害単位ごとの検知、影響、対処、現状を構成ファイルと実装から読み取る。
- [x] T002 [Docs] 可用性設計を書き直す。
- [x] T003 [Docs] 復旧設計にデータの種類ごとの表と保護三段を書く。
- [x] T004 [Docs] キャパシティ設計に算出の前提と見直しの契機を足す。
- [x] T005 [Docs] スケーリング・負荷設計を書き直し、題名を改める。
- [x] T006 [Docs] 索引と参照元を追従させる。
- [x] T007 [Verify] リンク、SLO 参照、監視構成ファイル、仕様、全体検証を通す。

## Verification

- `mise run check-links`
- `mise run check-slo-references`
- `mise run check-spec`
- `mise run verify`

## Risk Notes

設計上の想定を検証済みと読める書き方にすると、この文書は無いより悪くなる。切り替え時間、RPO、RTO のような値は、目標なのか実測なのか参照構成の公称値なのかを行ごとに区別する。現在の文書が守っているこの区別を緩めない。

未実装を「未実装」と書くだけでは、何が無いのかが伝わらない。欠落は、それが無いことで何が守られていないかとともに書く。早期検知の段が空であることが典型である。

目標値を設計文書へ写すと、[品質要求](../../docs/requirements/quality.md)と二重になる。値は参照し、この四文書には機構だけを書く。

題名の変更はアンカーと索引を変える。`docs/design/performance/README.md`、`docs/architecture/deployment.md`、`docs/design/reliability/availability.md` がこの文書を名指しているため、`mise run check-links` で確かめる。

## Completion

- **Completed At**: 2026-09-18
- **Summary**:
  信頼性と性能の四文書が、結論だけでなく、結論に至る機構と現状を読める水準になった。
  可用性設計は、障害単位ごとに検知、影響、対処、現状を表で持ち、現状を「構成ファイルあり」「設計上の想定」「未実装」に分けて書く。`/livez` がつねに成功すること、ワーカーが Probe を持たないこと、Pod の分散配置の制約がないこと、外形監視がないことを、それぞれが守っていないものとともに欠落として書いた。PostgreSQL の可用性は三つの手段と、デプロイプロファイルごとの選択を持つ。
  復旧設計は、利用者の指定により「リカバリ設計」へ改題した。本文の行為としての「復旧」は残した。保護の三段が受け持つ失われ方、早期検知の段がほぼ空であること、データの種類ごとの取得方式、RPO と RTO（未確定）、復元試験の頻度、鍵素材の喪失、ロードシェディング順序の逆としての復元の順序を持つ。
  キャパシティ設計は、前提ごとの出どころと置き換えに使う実測、見直しの契機、レプリカ上限を上げる手順を持つ。`DB_MAX_CONNS` のデフォルト 20 が計算式の仮定値と一致しないことを記録した。
  「スケーリングと過負荷の設計」は「スケーリング・負荷設計」へ改題し、スケーリング単位、HorizontalPodAutoscaler の条件と CPU 使用率が飽和を表さない場合、アドミッションコントロールの機構とロードシェディング順序との対応、バックプレッシャー、下流の遮断、再試行を所有する層の表を持つ。どの再試行も間隔にばらつきを付けていないことを欠落として書いた。
  目標値、構成ファイル、実装、規範文書は変えていない。`mise run spec-diff` は main に対して規範の変更なしと報告した。
  書いた欠落の解消（Probe、分散配置、外形監視、再試行のばらつき、早期検知、アドミッションの拒否をオートスケールの入力にすること、ランブックの起動順序）は、この記録では扱っていない。
- **Acceptance RED Evidence**:
  - **Test**: N/A: 文書だけの変更で、観測できるプロダクトの境界を持たない
  - **Requirement**: N/A: 規範の要求を変えない
  - **Observed Failure**: 代わりに `mise run check-links` へ故障を注入した。`scaling.md` のリンク `capacity.md#見直しの契機` を存在しないアンカーへ書き換えると、`Markdown anchor does not exist in docs/design/performance/capacity.md` で終了コード 1 になった。元に戻して通ることを確かめた
  - **Detection Reason**: 改題と新しい節へのアンカーが、参照元の追従漏れで切れたときに検出されることを確かめた
- **Unit RED Evidence**:
  - **Test**: N/A: 文書だけの変更で、単体の境界を持たない
  - **Requirement**: N/A: 規範の要求を変えない
  - **Observed Failure**: N/A: 上記の `check-links` への故障注入を代わりの検査とした
  - **Detection Reason**: N/A: 同上
- **Change-Resistance Results**:
  `risk: low` の文書変更なので変異試験は行っていない。記述の事実は、`infra/k8s/`、`backend/shared/resilience/`、`backend/shared/http/support_http/admission.go`、`backend/cmd/internal/bootstrap/`、`backend/jobs/`、`backend/cmd/idmagic-batch/restore_consistency.go` と照合した。照合の途中で、サーキットブレーカーの半開状態の説明と、整合性検査が確かめる内容の説明を実装に合わせて直した。
- **Verification Results**:
  - `mise run check-links` - passed（855 文書）
  - `mise run check-slo-references` - passed
  - `mise run check-terminology` - passed（226 文書）
  - `mise run check-work-item-references` - passed（183 文書）
  - `mise run check-spec` - passed
  - `mise run spec-diff` - no normative specification change against main
  - `mise run verify` - passed
