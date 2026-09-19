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
  reason: "利用者が観測できる振る舞い、公開契約、運用手順を変えない。設計文書の記述だけを厚くする。"
  references: []
spec_impact:
  kind: none
  reason: "監視、ログ、トレースの設計記述を厚くするだけで、計装、監視構成ファイル、SLO の値は変えない。"
initial_context:
  specification: []
  typespec: []
  source:
    - docs/design/observability
    - docs/requirements/quality.md
    - docs/architecture/deployment.md
    - docs/design/infrastructure/platform.md
    - infra/k8s/monitoring
    - infra/docker/prometheus.yml
    - infra/docker/otel-collector.yaml
    - backend/shared/observability
    - backend/shared/logging/logging.go
  tests: []
  stop_before_reading:
    - docs/runbooks
    - backend/oauth2
    - frontend
---

# 監視・ログ・トレースの設計を、当番担当者が使える水準まで書く

## Motivation

[オブザーバビリティ設計](../../docs/design/observability/README.md)は信号モデル、相関、所有と流れを持ち、この階層では十分である。
子文書が足りていない。

[監視設計](../../docs/design/observability/monitoring.md)は 17 行である。監視する層を列挙し、API とワーカーが何を公開するかを二文で述べ、Prometheus ルールが SLO ID を注釈に持つと書いて終わる。
何を検知したいのか、その検知に使う指標は何か、閾値を誰が決めるのか、通知先はどこか、対応する runbook は何かという対応が、どこにも一覧として存在しない。
`infra/k8s/monitoring/prometheus-rule.yaml` を読めば現行のルールは分かるが、**なぜその検知を選んだか**は構成ファイルからは読めない。

[トレース設計](../../docs/design/observability/tracing.md)は 11 行である。`OBSERVABILITY=otel` のときの動作を三文書き、伝播、標本化、保持の設計は未完成だと述べて [[wi-107-opentelemetry-distributed-tracing]] へ委ねている。
実装が未完成であることと、目標の設計が書かれていないことは別である。

[ログ設計](../../docs/design/observability/logging.md)は 73 行あり、共通フィールド、重大度の使い分け、ミドルウェアとアプリケーションの境界、秘匿と容量を持つ。
足りないのは、`event_name` の命名規約（安定した名前を使えとは書かれているが、どう作るかがない）、サンプリング規則の具体、保持期間と閲覧権限、容量超過と収集停止時の動作である。最後の三つは `DOCUMENTATION_GUIDE.md` §4.6 が `logging.md` の担当と定めている。

## Scope

- [監視設計](../../docs/design/observability/monitoring.md)に、検知したいこと、使う指標、閾値の所有者、通知、runbook の対応表を書く。
- [ログ設計](../../docs/design/observability/logging.md)に、`event_name` の命名規約、サンプリング規則、保持と閲覧権限、容量超過と収集停止時の動作を書く。
- [トレース設計](../../docs/design/observability/tracing.md)に、伝播、標本化、属性、保持の目標設計を書き、実装の現状と分けて示す。
- オブザーバビリティ設計の索引と未確定事項を追従させる。
- オブザーバビリティの用語を、分野で実際に使われている表記へ合わせる。

## Out of Scope

- 計装の実装。分散トレースの実装と検証は [[wi-107-opentelemetry-distributed-tracing]] が扱う。
- アラートの追加と runbook の整備。[[wi-290-alert-runbook-catalog-and-on-call-operations]] が扱う。
- SLO の値と測定境界。[品質要求](../../docs/requirements/quality.md)が正本である。
- 監査イベント。診断ログではなく業務記録であり、[Audit](../../docs/domain/audit/README.md) が持つ。
- 監視基盤のデプロイ構成。[[wi-584-ground-deployment-design-in-reference-profiles]] が扱う。

## Design

### 監視

監視設計が持つべきは指標の一覧ではなく、**検知したいことと、それを検知する手段の対応**である。
指標名の一覧は計装のコードと `infra/k8s/monitoring/` にあり、写すと二重になる。

| 列 | 内容 |
| --- | --- |
| 検知したいこと | 利用者から見て何が壊れているか |
| 層 | 公開入口、API、Worker、PostgreSQL、外部依存、Kubernetes、監視基盤自体 |
| 信号 | どの指標、ログ、トレースで見るか |
| 判断 | 瞬間値か、消費の速さか、欠落か |
| 通知 | ページか、通知だけか、ダッシュボードのみか |
| runbook | 対応する手順 |

この形にすると、**監視していないもの**が行の欠落として見える。
現在「監視基盤自体の自己監視が十分かは未検証」と散文で書かれている状態が、行として残る。

ダッシュボードの構成も持つ。何を一枚目に置くか（利用者から見た成功、次に飽和）、テナント単位の切り分けをどの信号で行うか。高カーディナリティ値を指標のラベルにしない既存の規則との関係を書く。

### ログ

四つを足す。

- **`event_name` の命名規約**。構成要素の順序（対象、操作、結果）、使う語彙、変わる値を名前へ入れないこと。一覧は持たない。名前の作り方だけを決める。
- **サンプリング規則**。既存の文書が「事象名、対象、割合、例外を明示する」と要求しているので、その形の表を置き、現在サンプリングしている事象があるかを書く。無いなら無いと書く。
- **保持と閲覧権限**。デプロイプロファイルごとに、保持期間、保存先、誰が読めるかを書く。ローカルの Loki 単一レプリカを本番の保証として扱わない既存の注意は維持する。
- **容量超過と収集停止**。ディスクが尽きたとき、Loki が落ちたとき、Alloy が止まったときに何が起きるか。アプリケーションは標準出力へ書き続けるのか、ブロックするのか。ログの欠落をサービスの正常性と混同しない仕組みは何か。

### トレース

**ログ設計へ統合しない。** 統合案は検討したうえで採らない。

理由は三つある。
[オブザーバビリティ設計](../../docs/design/observability/README.md)が信号を三つ（メトリクス、ログ、トレース）に分けて所有者を宣言しており、その分割は OpenTelemetry の信号区分と一致する。統合すると、README の信号モデルと子文書の構成が食い違う。
トレースが答える問いは「実行単位と外部依存をまたぐ因果関係と所要時間」で、ログが答える「個別リクエストとジョブの事実」とは別である。読む場面も違う。
そして、いま `tracing.md` が薄いのは統合すべきだからではなく、目標設計が書かれていないからである。統合すると、薄さが解消したように見えて、伝播と標本化の設計は依然としてどこにもない状態になる。

書く内容は次のとおりで、それぞれ目標の設計と実装の現状を分けて示す。

- **伝播**。入口での `traceparent` の扱い（既存記述を維持）、データベース呼び出し、外部 HTTP、非同期ジョブへの文脈の引き継ぎ。ジョブでは HTTP リクエストのトレースを継続するのか、新しいトレースにして link で結ぶのか。
- **標本化**。head か tail か、デフォルトのレート、常に採るもの（エラー、遅いリクエスト）、環境ごとの差。
- **属性**。OpenTelemetry の意味規約にあるものを使い、自分で決めたものだけをここへ書くという既存の原則の適用結果。span 名の付け方（高カーディナリティな値を span 名に入れない）。
- **保持とコスト**。保持期間、保存先、標本化率と保持期間の関係。
- **相関**。`X-Request-ID` が別の相関軸として常に存在することと、trace が無い環境での調査経路。

### 用語

レビューで、この階層が使っていた「信号」「相関」「所有境界」が造語であると指摘を受けた。
造語かどうかを一次資料で確かめ、分野の表記へ合わせる。

| これまでの表記 | 採る表記 | 根拠 |
| --- | --- | --- |
| 信号 | シグナル | OpenTelemetry 日本語ドキュメントの[シグナル](https://opentelemetry.io/ja/docs/concepts/signals/)が、テレメトリーの種類を指してこの語を使う |
| 相関 | 相関（維持） | Datadog 日本語ドキュメントの「ログとトレースの相関」が同じ意味で使う。OpenTelemetry 日本語ドキュメントは「関連付け」と書く |
| 所有境界 | 採らない | 一次資料に対応する語が無い。文書の分担の話であり、オブザーバビリティの用語ではないため、平易な日本語で書く |
| 公開入口 | エッジ | [デプロイメントアーキテクチャ](../../docs/architecture/deployment.md)がロードバランサーと Ingress の層をエッジと呼んでいる |
| 重大度 | ログレベル | 一般的な表記であり、`level` フィールドの値そのものを指す |

「シグナル」は OS のシグナルと衝突する。
語を変えるのではなく、オブザーバビリティ設計の冒頭で、この文書群のシグナルがテレメトリーの種類を指すと明示して区別する。

## Plan

1. `infra/k8s/monitoring/` と計装コードから、現在の検知とその意図を読み取る。
2. 監視設計を対応表の形へ書き直し、行の欠落として監視していないものを残す。
3. ログ設計に命名規約、サンプリング、保持、容量超過を足す。
4. トレース設計に目標設計を書き、実装の現状と分けて示す。
5. オブザーバビリティ設計の未確定事項を、子文書へ移した分だけ整理する。

## Tasks

- [x] T001 [Design] 現行の監視ルールと計装から、検知の意図を読み取る。
- [x] T002 [Docs] 監視設計を検知と手段の対応表へ書き直す。
- [x] T003 [Docs] ログ設計に命名規約、サンプリング、保持、容量超過を足す。
- [x] T004 [Docs] トレース設計に伝播、標本化、属性、保持の目標設計を書く。
- [x] T005 [Docs] オブザーバビリティ設計の索引と未確定事項を追従させる。
- [x] T006 [Verify] リンク、SLO 参照、監視構成ファイル、仕様、全体検証を通す。

## Verification

観測可能な境界を持つプロダクトの振る舞いを変えないため、受け入れ RED と単体 RED は持たない。
代わりに RED を取る検査は `mise run check-links` である。
監視設計の対応表は行ごとに runbook を指し、対応する `docs/runbooks/*.md` が無い行を書くとこの検査が落ちる。
落ちる行は、手順が無いことを示す値へ書き換え、リンクにしない。
これにより、「runbook がある」と読める行が、実在する手順とだけ対応する。

- `mise run check-links`
- `mise run check-slo-references`
- `mise run check-monitoring`
- `mise run check-spec`
- `mise run verify`

## Risk Notes

監視設計に指標名を並べると、計装と `infra/k8s/monitoring/` に続く三つ目の複製になる。対応表が持つのは検知したいことと手段の対応であって、指標名の一覧ではない。行に指標を書くときは、構成ファイルが正本であることを冒頭で明示する。

トレースの目標設計は未実装である。実装済みと読めると、`request_id` があることを理由にスパンが取れていると誤解される既存の危険がそのまま残る。目標と現状を行または節で分け、現状の側を先に読ませる。

ログの保持と閲覧権限はデプロイ先で決まる。デプロイプロファイルが確定していない状態で値を書くと、決まっていないものが決まったように見える。[[wi-584-ground-deployment-design-in-reference-profiles]] が定めるプロファイルごとに書き、未確定は未確定と書く。

## Completion

- **Completed At**: 2026-09-19
- **Summary**:
  `mise run spec-diff` reports no normative specification change against `main`, which matches
  `spec_impact: none`. The semantic difference is in the observability design documents.
  Monitoring now states what it wants to detect, the signal it detects it with, how the signal is judged,
  which kind of notification it raises, and which runbook it leads to, as one table per detection. A second
  table records the detections that have no rule, so a missing detection is a visible row instead of an
  absence. Threshold ownership is split by origin (SLO, capacity, observed normal range), and the document
  states that the alerts named after error-budget burn actually judge a five-minute instantaneous value.
  Logging gains an `event_name` naming rule (a closed result vocabulary, nothing that varies in the name),
  a declaration form for sampling with the statement that nothing is sampled today, a per-deployment-profile
  table of storage, redundancy, retention, read access, and deletion, and a table of what happens to the
  application and to the logs when Loki, Alloy, or the node disk fails.
  Tracing now separates the implemented state from the target design and puts the implemented state first:
  only `idmagic-api` creates spans, only for the OAuth/OIDC endpoints, only under `OBSERVABILITY=otel`, with
  no propagation past the ingress and no store to search. The target design fixes propagation per boundary
  (jobs start a new trace linked to the enqueue, with the reason), head sampling over tail sampling with the
  cost of that choice stated, span-name cardinality and the three self-declared attributes, and the rule that
  retention is shortened before the sampling rate is lowered.
  The observability README gains an index of what each child document owns and points each open item at the
  child that records it.
  A review pass then replaced the vocabulary this layer had coined. `信号` becomes `シグナル`, the term the
  Japanese OpenTelemetry documentation uses for a category of telemetry, with a sentence in the README
  separating it from an operating-system signal. `相関` is kept, because the Japanese Datadog documentation
  uses it for the same idea. `所有境界` is dropped and written out in plain Japanese, because no primary
  source has a matching term and it describes which document decides what, not observability. `公開入口`
  becomes `エッジ`, the name the deployment architecture already gives that layer, and `重大度` becomes
  `ログレベル`. The rename reaches the glossary entry, the documentation guide, and the design documents that
  referred to the old words, so no definition is left behind. Tracing also gains a section stating what it
  answers that logs cannot, which is why it is a document of its own rather than a section of the logging
  design, and its implemented state moves from four prose subsections into one table.
- **Acceptance RED Evidence**:
  - **Test**: `N/A: この変更は設計文書だけを変え、観測可能なプロダクトの境界を持たない。`
  - **Requirement**: N/A: spec_impact は none であり、規範シナリオも標準 id も変えていない。
  - **Observed Failure**: `mise run check-links` was GREEN before the change and stayed GREEN after it.
    Nothing was broken beforehand; the documents were thin, which no check measures.
  - **Detection Reason**: The check that owns this change is `mise run check-links`, because the monitoring
    table claims a runbook per detection row. Replacing `../../runbooks/async-jobs.md` with a plausible but
    absent `../../runbooks/jobs-queue-saturation.md` made it fail with
    `relative link target does not exist: docs/runbooks/jobs-queue-saturation.md` on all three rows that
    carry it. A row that reads "a runbook exists" therefore corresponds to a procedure that exists.
- **Unit RED Evidence**:
  - **Test**: `N/A: 変更したのは Markdown だけで、単体の境界を持つコードを変えていない。`
  - **Requirement**: N/A: 同上。
  - **Observed Failure**: `mise run check-terminology` failed with
    `docs/design/observability/tracing.md:140:14: 「既定」は採らない表記。「デフォルト」 を使う。` on the
    first draft of the tracing document, and passed after the wording was corrected.
  - **Detection Reason**: The terminology check is the unit-level gate for this change: it reads each
    sentence for the adopted spelling rather than the document as a whole, and it distinguished one word in
    one line out of three rewritten documents.
- **Change-Resistance Results**:
  Risk is `low` and no Go or TypeScript changed, so `mise run test-go-mutation` does not apply.
  The hand-written fault is the one the mutation operators cannot express: redirecting a link to a
  procedure that does not exist. Injected into the three `async-jobs.md` rows, `mise run check-links`
  detected it and named every affected row; reverting restored `ok Markdown links (865 document(s))`.
- **Verification Results**:
  - `mise run check-links` - passed
  - `mise run check-terminology` - passed
  - `mise run check-work-item-references` - passed
  - `mise run check-slo-references` - passed (19 objective(s), 2 monitoring asset(s))
  - `mise run check-monitoring` - passed (19 rules, both Alloy configurations, three kustomize builds)
  - `mise run check-spec` - passed
  - `mise run verify` - passed
