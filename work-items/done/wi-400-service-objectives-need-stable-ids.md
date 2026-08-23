---
depends_on: []
status: completed
authors: [tn]
initial_context:
  specification: [docs/capacity.md, docs/observability.md]
  source: [infra/docker/prometheus-rules.yml, infra/k8s/monitoring/prometheus-rule.yaml, infra/README.md, load/k6/oauth-smoke.js]
  tests: [tools/check/src]
  stop_before_reading: [backend, frontend, spec]
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

3 点とも着手時に確定した。

1. **ID は `capacity.md` に置き、`SLO-<SUBJECT>-<ASPECT>` という記憶しやすい形にする。**

   置き場所は `capacity.md` のままとした。[[wi-407-name-the-directory-after-the-kind]] が §11.1 を「`capacity.md` が兼ねてよい。独立させるなら `reliability.md` として正規文書の集合へ加える」と両方許す形にしたので、`ROOT_DOCUMENTS` を触らない側を選べる。**目標と容量の前提は同じ測定境界（[Measurement boundary](../docs/capacity.md#measurement-boundary)）を共有しており、離すとその節を二重に持つことになる。**

   ID の形は連番ではなく記憶しやすい語にした。このリポジトリは連番（`REQ-<CONTEXT>-NNN`）と語（`RFC7643-CORE-RESOURCES`、`WCAG22-KEYBOARD`）の両方を使っており、**SLO は後者に近い。** 名前の付いた恒久的な約束であって、順序に意味のある一覧ではない。決め手はアラートの `summary` での読みやすさである。`SLO-LATENCY-004 を超過` より `SLO-TOKEN-LATENCY を超過` のほうが、当番担当者が何を見ているか分かる。

   Context を接頭辞にする案（`SLO-OAUTH2-001`）は却下した。**非 5xx 比率と可用性の母集団が Context をまたぐ**ためである。`/api/auth/login`（Authentication）と `/token`（OAuth2）は同じ 1 行に入っており、どちらの Context を名乗らせても嘘になる。

2. **しきい値の複製は畳まない。名指しの整合だけを検査する。**

   Prometheus の式から数値を消す方法は無いので、`0.3` と `0.001` は資材側に残る。**代わりに、資材が名指しする ID が実在することを検査する。** これで宙に浮いた参照（今回見つかった `oauth2.yaml` など）は二度と入らない。

   **数値の一致は検査しない。** 3 のとおりアラートと目標は別のものを判定しており、同じ数を使っているのは現時点の設計判断にすぎない。一致を強制すると、バーンレート窓を変えたときに「検査を通すためだけに目標を動かす」誘因が生まれる。

   検査は `check-monitoring` ではなく `tools/check/` に置く。前者は docker を要求するので、`mise run check` の速い経路から外れる。

3. **アラートは目標の error budget の消費を検知するのであって、目標の判定そのものではない。**

   非 5xx 比率の目標は 30 日の移動窓、アラートは 5 分のバーンレート窓である。**同じ `0.001` を使っているが、判定しているものが違う。** アラートの `summary` は「どの目標の予算を燃やしているか」を名指しし、目標そのものを再掲しない。目標を緩めたときにアラートを直すべきかは、この関係が書いてあれば読める。

   SLO 由来でないアラート（`LoginThrottleHitRatioHigh`、`Jobs*`）は ID を名乗らない。**したがって検査は「ID を名乗るなら実在すること」を要求し、「すべてのアラートが ID を名乗ること」は要求しない。**

## Plan

- ID の形を決め、`capacity.md` に付けるところまでを最初の 1 手にする。ここが決まらないと他はすべて宙に浮く。
- 参照の張り替えは資材ごとに独立しているので、ID が付いた後は順不同で進む。
- runbook は `page` の 5 件を対象にする。既存の `async-jobs.md` が `JobsLatencySensitiveClaimLatencyHigh` を実際に扱えているかを先に確かめ、扱えていれば型の参考にする。
- 検査を足すかどうかは 2 の判断次第なので、最後に回す。

## Tasks

- [x] T001 [Design] ID の形と置き場所、複製の扱い、アラートと目標の関係を確定し `## Design` に記録する。
- [x] T002 [Spec] `docs/capacity.md` の 19 行のサービス目標と容量受入れ目標に ID を付けた。
- [x] T003 [Spec] `docs/observability.md` の参照を ID を指す形にした。
- [x] T004 [Ops] `infra/README.md` の数値の再掲を ID 参照へ置き換えた。
- [x] T005 [Ops] `prometheus-rules.yml` と `prometheus-rule.yaml` のコメントと annotations を ID へ張り替えた。
- [x] T006 [Ops] `load/k6/oauth-smoke.js` のコメントを張り替えた。
- [x] T007 [Ops] `page` の 5 件に対応する runbook を `docs/runbooks/` へ置き、全 8 アラートへ `runbook_url` を足した。
- [x] T008 [Tooling] `check-slo-references` を新設した。`check-monitoring` は docker を要求するため、そちらではなく `mise run check` の経路に置いた。
- [x] T009 [Verify] `mise run verify` を通した。

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

## Completion

- **Completed At**: 2026-08-23
- **Summary**:
  `docs/capacity.md` のサービス目標 16 件と容量受入れ目標 3 件に、`SLO-TOKEN-LATENCY` のような記憶しやすい安定 ID を与えた。連番ではなく語にした決め手はアラートの `summary` での読みやすさで、`SLO-LATENCY-004 を超過` より `SLO-TOKEN-LATENCY を超過` のほうが当番担当者に伝わる。Context 接頭辞は、非 5xx 比率と可用性の母集団が Context をまたぐため採らなかった。参照側は 4 か所すべてを ID 参照へ張り替え、`OAuth2/objective/*`・`oauth2.yaml`・`authentication.yaml` という実体の無い名前を消した。8 件のアラートすべてに `runbook_url` を付け、`page` 級が指す runbook 5 件を新しく書いた。`check-slo-references` が、資材の名指しする ID の実在と `page` 級の runbook の到達を強制する。**しきい値の数値は照合しない。** アラートは 5 分のバーンレート窓、目標は 30 日の移動窓で別のものを判定しており、一致の強制は「検査を通すために目標を動かす」誘因になる。
- **Verification Results**:
  - `mise run verify` - passed（exit 0）
  - `mise run check-slo-references` - ok 19 objective(s), 2 monitoring asset(s)
  - `mise run test-tools` - 176 pass / 0 fail（`slo-references.test.ts` の 9 件を含む）
  - 手動: ID と runbook パスをそれぞれ壊して検査が落ちることを確認 - passed（`names SLO-GONE-LATENCY, which no objective declares` / `points at a missing runbook`）
  - 手動: 宙に浮いた参照が 0 件であることを確認 - passed

## Left Undone

- **`async-jobs.md` はアラート runbook の型になっていない。** ジョブ運用の手引きであり、発火条件・最初に確認すること・緩和・確認・エスカレーションの構成を持たない。Jobs 系のアラート 3 件がここを指しているが、当番担当者が障害の最中に開いて初動を判断できる形ではない。[[wi-290-alert-runbook-catalog-and-on-call-operations]] が扱う。
- **`tenant-quotas.md` だけ「です・ます」調である。** 他の runbook は「である」調。文体の統一は本 work item の Scope に無い。
- **`CAP-*` の 3 件は誰も参照していない。** 容量受入れ試験がまだ実施されていないためで、ID は試験を定義する変更のために用意した状態である。
- **`reliability.md` は作らなかった。** [[wi-407-name-the-directory-after-the-kind]] がガイド §11.1 を「`capacity.md` が兼ねてよい」と両方許す形にしたので、`ROOT_DOCUMENTS` を触らずに済む側を選んだ。目標と容量の前提は同じ Measurement boundary を共有しており、離すとその節が二重になる。
