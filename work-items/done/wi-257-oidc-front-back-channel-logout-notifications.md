---
depends_on: []
status: completed
authors: [tn]
risk: medium
reversibility: irreversible
created_at: 2026-07-19
priority: p1
change_kind: feature
evidence_policy: risk-based-v3
documentation_impact:
  level: release_note
  reason: OIDC のログアウトが接続済み RP へ伝播する新しい利用者向け能力を追加するため、リリースの読み手へ知らせる。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-257.md }
initial_context:
  specification:
    - docs/contexts/oauth2/scenarios.feature.md#REQ-OAUTH2-025
    - docs/contexts/oauth2/standards.md
    - docs/contexts/oauth2/internals.md
    - docs/contexts/jobs/decisions.md
  typespec:
    - spec/contexts/oauth2/models.tsp
    - spec/contexts/oauth2/main.tsp
    - spec/contexts/jobs/models.tsp
  source:
    - backend/oauth2
    - backend/jobs
    - backend/shared/security/tokens_jose
    - backend/shared/security/safehttp
  tests:
    - backend/oauth2
    - backend/shared/http/server_http
  stop_before_reading:
    - frontend
    - backend/saml
    - backend/wsfederation
affected_spec:
  - { path: spec/contexts/oauth2/models.tsp, symbol: IdMagic.Contract.OAuth2Client }
  - { path: spec/contexts/oauth2/models.tsp, symbol: IdMagic.Contract.ClientRegistrationRequest }
  - { path: spec/contexts/oauth2/models.tsp, symbol: IdMagic.Contract.ClientRegistrationResponse }
  - { path: spec/contexts/oauth2/models.tsp, symbol: IdMagic.Contract.AdminOAuth2ClientCreateRequest }
  - { path: spec/contexts/oauth2/models.tsp, symbol: IdMagic.Contract.AdminOAuth2ClientUpdateRequest }
  - { path: spec/contexts/oauth2/models.tsp, symbol: IdMagic.Contract.AdminOAuth2ClientResponse }
  - { path: spec/contexts/oauth2/models.tsp, symbol: IdMagic.Contract.ClientSession }
  - { path: spec/contexts/oauth2/models.tsp, symbol: IdMagic.Contract.LogoutNotification }
  - { path: spec/contexts/oauth2/models.tsp, symbol: IdMagic.Contract.FrontChannelLogoutTarget }
  - { path: spec/contexts/oauth2/models.tsp, symbol: IdMagic.Contract.DiscoveryDocument }
  - { path: docs/contexts/oauth2/scenarios.feature.md, requirement: REQ-OAUTH2-023 }
  - { path: docs/contexts/oauth2/scenarios.feature.md, requirement: REQ-OAUTH2-025 }
  - { path: spec/contexts/oauth2/main.tsp, symbol: IdMagic.Contract.CheckSessionIframe }
  - { path: docs/contexts/oauth2/standards.md, requirement: OIDC-FRONTCHANNEL-IFRAME }
  - { path: docs/contexts/oauth2/standards.md, requirement: OIDC-BACKCHANNEL-LOGOUT-TOKEN }
  - { path: spec/contexts/jobs/models.tsp, symbol: IdMagic.Contract.JobKind }
primary_use_cases:
  - id: back-channel-logout-delivery
    requirement: REQ-OAUTH2-025
    observable_result: ローカルログアウト後、対象 RP が同じ sid と sub を持つ署名済み logout token を受信する。
    unit_test:
      path: backend/oauth2/logout/usecases/logout_test.go
      name: TestStartBackChannelLogout_REQ_OAUTH2_025
      task: test-go-race
    e2e_test:
      path: backend/shared/http/server_http/logout_propagation_e2e_test.go
      name: TestEndSessionBackChannelLogout_REQ_OAUTH2_025
      task: test-go-race
    unit_fault_model: ClientSession から通知と永続ジョブを作らない実装は、保存件数とジョブ引数の検査で検出する。
    e2e_fault_model: end_session の本番配線が通知を開始しない実装は、TLS の test RP が logout_token を受信しないことで検出する。
  - id: front-channel-logout-iframes
    requirement: OIDC-FRONTCHANNEL-IFRAME
    observable_result: ローカルログアウト応答が対象 RP ごとの iframe を含み、必要な RP だけへ iss と sid を渡す。
    unit_test:
      path: backend/oauth2/logout/usecases/logout_test.go
      name: TestFrontChannelLogoutTargets_OIDC_FRONTCHANNEL_IFRAME
      task: test-go-race
    e2e_test:
      path: backend/shared/http/server_http/logout_propagation_e2e_test.go
      name: TestEndSessionFrontChannelLogout_OIDC_FRONTCHANNEL_IFRAME
      task: test-go-race
    unit_fault_model: session_required の条件を無視する実装は、生成 URL のクエリパラメータ比較で検出する。
    e2e_fault_model: end_session 応答へ iframe を接続しない実装は、HTML に RP の URL が無いことで検出する。
---

# OIDC Front-Channel / Back-Channel Logout の通知配送を実装する

## Motivation
[[wi-28-session-management-and-oidc-logout-completion]] の T004/T005 で、session
revoke に伴う refresh token family の失効と `/end_session` の `id_token_hint`
検証は実運用相当まで完成した。しかし、接続済み RP (Relying Party) へ「ユーザーが
ログアウトしたこと」を伝播する OpenID Connect Front-Channel Logout 1.0 /
Back-Channel Logout 1.0 は未実装であり、`docs/contexts/oauth2/` の
`standards.OpenIDConnectFrontChannelLogout` / `OpenIDConnectBackChannelLogout` は
既に `adoption: required` として宣言済みである (wi-28 T001)。宣言した標準と実装の
不一致を解消するため、通知配送を実装する。

Keycloak / Okta / Google 相当の IdP では、ユーザーが idmagic からログアウトした際に
接続先アプリ (RP) 側のセッションも連動して終了することが期待される。これが無いと、
idmagic 上ではログアウト済みでも RP 側では認証済みのまま残り続ける。

## Scope
以下は wi-28 の T001 (specification-first) で既に `docs/contexts/oauth2/` /
`docs/contexts/jobs/` に追加・コミット済みであり、本 WI はそれらに対する
Go 実装を担当する (追加の specification 変更が必要になった場合のみ `spec-change` に戻る)。

- `models.ClientSession` — sid が発行結果を渡した RP (client_id) の参加記録。
- `models.LogoutNotification` / `models.LogoutNotificationState` /
  `state_machines.LogoutNotificationLifecycle` — back-channel logout 配送の
  outbox 行とその状態機械。
- `models.FrontChannelLogoutTarget` — front-channel logout の iframe target。
- `models.OAuth2Client` の `backchannel_logout_uri` /
  `backchannel_logout_session_required` / `frontchannel_logout_uri` /
  `frontchannel_logout_session_required`。
- `models.DiscoveryDocument` の `frontchannel_logout_supported` /
  `frontchannel_logout_session_supported` / `backchannel_logout_supported` /
  `backchannel_logout_session_supported` / `check_session_iframe`。
- `interfaces.FrontChannelLogout` (internal) / `interfaces.BackChannelLogout`
  (internal) / `interfaces.CheckSessionIframe` (public)。
- `spec/contexts/jobs/models.tsp` の `JobKind.backchannel_logout_delivery`。

Go 実装スコープ:
- 管理者向けクライアント編集 (`RegisterClient` / `UpdateAdminOAuth2Client` /
  admin client handler) に上記 4 メタデータフィールドの CRUD を追加する。
- token 発行成功時 (`ExchangeCodeForToken`、authorization_code グラント) に
  `ClientSession` を upsert する。
- ローカル revoke 確定後 (`/end_session` の local logout、wi-28 T005 が実装した
  経路) に、対象 sid の `ClientSession` から `backchannel_logout_uri` 登録済み
  RP ごとの `LogoutNotification` を作成し、Jobs (`kind=backchannel_logout_delivery`)
  へ enqueue する。
- logout token signer (`iss`, `sub`, `aud`, `iat`, `jti`,
  `events: {http://schemas.openid.net/event/backchannel-logout: {}}`, `sid`) を
  実装する。
- `BackChannelLogout` job handler (Jobs 経由の retry、2xx=成功、その他/timeout/
  接続失敗=再試行) を実装し `cmd/idmagic-worker` に登録する。
- `FrontChannelLogout` (iframe target 一覧の算出) を実装し、`/end_session`
  応答へ埋め込む。
- `CheckSessionIframe` (静的ページ、`session_state` 相関アルゴリズムは実装しない
  — `docs/contexts/oauth2/internals.md` の「広告と静的検査だけを提供する」に従う) を実装する。
- `LogoutNotification` の状態遷移を監査可能にする (配送成功/失敗の追跡)。

## Out of Scope
- CAEP / Shared Signals。別 WI で扱う (wi-28 と同じ整理)。
- access token の即時失効 (denylist)。`docs/contexts/oauth2/internals.md` が
  「アクセストークンの失効は対象外とし、最大 600 秒の残存リスクを受け入れる」と定めている。
- `check_session_iframe` の `session_state` salted hash 相関アルゴリズム
  (Draft 28 のため `adoption: optional`)。
- SAML / WS-Federation の Single Logout。本 WI は OIDC のみを扱う。

## Design

`ClientSession` は認可コード交換が成功し `sid` がある場合だけ記録し、ログアウト時に同じ `sid` へ参加したクライアントを引く索引とする。

front-channel は HTTP 応答内で iframe target を算出する同期処理とし、配送保証を持たせない。

back-channel は `LogoutNotification` を先に保存し、その ID と token 作成に必要な `sub`、`iss` を永続 Job へ渡す。

通知作成時に生成した `jti` を retry 間で固定し、2xx だけを Delivered、再試行余地のある失敗を Pending、試行上限の失敗を Failed とする。

時刻、ID 生成、署名、永続化、ジョブ投入、外部 HTTP 配送は port の境界へ置く。

登録済みの配送先であっても outbound HTTP の SSRF 境界になるため、URI は HTTPS かつ host 必須、userinfo と fragment 禁止とする。

配送はプロキシを使わず private IP を拒否し、redirect ごとに同じ検証を行う既存の `safehttp` client を使う。

この制御は `docs/contexts/oauth2/internals.md` と `docs/design/security/threat-model.md` に反映した。

## Plan
- 設計判断は `docs/contexts/oauth2/internals.md` の「OIDC session binding and logout propagation」に
  既に書かれている (back-channel の配送は永続的で冪等な `Job`、front-channel は同じリクエスト内で
  計算する `iframe` 送信先一覧で配送保証なし、`check_session_iframe` は広告と静的検査だけ)。
  つまり本 work item に残っているのは設計ではなく実装と規範シナリオである。
  仕様は既に「そう動く」と書いているのに実装が無い状態なので、この乖離を閉じることが目的になる。
  新規の設計判断が必要になった場合のみ `docs/contexts/oauth2/decisions.md` に追記する。
- `LogoutNotification` の specification モデルは `sub` (対象ユーザー) を持たない
  (session 状態の複製を避けるため)。しかし logout token は `sub` claim を
  必須とする (`OIDC-BACKCHANNEL-LOGOUT-TOKEN`)。ワーカープロセス内でテナント別
  issuer やユーザーを再解決する複雑さを避けるため、`sub` と `iss` は
  `LogoutNotification` 作成時点 (HTTP リクエストコンテキスト内、
  `tenancy.Issuer`/session 解決が可能な時点) に Jobs の job params
  (`{"notification_id":..., "sub":..., "iss":...}`) として運ぶ。job params は
  Jobs 側の永続化にそのまま乗るため、追加の永続化は不要。
- `logout_token_jti` は `LogoutNotification` 作成時に一度だけ生成し、retry の
  たびに再生成しない (同一論理配送に対し複数の jti が生まれることを避ける)。
- `ClientSession` は index のみを持つため、front/back-channel 双方の対象解決は
  `ClientSession.ListBySid(sid)` → 各 `client_id` の `OAuth2Client` を引いて
  `backchannel_logout_uri`/`frontchannel_logout_uri` の有無で振り分ける。
- 永続化は `LogoutNotification` も `ClientSession` も memory + postgres の
  両方を実装する (`refresh_tokens.sid` と同じ理由: 通知配送はプロセス再起動を
  跨いで再試行される必要があり、Jobs 自体が postgres 永続化されている以上、
  参照先の `LogoutNotification` もメモリのみでは再起動後に追跡不能になる)。

## Tasks
- [x] T001 [Client Metadata] `domain.OAuth2Client` に4フィールドを追加し、
      `RegisterClient` / `UpdateAdminOAuth2Client` / admin client handler /
      memory・postgres adapter (`clients` テーブルへのカラム追加) で
      CRUD できるようにする。
- [x] T002 [ClientSession] `domain.ClientSession` + `ports.ClientSessionStore`
      (Upsert/ListBySid) を追加し、memory・postgres 両adapterを実装する。
      `ExchangeCodeForToken` (authorization_code グラント成功時、sid が
      non-nil のときのみ) から upsert する。
- [x] T003 [LogoutNotification] `domain.LogoutNotification` /
      `LogoutNotificationState` + `ports.LogoutNotificationStore`
      (Save/FindByID) を追加し、memory・postgres 両adapterを実装する。
      local logout (wi-28 T005 の `end_session_handler.go` 経路) から、
      対象 sid の `ClientSession` を引いて `backchannel_logout_uri` 登録済み
      RP ごとに `LogoutNotification` を作成し、Jobs へ enqueue する use case
      を実装する。
- [x] T004 [Logout Token] logout token signer (`iss`/`sub`/`aud`/`iat`/`jti`/
      `events`/`sid`) を実装し、`BackChannelLogout` job handler
      (`kind=backchannel_logout_delivery`) を `cmd/idmagic-worker` に登録する。
      2xx=Delivered、それ以外/timeout/接続失敗は Jobs の attempts/max_attempts
      に応じて Pending (再試行) または Failed (dead-letter) に遷移させる。
- [x] T005 [Front-Channel] `FrontChannelLogout` use case
      (対象 sid の `ClientSession` から `frontchannel_logout_uri` 登録済み
      RP の iframe target 一覧を算出、`frontchannel_logout_session_required`
      なら iss/sid クエリパラメータを付与) を実装し、`/end_session` の応答へ
      埋め込む。
- [x] T006 [Session Management] `CheckSessionIframe` (`GET /session/check`) —
      RED: `TestCheckSessionIframe_noSession_respondsChanged` /
      `TestCheckSessionIframe_validSession_respondsUnchanged` を先に 404 で
      fail 確認 (`backend/oauth2/handlers_http/check_session_iframe_handler_test.go`)
      → GREEN (`check_session_iframe_handler.go`)。静的ページ + 現在の browser
      cookie が有効な LoginSession に解決できるかどうかだけを埋め込んで返す
      最小実装 (`docs/contexts/oauth2/internals.md`)。`d.AuthnResolver.Resolve` の結果 (nil または
      `AuthenticationPending`) を fail-safe 側 ("changed") に倒す。
      route を `backend/oauth2/handlers_http/routes.go` に登録し、
      `TestAssembledRoutesMatchGeneratedOpenAPI` の `GET /session/check` 差分を
      解消した (T007 verification の一部を前倒しで満たす)。wi-56 のブランチ作業中に
      発見した specification/実装 drift の修正として先行実装。
- [x] T007 [Verify] 複数 RP への配送、一時的配送失敗からの再試行、
      max_attempts 到達による dead-letter、同一 `LogoutNotification` の
      retry を跨いだ jti 不変性、ワーカー再起動後の配送継続を検証する。
      `GET /session/check` の route 契約差分は T006 で解消済み。

## Verification
- `mise run check`
- `mise run test-go` / `mise run lint-go` / `mise run build-go`
- 手動: `backchannel_logout_uri` を登録した test RP に対し、session revoke で
  logout token が POST されることを確認する。
- 手動: `frontchannel_logout_uri` を登録した RP への iframe が `/end_session`
  応答に含まれることを確認する。

## Risk Notes
`LogoutNotification`/`ClientSession` を postgres へ正しく永続化しないと、
ワーカー再起動時に配送中の通知を追跡できなくなる (Jobs 自体は永続化されるが
参照先の notification 行が失われると job params だけでは復元できない属性
(`target_uri` 等) が欠落する)。

logout token 配送は RP という外部境界への outbound HTTP であり、RP 側の
応答本体を解釈しない (2xx/非2xx の判定のみ) ため、未信頼入力のパースリスクは
無い。id_token_hint 検証 (wi-28 T005) と同様の理由で fuzz/property test は
本 WI でも採用しない。

`sub`/`iss` を job params (JSONB) 経由で運ぶ設計は、Jobs の汎用性
(`docs/contexts/jobs/decisions.md`: Jobs は汎用の永続キュー) を維持しつつ、ワーカー内でのテナント別
issuer 再解決という複雑さを避けるための選択。将来 Jobs 側で機微情報を job params
に含めることが問題になった場合は再検討する。

## Completion

- **Completed At**: 2026-09-09
- **Summary**:
  既存の TypeSpec と規範文書が宣言していた OIDC Front-Channel / Back-Channel Logout を実装した。
  認可コード交換で RP と `sid` の参加関係を保存し、`/end_session` のローカル失効後に iframe 応答と永続ジョブによる logout token 配送を開始する。
  配送失敗はローカルログアウトを巻き戻さず、通知状態とジョブの試行回数へ残る。
  `mise run spec-diff` 相当の確認では TypeSpec の規範差分は無く、実装と既存仕様の乖離を閉じた。
- **Primary Use Case Evidence**:
  - id: back-channel-logout-delivery
    unit_red: TestStartBackChannelLogout_REQ_OAUTH2_025 は、未実装時に logout package と通知作成 API が存在せずコンパイル失敗した。
    e2e_red: TestEndSessionBackChannelLogout_REQ_OAUTH2_025 は、未実装時に RP が logout_token を受信せず失敗した。
    unit_fault_injection: ジョブ handler の型と配線を外した状態では対象 package がコンパイルせず、通知作成と永続ジョブの結合を検出した。
    e2e_fault_injection: end_session の通知開始を持たない既存配線では TLS test RP の受信チャネルが timeout し、正式な HTTP 入口からの未配送を検出した。
  - id: front-channel-logout-iframes
    unit_red: TestFrontChannelLogoutTargets_OIDC_FRONTCHANNEL_IFRAME は、未実装時に target 算出 API が存在せずコンパイル失敗した。
    e2e_red: TestEndSessionFrontChannelLogout_OIDC_FRONTCHANNEL_IFRAME は、未実装時の /end_session 応答に RP iframe が無く失敗した。
    unit_fault_injection: frontchannel_logout_session_required を無視する実装では、期待した iss と sid の有無が URL 比較と一致せず失敗する。
    e2e_fault_injection: end_session の HTML 描画へ target を渡さない既存配線では、応答本文に登録済み RP URL が現れず失敗した。
- **Change-Resistance Results**:
  retry の最初の失敗後も状態が Pending で、次の成功が Delivered になり、両試行で同じ `jti` を使うことを `TestBackChannelLogoutHandlerRetriesWithStableJTI` が固定する。
  試行上限の失敗が Failed になることを `TestBackChannelLogoutHandlerMarksFinalFailure` が固定する。
  signer のテストは logout token の `typ`、`iss`、`sub`、`aud`、`iat`、`jti`、`sid`、events と nonce 不在を直接読む。
  PostgreSQL adapter のテストは ClientSession と LogoutNotification が再生成した store から読めるため、プロセス再起動後もジョブの参照先が残ることを検出する。
- **Timing Analysis**:

  | 細粒度の作業 | 実測または能動作業時間 | 分析 |
  |---|---:|---|
  | 仕様・既存設計・境界の読み取り | 約9分33秒 | 最大の思考時間。既存 TypeSpec がすでに完全だったため、新しい規範編集を避けられた。対象文書を work item の `initial_context` から先に読む方法が最短である。 |
  | T001 クライアントメタデータ | 約4分26秒 | DB、domain、管理 API の同時変更が必要だった。sqlc 1.58秒、整形1.02秒、対象テスト4.04秒で、実行時間より編集と配線確認が支配した。 |
  | T002 ClientSession | 約8分38秒 | token exchange から memory/PostgreSQL の両 adapter まで横断したため時間を使った。sqlc 1.45秒、GREEN 1.01秒だった。 |
  | T003/T004 通知・token・worker | 約12分 | 永続状態、署名、retry、worker DI の境界が多い。主要コマンドは signer 1.25秒、push 2.03秒、use case 2.32秒、handler package 10.62秒だった。 |
  | T005 front-channel と E2E | 約8分 | `/end_session` の失効順序と HTML/redirect を保った配線が中心。front E2E 3.09秒、back E2E は fixture 修正後1.58秒だった。 |
  | PostgreSQL・discovery | 約5分 | DB test 4.70秒、discovery test 1.25秒で、実行より fixture と生成コード確認が中心だった。 |
  | 変更 package の race test | 初回76.70秒、最終84.22秒 | 最長の単一コマンド。共有 PostgreSQL schema により多数 context の `models.go` が機械更新され、reverse dependency の対象が広がったことが原因である。 |
  | 仕様生成 | 初回3.61秒、setup 1.24秒、再実行3.45秒 | Mermaid 未導入で初回だけ失敗した。作業開始時に `setup-tools` を一度完了すれば、初回3.61秒は不要だった。 |
  | SSRF 対策後の対象検証 | format 1.21秒、各テスト1.93〜13.88秒、check-spec 3.70秒、境界0.35秒 | 4テストを並列実行し、直列なら約26秒のところを約14秒に短縮した。 |
  | 完了記録の形式修正 | 約8秒 | Completion の YAML 値を Markdown のバッククォートで始めたことと、テスト内の要件 ID 注記不足による不要な手戻り。既存の完了例を先に複製すれば避けられた。 |
  | PostgreSQL schema 収束検査 | 9.29秒 | 空 DB への適用と二度目の no-op を検証した。schema を変えた項目では省けない。 |
  | API DTO の TypeSpec 漏れ修正 | 約4分 | 契約差分検査が3 operation の4項目不足を検出した。修正後は仕様生成2.64秒、互換性検査1.21秒、契約差分検査1.27秒で通過した。 |
  | Go lint 修正 | 約3分 | 初回11件、修正後の見落とし1件と3件を局所実行で解消し、最終 lint は1.95秒で0件だった。 |
  | UI 依存関係の導入 | 1.72秒 | 永続 worktree に `node_modules` が無かったため必要になった。worktree 作成直後の共通 setup に含めれば最終ゲートの失敗は避けられた。 |
  | 最終集約ゲート | 初回37.90秒、再実行19.92秒 | 初回は lint 11件、API DTO の TypeSpec 漏れ、未導入 UI 依存関係を同時に検出した。個別修正後の再実行は全ゲートを通過した。 |
  | 一時 worktree 消失からの復旧 | 約35分 | 純粋な不要コスト。OS が `/tmp` の worktree を除去し、未コミット変更を再実装した。リポジトリ配下の永続 worktree と小さい checkpoint commit を使えば回避できる。 |

  約20時間の利用者待ち・セッション休止は能動作業時間から除外した。
  sandbox で Go cache が拒否された短い失敗も複数あり、最初から承認済みの `mise` task を同じ実行境界で使えば往復を減らせる。
  sqlc は GREEN の局所性を保つため段階ごとに実行したが、合計は約4.5秒であり、最後に一括生成しても節約は小さい。
  一方、共有 schema が全 context の生成モデルへ同じ列を複製する構成は検証範囲を大きくするため、context 別 schema または生成モデル分離を別 work item で検討する価値がある。
  開発中は対象テスト、最後は `test-go-changed` と `verify` を各1回にする方法が、診断の速さと重複処理の少なさを両立する。
- **Verification Results**:
  - `mise run spec-render` - passed（180文書、336 operation、19 API tag、889 TypeSpec symbol）
  - `mise run check-api-compat` - passed
  - `mise run check-spec` - passed
  - `mise run check-boundaries` - passed
  - `mise run test-go-package -- ./backend/oauth2/logout/usecases` - passed
  - `mise run test-go-package -- ./backend/oauth2/logout/push_http` - passed
  - `mise run test-go-test -- ./backend/shared/http/server_http TestEndSessionFrontChannelLogout_OIDC_FRONTCHANNEL_IFRAME` - passed
  - `mise run test-go-test -- ./backend/shared/http/server_http TestEndSessionBackChannelLogout_REQ_OAUTH2_025` - passed
  - `mise run check-schema` - passed
  - `mise run test-go-changed` - passed（84.22秒）
  - `mise run verify` - passed（19.92秒）
