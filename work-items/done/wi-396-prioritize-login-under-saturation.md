---
depends_on: []
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-08-23
priority: p1
change_kind: operations
evidence_policy: risk-based-v3
documentation_impact:
  level: upgrade_note
  reason: 固定レプリカ数を HPA へ移し、飽和時に新しい 503 を返すようになるので、配備側の操作と互換性の両方が読者に要る。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-396.md }
    - { kind: upgrade_note, path: docs/releases/upgrades/wi-396.md }
initial_context:
  specification:
    - docs/contexts/system/scenarios.md#REQ-SYSTEM-001
    - docs/contexts/system/scenarios.md#REQ-SYSTEM-016
    - docs/contexts/system/scenarios.md#REQ-SYSTEM-018
    - docs/contexts/system/scenarios.md#REQ-SYSTEM-019
    - docs/capacity.md
    - docs/api-rules.md
    - docs/deployment.md
    - docs/observability.md
    - docs/contexts/system/decisions.md
    - docs/contexts/system/internals.md
  typespec: []
  source:
    - backend/shared/http/support_http/admission.go
    - backend/shared/http/support_http/metrics.go
    - backend/shared/http/server_http/priority_class.go
    - backend/shared/http/server_http/priority_class_reference.go
    - backend/cmd/idmagic-route-reference/main.go
    - backend/shared/http/server_http/routes.go
    - backend/cmd/internal/bootstrap/apiconfig.go
    - backend/cmd/internal/bootstrap/configreference.go
    - backend/cmd/idmagic/server.go
    - backend/shared/observability/metrics_prometheus/metrics.go
    - infra/k8s/base/hpa.yaml
  tests:
    - backend/shared/http/support_http/admission_test.go
    - backend/shared/http/server_http/priority_class_test.go
    - backend/shared/http/server_http/admission_saturation_test.go
    - backend/cmd/idmagic/admission_e2e_test.go
    - backend/cmd/internal/bootstrap/apiconfig_test.go
  stop_before_reading:
    - frontend
    - backend/oauth2
    - backend/authentication
    - spec/generated
affected_spec:
  - { path: docs/contexts/system/scenarios.md, requirement: REQ-SYSTEM-001 }
  - { path: docs/contexts/system/scenarios.md, requirement: REQ-SYSTEM-016 }
  - { path: docs/contexts/system/scenarios.md, requirement: REQ-SYSTEM-018 }
  - { path: docs/contexts/system/scenarios.md, requirement: REQ-SYSTEM-019 }
---

# 容量が足りないときにログインが管理系トラフィックより先に生き残るようにする

## Motivation

IdP としての idmagic は、止まると依存する全システムのログインが止まる。`docs/capacity.md` のサービス目標も、対話的な認証の母集団に対して定義されている。

**その優先度を実現する機構が、現在ひとつも無い。**

`docs/capacity.md` の Degradation order は、容量を超過したときの縮退順序を規範として既に定めている。

> 1. `bulk` レーンの新規取得と外部スケジューラーの保守バッチを遅延させる。
> 2. `default` レーンの新規取得を遅延させ、`latency_sensitive` レーンの専用枠を維持する。
> 3. 管理用の集計、エクスポート、再同期など対話的な認証に不要な高コスト処理を明示的な 429 または 503 で拒否する。
> 4. 新しい動的クライアント登録など、既存セッションの認証とトークン処理に不要な書き込みを明示的に拒否する。
> 5. `/authorize`、`/token`、`/introspect`、ログインを受け付けられない場合は、状態を部分的に更新せず 429 または 503 で拒否する。

1 と 2 はワーカーの実行レーン（[[wi-261-job-execution-lanes]]）として実装済みである。**3 から 5 に対応する実装は見当たらない。** `backend/shared/ratelimit` はエンドポイント別の固定時間枠カウンターであって負荷に連動せず、同時実行数の上限も、負荷に応じた要求の切り捨ても無い。`postgres.NewResilientDB` のサーキットブレーカーは依存先障害時の遮断であって、優先度ではない。

つまり API プロセスは、飽和したときに**先着順で**倒れる。管理者が 10 万件のエクスポートを叩いた瞬間に、ログインがその後ろに並ぶ。

**ただしその例そのものは、本 work item が扱う問題ではない。** [[wi-459-api-process-plane-separation-decision]] の棚卸しで、API プロセスに残る上限のない同期処理は `GET /api/admin/v1/audit_events/export` の 1 経路だけだと分かった。これは入場制御が扱う「総容量が足りないときに何を捨てるか」ではなく、1 リクエストが際限なく資源を取るという別の欠陥で、[[wi-464-bound-the-synchronous-audit-event-export]] が固定の上限として塞ぐ。**到達率で見れば管理系は認証系の数%にすぎないので、この形の枯渇は本 work item の入場制御からは「飽和」に見えない。** 入場制御が要るのは、個々の要求に上限を置いてもなお総需要が容量を超える状況である。

容量そのものも自動では増えない。**`infra/` に HorizontalPodAutoscaler が 1 つも無く**、本番は `infra/k8s/overlays/prod/api.yaml` の `replicas: 3` 固定である。

この 2 つは別の問題である。前者は「足りないときに誰が勝つか」、後者は「足りなくならないようにするか」であり、後者だけを解いても前者は残る。

## Scope

- API プロセスの水平スケールを自動化する（HPA）。
- `docs/capacity.md` の Degradation order 3 から 5 を実装する。要求を優先度クラスに分類し、飽和時に低優先度から明示的に拒否する。
- 優先度クラスごとの PostgreSQL 接続予算。
- 縮退が発動したことを観測できるメトリクスと、`docs/capacity.md` のサービス目標に対する影響の測定。
- 縮退の閾値を運用者が調整できる起動時設定（REQ-SYSTEM-016 に従って検証する）。
- 縮退の振る舞いを規範的シナリオとして `docs/contexts/system/scenarios.md` に追加する。

## Out of Scope

- **API プロセスを認証プレーンと管理プレーンに分けてデプロイすること。** 本 work item では採らない。判断そのものは [[wi-459-api-process-plane-separation-decision]] が持ち、結論と再検討の条件は `docs/contexts/system/decisions.md` にある。
- ソースツリーとバイナリの分割。`backend/` の Context 構成も `backend/cmd/` の構成も変えない。
- マルチ AZ、自動フェイルオーバー、障害種別ごとの遷移。[[wi-165-high-availability-and-failover-resilience-topology]] が持つ。
- データ層の分割、読み取りレプリカ、接続プール製品の選定。[[wi-164-data-tier-scalability-partitioning-read-replica-pooling]] が持つ。
- エンドポイント別のレート制限（`backend/shared/ratelimit`）の変更。目的が異なる。Design を参照。
- **個々の要求に対する負荷非連動の固定上限。** 監査イベントのエクスポートの同時実行数と応答量の上限は [[wi-464-bound-the-synchronous-audit-event-export]] が持つ。入場制御は総容量が足りないときに働く機構で、平常時に 1 リクエストが取る資源は縛らない。両者は補完である。
- **種別（認証・プロトコル系、ポータル系、管理系、SCIM、Shared Signals）ごとの到達率と処理時間の観測。** [[wi-466-observe-request-families-against-capacity-assumptions]] が持つ。本 work item は縮退が発動したことを観測できるようにするが、種別ごとの容量の内訳は扱わない。ただし両者は経路の分類を必要とする点で重なるので、Design の「分類の置き場所」を参照。
- ステージングでの負荷試験基盤。[[wi-282-staging-load-testing-and-capacity-validation]] が持つ。本 work item はその基盤を使う側である。

## Design

### 解くべきは容量ではなく優先度である

API はステートレスなので、水平にスケールできる。だから第一の答えは「レプリカを増やす」であり、それは HPA で自動化できる。それでも次の 2 点は残る。

- **HPA は反応的で、隔離は即時である。** 判定間隔と Pod 起動で数十秒かかる。その窓の間、ログインは管理系トラフィックと同じ待ち行列に並ぶ。
- **束縛条件が PostgreSQL 側にあるとき、API を増やすと事態が悪化する。** Pod が増えれば接続プールの総数も増え、DB の競合は激しくなる。管理系のバーストに対して API をスケールアウトすることが、ログインにとって逆効果になりうる。

したがって HPA は入れるが、それだけでは目的を満たさない。**足りないときに何を先に捨てるかを、プロセス自身が決められる必要がある。** それが Degradation order が既に定めていることである。

### 優先度クラス

要求を分類し、飽和時に低い側から拒否する。分類は `docs/capacity.md` の Degradation order のステージをそのまま写す。

| クラス | ステージ | 対象 |
| --- | --- | --- |
| `interactive_auth` | ステージ 5 | OAuth2/OIDC のプロトコル経路、Discovery、JWKS、`/api/auth/**`、SAML と WS-Federation の SSO、テナントブランディング資材、ポータルの段階的認証 |
| `management` | ステージ 4 | 管理 API、ポータル API、SCIM、Shared Signals の受信、動的クライアント登録 |
| `management_bulk` | ステージ 3 | 監査イベントの一覧・検索・エクスポート、利用者と Group の入出力、動的 Group の事前評価、ライフサイクルの試験実行、プロビジョニングの全同期、アカウントのデータエクスポート |
| `infrastructure` | — | 生存確認、受付可否、起動完了、メトリクス。入場制御の対象外 |

**中間のクラス名は work item 記載の `management_write` ではなく `management` とした。** 分類は経路の全量を覆わなければならないのに、管理 API とポータル API の読み取りは「書き込み」でも「一括処理」でもなく、どちらの名前にも収まらない。`docs/capacity.md` のステージ 4 が例として挙げるのは書き込みだが、ステージが定めているのは「既存セッションの認証とトークン処理に不要」という性質であって、その性質は管理系の読み取りにも等しく当てはまる。読み取りを含むクラスに `management_write` という名前を付けることは、Risk Notes が最も重い失敗として挙げた「分類の名前と中身の食い違い」そのものである。

**SCIM は一覧も含めて丸ごと `management` に置く。** 容量計画上いちばん大きな一括処理は SCIM の全同期（1 テナント 20〜10,000 リクエスト、同時 50 テナント）なので、当初はその読み取り源である `GET /scim/v2/Users` と `GET /scim/v2/Groups` だけを `management_bulk` に置いた。これは誤りだった。差分同期が変更を適用する前に必ず打つ 1 件解決（`GET /Users?filter=userName eq "..."`）が、全同期の列挙とまったく同じ経路だからである。分けると、ステージ 3 では解決が拒否されて `PATCH` に到達できず、差分同期は 1 歩も進まないまま再送を繰り返す。**「ステージ 4 まで生き残る」という分類の主張と、実際の振る舞いが食い違う。** 用途で分けるには要求の中身（`filter` の有無）を見るしかなく、それは経路の分類とは別の機構になる。分けられないものを分けたことにするより、SCIM は動くか止まるかのどちらかである、という単純な形を選ぶ。代償として、全同期は管理 API の書き込みと同じ枠を争う。

### 分類は参照できなければ無いのと同じである

分類をコードの 1 箇所に置いただけでは、「この経路はどのクラスか」に答えられるのは前方一致の優先順位を手で解ける人だけになる。実際、SCIM の誤りは、その解決を手でやって初めて見つかった。散文に一覧を写す解は採らない。経路を足したときに写しだけが古くなり、その食い違いは負荷が高いときにしか現れないからである。

`CONFIGURATION.md` が `Config` の定義から生成され `check-config-reference` が乖離を落とす形（REQ-SYSTEM-017）を、そのまま分類にも当てる。組み立て済みの router と `ClassifyRoute` から `ROUTE_PRIORITY.md` を生成し、`check-route-reference` を `mise run check` に入れる。規範は REQ-SYSTEM-019 が持つ。**これにより、`decisions.md` と `deployment.md` からクラスの所属を述べる散文を消せる。** 両文書に残すのは境界の引き方の理由だけである。

`infrastructure` を分類の外ではなく 4 つ目のクラスとして持つのは、経路の全量を覆う検査を維持するためである。受付可否のプローブを拒否すると、飽和した全レプリカが同時に負荷分散の対象から外れて完全な停止になる。したがってこの経路は拒否しないが、それは「分類し忘れた」のではなく「分類した結果として拒否しない」でなければならない。

### レート制限との違い

`backend/shared/ratelimit` は、`(tenant_id, policy_id, key_hash)` ごとの固定時間枠カウンターで、**負荷とは無関係に**閾値を超えた要求を拒否する（REQ-OAUTH2-040）。目的は濫用の抑止である。

本 work item が足すのは**負荷連動の入場制御**で、目的は飽和時の優先度である。平常時は何も拒否しない。両者は代替しない。実装でも設定でも混同させないため、別の機構として置き、メトリクスも別に出す。

### 分類が間違っていたときに何が起きるか

`interactive_auth` に入れるべきものを `management_bulk` に入れると、**負荷が高いときだけログインの一部が 503 になる**。平常時のテストでは出ない。したがって分類は設定ではなくコードに置き、ルート登録と同じ場所で宣言し、**分類の無いルートが存在しないことを検査するテスト**を分類の導入と同時に入れる。これが本 work item の実装コストの中心である。

### 分類の置き場所と、種別との関係

[[wi-466-observe-request-families-against-capacity-assumptions]] も経路の分類を要する。ただし分類の軸が違う。**優先度クラスは「飽和時に何を先に捨てるか」、種別は「用途は何か」である。** 管理系の中でも参照は残して集計を捨てるという分け方はありうるので、2 つの分類が 1 対 1 に対応するとは限らない。

一方で、両者が独立に経路の全量を列挙し、独立に網羅性を検査するのは重複である。**先に着手したほうが列挙と網羅性検査の仕組みを持ち、後から来るほうがそれを使う。** 本 work item が先なら、分類はルート登録の側に置き、種別はその上に別の軸として載せられる形にする。逆なら、[[wi-466-observe-request-families-against-capacity-assumptions]] の網羅性検査に優先度クラスの列を足す。どちらであっても、経路の全量を数える場所は 1 つに保つ。

### プレーン分割を採らなかった理由

当初は API を認証プレーンと管理プレーンに分け、同一イメージのまま Deployment を 2 つにする案を検討した。ワーカーが `JOB_WORKER_LANES` で実行レーンごとに分かれている前例もある。本 work item では採らない。判断そのものは [[wi-459-api-process-plane-separation-decision]] が持ち、結論は `docs/contexts/system/decisions.md` の `No API plane separation` にある。採らない理由は次のとおりである。

- **PostgreSQL の競合は分割しても残る。** 共有ストアが束縛条件である限り、API 側を分けても DB 側は分かれない。接続予算をクラス別に分けるほうが直接的である。
- **常時費用が先に出る。** 追加するプレーンは自分の可用性下限レプリカ数、PodDisruptionBudget、容量モデル、監視、障害対応の手順を 1 組ずつ持つ。それを必要とする容量も障害波及も、まだ実測で示されていない。
- **今は存在しない障害モードが生まれる。** 管理プレーンの過小サイズという新しい失敗の形が増える。
- **定義が 2 箇所に分かれる。** ルート分類はコード側（登録）とゲートウェイ側（パス振り分け）の両方に必要で、片方だけ更新されると、正しく起動したプロセスに届かないトラフィックが生まれる。乖離を検出する手段が無い。

**プレーン分割と入場制御は代替ではない。** 分割は総容量が足りている状況で種別ごとに資源を固定的に分け、入場制御は総容量が足りない状況で何を拒否するかを決める。効く状況が違うので、一方が他方の代わりになるとは書けない。分割にしか得られないのはプロセス資源と停止の隔離、種別ごとに違うレプリカ数、片方だけの更新であり、入場制御にしか得られないのは飽和時の選択的な拒否である。

**将来採るとしたらの条件**は [[wi-459-api-process-plane-separation-decision]] と `decisions.md` が持つ。本 work item の負荷試験でその条件が成立した場合は、入場制御だけを追加して完了扱いにせず、プレーン分割を再評価する。**分類をコードに置くのは、その移行を安くしておく意味もある。**

### 却下した案

- **HPA だけを入れる。** 反応時間の窓が残り、DB が束縛条件のときは逆効果になりうる。
- **エンドポイント別のレート制限の閾値を下げる。** 濫用には効くが正当な一括操作には効かない。抑えたいのは正当な管理操作の影響であって、拒否したいわけではない。
- **管理 API を別のバイナリにする。** ログイン経路は IdM 側の読み取りに依存しており、リンクされるコードはほとんど同じである。`docs/structure.md` が定める単一の `Config` も 2 系統になり、REQ-SYSTEM-016 が避けている「あるプロセスだけ検証されていない値を持つ」状態を作る。この案は [[wi-459-api-process-plane-separation-decision]] が D4 として改めて評価し、独立したデータ所有権、担当チーム、SLO が現れるまで採らないと結論している。

## Plan

- **測定を先に置く。** 管理系バースト下でログインのレイテンシーがどれだけ劣化するかを [[wi-282-staging-load-testing-and-capacity-validation]] の基盤で実測する。劣化が観測できなければ、この work item は根拠を持たない。その場合は HPA だけ入れて残りを取り下げる。**バーストの大きさは想像で決めない。** `docs/capacity.md` の Non-protocol request profile が管理コンソール、管理 API 自動化、ポータル、SCIM、Shared Signals の通常時・最繁時・集中実行時を持つようになったので、負荷構成はそこから組む。同書が言うとおりこれらはすべて Planning assumption で幅が 2 桁あるため、中央値だけでなく上限側でも走らせる。
- HPA は入場制御より先に入れる。安く、単独で効果があり、入場制御の測定条件を現実に近づける。
- 入場制御は、分類とその網羅性テストを同時に入れる。分類だけ先に入れると、抜けたルートが平常時のテストで見つからない。
- 規範的シナリオは実装より先に書く。「飽和時に `management_bulk` が 503 を返し、`interactive_auth` は受け付けられる」は観測可能な振る舞いである。

### 測定について実際にできたこと

**ステージングでの測定（T001 と T010）は行っていない。** [[wi-282-staging-load-testing-and-capacity-validation]] が持つ負荷試験基盤がまだ存在せず、参照運用プロファイルのデータを投入した環境も無い。`docs/capacity.md` の Evidence classes が言う Measurement は、日付、ソース版、実行環境、データ分布、負荷構成、試験時間、結果の保存先を伴うものであり、それを名乗れる材料が本 work item の作業環境には無い。**無いものを Measurement と呼ばないことが、この分類を持っている理由である。**

代わりに、組み立て済みの経路とミドルウェア列をそのまま使い、プロセス内で先着順の待ち行列が起きることと、入場制御がそれを解くことを観測する試験を `backend/shared/http/server_http/admission_saturation_test.go` に置いた。これは待ち行列の性質の実測であって、`SLO-LOGIN-LATENCY` に対する製品の実測ではない。前者は「機構が意図どおり働くか」に答え、後者は「その機構が要るか」に答える。

では取り下げの根拠はどうなるか。本 work item の Motivation が示したのは、**優先度を実現する機構がひとつも無い**という構造の事実であり、これは測定を要しない。測定が答えるのは「どれだけ深刻か」であって「起きるか」ではない。したがって実装は進め、`SLO-LOGIN-LATENCY` に対する実測は wi-282 の基盤が立った時点の作業として残す。**閾値の既定値が現実に合っているかは、その測定でしか分からない。** 既定値を Planning assumption として扱い、`docs/deployment.md` にそう書く。

### 着手時に決めた点

**1. 飽和の判定基準は、そのプロセスで実行中の要求数とする。** 要求 1 件は高々 1 本の DB 接続を同時に握るので、実行中の要求数は接続プールの待ち行列そのものの長さである。この量はプロセス内で数え切れ、レプリカ数を知らなくても意味を持つ。レプリカが増えれば 1 プロセス当たりの実行中数が下がり、どのプロセスも縮退しなくなる。これは判定がずれたのではなく、正しい結果である。CPU 使用率を採らないのは、それが同じ性質を持たないからではなく、GC と待ち時間を含む要求の混合では実行中数のほうが接続の競合に直結し、閾値を容量計画の数（`DB_MAX_CONNS`）と突き合わせて説明できるためである。

**2. Discovery と JWKS は分類の対象とし、`interactive_auth` に置く。** キャッシュ可能であることは拒否してよい理由にならない。`docs/capacity.md` はヒット率 90% を保証値ではないと明記し、コールドキャッシュでの容量検証を求めている。JWKS を返せなければ依存先はトークンを検証できないので、これはログイン経路そのものである。処理費用も小さく、`interactive_auth` に入れても他を圧迫しない。

**3. 拒否はステージによらず 503 とし、`Retry-After` を付ける。** 429 は `backend/shared/ratelimit` が濫用の抑止として返すコードであり、同じコードを負荷連動の入場制御にも使うと、呼び出し側からもメトリクスからも 2 つの機構を区別できなくなる。Design の「レート制限との違い」が両者を混同させないと決めた以上、状態コードでも分ける。本文は `docs/api-rules.md` の既定形式である Problem Details（`urn:idmagic:error:service_overloaded`）とする。5xx を選ぶことは、ステージ 5 の拒否が `SLO-PRIMARY-ERRORS` の失敗として数えられることを意味するが、それは正しい。認証を受け付けられなかったことは失敗である。数え方を変えて目標を守るのは、`docs/capacity.md` が禁じる「Specification target の暗黙の引き下げ」にあたる。`Retry-After` はクラスごとに変え、`management_bulk` を 5 秒、`management` を 2 秒、`interactive_auth` を 1 秒とする。最初に捨てたものが最初に戻ってくると、縮退が解けない。

**4. 接続予算はプールを分けず、クラス別の入場上限で 1 つのプールを配分する。** 実行中の要求数に上限を置くことは、そのクラスが同時に握れる接続数に上限を置くことと同じである。プールを分けると `docs/capacity.md` の論理接続予算がクラス数だけ増え、しかも空いているクラスの枠を他へ融通できない。1 つのプールを入場側で配分すれば、増分はゼロで、余っている枠は上位のクラスがそのまま使える。

### 入場制御の形

主要な型と操作は次のとおりである。判定は純粋な計算で、実行中数の増減だけが作用である。

```go
type PriorityClass string // interactive_auth / management / management_bulk / infrastructure / unclassified

type AdmissionBudget struct {
    Enabled             bool
    MaxConcurrent       int // ステージ 5。プロセス全体の同時実行上限
    ManagementLimit     int // ステージ 4
    ManagementBulkLimit int // ステージ 3
}

func (b AdmissionBudget) Limit(class PriorityClass) (limit int, controlled bool)
func (b AdmissionBudget) Admit(class PriorityClass, inFlight int) bool
func (c PriorityClass) RetryAfterSeconds() int
```

作用の境界は 3 つある。実行中数はミドルウェアが持つ `atomic.Int64` で、要求の入口で 1 増やし、拒否でも完了でも `defer` で 1 減らす。閾値は起動時設定から入り、実行中は変わらない。判定の記録はメトリクスのポートへ出る。`Admit` は時刻も乱数も永続化も見ない。

拒否はルーティングの後、ハンドラーの前で起きる。したがってステージ 5 の「状態を部分的に更新しない」は、ハンドラーが 1 行も走らないという最も強い形で成り立ち、そのことを検査できる。

分類は `server_http` が持ち、ミドルウェアは `func(route string) PriorityClass` として受け取る。分類に無い経路は `unclassified` として `interactive_auth` と同じ上限で扱う——本番で誤って拒否するより、拒否しないほうが安全側である——うえで、メトリクスには `unclassified` として現れる。分類漏れそのものは網羅性検査が起動前に落とす。

### 上限の既定値

| 設定 | 既定値 | 根拠 |
| --- | ---: | --- |
| `ADMISSION_CONTROL_ENABLED` | `true` | 既定で働かなければ、飽和した本番でも何も守らない |
| `ADMISSION_MAX_CONCURRENT_REQUESTS` | 256 | `DB_MAX_CONNS` の既定 20 の 12 倍以上であり、ここに達したときには接続待ちの行列がすでに深い |
| `ADMISSION_MANAGEMENT_MAX_CONCURRENT_REQUESTS` | 192 | 全体の 75% |
| `ADMISSION_MANAGEMENT_BULK_MAX_CONCURRENT_REQUESTS` | 128 | 全体の 50% |

`ADMISSION_CONTROL_ENABLED` を持つのは、閾値が誤っていたときに運用者が機構を切れるようにするためである。切れる手段が無ければ、障害の最中に機構を外す変更を配備することになる。

`bulk ≤ management ≤ max` を満たさない組み合わせは、REQ-SYSTEM-016 に従って集約エラーで起動を止める。順序が逆の設定は、縮退の順序が逆になるという最も重い誤りであり、既定値へ黙って戻してはならない。

## Tasks

- [x] T001 [Measure] 管理系バースト下でログインのレイテンシーがサービス目標をどれだけ侵すかを実測する。侵さないなら T003 以降を取り下げる。→ ステージング基盤が無いため製品の実測は不可。Plan の「測定について実際にできたこと」に、代わりに何を観測し、取り下げの判断をどう置いたかを記録した。待ち行列の観測は `TestAdmissionControlPreservesInteractiveAuthUnderBulkSaturation`（`backend/shared/http/server_http/admission_saturation_test.go`）。
- [x] T002 [Ops] API に HPA を入れる。`replicas` 固定をやめ、最小値、最大値、判定指標を `docs/capacity.md` の Sizing rules と整合させる。
- [x] T003 [Design] 飽和の判定基準、優先度クラスの境界、拒否の状態コード、接続予算の分け方を確定し `## Design` に記録する。
- [x] T004 [Spec] 縮退の振る舞いを `docs/contexts/system/scenarios.md` に規範的シナリオとして追加する。→ REQ-SYSTEM-018。`docs/api-rules.md` の Declared status codes に、ミドルウェアが返す 3 つ目の例外として 503 を記録した。
- [x] T005 [App] 優先度クラスの分類をルート登録と同じ場所で宣言し、**分類の無いルートが存在しないことを検査するテスト**を同時に入れる。ルートを 1 つ分類から外すと落ちることを確認する。→ `backend/shared/http/server_http/priority_class.go`、`TestEveryAssembledRouteDeclaresAPriorityClass`（REQ-SYSTEM-018）。
- [x] T012 [Ops] 分類を運用者が参照できる生成物にする。組み立て済みの router と `ClassifyRoute` から `ROUTE_PRIORITY.md` を生成し、`mise run check-route-reference` を `mise run check` に入れる（REQ-SYSTEM-019）。`decisions.md` と `deployment.md` からクラスの所属を述べる散文を消す。
- [x] T006 [App] 負荷連動の入場制御を実装する。Degradation order のステージ 3、4、5 に対応させる。→ `backend/shared/http/support_http/admission.go`、`TestAdmissionBudgetAdmit`、`TestAdmissionMiddlewareShedsLowerPriorityFirst`（REQ-SYSTEM-018）。
- [x] T007 [App] PostgreSQL 接続予算をクラス別に分け、`management_bulk` が `interactive_auth` の接続枠を奪えないようにする。→ プールは分けず、クラス別の入場上限で 1 つのプールを配分する（Design 4）。`TestAdmissionBudgetReservesHeadroomForInteractiveAuth`。
- [x] T008 [Config] 縮退の閾値を起動時設定に足し、REQ-SYSTEM-016 に従って検証する。不正値は集約エラーで起動を止める。→ `TestLoadAPIConfigRejectsInvertedAdmissionThresholds`（REQ-SYSTEM-016）。
- [x] T009 [Observability] 縮退の発動をクラス別のメトリクスとして出し、ダッシュボードとアラートに載せる。→ `http_admission_decisions_total`、`http_admission_in_flight_requests`、`ApiAdmissionSheddingInteractiveAuth` / `ApiAdmissionShedding`。
- [x] T010 [Measure] T001 と同じ条件で再測定し、ログインのサービス目標が保たれることを確認する。保たれなければ Design のプレーン分割の条件に照らして判断する。→ T001 と同じ理由でステージングの実測は不可。プロセス内の再測定は同じ試験の後半が持つ。
- [x] T011 [Verify] `mise run verify` と `mise run check-k8s` を通す。

## Verification

宣言する RED は次の 2 つである。本 work item は `change_kind: operations` なので、`WORK_ITEM_FORMAT.md` の Acceptance RED / Unit RED の形で記録する。

- **Acceptance RED**：`TestAdmissionMiddlewareShedsLowerPriorityFirst`（`backend/cmd/idmagic/admission_e2e_test.go`、`mise run test-go-race`）。REQ-SYSTEM-018。組み立て済みの router と起動時設定から入る入場制御を通し、飽和時に `management_bulk` が 503 で拒否され、同時に `interactive_auth` が受け付けられ、拒否された要求のハンドラーが 1 度も走らないことを観測する。
- **Unit RED**：`TestAdmissionBudgetAdmit`（`backend/shared/http/support_http/admission_test.go`、`mise run test-go-race`）。REQ-SYSTEM-018。クラスと実行中数から入場可否を決める純粋な計算を、ステージ 3 / ステージ 4 / ステージ 5 の境界の両側で検査する。

検査の一覧は次のとおりである。

- `mise run check-k8s`
  - reason: HPA を含むマニフェストが妥当であること。
- `mise run check-config-reference`
  - reason: 追加した閾値の設定が ConfigurationReference に反映され、REQ-SYSTEM-017 の乖離検査を通ること。
- `mise run check-route-reference`
  - reason: 経路の分類と `ROUTE_PRIORITY.md` が乖離していないこと（REQ-SYSTEM-019）。
- `mise run test-go`
- `mise run k6-smoke`
  - reason: 入場制御を入れた後も平常時の OAuth サービス目標を満たすこと。**平常時に 1 件も拒否が出ないこと**を含む。
- `mise run verify`
- 手動: ルートを 1 つ優先度クラスから外し、T005 のテストが落ちることを確認する。落ちなければ、分類の抜けは本番でしか見つからない。
- 手動: 飽和を人工的に起こし、`management_bulk` が拒否され `interactive_auth` が受け付けられることを確認する。逆になっていないことの確認である。
- 手動: 飽和時の `interactive_auth` の拒否で、状態が部分的に更新されていないことを確認する（Degradation order ステージ 5）。
- 手動: 不正な閾値設定を与え、プロセスが副作用のある初期化を始めずに集約エラーで停止することを確認する（REQ-SYSTEM-016）。

## Risk Notes

リスクは medium。

**最も重い失敗は、優先度クラスの分類が間違っていることである。** `interactive_auth` に入れるべきルートが `management_bulk` に入っていると、負荷が高いときだけログインの一部が 503 になる。平常時のテストでは絶対に出ない。T005 で分類の網羅性を機械検査し、外したら落ちることを手で確認する手順を Verification に入れたのはこのためである。

次に、**入場制御が平常時に発動してしまうこと**。閾値が低すぎると、負荷でも何でもない状況で管理操作が拒否され、運用者は機構を無効化して先へ進む。そうなれば飽和時にも何も守られない。`mise run k6-smoke` で平常時に拒否がゼロであることを確認条件に入れた。

飽和の判定基準をレプリカ数に依存する量（たとえばプロセス単体の CPU 使用率）に置くと、HPA でレプリカが増えたときに判定がずれる。T003 の未解決点 1 はこの理由による。

**この変更は高可用性ではない。** PostgreSQL は依然として単一の正であり、共通の障害域は解消しない。優先度を付けたことで可用性が確保されたと読まれると、[[wi-165-high-availability-and-failover-resilience-topology]] が扱う本来の課題が過小評価される。`docs/deployment.md` にはこの限界を明記する。

T001 の実測で劣化が観測できなかった場合に、それでも「入れておいたほうがよい」として T003 以降へ進むと、根拠の無い複雑さを恒久的に抱えることになる。**測定を取り下げの根拠として使えるようにしておく**ことが、この計画の要である。

## Completion
- **Completed At**: 2026-09-05
- **Summary**:
  `mise run spec-diff` が報告した規範的な差分は `added scenarios: REQ-SYSTEM-018, REQ-SYSTEM-019` の 2 件である。REQ-SYSTEM-019 は、経路の優先度クラスを実装を読まずに参照できることを、`CONFIGURATION.md` と同じ生成＋乖離検出の形で定める。飽和した API プロセスが優先度の低い要求から拒否すること、拒否がハンドラーへ到達しないこと、`infrastructure` の経路は拒否しないこと、飽和していないときは何も拒否しないことを、新しい振る舞いとして加えた。既存のシナリオは 1 つも変えていない。
  契約側の差分は無い。飽和時の 503 は routing の後、どのハンドラーの手前でも同じに返り、呼び出し側の対応も `Retry-After` の秒数だけ待って再送する 1 つしかないので、`docs/api-rules.md` の Declared status codes に、ミドルウェアが返す 3 つ目の例外として記録した。TypeSpec の operation は 1 つも変えていない。
  設定は 4 つ増えた（`ADMISSION_CONTROL_ENABLED`、`ADMISSION_MAX_CONCURRENT_REQUESTS`、`ADMISSION_MANAGEMENT_MAX_CONCURRENT_REQUESTS`、`ADMISSION_MANAGEMENT_BULK_MAX_CONCURRENT_REQUESTS`）。メトリクスは 2 つ増えた（`http_admission_decisions_total`、`http_admission_in_flight_requests`）。配備側では API の HorizontalPodAutoscaler が加わり、オーバーレイの固定 `replicas` が消えた。
- **Acceptance RED Evidence**:
  - **Test**: `TestAdmissionMiddlewareShedsLowerPriorityFirst`（`backend/cmd/idmagic/admission_e2e_test.go`）
  - **Requirement**: REQ-SYSTEM-018
  - **Observed Failure**: `Register` から入場制御ミドルウェアを外した状態で `saturated /realms/default/api/admin/v1/audit_events status = 401, want 503`。飽和していても管理 API の要求が guard まで届いてしまう。
  - **Detection Reason**: 飽和していないときの同じ要求は guard の 401 になる。飽和時に 503 へ変わることは、要求が guard にもハンドラーにも到達しなかったことを意味する。状態コードだけを見る検査は、拒否を書いてから操作を実行する実装にも通ってしまうが、401 から 503 への変化は「どこまで進んだか」を区別する。あわせて、飽和が去った後に同じ要求が 401 へ戻ることも確かめるので、実行中数を減らし忘れる実装も落ちる。
- **Unit RED Evidence**:
  - **Test**: `TestAdmissionBudgetAdmit`、`TestAdmissionBudgetReservesHeadroomForInteractiveAuth`（`backend/shared/http/support_http/admission_test.go`）
  - **Requirement**: REQ-SYSTEM-018
  - **Observed Failure**: 拒否しない実装（`Admit` が常に true）に対して `Admit("management_bulk", 4) = true, want false`、`Admit("management", 7) = true, want false`、`Admit("interactive_auth", 11) = true, want false`、および `in-flight 4: management_bulk was admitted past its limit 3`。
  - **Detection Reason**: 各クラスの上限のちょうど上と下を並べているので、「未満」で判定する実装と「以下」で判定する実装を区別する。`ReservesHeadroom` は上限のあいだのすべての実行中数について、下位が拒否され上位が受け付けられることを確かめるので、全クラスを同じ上限にする実装（優先順位の消失）を落とす。
- **Change-Resistance Results**:
  変更した論理に 9 つの誤りを順に入れ、どれが検出されるかを記録した。
  1. **拒否しない**（`Admit` が常に true、入場制御を入れる前の振る舞い）→ 検出。unit 4 件、middleware 4 件、E2E、飽和試験がいずれも落ちた。
  2. **配線の切断**（`Register` から `AdmissionMiddleware` を外す）→ 検出。E2E が `status = 401, want 503` で落ちた。unit は落ちない（配線を見ていないため）ので、E2E がこの誤りの唯一の検出点である。
  3. **優先順位の消失**（全クラスに `MaxConcurrent` を返す）→ 検出。unit 2 件と E2E が落ちた。
  4. **実行中数を減らし忘れる**（`defer inFlight.Add(-1)` の削除）→ 検出。`RecordsDecisionsPerClass` が `interactive_auth admitted count = 0, want 1`、E2E が `after recovery ... status = 503, want 401` で落ちた。**この誤りは飽和の検出ではなく回復の検査だけが捕らえる。**
  5. **認証の誤分類**（`/api/auth` を `management_bulk` へ）→ 検出。`TestClassifyRouteKeepsAuthenticationOutOfTheSheddableClasses` が 7 経路について落ちた。これが Risk Notes の言う最も重い失敗である。
  6. **網羅性の欠落**（分類表から `/api/admin/v1` を落とす）→ 検出。`TestEveryAssembledRouteDeclaresAPriorityClass` が `368 of 651 assembled route(s) declare no priority class` で落ちた。Verification の手動手順に対応する。
  7. **閾値の順序検証の削除**（`l.Require` を外す）→ 検出。`TestLoadAPIConfigRejectsInvertedAdmissionThresholds` が `err = nil` で落ちた。
  8. **`Retry-After` をクラスによらず同じにする** → 検出。unit 2 件と E2E が落ちた。
  9. **自分の record の `documentation_impact` を `none` へ弱める** → 検出。`check-work-items` が `documentation_impact none is weaker than inferred release_note` を報告した。検査ツールの修正（下記）が、直そうとした誤検出だけを止めて本来の検出を残していることの確認である。
  10. **生成物を手で書き換える**（`ROUTE_PRIORITY.md` の 1 行からメソッドを削る）→ 検出。`check-route-reference` が out of date で落ちた。
  11. **分類を変えて生成物を再生成し忘れる**（`/application-icons` を `management_bulk` へ）→ 検出。同じ検査が落ちた。両方向を試したのは、生成物と定義のどちらが動いても乖離を捕らえることを確かめるためである（REQ-SYSTEM-019）。
  加えて、1 と 4 を当てた時点で **2 つのテストが落ちずに固まる**ことが分かった。占有側のハンドラーが無期限に待っていたためで、止まる検査は落ちない検査である。ハンドラーの待ちと並行検査の待ちの両方に期限を置き、拒否されなかったことが assertion の失敗として現れるようにしてから、上の結果を取り直した。
- **Verification Results**:
  - `mise run verify` - passed
  - `mise run check-work-items` / `mise run check-ids` - passed
  - `mise run check-config-reference` - passed（`CONFIGURATION.md` を再生成）
  - `mise run check-route-reference` - passed（`ROUTE_PRIORITY.md` を生成。249 経路が bulk 26 / management 167 / interactive_auth 51 / infrastructure 5 に分かれる）
  - `mise run check-slo-references` - passed
  - `mise run test-go-race` - passed
  - `mise run check-k8s` / `mise run check-monitoring` / `mise run k6-smoke` - **未実行**。この作業環境の Docker（colima）が `vz driver is running but host agent is not` で起動せず、これら 3 つはいずれも Docker を要する。代わりに、追加・変更した YAML をすべて `yq` で解析し、HorizontalPodAutoscaler が `autoscaling/v2` の `scaleTargetRef` / `minReplicas` / `maxReplicas` / `metrics[].resource` / `behavior` を持つこと、Prometheus のルールが 5 グループ 11 アラートとして解析できること、Grafana のダッシュボード JSON が `jq` を通ることを確かめた。**kubeconform によるスキーマ検証と promtool によるルール検証、そして平常時に拒否がゼロであることの k6 による確認は残っている。**
- **Follow-ups**:
  - ステージングの負荷試験基盤（[[wi-282-staging-load-testing-and-capacity-validation]]）が立った時点で、T001 と T010 が求める `SLO-LOGIN-LATENCY` に対する実測を行い、`ADMISSION_*` の既定値を Planning assumption から置き直す。
  - **SCIM の全同期はステージ 3 で先に捨てられない。** 全同期の列挙と差分同期の 1 件解決が同じ経路なので、経路の分類では分けられず、SCIM は丸ごと `management` にした（Design 参照）。分けるには要求の中身を見る機構が要る。それが引き合うかは、[[wi-466-observe-request-families-against-capacity-assumptions]] が種別ごとの到達率を実測してから判断する。**それまでは、全同期が管理 API の書き込みと同じ枠を争う状態が残る。**
