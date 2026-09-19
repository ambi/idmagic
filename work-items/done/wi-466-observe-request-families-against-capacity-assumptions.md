---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-03
priority: p2
depends_on: []
change_kind: operations
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 記録規則、ダッシュボード、網羅性検査の追加であり、利用者に見える振る舞いも公開契約も変わらない。読み手は運用者だけなので、宛先はリリース文書ではなく docs/design/observability/monitoring.md である。
  references: []
initial_context:
  specification: [docs/domain/system/scenarios.feature.md#REQ-SYSTEM-001]
  typespec: []
  source:
    - backend/shared/http/server_http/priority_class.go
    - backend/shared/http/server_http/gateway_allowlist.go
    - backend/shared/http/support_http/metrics_middleware.go
    - backend/shared/observability/metrics_prometheus/metrics.go
    - backend/cmd/idmagic-gateway-routes/main.go
    - infra/docker/prometheus-rules.yml
    - infra/k8s/monitoring/prometheus-rule.yaml
    - infra/docker/grafana-dashboard.json
    - infra/k8s/monitoring/grafana-dashboard.yaml
    - docs/design/performance/capacity.md
    - docs/design/observability/README.md
    - docs/design/observability/monitoring.md
    - tools/check/src/registry.ts
    - tools/check/src/check-slo-references.ts
    - ROUTE_PRIORITY.md
    - mise.toml
  tests:
    - backend/shared/http/server_http/priority_class_test.go
  stop_before_reading:
    - work-items/done/wi-459-api-process-plane-separation-decision.md
    - docs/requirements/quality.md
    - frontend
affected_spec:
  - { path: docs/domain/system/scenarios.feature.md, requirement: REQ-SYSTEM-001 }
---

# 種別ごとの到達率と処理時間を観測し、容量の Planning assumption を Measurement へ置き換えられるようにする

## Motivation

[[wi-459-api-process-plane-separation-decision]] は API の Deployment を種別ごとに分けない判断を記録した。その判断は永久のものではなく、6 つの再検討条件（C1–C6）を伴う。**そのうち負荷に関わる 2 つは、いま検知できない。**

- **C1**「管理系、ポータル系、SCIM、Shared Signals のいずれかを含む混合負荷で認証系のサービス目標を満たせない」— 認証系のサービス目標は `docs/requirements/quality.md` の `SLO-*` として定義され、`infra/k8s/monitoring/prometheus-rule.yaml` が `/token` とログインについてバーンレートを見ている。しかし**目標が侵されたときに、どの種別がそれを侵したのかを示す観測がない。** 侵害は見えるが、原因の種別は見えない。
- **C2**「分離によって 1 レプリカ当たり持続処理能力が上がる」— 判定には混合ごとの Measurement が要る。

同時に、同 work item が `docs/design/performance/capacity.md` へ追加した Non-protocol request profile は、**18 個の入力すべてが Planning assumption で、Measurement が 1 つもない。** 幅は 2 桁に及ぶ。同書の Evidence classes は「ステージングの容量検証では Planning assumption を Measurement へ置き換える」と定めているが、置き換える先の観測が本番にもステージングにも存在しない。

現在あるのは `http_requests_total`、`http_request_duration_seconds`、`http_requests_in_flight` の 3 つと、`route`、`method`、`status_code` のラベルである。**種別で集約する手段がない。** `route` は登録済みのルートパターンなので、接頭辞から種別は導けるが、その導出をどこにも持っていない。記録規則にもダッシュボードにもアラートにも、管理系、ポータル系、SCIM、Shared Signals を母集団とするものは 1 つもない。

つまり「分けない」という判断は、それを覆す条件を観測できないまま置かれている。**再検討条件が検知できない判断は、判断ではなく既定値である。**

## Scope

- `route` ラベルから種別（認証・プロトコル系、ポータル系、管理系、SCIM、Shared Signals の受信、運用経路）を導く対応を 1 か所に定義する。
- 種別ごとの到達率、レイテンシー分位、非 5xx 比率、実行中要求数の記録規則を追加する。
- ダッシュボードに種別ごとの内訳を出す。`docs/design/performance/capacity.md` の Non-protocol request profile の各行に対応する観測を、同じ単位（通常時、最繁時、集中実行時）で読めるようにする。
- 認証系のサービス目標が侵されたときに、同じ時間窓の種別別内訳を並べて読めるようにする。C1 の判定に使う。
- 種別に対応しない `route` が存在しないことを検査する。分類漏れがあれば、その経路は観測から静かに消える。
- `docs/design/observability/README.md` に種別の対応と、この観測が `docs/design/performance/capacity.md` のどの行を置き換えるためのものかを記録する。

## Out of Scope

- 種別ごとのサービス目標（`SLO-*`）を新設すること。**観測を持つことと目標を約束することは別である。** 目標を置くかどうかは、実測が集まってから別に判断する。とくにポータル系については [[wi-459-api-process-plane-separation-decision]] が「対話的な利用者操作として認証系と同じ可用性優先度を与えるか、独立した容量と SLO を持たせるかを容量シナリオと正準文書で決める」として未決のまま残している。
- 優先度クラスの分類と、それに基づく入場制御。[[wi-396-prioritize-login-under-saturation]] で実装済みであり、本項目は既存の経路正規化と網羅性検査を再利用するが、優先度クラスの意味や対応は変えない。
- ステージングでの負荷試験と、混合負荷での 1 レプリカ当たり持続処理能力の実測。[[wi-282-staging-load-testing-and-capacity-validation]] が持つ。本 work item はその測定結果を読める形を用意する側である。
- `docs/design/performance/capacity.md` の Planning assumption を実際に Measurement へ書き換えること。観測が回ってからの作業であり、書き換えは測った人が行う。
- 新しいメトリクスの追加。既存の 3 つと既存のラベルで足りるかを Design で確かめ、足りない場合だけ Scope へ戻す。

## Design

### 着手時に確かめた前提の変化

起票から着手までに、この記録が名指す文書が動いた。

`docs/design/performance/capacity.md` の Non-protocol request profile は、日本語化と再編を経て「管理と連携」節になった。
行は管理コンソール、管理 API による自動化、アカウントポータル、SCIM、Shared Signals の 5 つ、列は平常時、最繁時、一括実行時の 3 つで、値は 13 個である。
18 個という起票時の数は再編より前の形を指す。
値がすべて仮定値で実測値が 1 つも無いという前提は変わっていない。

対応を記録する宛先は `docs/design/observability/README.md` から `docs/design/observability/monitoring.md` へ移す。
README は自分の担当を「個別のメトリクス、ログ、トレースの設計は子文書が持つ」と定め、子文書の担当表で指標ごとの設計を監視設計へ委ねている。
種別の対応は指標のラベルの設計なので、README へ置けば README が自分で定めた分担を破る。

### ラベルを増やすか、記録規則で導くか

| 案 | 内容 | 利点 | 欠点 |
| --- | --- | --- | --- |
| A | `http_requests_total` に `family` ラベルを足す | 集約が単純。分類が計測点にあるので取りこぼさない | カーディナリティが増える。既存の記録規則とアラートを見直す必要がある。分類を変えると過去の系列と接続しない |
| B | Prometheus の記録規則で `route` の正規表現から種別を導く | コードを変えない。分類を変えても過去のデータへ遡って適用できる | 分類が監視資材の側にあり、経路を足した人が気づかない。`route` の値と正規表現がずれても静かに落ちる |
| C | B に加えて、種別に対応しない `route` が無いことを検査する | B の欠点を塞ぐ | 検査が `route` の全量を知る必要がある |

**C を採る。** 分類の実体は記録規則に置き、その網羅性をリポジトリの検査で保証する。`route` の全量は `backend/shared/spec/operations_gen.go` から取れる。A を採らないのは、種別が用途による分類であって計測の属性ではないためで、同じ理由で分類は後から変わりうる。過去の系列と接続しないのは、その変更を高くつかせる。

`docs/design/performance/capacity.md` の Measurement boundary は「`route` は解決済みのパスではなく登録済みのルートパターンで集約し、realm 接頭辞を持つ同じ操作も同じエンドポイント群へ含める」と定めている。**種別の導出も同じ規則に従う。** つまり `/realms/{tenant_id}/api/admin/v1/...` と `/api/admin/v1/...` は同じ種別になる。

### 分類の対応

[[wi-459-api-process-plane-separation-decision]] が定めた種別をそのまま使う。分類の実体を 2 つ持たない。

| 種別 | `family` ラベルの値 | 経路 |
| --- | --- | --- |
| 認証・プロトコル系 | `authentication` | `/authorize`、`/token`、`/introspect`、`/revoke`、`/userinfo`、`/par`、`/bc-authorize`、`/device_authorization`、`/end_session`、`/register`、`/jwks`、`/.well-known/*`、`/session/check`、`/saml/*`、`/wsfed`、`/federationmetadata/*`、`/trust/*`、`/api/auth/*`、`/api/branding/*`、`/tenant-branding-assets/*`、`/application-icons/*` |
| ポータル系 | `portal` | `/api/account/v1/*` |
| 管理系 | `management` | `/api/admin/v1/*` |
| SCIM | `scim` | `/scim/v2/*` |
| Shared Signals | `shared_signals` | `/ssf/*` |
| 運用経路 | `operations` | `/livez`、`/readyz`、`/startupz`、`/health`、`/metrics` |

Shared Signals のストリーム管理は `/api/admin/v1/shared-signals/*` にあるので管理系に入る。`/ssf` に残るのは受信 1 経路だけである。

T001 で `ROUTE_PRIORITY.md` の全量と照合し、起票時の表が名指していなかった 2 経路を補った。
`/register` を認証・プロトコル系へ入れるのは、`docs/design/performance/capacity.md` の「認証とトークン発行」表がこの経路を自分の母集団として挙げているからである。
`/federationmetadata/*` は WS-Federation のメタデータなので `/wsfed` と同じ行に入る。

種別と優先度クラスが食い違う経路は 2 つある。
`/register` の優先度クラスは `management`、`/api/account/v1/step_up/*` の優先度クラスは `interactive_auth` で、どちらも種別とは別の行に落ちる。
食い違いは誤りではない。
優先度クラスは飽和時に何を先に捨てるかを決め、種別はどの見積もりの行に対応するかを決める。

### wi-396 の優先度クラスとの関係

[[wi-396-prioritize-login-under-saturation]] は要求を優先度クラスへ分類する `ClassifyRoute`、テナント接頭辞とパラメーター名をそろえる経路正規化、組み立て済み router の全経路が分類されることを確かめるテストを実装した。**本 work item の種別と、その優先度クラスは別の分類である。** 種別は用途、優先度クラスは飽和時に何を先に捨てるかで、たとえば管理系の中でも参照は残して集計を捨てるという分け方はありうる。

種別の値と Prometheus の記録規則は本項目で所有するが、経路の正規化規則と全量の取得は既存実装を再利用する。種別を優先度クラスから導くと異なる意味を一つに潰すため、対応表は別に持つ。網羅性テストは同じ組み立て済み router を入力にして、優先度クラスと種別をそれぞれ独立に検査する。

### 記録規則の形

1 つのグループ `idmagic-request-families` に 7 種類の規則を置く。
上 3 つが種別ごとに 1 本ずつ、下 4 つは種別を知らず `family` ラベルの付いた系列を集計するだけである。

| 記録規則 | 保持するラベル | 由来 |
| --- | --- | --- |
| `idmagic:http_requests_by_family_status:rate5m` | `family`、`status_code` | `http_requests_total` を種別の正規表現で絞る |
| `idmagic:http_request_duration_seconds_by_family:histogram_rate5m` | `family`、`le` | `http_request_duration_seconds_bucket` を同じ正規表現で絞る |
| `idmagic:http_requests_in_flight_by_family:sum` | `family` | `http_requests_in_flight` を同じ正規表現で絞る |
| `idmagic:http_requests_by_family:rate5m` | `family` | 1 行目を `family` で合計した到達率 |
| `idmagic:http_request_success_ratio_by_family:ratio_rate5m` | `family` | 1 行目のうち 5xx でないものの比 |
| `idmagic:http_request_duration_seconds_by_family:p50_5m` | `family` | 2 行目から求める中央値 |
| `idmagic:http_request_duration_seconds_by_family:p99_5m` | `family` | 2 行目から求める p99 |

種別の正規表現が現れるのは 1 種別につき 3 か所である。
ヒストグラムのバケットと実行中要求数は生の指標からしか導けないので、これ以上は減らせない。
そのかわり、同じ種別の 3 つの規則が同じ正規表現を使っていることを検査で固定する。
1 つだけずれた状態は、到達率は正しく分位だけが間違っているという読み違いを生み、系列そのものは出ているのでダッシュボードには現れない。

新しい指標は足さない。
`MetricsMiddleware` は 3 つの指標すべてに `route` を付けており、種別はそこから導ける。

### 網羅性検査の置き場所

検査は `backend/shared/http/server_http/request_family_test.go` に置く。
新しいコマンドも新しい `mise` タスクも作らない。

経路の全量を得られるのは `Register` が組み立てた echo の router だけである。
`route` ラベルの値は `MetricsMiddleware` が入れる `c.Path()`、つまり `/realms/:tenant_id` 接頭辞を含んだままのルートパターンだからである。
`backend/shared/spec/operations_gen.go` は照合の入力にならない。
生成物のパスは接頭辞を持たず、パラメーターも `{name}` 形式なので、`route` ラベルに現れる字面と一致しない。

同じ入力を使う検査が、同じパッケージに既にある。
`ops_assets_test.go` は `infra/` のマニフェストと収集設定を読み、組み立て済みの router が登録した経路と突き合わせる。
リポジトリ root への相対パス、複数ドキュメントの YAML の読み取り、空振りを落とす番人は、そこに揃っている。
種別の網羅性は同じ種類の検査なので、同じ形に従う。

起票時の Design はコマンドの形（`CheckGatewayAllowlists` と `backend/cmd/idmagic-gateway-routes` の対）を想定していた。
その形を採らないのは、検査の対象が `go test` の外から要る場面が無いためである。
ゲートウェイの検査がコマンドなのは、`tools/check` からも CI からも同じ判定を呼べるようにするためであり、ここにその要求は無い。
生成物も出さないので、`--check` と生成を兼ねる必要もない。
純関数を製品コードへ置いてテストだけが呼ぶ状態も作らない。

| 判定内容 | 失敗時に落とすもの |
| --- | --- |
| 記録規則が宣言する種別が、テストの宣言する 6 つと一致する | 種別の改名と削除 |
| 同じ種別の規則が同じ経路の正規表現を使う | 指標ごとに種別の範囲がずれた状態 |
| 2 つの規則ファイルが同じ種別へ同じ正規表現を与える | Docker Compose と Kubernetes の乖離 |
| 組み立て済みの経路が、ちょうど 1 つの種別に一致する | 分類漏れと二重計上 |
| 各種別が、少なくとも 1 つの経路に一致する | 何にも当たらない正規表現による空振り |

この検査は `REQ-SYSTEM-001` のどの具体例も主張しない。
`EX-SYSTEM-001-01` が定めるのは OAuth2/OIDC を母集団とする可用性、レイテンシー、非 5xx 比率の評価であり、種別別の観測はその母集団を変えない。
同じファイルの上にある 2 つのテストがその具体例を被覆しており、そこへ重ねて `//spec:covers` を書くと、被覆していないものを被覆したことにする。

| 判定内容 | 失敗時に落とすもの |
| --- | --- |
| 記録規則が宣言する種別が、Go の宣言する 6 つと一致する | 種別の改名と削除 |
| 同じ種別の規則が同じ経路の正規表現を使う | 指標ごとに種別の範囲がずれた状態 |
| 2 つの規則ファイルが同じ種別へ同じ正規表現を与える | Docker Compose と Kubernetes の乖離 |
| 組み立て済みの経路が、ちょうど 1 つの種別に一致する | 分類漏れと二重計上 |
| 各種別が、少なくとも 1 つの経路に一致する | 何にも当たらない正規表現による空振り |

種別の名前だけを検査側に置き、経路との対応は規則ファイルに残す。
検査が持つのは語彙であって 2 つ目の対応表ではない。
名前を検査に置かないと、`portal` を `account` へ改名した変更が経路の網羅性だけを見る検査を素通りし、ダッシュボードと文書だけが静かに空になる。

「ちょうど 1 つ」を見るのは、漏れと重なりが別の誤りだからである。
漏れた経路は集計から消え、重なった経路は 2 つの種別に数えられる。
後者は種別の合計が全体を超えるので合計と比べれば気づけるが、比べるのは人である。

### 読み取れないこと

`route` ラベルでは分けられない区別が 2 つある。

管理コンソールと管理 API による自動化は、どちらも `/api/admin/v1/*` に着地する。
分類の対応表が両者を管理系 1 つへ潰しているのはこのためで、`docs/design/performance/capacity.md` の 2 行に対して観測は 1 系列しか作れない。
これは分類の不足ではなく、経路で分けられないものを分けたことにしないという立場の帰結である。
`ClassifyRoute` が SCIM の全同期の列挙と差分同期の 1 件解決に対して取ったのと同じ判断になる。
両者を分けるには要求の出どころを見るしかなく、それは経路の分類とは別の機構になる。

もう 1 つは、どの経路にも一致しなかった要求である。
`MetricsMiddleware` は `c.Path()` が空のとき `route` へ `unmatched` を入れる。
組み立て済みの router にこの値は無いので網羅性の検査は見ないが、種別の合計は `http_requests_total` の合計より少なくなる。
差は経路に当たらなかった要求であり、欠測ではない。

どちらも `docs/design/observability/monitoring.md` に読み方として書く。

## Plan

1. `route` ラベルの全量を `ROUTE_PRIORITY.md` で確認し、分類がすべてを覆うことを確かめる。
2. 検査を先に書く。記録規則がまだ種別を 1 つも宣言していない状態で走らせ、全経路が未分類として並ぶことを確かめる。この出力が `route` ラベルの全量そのものなので、分類の見直しにも使う。
3. 記録規則を 2 つのファイルへ追加し、検査を GREEN にする。
4. ダッシュボードに種別別の内訳を出す。`docs/design/performance/capacity.md` の「管理と連携」の行と同じ単位で読めるようにする。
5. `docs/design/observability/monitoring.md` に対応と読み方を記録する。数値は写さず、`docs/design/performance/capacity.md` の節を名指しする。

### 証拠の選び方

`risk` は low、`change_kind` は `operations` であり、`primary_use_cases` の対象ではない。
プロダクトの振る舞いを変えないので、製品の受け入れ境界に置ける RED が無い。
代わりに次の 2 つを取る。

- Acceptance RED: `TestRequestFamilyRulesCoverEveryAssembledRoute`。実際の 2 つの規則ファイルがまだ種別を宣言していない状態で走らせ、組み立て済みの全経路が未分類として報告されることを観測する。
- Unit RED: `TestRequestFamilyFindingsNameEachKindOfDrift`。合成した規則の字面を入力に、漏れ、重なり、規則間の不一致、ファイル間の乖離、空振りのそれぞれが別々の所見として出ることを観測する。

検査が観測しているのは製品の振る舞いではなく、監視資材と経路表の整合である。
したがって `Requirement` は `N/A` とし、代わりに失敗する検査を名指しする。

## Tasks

- [x] T001 [Research] `route` ラベルの全量を確認し、分類の網羅性を確かめる。`ROUTE_PRIORITY.md` の全量と照合し、`/register` と `/federationmetadata/*` を分類の対応表へ補った。
- [x] T002 [Design] 再利用する境界を固定した。経路の全量は `echoRoutes` と同じ組み立て済み router、種別の対応は記録規則、語彙は Go。優先度クラスとは独立に検査する。
- [x] T003 [Acceptance] `request_family_test.go` を `ops_assets_test.go` の形で書き、種別を宣言していない規則ファイルに対して `TestRequestFamilyRulesCoverEveryAssembledRoute` が RED になることを確かめた。505 経路すべてが未分類として並んだ。
- [x] T004 [Unit] `TestRequestFamilyFindingsNameEachKindOfDrift` で、6 つの崩れ方がそれぞれ別の所見として出ることを確かめた。
- [x] T005 [Monitoring] 種別ごとの記録規則を `infra/docker/prometheus-rules.yml` と `infra/k8s/monitoring/prometheus-rule.yaml` へ追加し、T003 と T004 を GREEN にした。
- [x] T006 [Monitoring] ダッシュボード 2 枚へ種別別の内訳を 4 枚のパネルとして出した。Docker Compose 側では二段目に挿入し、以降の段を 1 段ずつ下げた。
- [x] T007 [Docs] `docs/design/observability/monitoring.md` に「要求の種別」節を追加し、分類、優先度クラスとの違い、読み取れないことを記録した。ダッシュボードの構成表にも二段目を足した。
- [x] T008 [Verify] `mise run check-monitoring`、`mise run check-slo-references`、`mise run verify` を通した。

## Risk Notes

リスクは low。観測を足すだけで、製品の振る舞いも公開契約も変えない。

**分類漏れは静かに効く。** 種別に対応しない経路は、集計から消えるだけで警告を出さない。それでは「管理系の到達率は低い」という観測が、実は分類漏れだったという読み違いを生む。網羅性の検査を先に入れ、検査が無い状態で記録規則だけを入れない。

**観測を持つことが目標を約束したことにならないよう注意する。** 種別別の系列が出ると、そこへ閾値を置きたくなる。`docs/requirements/quality.md` の Service level objectives は現在すべて認証・プロトコル系を母集団としており、その範囲は [[wi-459-api-process-plane-separation-decision]] が意図的に変えていない。目標の新設は実測が集まってから別に判断する。

`reversibility` は reversible。記録規則とダッシュボードは削除できる。ただし記録規則の名前は保存された系列の識別子になるので、名前を変えると過去のデータと接続しなくなる。命名は `infra/k8s/monitoring/prometheus-rule.yaml` の既存の `idmagic:` 接頭辞に揃え、後から変えない。

## Verification

- `mise run check-monitoring`
- `mise run check-slo-references`
- `mise run test-go-package -- ./backend/shared/http/server_http`
- `mise run verify`

## 完了

- **Completed At**: 2026-09-19
- **Summary**:
  `mise run spec-diff` reports no normative specification change against `main`: the work adds observation,
  not behaviour. The semantic difference is that requests can now be read by purpose. Six request families
  (`authentication`, `portal`, `management`, `scim`, `shared_signals`, `operations`) are derived from the
  `route` label by regular expressions in the two Prometheus rule files, and carried as a `family` label on
  seven recording rules covering arrival rate, non-5xx ratio, p50/p99 latency, and in-flight requests. Both
  Grafana dashboards show the breakdown directly beneath the route-level panels, so a burning authentication
  objective can be read against the same time window's family mix — the observation the C1 reconsideration
  condition of the API process-plane separation decision needs and did not have. The classification is
  exhaustive by construction: a Go test matches all 505 assembled route patterns against the rule files and
  fails when a route matches no family, matches more than one, when the three rules of one family disagree,
  or when the two rule files disagree. No service level objective was added, and no planning assumption in
  the capacity design was rewritten; both remain out of scope until measurements exist.
- **Acceptance RED Evidence**:
  - **Test**: `TestRequestFamilyRulesCoverEveryAssembledRoute` in
    `backend/shared/http/server_http/request_family_test.go`, run with
    `mise run test-go-test -- ./backend/shared/http/server_http TestRequestFamilyRulesCoverEveryAssembledRoute`.
  - **Requirement**: N/A: この検査は製品の振る舞いではなく、監視資材と経路表の整合を観測する。REQ-SYSTEM-001 の具体例は OAuth2/OIDC を母集団とする評価を定めており、種別別の観測はその母集団を変えないので、被覆を主張しない。
  - **Observed Failure**: 規則ファイルがまだ種別を 1 つも宣言していない状態で、2 つの資材それぞれについて
    「no recording rule labels family ...」が 6 件と、505 本すべての経路について
    「matches no family, so it vanishes from every by-family series without warning」が出た。
  - **Detection Reason**: 照合の入力が、生成物でも手書きの一覧でもなく `Register` が組み立てた router の
    全量である。経路を 1 本足して分類を足し忘れた状態は、その 1 本だけが所見として現れる。
    件数ではなく経路ごとの一致を見るので、数だけ合って中身が入れ替わった分類は通らない。
    経路が 0 本のときは空振りとして落とすので、router の組み立てが壊れた状態を合格と読まない。
- **Unit RED Evidence**:
  - **Test**: `TestRequestFamilyFindingsNameEachKindOfDrift` in the same file, run with
    `mise run test-go-test -- ./backend/shared/http/server_http TestRequestFamilyFindingsNameEachKindOfDrift`.
  - **Requirement**: N/A: 上と同じ理由による。合成した規則の字面を入力に、所見の切り分けだけを観測する。
  - **Observed Failure**: 所見を 1 種類へまとめた実装では、`family "portal" is selected by two different
    route expressions`、`matches no family`、`matches 2 families`、`matches no assembled route` の
    各部分文字列がどれも現れず、対応する部分テストが `no finding mentions ...` で落ちた。
  - **Detection Reason**: 網羅性だけを見る検査は、所見をまとめた実装でも通る。通ったうえで運用者は
    「どの経路が漏れたのか」にも「どちらのファイルがずれたのか」にも答えられない。部分テストは
    崩れ方ごとに別の所見を要求するので、区別を失った実装を通さない。
- **Change-Resistance Results**:
  `mise run test-go-mutation -- ./backend/shared/http/server_http` は 101 個の変異のうち 57 個を検出し、
  30 個が生き残った。生存はすべて `routes.go` (17)、`gateway_allowlist.go` (11)、`health_handler.go` (2)
  にあり、本項目が触れた行は 1 つも無い。本項目は製品コードを変更しないので、字面を書き換える変異器は
  この変更について何も言わない。スコアは付けない。
  検出能力は、資材への明示的な故障注入で測った。7 つすべてを検査が捕らえた。
  各回はファイルを複写して退避し、複写から書き戻した。

  | 注入した故障 | 観測された所見 |
  | --- | --- |
  | `scim` の記録規則 3 本を丸ごと削除 | `no recording rule labels family "scim"` と、SCIM の 7 経路の `matches no family` |
  | `portal` を `account` へ改名 | `labels family "account", which is not one of ...` |
  | `portal` の正規表現を `.*` へ広げる | `matches 2 families (authentication, portal)` |
  | `management` を `/api/admin/v2/.*` へ狭める | 管理系の全経路の `matches no family` と `family "management" matches no assembled route` |
  | `shared_signals` を Kubernetes 側だけ書き換える | `select family "shared_signals" differently` |
  | `scim` のヒストグラム 1 本だけを `/scim/.*` へ広げる | `family "scim" is selected by two different route expressions` |
  | PromQL 文字列の脱出解除を外す | `/.well-known/*` の 3 経路の `matches no family` |

  最後の 1 件は、検査自身の配線を外す故障である。資材の `\\.` を脱出解除せずに Go の正規表現へ渡すと、
  資材が正しいときに限って落ちる。この 1 段が効いていることを、外して確かめた。
- **Verification Results**:
  - `mise run verify` - passed
  - `mise run check-monitoring` - passed (promtool: 41 rules found)
  - `mise run check-slo-references` - passed (19 objective(s), 2 monitoring asset(s))
  - `mise run test-go-package -- ./backend/shared/http/server_http` - passed
  - `mise run test-ui-e2e` - N/A: 変更はブラウザーへ届かない。記録規則、ダッシュボードの定義、Go のテスト、
    設計文書だけを変更しており、フロントエンドの資材も API の応答も変えていない。
