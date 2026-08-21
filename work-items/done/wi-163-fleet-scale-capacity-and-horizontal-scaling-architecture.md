---
depends_on: []
status: completed
authors: [tn]
risk: high
created_at: 2026-07-10
change_kind: operations
initial_context:
  specification:
    - spec/contexts/system/scenarios.md#REQ-SYSTEM-001
    - spec/README.md
    - spec/deployment.md
    - spec/observability.md
    - spec/structure.md
    - spec/contexts/audit/decisions.md
    - spec/contexts/authentication/internals.md
    - spec/contexts/jobs/internals.md
    - spec/contexts/oauth2/internals.md
  typespec:
    - IdMagic.System.Operations.MetricsExposition
    - IdMagic.OAuth2.Operations.RegisterClient
    - IdMagic.OAuth2.Operations.Authorize
    - IdMagic.OAuth2.Operations.PushAuthorizationRequest
    - IdMagic.OAuth2.Operations.Token
    - IdMagic.OAuth2.Operations.Introspect
    - IdMagic.OAuth2.Operations.Revoke
    - IdMagic.OAuth2.Operations.UserInfo
    - IdMagic.OAuth2.Operations.PostUserInfo
    - IdMagic.OAuth2.Operations.DeviceAuthorization
    - IdMagic.OAuth2.Operations.GetOpenidConfiguration
    - IdMagic.OAuth2.Operations.GetOauthAuthorizationServer
    - IdMagic.Authentication.Operations.SubmitBrowserLogin
    - IdMagic.Authentication.Operations.CompleteFederatedLogin1
    - IdMagic.Authentication.Operations.CompleteFederatedLogin2
    - IdMagic.Authentication.Operations.ListMySessions
    - IdMagic.SigningKeys.Operations.GetJwks
    - IdMagic.SigningKeys.Operations.ListTenantJwks
  stop_before_reading:
    - backend
    - frontend
    - infra
affected_spec:
  - { path: spec/contexts/system/scenarios.md, requirement: REQ-SYSTEM-001 }
---

# 1000万ユーザー、10万テナント規模の容量目標と水平スケール参照構成を定義する

## Motivation

SCL の `objectives` は `dc0961d0` で削除され、後続の `1b7b2cef` で仕様が TypeSpec と正典文書へ移行した際にも移送先が作られなかった。このため、`REQ-SYSTEM-001` は Prometheus が OAuth2 の可用性、レイテンシー、エラー率の目標を評価すると規定している一方、現在の `spec/` にはその目標値が存在しない。`spec/observability.md` は評価に使うメトリクスを定め、`spec/structure.md` は SLO をサービス分割の判断材料に挙げているが、どちらも目標値の正典ではない。

フリート全体の容量前提も未定義である。1000万ユーザー、10万テナントを扱うという目標だけでは、同時アクティブ数、オブジェクト数、保持期間、ストレージ成長、集中時間帯の要求率が決まらず、API レプリカ数、PostgreSQL 接続数、ワーカー実行枠を算出できない。計画上の仮定と実測済みの上限を区別しなければ、参照構成が目標を満たすかどうかも判断できない。

現在の共有状態ストアは Valkey ではない。[[wi-278-consolidate-ephemeral-state-into-postgresql-remove-valkey]] により、永続状態と一時状態は PostgreSQL に統合され、`spec/deployment.md` も PostgreSQL を唯一の共有状態ストアとしている。本 WI はこの現行構成を出発点として、失われた SLO の正典、フリート規模の容量前提、水平スケールの参照構成を整備する。データ層の改修は [[wi-164-data-tier-scalability-partitioning-read-replica-pooling]]、高可用性とフェイルオーバーの実装は [[wi-165-high-availability-and-failover-resilience-topology]]、ステージングでの実測は [[wi-282-staging-load-testing-and-capacity-validation]] が受け持つ。

## Scope

- `spec/capacity.md` を作成し、`spec/README.md` の正典文書索引から参照できるようにする。
- 旧 SCL に存在した主要なレイテンシー、非 5xx 比率、可用性、スループットの数値を現在の API とメトリクスに照合し、測定対象、母集団、時間窓、除外条件を備えた目標として `spec/capacity.md` に復元する。
- 10万テナント、1000万ユーザーを参照運用プロファイルとし、テナント規模分布、同時アクティブセッション、主要オブジェクト総数、保持期間、ストレージ成長率、集中時間帯のエンドポイント別要求率を定める。
- 計画上の仮定、仕様上の目標、実測結果を区別し、API レプリカ数、ワーカー実行枠、PostgreSQL 接続予算、ストレージ予算を算出する式と安全余裕を `spec/capacity.md` に定める。
- 容量超過時はセキュリティ境界と監査の完全性を弱めず、対話的な認証処理より先にバルク処理を遅延させ、受け付けられない要求は黙って破棄せず明示的に拒否するという縮退順序を `spec/capacity.md` に定める。障害種別ごとの詳細な縮退マトリクスと実装は wi-165 に委ねる。
- `spec/deployment.md` に、ゲートウェイまたは負荷分散装置、水平スケールするステートレスな API レプリカ、独立して水平スケールするワーカー、外部スケジューラーから起動するバッチ、共有状態を持つ PostgreSQL から成る製品中立の参照トポロジを記載する。
- `REQ-SYSTEM-001` を、`spec/capacity.md` に定めた SLO を `spec/observability.md` のメトリクスで評価するシナリオとして整合させる。

## Out of Scope

- PostgreSQL のパーティショニング、読み取りレプリカ、接続プール、テナント分散配置の実装。[[wi-164-data-tier-scalability-partitioning-read-replica-pooling]] が扱う。
- マルチ AZ、マルチリージョン、自動フェイルオーバー、過負荷保護、ゼロダウンタイム移行の詳細設計と実装。[[wi-165-high-availability-and-failover-resilience-topology]] が扱う。
- 負荷試験基盤の構築と、本番相当環境で参照運用プロファイルを満たすことの実証。[[wi-282-staging-load-testing-and-capacity-validation]] が扱う。
- 大規模な単一テナントの検索と集計の最適化。[[wi-161-large-tenant-performance-foundation]] が扱う。
- Valkey または別の共有状態ストアの再導入。PostgreSQL だけでは目標に届かないことが実測で判明した場合は、ボトルネックと必要な性質を根拠に別の work item で再検討する。
- 特定クラウドのマネージド製品を前提とする Terraform、Helm chart、Kubernetes マニフェストの実装。
- アプリケーションロジック、HTTP API、メトリクス実装、アラート定義の変更。

## Design

### 正典文書の分担

フリート全体の決定は Context 固有の `decisions.md` には置かない。`spec/contexts/system/decisions.md` はシステム入口の境界を所有しており、データ層とワーカーを含むフリート全体はその責務を超える。実行単位と接続関係は `spec/deployment.md`、想定規模、目標値、限界の決め方、縮退順序は `spec/capacity.md` に置く。変更時に検討した代替案と判断履歴は本 work item に残し、ADR は作成しない。

`spec/capacity.md` だけにトポロジも集約する案は採らない。容量の数値を変えずに実行単位や接続関係だけを変更する場合があり、両者を同じ文書に置くと変更理由とレビュー対象が混ざるためである。

### 復元するサービス目標

SLO の復元を別 work item へ分けない。`REQ-SYSTEM-001` がすでに目標の評価を要求しており、フリートの負荷モデルもレイテンシー、エラー率、可用性を維持できる要求率として定義する必要があるため、本 WI 内で同じ正典へ戻す。

旧 SCL の値は次の復元基準として扱う。現在のルート、メトリクス、測定可能性と一致する値は維持し、一致しない場合は対象を黙って削除せず、後継となる対象または不採用の理由を本 work item に記録する。

| 種別 | 復元基準 |
| --- | --- |
| レイテンシー | 30 日窓で `/api/auth/login` は p99 300 ms、`/authorize` は p99 500 ms、`/par` は p99 200 ms、`/token` は p99 300 ms、`/introspect` は p99 50 ms、`/revoke` と `/userinfo` は p99 100 ms、Discovery と JWKS は p99 20 ms、`/register` は p99 500 ms、`/device_authorization` は p99 300 ms、外部連携ログインのコールバックは p95 2 s、セッション一覧の先頭ページは p95 100 ms |
| 非 5xx 比率 | 30 日窓で `/api/auth/login`、`/authorize`、`/par`、`/token`、`/revoke`、`/userinfo`、`/register`、`/device_authorization` は 99.9% 以上、`/introspect`、Discovery、JWKS は 99.99% 以上。認証失敗、流量制限、入力不正などの 4xx は 5xx に数えない |
| 可用性 | 5 分の時間区分を用いる 30 日窓で OAuth2/OIDC 全体は 99.9% 以上、`/token` は 99.95% 以上 |
| 容量受入れ目標 | 規定した参照負荷で `/token` は 5,000 rps、`/authorize` は 1,000 rps、`/introspect` は 20,000 rps を処理しながら、対応するレイテンシーと非 5xx 比率を満たす |

スループットは、要求が存在しない本番時間帯を含めて「測定区間の 99% が閾値以上」と評価する SLO には戻さない。規定したデータ分布、要求構成、試験時間、安全余裕の下で満たす容量受入れ目標として定義し、wi-282 の再現可能な負荷試験で検証する。

### 旧 SCL と現行仕様の対応

| 旧 SCL objective | 現行の操作とルート | 評価メトリクス | 判断 |
| --- | --- | --- | --- |
| `LoginLatency`, `LoginErrorRate` | `SubmitBrowserLogin`、`POST /api/auth/login` | `http_request_duration_seconds`, `http_requests_total` | 維持 |
| OAuth2 の各 `*Latency`, `*ErrorRate` | `Authorize`, `PushAuthorizationRequest`, `Token`, `Introspect`, `Revoke`, `UserInfo`, `PostUserInfo`, `RegisterClient`, `DeviceAuthorization` | `http_request_duration_seconds`, `http_requests_total` | 現行 TypeSpec の操作とメソッドへ移送。`/userinfo` は GET と POST を同じ目標へ含める |
| `DiscoveryLatency`, `DiscoveryErrorRate` | `GetOpenidConfiguration`, `GetOauthAuthorizationServer` | `http_request_duration_seconds`, `http_requests_total` | OIDC Discovery に加え、後から追加された RFC 8414 の認可サーバーメタデータも同じ公開文書群として含める |
| `JwksLatency`, `JwksErrorRate` | `GetJwks`, `ListTenantJwks`、`GET /jwks`, `GET /realms/{tenant_id}/jwks` | `http_request_duration_seconds`, `http_requests_total` | 既定テナントだけでなく明示テナントの後継ルートも含める |
| `FederatedCallbackLatency` | `CompleteFederatedLogin1`, `CompleteFederatedLogin2`、OIDC と SAML の各コールバック | `http_request_duration_seconds` | TypeSpec 上の二つの操作へ移送し、上流交換を含む p95 2 秒を維持 |
| `SessionListLatency` | `ListMySessions`、`GET /api/account/v1/sessions` | 現行 HTTP RED は先頭ページと後続ページを区別できない | 不採用。旧 `/api/account/sessions` からのパス変更は確認したが、先頭ページだけの母集団を現在の低カーディナリティラベルで評価できない |
| `OverallAvailability`, `TokenEndpointAvailability` | HTTP RED 対象全体、`Token` | `http_requests_total` と Prometheus のスクレイプ状態 | 5 分の時間区分と無通信区分の扱いを明示して維持 |
| `TokenThroughput`, `AuthorizeThroughput`, `IntrospectThroughput` | `Token`, `Authorize`, `Introspect` | wi-282 の負荷生成結果と HTTP RED | 30 日の発生回数 SLO には戻さず、60 分の測定時間を持つ容量受入れ目標へ移行 |

`oauth2_token_issuance_total` と `oauth2_token_issuance_duration_seconds` は grant 別の診断には使えるが、ルート、HTTP 状態、他の OAuth2 操作を同じ母集団で評価できないため、復元した SLO の唯一の根拠にはしない。

### 容量目標と実測の区別

10万テナントと1000万ユーザーは、すべてのデプロイに課す最低構成でも、製品がそれ以上を拒否するハード上限でもない。参照運用プロファイルに対して SLO を満たす構成を算出するための設計目標である。単一テナントの quota、API ごとの入力上限、保持期間とは別に扱う。

各数値には「仕様上の目標」「計画上の仮定」「日時と環境を伴う実測」のいずれかを付ける。1 レプリカ当たりの処理能力が未測定なら仮定として明示し、必要レプリカ数は `ピーク要求率 ÷ 1 レプリカ当たりの持続処理能力 × 安全係数` から切り上げる。実測値へ置き換える責務は wi-282 に持たせ、仮定を実測済みの保証として記述しない。

### PostgreSQL を共有状態の正とする参照構成

API とワーカーはプロセス内に正となる状態を持たず、永続状態と一時状態を PostgreSQL で共有する。プロセス内キャッシュ、ゲートウェイ、CDN は Discovery や JWKS などから導出できる応答の負荷軽減に利用できるが、認可コード、セッション、再送防止、流量制限の正にはしない。キャッシュを失っても正しさを損なわず、失効や更新を TTL だけに依存させない境界を `spec/deployment.md` に記載する。

Valkey を参照構成へ戻す案は採らない。現在の実装と `spec/deployment.md` は PostgreSQL への統合を完了しており、本 WI には新しい共有ストアを必要とする実測根拠がない。PostgreSQL の論理的な単一の正を、どの物理トポロジと接続方式で支えるかは wi-164 と wi-165 が決める。

## Plan

1. 削除前の SCL から目標値、測定窓、対象操作を棚卸しし、現在の TypeSpec の操作、HTTP ルート、`spec/observability.md` のメトリクスへ対応づける。対応先がない項目は後継または不採用理由を記録する。
2. `spec/capacity.md` を作成し、サービス目標、参照運用プロファイル、オブジェクト数と保持前提、ピーク負荷、算出式、安全余裕、容量超過時の縮退順序を記載する。未実測値には仮定であることと、wi-282 で置き換える条件を付ける。
3. `spec/deployment.md` に現行の PostgreSQL 単一共有ストアを前提とする参照トポロジ、各実行単位の水平スケール境界、キャッシュの責務、wi-164 と wi-165 に委ねる物理トポロジの境界を記載する。
4. `spec/README.md` の索引と `REQ-SYSTEM-001` を新しい正典へ接続し、削除済みの SCL、ADR、Valkey を参照していないことを確認する。
5. 仕様を生成して検証し、規範的な差分が容量目標と参照構成の追加に限られることを確認する。

## Tasks

- [x] T001 [Inventory] 旧 SCL のサービス目標を現在の TypeSpec 操作、HTTP ルート、メトリクスへ対応づけ、維持、後継への移行、不採用を決める。
- [x] T002 [Spec] `spec/capacity.md` を作成し、復元した SLO と容量受入れ目標の測定対象、母集団、時間窓、除外条件を定義する。
- [x] T003 [Spec] `spec/capacity.md` に10万テナント、1000万ユーザーの参照運用プロファイル、主要オブジェクト数、保持期間、ストレージ成長、ピーク要求率、算出式、安全余裕、縮退順序を定義する。
- [x] T004 [Spec] `spec/deployment.md` に PostgreSQL を唯一の共有状態ストアとする水平スケール参照トポロジと、API、ワーカー、バッチ、キャッシュ、データ層の責務境界を記載する。
- [x] T005 [Spec] `spec/README.md` の索引と `REQ-SYSTEM-001` を `spec/capacity.md` に接続し、`spec/observability.md` のメトリクスで各 SLO を評価できることを確認する。
- [x] T006 [Render] `mise run spec-render` で TypeSpec と正典文書から派生成果物を再生成する。
- [x] T007 [Verify] `mise run check-spec`、`mise run check-work-items`、`mise run check-ids`、`mise run verify-spec`、`mise run spec-diff` を通す。

## Verification

- `mise run check-spec`
- `mise run spec-render`
- `mise run check-work-items`
- `mise run check-ids`
- `mise run verify-spec`
- `mise run spec-diff`
- 手動で、各 SLO に測定対象、母集団、時間窓、除外条件、対応するメトリクスがあり、スループットが負荷条件を伴う容量受入れ目標として読めることを確認する。
- 手動で、参照運用プロファイルのユーザー数、テナント数、オブジェクト数、保持期間、ストレージ成長、ピーク要求率からレプリカ数と接続予算を再計算できることを確認する。
- 手動で、`spec/deployment.md` と `spec/capacity.md` が削除済みの SCL、ADR、Valkey を参照せず、`spec/deployment.md` の PostgreSQL 共有状態方針と矛盾しないことを確認する。
- 手動で、wi-164 がデータ層の容量機構、wi-165 が高可用性と障害時の詳細な縮退、wi-282 が実測を所有する分界を一意に読めることを確認する。

## Risk Notes

旧 SCL の数値は測定モデルが不安定だったために `objectives` とともに削除されており、値だけを貼り戻すと同じ問題を再現する。対象母集団、時間窓、除外条件、負荷条件を各目標に付け、現在のメトリクスで評価できない目標は評価方法を定めるまで仕様上の保証にしない。

1000万ユーザー、10万テナントという設計目標と、その規模を現在の実装が処理できるという実証は別である。本 WI は前者を定義し、後者は wi-282 の実測結果で更新する。未実測の 1 レプリカ当たり処理能力や安全係数は仮定として表示し、保証値と混同させない。

PostgreSQL が最初のボトルネックになる可能性はあるが、ボトルネックの予想だけで共有状態ストアを増やすと、現在の単一依存という設計を根拠なく覆す。wi-282 の実測で不足を特定し、wi-164 または新しい work item で必要な機構を選ぶ。

容量超過時の縮退順序は wi-165 の障害時縮退と重なるため、本 WI は守る優先順位と禁止事項までを定め、障害種別ごとの遷移、冗長化方式、運用手順は wi-165 に残す。

## Completion

- **Completed At**: 2026-08-22
- **Summary**:
  - 主要な OAuth2/OIDC 操作の SLO と容量受入れ目標を、測定母集団、時間窓、除外条件を含む正典として `spec/capacity.md` に復元した。
  - 10万テナント、1000万ユーザーの参照運用プロファイル、オブジェクト数、保持期間、ストレージ成長、ピーク要求率、算定式、安全余裕、縮退順序を定義した。
  - PostgreSQL を唯一の共有状態ストアとし、API、ワーカー、バッチ、キャッシュの水平スケール境界を `spec/deployment.md` に同期した。
- **Verification Results**:
  - `mise run verify` - passed
  - `mise run spec-render` - passed
  - `mise run spec-diff` - passed; normative scenario change is limited to `REQ-SYSTEM-001`
  - 手動確認 - 各目標の測定可能性、容量算定の再現性、後続 WI との責務分界を確認
