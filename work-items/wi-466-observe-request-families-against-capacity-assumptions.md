---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-03
priority: p2
depends_on: []
change_kind: operations
affected_spec:
  - { path: docs/contexts/system/scenarios.md, requirement: REQ-SYSTEM-001 }
---

# 種別ごとの到達率と処理時間を観測し、容量の Planning assumption を Measurement へ置き換えられるようにする

## Motivation

[[wi-459-api-process-plane-separation-decision]] は API の Deployment を種別ごとに分けない判断を記録した。その判断は永久のものではなく、6 つの再検討条件（C1–C6）を伴う。**そのうち負荷に関わる 2 つは、いま検知できない。**

- **C1**「管理系、ポータル系、SCIM、Shared Signals のいずれかを含む混合負荷で認証系のサービス目標を満たせない」— 認証系のサービス目標は `docs/capacity.md` の `SLO-*` として定義され、`infra/k8s/monitoring/prometheus-rule.yaml` が `/token` とログインについてバーンレートを見ている。しかし**目標が侵されたときに、どの種別がそれを侵したのかを示す観測がない。** 侵害は見えるが、原因の種別は見えない。
- **C2**「分離によって 1 レプリカ当たり持続処理能力が上がる」— 判定には混合ごとの Measurement が要る。

同時に、同 work item が `docs/capacity.md` へ追加した Non-protocol request profile は、**18 個の入力すべてが Planning assumption で、Measurement が 1 つもない。** 幅は 2 桁に及ぶ。同書の Evidence classes は「ステージングの容量検証では Planning assumption を Measurement へ置き換える」と定めているが、置き換える先の観測が本番にもステージングにも存在しない。

現在あるのは `http_requests_total`、`http_request_duration_seconds`、`http_requests_in_flight` の 3 つと、`route`、`method`、`status_code` のラベルである。**種別で集約する手段がない。** `route` は登録済みのルートパターンなので、接頭辞から種別は導けるが、その導出をどこにも持っていない。記録規則にもダッシュボードにもアラートにも、管理系、ポータル系、SCIM、Shared Signals を母集団とするものは 1 つもない。

つまり「分けない」という判断は、それを覆す条件を観測できないまま置かれている。**再検討条件が検知できない判断は、判断ではなく既定値である。**

## Scope

- `route` ラベルから種別（認証・プロトコル系、ポータル系、管理系、SCIM、Shared Signals の受信、運用経路）を導く対応を 1 か所に定義する。
- 種別ごとの到達率、レイテンシー分位、非 5xx 比率、実行中要求数の記録規則を追加する。
- ダッシュボードに種別ごとの内訳を出す。`docs/capacity.md` の Non-protocol request profile の各行に対応する観測を、同じ単位（通常時、最繁時、集中実行時）で読めるようにする。
- 認証系のサービス目標が侵されたときに、同じ時間窓の種別別内訳を並べて読めるようにする。C1 の判定に使う。
- 種別に対応しない `route` が存在しないことを検査する。分類漏れがあれば、その経路は観測から静かに消える。
- `docs/observability.md` に種別の対応と、この観測が `docs/capacity.md` のどの行を置き換えるためのものかを記録する。

## Out of Scope

- 種別ごとのサービス目標（`SLO-*`）を新設すること。**観測を持つことと目標を約束することは別である。** 目標を置くかどうかは、実測が集まってから別に判断する。とくにポータル系については [[wi-459-api-process-plane-separation-decision]] が「対話的な利用者操作として認証系と同じ可用性優先度を与えるか、独立した容量と SLO を持たせるかを容量シナリオと正準文書で決める」として未決のまま残している。
- 優先度クラスの分類と、それに基づく入場制御。[[wi-396-prioritize-login-under-saturation]] が持つ。同 work item も分類をコードへ置くが、目的が違う（拒否の判断に使う）ので、対応が同じになるとは限らない。Design で関係を決める。
- ステージングでの負荷試験と、混合負荷での 1 レプリカ当たり持続処理能力の実測。[[wi-282-staging-load-testing-and-capacity-validation]] が持つ。本 work item はその測定結果を読める形を用意する側である。
- `docs/capacity.md` の Planning assumption を実際に Measurement へ書き換えること。観測が回ってからの作業であり、書き換えは測った人が行う。
- 新しいメトリクスの追加。既存の 3 つと既存のラベルで足りるかを Design で確かめ、足りない場合だけ Scope へ戻す。

## Design

### ラベルを増やすか、記録規則で導くか

| 案 | 内容 | 利点 | 欠点 |
| --- | --- | --- | --- |
| A | `http_requests_total` に `family` ラベルを足す | 集約が単純。分類が計測点にあるので取りこぼさない | カーディナリティが増える。既存の記録規則とアラートを見直す必要がある。分類を変えると過去の系列と接続しない |
| B | Prometheus の記録規則で `route` の正規表現から種別を導く | コードを変えない。分類を変えても過去のデータへ遡って適用できる | 分類が監視資材の側にあり、経路を足した人が気づかない。`route` の値と正規表現がずれても静かに落ちる |
| C | B に加えて、種別に対応しない `route` が無いことを検査する | B の欠点を塞ぐ | 検査が `route` の全量を知る必要がある |

**C を採る。** 分類の実体は記録規則に置き、その網羅性をリポジトリの検査で保証する。`route` の全量は `backend/shared/spec/operations_gen.go` から取れる。A を採らないのは、種別が用途による分類であって計測の属性ではないためで、同じ理由で分類は後から変わりうる。過去の系列と接続しないのは、その変更を高くつかせる。

`docs/capacity.md` の Measurement boundary は「`route` は解決済みのパスではなく登録済みのルートパターンで集約し、realm 接頭辞を持つ同じ操作も同じエンドポイント群へ含める」と定めている。**種別の導出も同じ規則に従う。** つまり `/realms/{tenant_id}/api/admin/v1/...` と `/api/admin/v1/...` は同じ種別になる。

### 分類の対応

[[wi-459-api-process-plane-separation-decision]] が定めた種別をそのまま使う。分類の実体を 2 つ持たない。

| 種別 | 経路 |
| --- | --- |
| 認証・プロトコル系 | OAuth2/OIDC のプロトコル経路、`/.well-known/*`、`/jwks`、SAML、WS-Federation、`/trust/*`、`/api/auth/*`、`/api/branding`、`/session/check`、`/tenant-branding-assets/*`、`/application-icons/*` |
| ポータル系 | `/api/account/v1/*` |
| 管理系 | `/api/admin/v1/*` |
| SCIM | `/scim/v2/*` |
| Shared Signals | `/ssf/*` |
| 運用経路 | `/livez`、`/readyz`、`/startupz`、`/health`、`/metrics` |

Shared Signals のストリーム管理は `/api/admin/v1/shared-signals/*` にあるので管理系に入る。`/ssf` に残るのは受信 1 経路だけである。

### wi-396 の優先度クラスとの関係

[[wi-396-prioritize-login-under-saturation]] は要求を優先度クラスへ分類し、その分類をルート登録と同じ場所へ置き、分類の無いルートが無いことを検査する。**本 work item の種別と、その優先度クラスは別の分類である。** 種別は用途、優先度クラスは飽和時に何を先に捨てるかで、たとえば管理系の中でも参照は残して集計を捨てるという分け方はありうる。

一方で、2 つの分類が独立に経路の全量を列挙し、独立に網羅性を検査するのは重複である。**先に入るほうが列挙と検査の仕組みを持ち、後から入るほうがそれを使う。** どちらが先かは着手時に決める。現在 wi-396 は `pending` である。

## Plan

1. `route` ラベルの実際の値の全量を確認し、上の分類がすべてを覆うことを確かめる。覆えない経路があれば分類を直す。
2. wi-396 の進行状況を見て、列挙と網羅性検査をどちらが持つかを決める。
3. 記録規則を追加する。分類漏れがある状態で検査が RED になることを先に確かめる。
4. ダッシュボードに種別別の内訳を出す。`docs/capacity.md` の行と対応が読めるようにする。
5. `docs/observability.md` に対応を記録する。数値は書かず、`docs/capacity.md` の行を ID で名指しする。

## Tasks

- [ ] T001 [Research] `route` ラベルの全量を確認し、分類の網羅性を確かめる。
- [ ] T002 [Design] wi-396 の優先度クラスとの分担を決める。
- [ ] T003 [Acceptance] 分類に対応しない `route` がある状態で失敗する検査を書き、RED を確かめる。
- [ ] T004 [Monitoring] 種別ごとの記録規則を追加する。
- [ ] T005 [Monitoring] ダッシュボードに種別別の内訳を出す。
- [ ] T006 [Docs] `docs/observability.md` に対応を記録する。
- [ ] T007 [Verify] 検査を通す。

## Risk Notes

リスクは low。観測を足すだけで、製品の振る舞いも公開契約も変えない。

**分類漏れは静かに効く。** 種別に対応しない経路は、集計から消えるだけで警告を出さない。それでは「管理系の到達率は低い」という観測が、実は分類漏れだったという読み違いを生む。網羅性の検査を先に入れ、検査が無い状態で記録規則だけを入れない。

**観測を持つことが目標を約束したことにならないよう注意する。** 種別別の系列が出ると、そこへ閾値を置きたくなる。`docs/capacity.md` の Service level objectives は現在すべて認証・プロトコル系を母集団としており、その範囲は [[wi-459-api-process-plane-separation-decision]] が意図的に変えていない。目標の新設は実測が集まってから別に判断する。

`reversibility` は reversible。記録規則とダッシュボードは削除できる。ただし記録規則の名前は保存された系列の識別子になるので、名前を変えると過去のデータと接続しなくなる。命名は `infra/k8s/monitoring/prometheus-rule.yaml` の既存の `idmagic:` 接頭辞に揃え、後から変えない。

## Verification

- `mise run check-monitoring`
- `mise run check-slo-references`
- `mise run verify`
