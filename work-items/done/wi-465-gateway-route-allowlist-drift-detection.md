---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-03
priority: p1
depends_on: []
change_kind: bugfix
evidence_policy: risk-based-v3
documentation_impact:
  level: release_note
  reason: 参照ゲートウェイの背後で届いていなかった経路が届くようになる。読者は自分のデプロイでも同じ許可リストを直す必要があるかを判断できなければならない。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-465.md }
initial_context:
  specification:
    - docs/domain/system/scenarios.feature.md#REQ-SYSTEM-019
    - docs/domain/system/scenarios.feature.md#REQ-SYSTEM-021
  typespec: [IdMagic.SharedSignals.Operations.ReceiveSecurityEvent]
  source:
    - frontend/Caddyfile
    - frontend/vite.config.ts
    - backend/shared/http/server_http/routes.go
    - backend/shared/http/server_http/priority_class.go
    - backend/shared/http/server_http/priority_class_reference.go
    - backend/cmd/idmagic-route-reference/main.go
    - ROUTE_PRIORITY.md
    - docs/design/architecture/deployment.md
    - mise.toml
  tests:
    - frontend/src/devProxy.test.ts
    - backend/shared/http/server_http/priority_class_test.go
  stop_before_reading:
    - backend/sharedsignals
    - spec/contexts/sharedsignals/main.tsp
affected_spec:
  - { path: docs/domain/system/scenarios.feature.md, requirement: REQ-SYSTEM-021 }
  - { path: docs/domain/system/scenarios.feature.md, requirement: REQ-SYSTEM-001 }
  - { path: spec/contexts/sharedsignals/main.tsp, symbol: IdMagic.SharedSignals.Operations.ReceiveSecurityEvent }
primary_use_cases:
  - id: reach-required-routes-through-gateway
    requirement: REQ-SYSTEM-021
    observable_result: 参照ゲートウェイ経由で `/ssf/streams/{stream_id}/events` と `/session/check` を呼ぶと API の応答が返り、SPA の `index.html` は返らない。
    unit_test: { path: backend/shared/http/server_http/gateway_allowlist_test.go, name: TestGatewayAllowlistsCarryEveryRequiredRoute, task: test-go-race }
    e2e_test: { path: frontend/tests/e2e/gateway-routes.spec.ts, name: gateway proxies the runtime routes the SPA fallback would otherwise swallow, task: test-ui-e2e }
    unit_fault_model: 照合が必須経路の欠落を見落とす。許可リストの抽出が空を返しても突き合わせが成功する。
    e2e_fault_model: ゲートウェイの設定が経路を中継せず、要求が SPA の `index.html` に落ちる。
---

# ゲートウェイの経路許可リストが実行時の経路表からずれていることを検出し、現在のずれを塞ぐ

## Motivation

ブラウザーから見える境界を同一オリジンに揃えるのはゲートウェイの役目である（`docs/design/architecture/deployment.md`）。そのゲートウェイは、どのパスを Go へ渡すかを**手書きの接頭辞列挙**で決めている。列挙は 3 か所にあり、いずれも実行時の経路表と照合されていない。

| 場所 | 形 |
| --- | --- |
| `frontend/Caddyfile` の `@backend` | パス接頭辞の列挙 |
| `frontend/Caddyfile` の `@realmBackend` | `^/realms/[^/]+/(api\|scim\|saml\|...)` の正規表現 |
| `frontend/vite.config.ts` の `server.proxy` | 上の 2 つとほぼ同じ内容を JavaScript で再掲 |

**3 つとも、実行時に登録されている次の経路を含まない。**

| 欠落している経路 | 何が壊れるか |
| --- | --- |
| `/ssf/streams/{stream_id}/events` | 顧客側の IdP が送るセキュリティイベントの受信。Shared Signals の受信経路はこれ 1 本しかないので、参照ゲートウェイの背後では受信機能そのものが届かない |
| `/session/check` | OIDC Session Management の `check_session_iframe`。RP が埋め込む iframe が SPA の `index.html` を受け取る |
| `/application-icons/{application_id}/{id}` | 管理コンソールが `support.TenantRoute` で組み立てるアプリケーションアイコンの URL |
| `/livez`、`/readyz`、`/startupz`、`/metrics` | ゲートウェイ経由での到達。Kubernetes は Pod へ直接当てるので本番の判定には影響しないが、外形監視は届かない |

**このずれは静かに失敗する。** ゲートウェイの `handle` は最後に SPA へ落ちるので、外れた経路は 404 ではなく `index.html` を 200 で返す。`Content-Type` は `text/html` になり、SET の受信も iframe も画像も、エラーではなく「意味のない成功」として観測される。

`frontend/src/devProxy.test.ts` に検査が 1 つあるが、Vite の開発プロキシだけを対象にしており、しかも 8 本の経路を手で列挙している。`Caddyfile` は何にも検査されていない。列挙を手で足す検査は、列挙を手で足す設定と同じ速度でずれる。

[[wi-459-api-process-plane-separation-decision]] は API の Deployment を種別ごとに分けない判断を記録し、その Out of Scope でこの照合を別の work item へ渡した。同じ work item は、**このずれが存在する限り「後から分けるのは安い」とはみなさない**とも書いている。分割は後段を複数の上流へ振り分ける変更なので、振り分けの正しさを確かめる手段がなければ移行の検証対象が読めない。ただし本 work item が塞ぐのは分割の準備ではなく、単一プロセスのままでも現に壊れている経路である。

## Scope

- 実行時の経路表とゲートウェイの許可リストを照合し、ずれを機械的に検出する検査を追加する。`mise run check-*` の 1 つとして常時走らせる。
- 検査の対象に `frontend/Caddyfile` の `@backend` と `@realmBackend`、`frontend/vite.config.ts` の `server.proxy` の 3 つすべてを含める。
- 現在のずれを塞ぐ。`/ssf/*`、`/session/check`、`/application-icons/*` と、運用経路の扱いを決める。
- 経路表の正をどこから取るかを決める。`backend/shared/spec/operations_gen.go` は TypeSpec から生成された 323 件の経路表を持ち、`mise run check-generated-contract` が実行時の経路メタデータと照合している。この既存の正を使えるかを Design で確かめる。
- ホスト形式（`/` 直下）とパス形式（`/realms/{tenant_id}/` 配下）の両方を照合する。`@realmBackend` の正規表現は `@backend` と別の列挙になっているので、片方だけの照合では足りない。

## Out of Scope

- ゲートウェイ設定そのものを必須のランタイムにすること。`docs/design/architecture/deployment.md` の「Caddy は参照用の設定であり、必須のランタイムではない」という判断は変えない。検査するのはリポジトリが同梱する参照設定である。
- 列挙を 1 か所へ統合し、Caddyfile と Vite の設定を生成物にすること。有力だが、生成に倒すかどうかは Design で判断する。判断が生成なら本 work item で実施し、そうでなければ照合だけを持つ。
- 経路ごとの認可、ヘッダー、キャッシュ方針の照合。本 work item は「Go へ届くか」だけを見る。
- 同一オリジンの前提が満たされているかの実行時検証。[[wi-426-same-origin-deployment-assumption-detection]] が持つ。
- API プロセスを種別ごとに分けること、およびそのときの振り分け設計。[[wi-459-api-process-plane-separation-decision]] が判断を持つ。

## Design

### 正をどこから取るか

**組み立て済みの router を正とする。** `Register(echo.New(), Deps{})` が登録した経路パターンの全量であり、`RenderPriorityClassReference` と `TestEveryAssembledRouteDeclaresAPriorityClass` が既に同じ正を使っている。`Deps` は各ハンドラーが何をできるかを決めるだけで、どの経路が存在するかは決めないので、依存を持たない零値でも本番と同じ集合が並ぶ。

`backend/shared/spec/operations_gen.go` は採らない。2 点で足りない。

| 足りない点 | 内容 |
| --- | --- |
| 非宣言経路 | `/livez`、`/readyz`、`/startupz`、`/health` は `Register` が直接登録し、生成表には現れない。生成表を正にすると、ゲートウェイを通してはならない経路の分類がそもそも書けない |
| 形の区別 | 生成表は既定テナントの形だけを持つ。realm 形は `e.Group("/realms/:tenant_id")` への再登録で生まれるので、表からは読めない。`@realmBackend` が別の列挙である以上、両方の形が要る |

`ROUTE_PRIORITY.md` も採らない。`referencePath` が `/realms/:tenant_id` を落として両形を 1 行に潰すため、realm 形の照合に使えない。

網羅性は `TestEveryAssembledRouteDeclaresAPriorityClass` が支えている。router に載る経路はすべて優先度クラスを持つので、クラスから導く公開可否の分類にも抜けが出ない。router に載らない経路（ミドルウェアだけが応答する経路）は現在存在せず、増えればこの前提が崩れる。それは Risk に残す。

### 照合の向き

照合は経路表の側から引く。組み立て済みの経路それぞれについて、許可リストが通すかどうかを問い、分類が `required` なら通らないことを、`forbidden` なら通ることを失敗とする。

**許可リストの側から引く向き、つまり「ゲートウェイが通すが実行時に経路が無い」は見ない。** 廃止した経路の掃除漏れであって、機能は壊れないからである。警告としても出さない。設定の側から経路の不在を言うには、接頭辞 1 件が経路 0 件を覆っていることを判定する必要があり、それは `/api/*` のような広い接頭辞では常に偽になる。**判定できない主張を警告として出すと、出ないことが根拠として読まれてしまう。**

失敗の意味は 2 つに保つ。「利用者に届かない機能がある」と「公開してはならないものが公開入口にある」である。どちらも設定を直すまで直らない。

### 生成に倒すか、照合にとどめるか

| 案 | 利点 | 欠点 |
| --- | --- | --- |
| 照合だけを持つ | 小さい。既存の設定ファイルの形と読みやすさを変えない。参照設定であるという位置付けとも整合する | ずれは検出できるが、直すのは人である。3 か所を手で同期する作業は残る |
| 接頭辞の列挙を生成物にする | 3 か所のずれが構造的に起きなくなる | 生成物を追跡するか、生成を検査に含めるかを決める必要がある。`Caddyfile` は運用者が読んで理解する参照設定でもあるので、生成物にすると読みにくくなりうる |

**照合を先に入れる。** 現在のずれを塞ぐことと、再発を検出することが目的であり、それは照合で達成できる。生成は、照合が「毎回同じ 3 か所を直す」作業に落ちたときに改めて判断する。判断材料が出るまで、読みやすさを確実に失う変更はしない。

### 運用経路の扱いと分類の置き場所

`/livez`、`/readyz`、`/startupz`、`/metrics` は Kubernetes が Pod へ直接当てるので、ゲートウェイを通す必要は必ずしもない。`/metrics` は認証を持たないため、`docs/domain/system/decisions.md` が「公開先は折り返しアドレス、管理用ネットワーク、認証付きプロキシの背後に限る」と定めている。**つまり `/metrics` はゲートウェイを通してはならない経路である。**

分類は `ClassifyRoute` と同じ場所、すなわち経路の登録と同じパッケージに置く。起動時設定にも照合側の設定にも置かない。置けばデプロイごとに公開範囲が変わる状態ができ、`ClassifyRoute` が起動時設定を避けたのと同じ理由でそれを避ける。分類の値は 3 つとする。

| 値 | 意味 | 照合の判定 |
| --- | --- | --- |
| `required` | 利用者または外部の当事者がゲートウェイ越しに呼ぶ | どれか 1 つの許可リストに一致しなければ失敗 |
| `optional` | 通しても通さなくてもよい | 一致の有無を判定しない |
| `forbidden` | ゲートウェイを通してはならない | いずれかの許可リストに一致したら失敗 |

明示規則を先に引き、当たらなければ優先度クラスから導く。`/health` は `required`（現行の許可リストが既に通しており、外形監視が当てる先である）、`/livez`、`/readyz`、`/startupz` は `optional`。当たらない経路は、優先度クラスが `infrastructure` または `unclassified` なら `forbidden`、それ以外なら `required` とする。

この既定の向きが、Risk Notes の求める安全側である。名指ししていない `/metrics` は `infrastructure` なので `forbidden` になり、将来増える運用経路も既定で `forbidden` になる。一方、将来増える製品経路は既定で `required` になり、許可リストへ足すまで検査が落ちる。それは「利用者に届かない機能がある」という、この work item が塞ごうとしている失敗そのものである。

### 照合の単位

接頭辞の集合どうしを引き算しない。設定が表す意味を復元し、**組み立て済みの経路パターンごとに具体的なパス 1 本を作って、そのパスを各許可リストの照合器に与える。** 問いは「この設定はこのパスを中継するか」の 1 つだけになり、設定が接頭辞で書かれていても正規表現で書かれていても同じ問いで答えられる。

| 許可リスト | 意味 | 復元のしかた |
| --- | --- | --- |
| `Caddyfile` の `@backend` | `path` マッチャー。`*` はワイルドカード、それ以外は完全一致 | 継続行をまとめてトークン列を読み、glob として照合する |
| `Caddyfile` の `@realmBackend` | `path_regexp` | 正規表現をそのまま取り出して照合する |
| `vite.config.ts` の `server.proxy` | キーが `^` で始まれば正規表現、そうでなければ前方一致 | `proxy` ブロック内の文字列リテラルのキーを取り出す |

3 つのうち `vite.config.ts` だけは、設定の値ではなくソースの字面から読む。字面の解釈がずれると照合は空振りし、しかも成功に見える。**抽出が 1 件も取れなかった場合は、突き合わせを成功させずに失敗させる**（EX-SYSTEM-021-04）。`frontend/src/devProxy.test.ts` は、Vite が実際に読み込む設定オブジェクトのキー集合と、同じ規則でソースから取り出したキー集合が一致することを確かめ、字面の解釈が嘘をついていないことを反対側から固定する。

### 主要な型と関数

```go
type GatewayExposure string // "required" | "optional" | "forbidden"

func ClassifyGatewayExposure(pattern string) GatewayExposure
func GatewayRouteExpectations() []GatewayRouteExpectation // {Pattern, SamplePath, Exposure}
func CheckGatewayAllowlists(caddyfile, viteConfig string) []string
```

効果は入口に置く。`CheckGatewayAllowlists` は 2 つの設定の**中身**を受け取り、ファイルを読まない。読むのは `backend/cmd/idmagic-gateway-routes` だけである。経路の列挙も乱数も時刻も使わないので、検査はこの 2 つの文字列だけで決まる。

## Plan

着手時の調査で T001 から T003 は解決した。結論は Design と下の表が持つ。残りは実装である。

**現在のずれ（組み立て済みの router の第 1 セグメントで測った全量）。** 記録時に挙げた 7 件のうち運用経路 4 件は `forbidden` と `optional` に分類され、ずれではなくなった。代わりに記録が挙げていなかった `/authorize/resume` が出た。

| 許可リスト | 欠けている経路 |
| --- | --- |
| `Caddyfile` の `@backend` | `/ssf/*`、`/session/check`、`/application-icons/*`、`/authorize/resume` |
| `Caddyfile` の `@realmBackend` | `ssf`、`session`、`application-icons` |
| `vite.config.ts` の `server.proxy`（ホスト形式） | `/ssf`、`/session/check`、`/application-icons` |
| `vite.config.ts` の `server.proxy`（realm 形式） | `ssf`、`session`、`application-icons` |

`/authorize/resume` が Caddy でだけ落ちるのは、`@backend` の `/authorize` が完全一致で、Vite のキー `'/authorize'` が前方一致だからである。同じ列挙を 2 つの言語へ書き写しても、マッチャーの意味までは写らない。

1. 分類と照合を書き、現在のずれで RED になることを確かめる。
2. 3 つの設定を直して GREEN にする。
3. ゲートウェイ越しに `/ssf/streams/{stream_id}/events` と `/session/check` が API へ届くことを、観測可能な境界で確かめる。
4. `mise` タスクにし、`check` の依存へ加える。

## Tasks

赤緑の 1 往復ごとに選び直さないよう、使うレシピをここで決めておく。Go の 1 件は `mise run test-go-test -- ./backend/shared/http/server_http <test>`、パッケージ単位は `mise run test-go-package -- ./backend/shared/http/server_http`、GREEN になったら `mise run lint-go`、この記録を触ったら `mise run check-work-items`、ゲートウェイ越しの確認は `mise run test-ui-e2e-file -- tests/e2e/gateway-routes.spec.ts`、UI 単体は `mise run test-ui-unit-file -- src/devProxy.test.ts`。

- [x] T001 [Research] 実行時経路の正を確定し、realm 形と非宣言経路の扱いを決める。→ 組み立て済みの router。Design の「正をどこから取るか」。
- [x] T002 [Design] 「通すべき/通してはならない/どちらでもよい」の分類の置き場所を決める。→ `ClassifyRoute` と同じパッケージ。Design の「運用経路の扱いと分類の置き場所」。
- [x] T003 [Research] 3 つの設定と経路表のずれを全量出す。→ 上の表。
- [x] T004 [Acceptance] `mise run check-spec` が EX-SYSTEM-021-01 から -04 を未被覆として拒否することを確かめる（REQ-SYSTEM-021）。
- [x] T005 [Domain] `ClassifyGatewayExposure` を書く。`TestClassifyGatewayExposureKeepsMetricsOffTheGateway` ほか（REQ-SYSTEM-021 / EX-SYSTEM-021-01、EX-SYSTEM-021-03）。
- [x] T006 [Domain] 許可リストの抽出と照合を書く。`TestCheckGatewayAllowlistsReportsAMissingRequiredRoute`、`TestCheckGatewayAllowlistsFailsWhenAnAllowlistCannotBeRead`（EX-SYSTEM-021-02、EX-SYSTEM-021-04）。
- [x] T007 [Acceptance] `TestGatewayAllowlistsCarryEveryRequiredRoute` がリポジトリの 3 つの設定で RED になることを確かめる（EX-SYSTEM-021-02）。
- [x] T008 [Gateway] `frontend/Caddyfile` の `@backend` と `@realmBackend` を直す。
- [x] T009 [Gateway] `frontend/vite.config.ts` の `server.proxy` を直す。
- [x] T010 [UI] `frontend/src/devProxy.test.ts` の手書き列挙をやめ、字面から取り出したキー集合と実行時の設定オブジェクトの一致を固定する。
- [x] T011 [E2E] `frontend/tests/e2e/gateway-routes.spec.ts` で、ゲートウェイ経由の `/ssf/streams/{stream_id}/events` と `/session/check` が SPA の `index.html` ではなく API の応答を返すことを確かめる（REQ-SYSTEM-021）。
- [x] T012 [Tooling] `backend/cmd/idmagic-gateway-routes` と `mise run check-gateway-routes` を作り、`check` の依存へ加える。
- [x] T013 [Docs] `docs/releases/changes/wi-465.md` を書く。
- [x] T014 [Verify] `mise run verify` と `mise run test-ui-e2e` を通す。

## Risk Notes

リスクは medium。壊れているのは Shared Signals の受信と OIDC Session Management という、どちらも外部の当事者が呼ぶ経路である。届いていないことに現在誰も気づいていないのは、失敗が 200 の HTML として返るためである。**修正すると、これまで届かなかった要求が届くようになる。** 受信側の処理が本番の負荷を受けるのは初めてになるので、`docs/design/performance/capacity.md` の Non-protocol request profile に置いた Shared Signals の見積もりが最初に試されるのはこの変更の後である。

**照合の網羅性そのものは検査できない。** 「実行時経路の正」が実際にすべての経路を含んでいることは、その正の作り方に依存する。`operations_gen.go` に現れない経路があれば、照合を通っても届かない経路が残る。T001 でこの網羅性を確かめ、確かめられない部分は Risk として残す。

**運用経路の分類を誤ると、`/metrics` を公開しうる。** `docs/domain/system/decisions.md` は `/metrics` を認証なしと定めているので、「通すべき経路」の集合へ誤って入れると認証のない指標が公開入口から読める。分類は既定を「通してはならない」にし、通す経路だけを明示する向きにする。

`reversibility` は reversible。設定の変更であり、データも公開契約も変えない。

## Verification

- `mise run check`
- `mise run verify`
- `mise run test-ui-e2e`

## Completion

- **Completed At**: 2026-09-19
- **Summary**:
  The normative diff against `main` adds one scenario, `REQ-SYSTEM-021`: the reference frontend gateway's
  route allowlists are compared against the assembled route table, and drift is reported instead of being
  served as the SPA. Four routes the allowlists did not carry now reach the API through both the host form
  and the `/realms/{tenant_id}` path form: the Shared Signals receiver
  `/ssf/streams/{stream_id}/events`, the OIDC `check_session_iframe` at `/session/check`, the application
  icon URL at `/application-icons/{application_id}/{id}`, and `/authorize/resume`, which the Caddyfile
  alone was missing because its `/authorize` entry matches exactly while the Vite key is a prefix. Whether
  a route may pass the gateway is now classified beside the route registration, defaulting to forbidden
  for infrastructure and unclassified routes, which keeps the unauthenticated metrics endpoint off the
  public entry point without anyone naming it. `mise run check-gateway-routes` joins `mise run check`.
- **Primary Use Case Evidence**:
  - id: reach-required-routes-through-gateway
    unit_red: TestGatewayAllowlistsCarryEveryRequiredRoute failed with 13 findings, naming every route the
      three allowlists did not proxy in either form, including the unpredicted `/authorize/resume`.
    e2e_red: gateway proxies the runtime routes the SPA fallback would otherwise swallow failed because
      `GET /session/check` returned the SPA `index.html` with 200 instead of the check_session_iframe body.
    unit_fault_injection: Accepting an empty `server.proxy` extraction instead of failing on it made
      TestCheckGatewayAllowlistsFailsWhenAnAllowlistCannotBeRead fail. Disabling the forbidden arm of the
      comparison made TestCheckGatewayAllowlistsRejectsAnExposedMetricsRoute fail.
    e2e_fault_injection: Restoring the pre-change `frontend/vite.config.ts` made the spec fail, because
      `/ssf/streams/{stream_id}/events` no longer reached the API and returned an empty body.
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`
  - **Requirement**: REQ-SYSTEM-021
  - **Observed Failure**: `EX-SYSTEM-021-01` through `-04` are declared, but no test names them.
  - **Detection Reason**: The declared examples separate the four outcomes the comparison must tell apart:
    a clean comparison, a required route missing, a forbidden route exposed, and an allowlist that cannot be
    read. An implementation that only subtracts prefix sets satisfies none of the last three.
- **Unit RED Evidence**:
  - **Test**: `TestClassifyGatewayExposureKeepsMetricsOffTheGateway`
  - **Requirement**: REQ-SYSTEM-021
  - **Observed Failure**: `undefined: httpadapter.GatewayExposure`.
  - **Detection Reason**: The table asserts that `/metrics` and an unclassified route both come out
    `forbidden` without being named, which is what separates a default-deny classification from one that
    returns `required` for anything it does not recognise.
- **Change-Resistance Results**:
  `mise run test-go-mutation -- backend/shared/http/server_http` produced 101 mutants: 56 killed, 31 lived,
  8 not covered, 5 not viable, 1 infrastructure error. The survivors in `gateway_allowlist.go` fall into
  three groups.

  | 生存した変異 | 読み方 |
  | --- | --- |
  | `sort.Slice` の比較 (L101) | 乖離は集合であり、報告の順序は意味を持たない。等価な変異である |
  | `strings.Index` と `len(fields)` の境界 (L230、L243、L333) | どれも「読み取れない」側の二重の防壁と重なる。L243 だけは実在する入力形（マッチャー名を省いた `path_regexp`）を指していたので、`TestCheckGatewayAllowlistsReadsAnUnnamedPathRegexp` を足して塞いだ |
  | `unescapeJSString` の添字 (L342、L343、L355) | 逆斜線 1 個で終わる文字列リテラルを与えれば殺せるが、その入力は JavaScript の字句として存在しない。閉じ引用符を脱出してしまい、キーを取り出す正規表現にも一致しない。構成上到達しない |

  変異器が表現できない故障は手で入れた。公開禁止の判定を外すと
  `TestCheckGatewayAllowlistsRejectsAnExposedMetricsRoute` が落ち、許可リストの抽出が空でも成功させると
  `TestCheckGatewayAllowlistsFailsWhenAnAllowlistCannotBeRead` が落ち、`devProxy.test.ts` の字面抽出の
  深さを 1 段ずらすとその検査自身が落ちた。
- **Verification Results**:
  - `mise run verify` - passed
  - `mise run test-ui-e2e` - passed (30 tests across 7 files)
