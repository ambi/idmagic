---
status: pending
authors: [tn]
risk: medium
created_at: 2026-07-25
priority: p1
depends_on: []
change_kind: operations
initial_context:
  source:
    - infra/k8s/monitoring/prometheus-rule.yaml
    - infra/docker/prometheus-rules.yml
    - docs/operations
  tests:
    - infra/k8s
  stop_before_reading:
    - backend
    - frontend
affected_spec:
  - { path: spec/contexts/system/main.tsp, symbol: IdMagic.Contract.MetricsExposition }
---

# アラートごとの運用 runbook を整備し、アラートから手順へ辿れるようにする

## Motivation

観測とアラートの土台は既にある。[[wi-112-prometheus-metrics-and-authentication-golden-signals]]
で golden signals が出ており、`infra/k8s/monitoring/prometheus-rule.yaml` には SLO ベースの
アラートが 8 件定義されている (TokenErrorRateBudgetBurn / LoginLatencyBudgetBurn /
LoginThrottleHitRatioHigh / JobsLatencySensitiveClaimLatencyHigh など)。

しかし **アラートが鳴ったときに何をするかが書かれていない**。
`docs/operations/` にあるのは `tenant-quotas.md` 1 本だけで、アラート定義に
`runbook_url` annotation も無い。結果として:

1. **オンコールが機能しない**。深夜に `TokenErrorRateBudgetBurn` が鳴っても、
   最初に何を見るか (どのダッシュボード / どのログ / どの `mise` タスク) が
   コードベースの知識に依存する。IdP は停止が全依存システムのログイン停止に直結するため、
   復旧の初動が属人的なのは可用性目標 (`OverallAvailability` 99.9%) と整合しない。
2. **エンタープライズ調達で不足を突かれる**。運用引き渡し (運用移管) の審査では
   「アラート → 手順 → エスカレーション」の一式が要求される。Okta / Entra は
   サービスとしてこれを内部化しており、自社運用する IdMagic は文書で示す必要がある。
3. **他の WI が前提にしている**。[[wi-101-backup-restore-and-disaster-recovery]] は DR runbook を、
   [[wi-165-high-availability-and-failover-resilience-topology]] は縮退マトリクスを扱うが、
   日常的なアラート対応の runbook はどの WI にも属していない。

本 WI は「定義済みアラート 1 件ごとに runbook が存在し、アラート本体から
`runbook_url` で辿れ、内容が実際のコマンドで検証されている」状態を作る。

## Scope

- **decision**:
  - `decisions.md` へ新しい決定は起こさない。運用文書の構造とアラート命名規約は本 WI の成果物
    (`docs/operations/README.md`) に置き、`docs/observability.md` の
    観測インタフェース方針を参照する。
- **specification**:
  - `System.interfaces.MetricsExposition` の記述に「各 SLO アラートは runbook 参照を持つ」
    という運用要件を追記する。
  - `System` に `scenarios` として「SLO 逸脱アラート発火時、オンコールは runbook を辿って
    初動判断ができる」を追加する (検証可能な形にするため、runbook の存在と参照整合性を
    CI で検査する対象として書く)。
  - 既存 objectives (OAuth2 / Authentication / SigningKeys / Tenancy / Jobs) と
    アラートの対応表を保守できるよう、objective 名をアラート annotation に含める規約を書く。
- **operations / infra**:
  - `docs/operations/` に定義済みアラート 1 件ごとの runbook を追加する。各 runbook は
    固定構成にする: **影響 (ユーザーに何が起きているか) / 確認 (見るべき metric・ログ・
    エンドポイント) / 切り分け (分岐条件) / 一次対応 (実行するコマンド) /
    エスカレーション (誰に・何を渡すか) / 誤検知条件 / 関連 objective と WI**。
  - 対象アラート: `TokenErrorRateBudgetBurn` / `TokenLatencyBudgetBurn` /
    `LoginErrorRateBudgetBurn` / `LoginLatencyBudgetBurn` / `LoginThrottleHitRatioHigh` /
    `JobsLatencySensitiveClaimLatencyHigh` および `prometheus-rule.yaml` の残りのアラート全件。
  - 加えて、アラートは無いが初動が必要な運用事象の runbook を追加する:
    署名鍵プロバイダ (OpenBao) 到達不能による fail-closed、DB 接続枯渇 /
    circuit breaker open、ジョブ queue 滞留、テナント quota 超過 (既存文書を統合)。
  - `prometheus-rule.yaml` と `infra/docker/prometheus-rules.yml` の各アラートに
    `runbook_url` annotation と対応する specification objective 名を付ける。
  - `docs/operations/README.md` に索引 (アラート名 → runbook) と runbook の書式規約を置く。
- **tooling**:
  - `mise run check-monitoring` を拡張し、(1) すべての `alert:` が `runbook_url` を持つこと、
    (2) `runbook_url` の指す runbook ファイルが存在すること、(3) runbook が必須見出しを
    持つこと、を検査して CI で落とす。
  - `mise.toml` に `check-runbooks` を追加する (または `check-monitoring` に統合する)。
- **documentation**:
  - README の Documentation Guide に runbook 索引へのリンクを追加する。

## Out of Scope

- アラートの閾値・SLO 自体の見直し。実測に基づく調整は
  [[wi-282-staging-load-testing-and-capacity-validation]] と
  [[wi-161-large-tenant-performance-foundation]] が扱う。
- 新規メトリクスの追加。既存の golden signals で書ける範囲に留める。
- DR / バックアップ復旧手順。→ [[wi-101-backup-restore-and-disaster-recovery]]
- 障害時の機能縮退マトリクスと load shedding。→
  [[wi-165-high-availability-and-failover-resilience-topology]]
- ページャ / オンコールローテーションのツール設定 (組織側の設定)。
- 分散トレーシングを使った切り分け手順。→ [[wi-107-opentelemetry-distributed-tracing]]
  (導入後に runbook の「確認」節へ追記する)。

## Plan

- **runbook の構成を固定し、CI で検査する**。文書はレビュー圧が弱いと腐るため、
  必須見出しを機械検査する。「アラートに runbook_url が無ければ CI が落ちる」ようにすれば、
  今後アラートを追加する WI が自動的に runbook を伴う。
- **一次対応は必ず実行可能なコマンドで書く**。`mise` タスク・`kubectl` コマンド・
  確認用エンドポイント (`/readyz`, `/metrics`) を具体的に書き、抽象的な「調査する」を禁じる。
  このリポジトリは `mise.toml` が単一のコマンドマップなので、runbook のコマンドも `mise run` 経由に寄せる。
- **既存の 1 本 (`tenant-quotas.md`) を新書式の参照実装にする**。まずこれを新構成に書き換え、
  検査ツールを通し、残りをその型で量産する。
- **誤検知条件を必ず書く**。オンコールが「これは無害」と判断できないアラートは
  無視される (alert fatigue)。各 runbook に「鳴っても正常な条件」(デプロイ中、
  seed 実行中、負荷テスト中など) を明示する。
- 未決定: runbook を英語で書くか日本語で書くか。この repo の規約では README のみ英語なので、
  runbook は日本語で書き、コマンドと固有名詞は原文のままにする。

## Tasks

- [ ] T001 [Spec] `System.interfaces.MetricsExposition` に runbook 参照要件を追記し、
      対応する scenario を追加して `mise run check-spec` を通す。
- [ ] T002 [Format] `docs/operations/README.md` に索引と書式規約 (必須見出し 7 項目) を作る。
      既存 `tenant-quotas.md` を新書式に書き換えて参照実装にする。
- [ ] T003 [Tooling] `mise run check-monitoring` を拡張し、alert → runbook_url → ファイル存在 →
      必須見出しの検査を実装する。RED: runbook_url の無いアラートを一時的に混ぜて検査が
      落ちることを確認 → 実装 → GREEN (混ぜた分は戻す)。
- [ ] T004 [Runbook] `prometheus-rule.yaml` の SLO アラート全件 (Token / Login の
      error rate・latency、LoginThrottleHitRatioHigh、Jobs 系) の runbook を書く。
- [ ] T005 [Runbook] アラート未定義だが初動が必要な事象の runbook を追加する
      (署名鍵プロバイダ到達不能による fail-closed、DB 接続枯渇 / circuit open、
      ジョブ queue 滞留、readiness 失敗)。必要なら対応するアラートも追加する。
- [ ] T006 [Annotation] `infra/k8s/monitoring/prometheus-rule.yaml` と
      `infra/docker/prometheus-rules.yml` の全アラートに `runbook_url` と対応 objective 名を
      付与する。
- [ ] T007 [CI] `mise run check-monitoring` (または新設 `check-runbooks`) を
      `.github/workflows/idmagic-ci.yaml` の Verify Core に組み込む。
- [ ] T008 [Docs] README の Documentation Guide に runbook 索引を追加する。
- [ ] T009 [Verify] 下記 Verification を緑にする。runbook のコマンドを 1 件ずつローカルで
      実行し、記載どおり動くことを確認する。

## Verification

- `mise run check` / `mise run check-monitoring` / `mise run check-spec` / `mise run check-work-items`
- `mise run check-k8s` (annotation 追加後も overlay が妥当であること)
- 手動: `mise run dev-compose` で Prometheus を起動し、(1) 各 record rule / alert rule が
  読み込まれること、(2) 意図的に負荷や失敗を作って 1 件アラートを発火させ、
  runbook の「確認」節のコマンドがそのまま動くこと、を確認する。
- 手動: runbook 1 本を、コードベースを知らない読者が辿れるかレビューする
  (前提知識に依存した記述を排除する)。

## Risk Notes

これは文書とツーリング中心の変更で、実行時の振る舞いは変えない。主なリスクは
「書いたが腐る」ことである。CI 検査 (alert → runbook 存在 → 必須見出し) を入れて、
アラート追加時に runbook が伴うことを構造的に強制する。
runbook のコマンドが実際に動かないと、障害時にオンコールを誤誘導して復旧を遅らせる。
記載したコマンドは全件ローカルで実行して確認し、`mise run` 経由に寄せて
コマンド変更時に `mise.toml` 側で追随できるようにする。
`runbook_url` に外部 URL を書くと repo とドキュメントが乖離する。repo 内の相対参照を
既定とし、公開ホスティングが必要になった時点で索引だけを外部化する。
