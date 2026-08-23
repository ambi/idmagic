---
depends_on: []
status: pending
authors: [tn]
risk: low
created_at: 2026-08-23
priority: p2
change_kind: docs
spec_impact: { kind: none, reason: "サービス目標に安定 ID を与え、資材側の参照を張り替える作業である。目標値も製品の振る舞いも変えない。affected_spec が索引するのは既存の normative scenario / standard ID または TypeSpec symbol だが、docs/capacity.md には現在いかなる規範 ID も無く、この変更こそが最初の ID を作るため、指せる既存要素が存在しない。" }
---

# サービス目標に安定 ID を与え、しきい値の複製と宙に浮いた参照を畳む

## Motivation

[docs/capacity.md](../docs/capacity.md) の `Service level objectives` は、母集団・時間窓・除外条件まで踏み込んだ目標を持っている。**しかしどの行にも ID が無い。** 参照する手段が無いので、他の文書と資材は目標を指すかわりに数値を写している。

2026-08-23 時点で `/token` の p99 300 ms は 4 か所、非 5xx 99.9% は 3 か所に現れる。

| 場所 | 書かれているもの |
|---|---|
| `docs/capacity.md:34` | `/token` p99 ≤ 300 ms（正本） |
| `docs/capacity.md:50` | `/token` 非 5xx 比率 ≥ 99.9%（正本） |
| `infra/docker/prometheus-rules.yml:63` | `> 0.3` |
| `infra/docker/prometheus-rules.yml:57` | `> 0.001` |
| `infra/README.md:34` | 「p99 トークンレイテンシーを 300 ms 未満、エラー率を 0.1% 未満とする」 |
| `load/k6/oauth-smoke.js:22` | `idmagic_token_latency: ['p(99)<300']` |

[DOCUMENTATION_GUIDE.md](../DOCUMENTATION_GUIDE.md) §3 は「同じ数値を二か所に書かない」、§11.1 は「他の文書は SLO ID を参照し、数値を再掲しない」と定める。現状はその逆で、正本を変えても他の 4 か所は黙って古くなる。

**さらに、目標を指そうとした参照はすべて既に壊れている。** 参照先の名前空間が撤去済みだからである。

| 参照元 | 参照している名前 | 実体 |
|---|---|---|
| `infra/README.md:34` | `OAuth2/objective/TokenLatency`、`OAuth2/objective/TokenErrorRate` | 無い |
| `load/k6/oauth-smoke.js:22`（コメント） | 同上 | 無い |
| `infra/docker/prometheus-rules.yml:50-52`（コメント） | `oauth2.yaml`、`authentication.yaml` | 無い |
| `prometheus-rules.yml` の annotations 4 件 | `oauth2.yaml TokenErrorRate` ほか | 無い |

読み手はアラートの `summary` から正本へ辿れず、辿ろうとすると存在しないファイルを探すことになる。数値が一致しているのは現時点で偶然そうなっているだけで、それを保証している仕組みは無い。

**page 級のアラートに手順が無い。** `prometheus-rules.yml` の `severity: page` は 5 件（`TokenErrorRateBudgetBurn`、`TokenLatencyBudgetBurn`、`LoginErrorRateBudgetBurn`、`LoginLatencyBudgetBurn`、`JobsLatencySensitiveClaimLatencyHigh`）。このうち runbook があるのはジョブ系だけで、`docs/runbooks/` は `async-jobs.md`、`backup-restore-dr.md`、`tenant-quotas.md` の 3 件しかない。**トークン発行とログインという製品の中核が落ちたときに、当番が開くものが存在しない。** どのアラートにも `runbook_url` の annotation は無い。

## Scope

- `docs/capacity.md` のサービス目標の各行に安定 ID を与える。レイテンシー、非 5xx 比率、可用性、容量受入れ目標のすべてを対象にする。
- ID を付けたうえで、`infra/README.md` の数値の再掲を目標 ID の参照に置き換える。
- `prometheus-rules.yml` と `infra/k8s/monitoring/prometheus-rule.yaml` のコメントと annotations から、存在しないファイル名を落とし、目標 ID を名指しする。
- `load/k6/oauth-smoke.js` のコメントを目標 ID へ張り替える。しきい値そのものは k6 の設定なので残す。
- `severity: page` のアラートに対応する runbook を置き、各アラートに `runbook_url` を付ける。
- `docs/observability.md` の「サービス目標の母集団、時間窓、除外条件、目標値は capacity.md が定める」という記述を、ID を指す形にする。

## Out of Scope

- 目標値そのものの見直し。数値は変えず、参照可能にすることだけを扱う。
- `Measurement` の取得。`capacity.md` が「本書に Measurement はまだない」と書いているとおり、ステージングでの実測は別の作業である。
- `operations/release-and-rollback.md` の新設。DOCUMENTATION_GUIDE §11.2 が挙げる文書だが、このリポジトリはまだリリースを行っていない。§3 の「必要が生じていない文書を作らない」に従い、最初のリリースを定義する変更が持つ。
- 縮退順序の実装。`capacity.md` の `Degradation order` は方針として既にあり、これを資材へ落とすのは別の作業である。

## Design

未定。着手時に次の 3 点を確定して本節に記録する。

1. **ID の形と置き場所。** DOCUMENTATION_GUIDE §11.1 は `operations/reliability.md` を SLO の正本とするが、このリポジトリは目標を `capacity.md` に統合しており、[SPECIFICATION_FORMAT.md](../SPECIFICATION_FORMAT.md) §1 の `ROOT_DOCUMENTS` に `reliability.md` は無い。ファイル集合は *(checked)* なので、**文書を増やすなら `tools/check/src/specification-doc.ts:30` の集合を変える判断が要る。** 増やさず `capacity.md` に `SLO-<AREA>-NNN` を足す側に寄せるのが素直だが、そうすると「容量の前提」と「守るべき目標」が同じファイルに同居し続ける。どちらを取るかを決める。

2. **複製をどこまで畳めるか。** Prometheus の式に数値を書かずに済ませる方法は無いので、`0.3` と `0.001` は資材側に残る。残る複製を **検査で一致を強制する**（`mise run check-monitoring` に、アラートが名指しする ID の実在と、しきい値が正本と一致することの検査を足す）か、**注記で正本を指すに留める**かを決める。前者は目標の書式を機械が読める形に縛る。

3. **アラートは目標を参照するのか、別の判定基準なのか。** 非 5xx 比率の目標は 30 日窓、アラートは 5 分のバーンレート窓である。`> 0.001` という同じ数を使っているだけで、**判定しているものが違う。** ID で結ぶときに「この目標の消費を検知するアラート」と書くのか、アラート側に独立した閾値の根拠を持たせるのかを決める。ここを曖昧にしたまま ID で結ぶと、目標を緩めたときにアラートを直すべきかどうかが読めない。

## Plan

- ID の形を決め、`capacity.md` に付けるところまでを最初の 1 手にする。ここが決まらないと他はすべて宙に浮く。
- 参照の張り替えは資材ごとに独立しているので、ID が付いた後は順不同で進む。
- runbook は `page` の 5 件を対象にする。既存の `async-jobs.md` が `JobsLatencySensitiveClaimLatencyHigh` を実際に扱えているかを先に確かめ、扱えていれば型の参考にする。
- 検査を足すかどうかは 2 の判断次第なので、最後に回す。

## Tasks

- [ ] T001 [Design] ID の形と置き場所、複製の扱い、アラートと目標の関係を確定し `## Design` に記録する。
- [ ] T002 [Spec] `docs/capacity.md` のサービス目標に ID を付ける。
- [ ] T003 [Spec] `docs/observability.md` の参照を ID を指す形にする。
- [ ] T004 [Ops] `infra/README.md` の数値の再掲を ID 参照へ置き換える。
- [ ] T005 [Ops] `prometheus-rules.yml` と `prometheus-rule.yaml` のコメントと annotations を ID へ張り替える。
- [ ] T006 [Ops] `load/k6/oauth-smoke.js` のコメントを張り替える。
- [ ] T007 [Ops] `page` の 5 件に対応する runbook を `docs/runbooks/` へ置き、各アラートへ `runbook_url` を足す。
- [ ] T008 [Tooling] 2 で検査を足すと決めた場合、`check-monitoring` を拡張する。
- [ ] T009 [Verify] `mise run check-spec`、`mise run check-monitoring`、`mise run verify` を通す。

## Verification

- `mise run check-spec`
  - reason: `capacity.md` は正規文書なので、ID を足した後も書式検査を通る必要がある。
- `mise run check-monitoring`
  - reason: アラート定義の変更が資材として妥当であることを確かめる。
- `mise run verify`
- 手動: `rg 'OAuth2/objective|oauth2\.yaml|authentication\.yaml' --glob '!work-items/**'` が 0 件になることを確認する。宙に浮いた参照が残っていないことの指標にする。
- 手動: `capacity.md` の `/token` の p99 を一時的に別の数へ変え、参照側 4 か所のどれが追随しないかを見る。追随しない箇所が残るなら、それは畳めていない複製である。

## Risk Notes

リスクは low。仕様の数値も製品の振る舞いも変えず、参照可能にする作業である。

失敗の形は 2 つある。1 つは **ID を付けただけで複製を残すこと**である。参照が壊れていた原因は ID が無かったことだが、ID を付けても数値の写しが残れば、次に目標を変えたときに同じ乖離が起きる。ID の付与と参照の張り替えを同じ変更で終える。

もう 1 つは **runbook を型だけ埋めること**である。`tenant-quotas.md` は 30 行で発火条件もエスカレーションも持たない。同じものを 5 件増やしても、当番が障害の最中に開く価値は無い。書けない手順があるなら、書けないことを書くほうがよい（`backup-restore-dr.md` がリージョン喪失について実際にそうしている）。
