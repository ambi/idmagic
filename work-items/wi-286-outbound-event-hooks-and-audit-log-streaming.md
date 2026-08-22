---
status: pending
authors: [tn]
risk: high
created_at: 2026-07-25
priority: p2
depends_on: []
change_kind: feature
initial_context:
  source:
    - backend/audit/usecases
    - backend/audit/handlers_http
    - backend/shared/events
    - backend/jobs/usecases
    - backend/provisioning/usecases
  tests:
    - backend/audit/usecases
    - backend/audit/handlers_http
  stop_before_reading:
    - backend/saml
    - backend/wsfederation
affected_spec:
  - { path: spec/contexts/audit/main.tsp, symbol: IdMagic.Contract.ListAdminAuditEvents }
  - { path: spec/contexts/audit/main.tsp, symbol: IdMagic.Contract.ExportAdminAuditEvents }
---

# テナント設定の outbound event hook (webhook) と監査ログストリーミングを導入する

## Motivation

現状、監査イベントを外へ出す経路は **管理 API での検索と CSV エクスポートだけ**である。
これは pull 型で人間向けの経路であり、テナントが自分のイベントだけを自分の宛先へ購読する手段は無い。

起票時にはもう 1 本、`EVENT_SINK=outbox` で Kafka へリレーする内部トランスポートがあったが、
これは [[wi-305-remove-external-message-brokers]] で撤廃された。そのとき「IdP から RP へのイベント通知は
SSF/CAEP の HTTP Webhook で、汎用の外部連携は本 work item の outbound webhook で担う」と分界が決まっており、
wi-305 は Out of Scope で明示的に本 work item を指している。つまりブローカー撤廃後、
**汎用の push 経路はどこにも存在しない**。この穴を埋めるのが本 work item である。

エンタープライズ導入では、IdP イベントを外部システムへ push できることが要件になる:

- **Okta**: Event Hooks (HTTPS webhook + HMAC 検証 + retry) と Log Streaming
  (AWS EventBridge / Splunk への継続配送) を別機能として提供する。
- **Entra ID**: diagnostic settings で監査 / サインインログを Log Analytics / Event Hub /
  Storage へ継続エクスポートする。
- **Keycloak**: Event Listener SPI で任意の外部配送を実装できる (ただし自作)。

これが無いと、(1) SIEM への取り込みが人手の CSV エクスポート運用になり、
セキュリティ運用の検知遅延が「日」単位になる、(2) 顧客側の ITSM / プロビジョニング連携が
IdMagic の DB をポーリングする以外に手段が無い、(3) コンプライアンス監査で要求される
「ログの外部保全 (WORM ストレージへの継続転送)」を満たせない。

本 WI は「テナントが宛先とイベント種別を登録し、署名付き・再試行付き・配送可視化付きで
push される」汎用 outbound hook を導入する。

## Scope

- **decision**:
  - `spec/contexts/audit/decisions.md` へ記録する決定 (outbound event hook と配送保証): 配送セマンティクス (at-least-once、順序保証は
    しない、重複は購読側が `event_id` で冪等化)、HMAC 署名方式とヘッダ仕様、再試行と指数
    バックオフ、連続失敗時の自動無効化 (circuit) 条件、SSRF 対策 (宛先の URL 検証と private
    IP 範囲の拒否)、payload に載せる PII の範囲 (`spec/contexts/authentication/decisions.md` の認証イベント PII 方針 と
    整合)、[[wi-58-continuous-access-evaluation-agent-revocation]] の CAEP/SSF との分界
    (CAEP は標準準拠の security event 配信、本 WI は汎用 hook) を記録する。
- **specification**:
  - `Audit` に `EventHookSubscription` model (id / tenant_id / name / target_url /
    event_type_filter / secret_ref / state / created_at)、`EventHookDelivery` model
    (id / subscription_id / event_id / attempt / status / response_code / next_attempt_at /
    last_error)、`EventHookState` enum (Enabled / Disabled / Suspended) を追加する。
  - `RegisterEventHook` / `ListEventHooks` / `GetEventHook` / `UpdateEventHook` /
    `DeleteEventHook` / `TestEventHook` / `ListEventHookDeliveries` /
    `RetryEventHookDelivery` interface を追加する。
  - `states` に EventHookRegistered / EventHookDelivered / EventHookDeliveryFailed /
    EventHookSuspended event を追加する。
  - `objectives` に配送捕捉レイテンシ (イベント発生から delivery 行の作成まで) の目標を追加する
    (`spec/contexts/provisioning/decisions.md` の
    ProvisioningDeliveryCaptureLatency と同じ考え方)。
  - `authorization` に hook 管理を `audit:write` 相当の scope / tenant admin に限定する規則を追加する。
  - `scenarios`: 正常配送 / 4xx で再試行せず失敗記録 / 5xx で指数バックオフ再試行 /
    連続失敗で自動 Suspend / private IP 宛先の登録拒否 / 他テナントの delivery が見えない。
- **go**:
  - `EventHookSubscription` の domain (状態遷移、URL 検証、イベント種別フィルタ照合) と
    repository (memory / postgres) を追加する。
  - 配送は **durable job** (`kind=event_hook_delivery`) として実装する。監査イベント書き込みと
    同一トランザクションで delivery 行を作り (transactional outbox)、worker が送信する。
    lane は `default` を使い、認証ホットパスを遅くしない。
  - HMAC-SHA256 署名 (`X-Idmagic-Signature: t=<ts>,v1=<hex>`) とタイムスタンプ付き署名基底文字列で
    replay を防ぐ。secret は書き込み専用で API から読み出せない。
  - 宛先 URL は登録時と送信時の両方で検証する (https 必須、DNS 解決結果が private /
    link-local / loopback 範囲なら拒否、リダイレクト追従なし)。
  - 監査ログストリーミングは同じ subscription 機構の「全 event 種別 + バッチ配送」プロファイルとして
    表現し、専用の別機構を作らない。
- **http**:
  - hook CRUD / テスト配送 / 配送履歴 / 再送の管理 API を追加する。secret は登録時のみ返す。
- **ui**:
  - 管理コンソールに「イベント連携」画面を追加する。購読作成 (宛先 URL / イベント種別選択 /
    secret 生成)、テスト配送の結果表示、配送履歴 (成功・失敗・次回再試行時刻)、失敗の
    エラー本文、手動再送、Suspend からの復帰を提供する。
- **documentation**:
  - README に webhook の署名検証手順 (受信側の擬似コード)、再試行方針、SIEM 連携の
    設定例を追記する。

## Out of Scope

- 特定 SaaS 専用コネクタ (Splunk HEC / Datadog / EventBridge 固有の認証)。汎用 HTTPS + HMAC を
  必達とし、専用コネクタは需要が出た時点で adapter として別 WI にする。
- CAEP / SSF (Shared Signals Framework) 準拠の標準 security event 配信。
  → [[wi-58-continuous-access-evaluation-agent-revocation]]
- inbound webhook (外部からの受信)。→ [[wi-30-inbound-federation-and-identity-broker]] /
  [[wi-95-ldap-ad-user-federation]] の領域。
- イベント payload の任意テンプレート / 変換式。固定スキーマから始める。
- テナントを跨いだ集約配送 (system admin 用の全テナント購読)。

## Plan

- **transactional capture を先に決める**。「イベントは発生したが配送行が作られていない」状態を
  作らないため、監査イベント永続化と delivery enqueue を同一トランザクションに入れる。
  `spec/contexts/provisioning/decisions.md` が
  Provisioning で既に同じ形を採っているので、その構造を踏襲し新しい発明をしない。
- **配送は既存 Jobs 基盤に載せる**。retry / lease / 進捗可視化は
  `spec/contexts/jobs/decisions.md` の永続キュー と
  `spec/contexts/jobs/decisions.md` のワーカー実行モデル が既に提供しているので、
  独自の retry ループを書かない。lane は `default` を使い、`latency_sensitive` を汚さない。
- **SSRF を最初のテストにする**。webhook は「テナント管理者が任意 URL を指定できる」機能で、
  内部メタデータエンドポイントへの到達に使われうる。登録時検証だけでは DNS rebinding を
  防げないため、送信時にも解決 IP を検証する。ここを最初に落とすテストにする。
- **PII は既定で最小**。payload には event_id / type / 時刻 / tenant / actor 識別子 /
  対象リソース識別子を入れ、メール本文・トークン・資格情報・生の IP は入れない。
  詳細が必要な購読者は管理 API で引く前提にする (`spec/contexts/authentication/decisions.md` の認証イベント PII 方針)。
- **自動 Suspend を入れる**。宛先が長期間死んでいるときに無限に job を積むと queue が
  汚染される。連続失敗回数と経過時間の閾値で Suspend し、UI から明示復帰させる。
- 未決定: バッチ配送 (ストリーミング用途) のバッチサイズと最大遅延は、
  [[wi-282-staging-load-testing-and-capacity-validation]] の実測前は保守的な既定値に置く。

## Tasks

- [ ] T001 [Spec] `Audit` に EventHookSubscription / EventHookDelivery / EventHookState、
      interface 8 件、event 4 件、objective、authorization、scenario 6 件を追加し
      `mise run check-spec` を通す。
- [ ] T002 [Spec] outbound event hook と配送保証の決定を `spec/contexts/audit/decisions.md` に記録する (署名方式・再試行・
      自動 Suspend・SSRF 対策・PII 範囲・CAEP との分界)。
- [ ] T003 [Domain] subscription の状態遷移、イベント種別フィルタ照合、宛先 URL 検証を実装する。
      RED: private IP / http / リダイレクト宛先が拒否されるテストを先に書く
      (scenario `Audit.event_hook_ssrf_rejected`) → GREEN。
- [ ] T004 [Persistence] `event_hook_subscriptions` / `event_hook_deliveries` を
      `infra/schema/postgres.sql` に追加し、sqlc クエリと memory / postgres repository を実装する。
      RED: tenant 越え参照が 0 件になるテスト → GREEN。
- [ ] T005 [Capture] 監査イベント永続化と同一トランザクションで delivery を作る capture を
      実装する。RED: 監査書き込みがロールバックしたとき delivery も残らないテスト → GREEN。
- [ ] T006 [Job] `event_hook_delivery` job handler を実装する。HMAC 署名、タイムアウト、
      指数バックオフ、4xx は再試行しない、連続失敗で Suspend。RED: 5xx で再試行し 4xx で
      再試行しないテスト → GREEN。
- [ ] T007 [Usecase] hook CRUD / テスト配送 / 配送履歴 / 手動再送の usecase と audit event を
      実装する。secret は書き込み専用で応答に含めない。RED: secret 非開示テスト → GREEN。
- [ ] T008 [HTTP] 管理 API を追加し、scope / tenant admin 認可を配線する。
      RED: 権限不足で 403 になる handler テスト → GREEN。
- [ ] T009 [UI] 「イベント連携」画面 (購読 CRUD / テスト配送 / 配送履歴 / 再送 / 復帰) を追加する。
      RED: presentation logic の unit test → GREEN。
- [ ] T010 [Streaming] 全 event 種別 + バッチ配送プロファイルを subscription の設定として
      表現し、SIEM 連携ケースを同じ機構で満たす。RED: バッチ境界のテスト → GREEN。
- [ ] T011 [Docs] README に署名検証の擬似コード、再試行方針、SIEM 連携例を追記する。
- [ ] T012 [Verify] 下記 Verification を緑にする。`mise run spec-render` を実行する。

## Verification

- `mise run check` / `mise run check-spec` / `mise run check-work-items` / `mise run check-ids`
- `mise run test-go` / `mise run test-go-race` / `mise run verify-go`
- `mise run verify-ui` / `mise run test-ui-unit`
- 手動: `mise run dev` でローカル受信サーバ (署名検証付き) を立て、(1) ログイン失敗イベントが
  配送されること、(2) 受信側 500 応答で再試行されること、(3) 400 応答で再試行されないこと、
  (4) `http://169.254.169.254/` を宛先に登録できないこと、を確認する。

## Risk Notes

**SSRF が最大のリスク**である。テナント管理者が指定した URL へサーバが接続するため、
クラウドのインスタンスメタデータや内部管理エンドポイントへの到達に転用されうる。
登録時検証に加えて送信時の解決 IP 検証・リダイレクト非追従を必須とし、これを最初に
落とすテストで固定する。
配送 job が queue を汚染するリスクがある (死んだ宛先へ無限に積む)。自動 Suspend と
lane 分離で認証ホットパスへの影響を防ぐ。
payload の PII 過多は情報漏洩面を広げる。既定を最小にし、拡張は明示的な `spec/contexts/audit/decisions.md` 更新を要する。
監査書き込みトランザクションに delivery 作成を含めるため、hook 定義が多いテナントで
書き込みレイテンシが伸びうる。capture は「1 行の delivery + job」に留め、
宛先ごとの展開は worker 側で行う。
