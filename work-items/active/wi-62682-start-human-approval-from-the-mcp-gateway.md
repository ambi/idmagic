---
status: pending
authors: [tn]
risk: high
created_at: 2026-11-11
priority: p2
group: mcp-gateway
depends_on: [wi-34766-return-structured-approval-requests-from-evaluation]
change_kind: feature
affected_spec:
  - { path: docs/modules/oauth2/features/resource-access.md, requirement: REQ-OAUTH2-065 }
  - { path: docs/modules/oauth2/features/resource-access.md, requirement: REQ-OAUTH2-062 }
  - { path: docs/modules/oauth2/features/approval.md, requirement: REQ-OAUTH2-043 }
  - { path: docs/modules/mcp-gateway/features/runtime.md, requirement: REQ-MCPGATEWAY-003 }
  - { path: docs/modules/mcp-gateway/features/runtime.md, requirement: REQ-MCPGATEWAY-013 }
  - { path: spec/modules/oauth2/models.tsp, symbol: ResourceAccessEvaluationRequest }
  - { path: spec/modules/oauth2/models.tsp, symbol: ResourceAccessEvaluationResponse }
---

# MCP ゲートウェイが承認を起票し、汎用の MCP クライアントを人の承認へ導く

## 動機

REQ-OAUTH2-065 と REQ-MCPGATEWAY-013 により、人が承認した操作を最大一回だけ転送できるようになった。
しかし承認を始められるのは、自分で `/bc-authorize` を呼び、引数と完全に一致する `fields` を組み立て、承認後のトークンへ差し替えて再送する Agent だけである。
Claude や Cursor などの汎用の MCP クライアントは、認可コードで得た一つのトークンを使い続け、CIBA もトークンの差し替えもしない。
このため、ゲートウェイの背後で承認を求める操作は、汎用のクライアントからは常に 403 で終わる。

[[wi-34766-return-structured-approval-requests-from-evaluation]] は、Auth0 や Okta と同じく、登録した Agent のアプリが CIBA と RAR で承認を求める経路を整える。
この項目はそれを置き換えず、CIBA を呼べない汎用のクライアントのための経路を補う。
主要な IdP にこの形の前例はなく、AuthZEN AARP（WG Draft）と MCP の URL モードの elicitation（今後変わり得る機能）を組み合わせた設計である。
そのため、着手時に T000 で MCP 本体のステップアップ認可と比べ、採否を確定する。

## 対象範囲

- 評価の入口で、執行点の求めに応じて、規則と実際の引数から IdMagic が明細を組み立てた承認リクエストを起票し、本人へ既存のメールで通知する。
- 起票した承認リクエストの識別子を伴う評価で、トークンを差し替えずに、REQ-OAUTH2-065 と同じ照合と一回限りの使用を適用する。
- `proxy` モードのゲートウェイで、承認が必要なとき、MCP 2026-07-28 の `InputRequiredResult`（URL モードの elicitation と `requestState`）を返し、再送を評価して一回だけ転送する。
- アカウントポータルで、承認の URL から該当する承認リクエストを開き、起票したリソースサーバーを表示する。

## 対象外

- `envoy_ext_authz` モード。HTTP 版の `ext_authz` は 200 を許可として読むので、200 の `InputRequiredResult` を返せない。Envoy では従来の 403 を維持する。対応するには `ext_proc` か、Envoy の前段での応答の書き換えが要る。
- 主体が Agent のトークン（`principal_type` が `agent`）。承認者を所有者などへ解決する規則が要り、MCP クライアントの前に承認する本人がいない非同期の流れになる。AuthZEN AARP のタスクとコールバックとあわせて検討する。
- AuthZEN の Access Request の入口と、AARP への適合の主張（[[wi-80811-decide-authzen-compatible-evaluation]]）。今回の応答の項目名は AARP に寄せ、後で互換の入口から再利用できる形にする。
- 承認リクエストの取り消しと、利用者が elicitation を辞退したときの即時の失効。辞退後も承認リクエストは期限で `Expired` になる。
- `-32000`〜`-32019` の範囲の既存のエラーコード（`-32000`、`-32001`、`-32002`、`-32004`、`-32005`）。2026-07-28 版は、新しい実装にこの範囲を使わないよう求めている。今回の新しい応答はこの範囲を使わず、既存のコードの移行は別の項目で扱う。
- CIBA の `push` と、メール以外の通知の経路。
- Agent が自分で始める CIBA の経路。wi-34766 が扱う。

## 設計

### 採用する流れ

```text
tools/call(args)          → Gateway → evaluate(token, 操作, args, approval_start=true)
                                       IdMagic：ほかの論理積を満たし、承認が必要
                                                → ApprovalRequest(Pending, 起票者=リソースサーバー) を作り、本人へメール
                          ← InputRequiredResult{ inputRequests.approval = elicitation/create(mode=url, url=approval_uri),
                                                 requestState = AEAD(approval_request_id, ツール名と引数の digest, 期限) }
利用者は承認の URL を開き、セッション、ステップアップ、CSRF を経て承認する（REQ-OAUTH2-043）
tools/call(args, requestState, inputResponses) → Gateway：状態を検証し、digest の一致を確かめる
                                       → evaluate(token, 操作, args, approval_request_id, operation_key=新しい鍵)
                                       IdMagic：Approved、本人、クライアント、Agent、明細、引数、未使用を照合し、使用権を記録する
                          ← 上流へ一回だけ転送（REQ-MCPGATEWAY-013 のまま）
```

### 判断

- **起票を評価の入口に統合する。** 別の起票の入口を設けると、ゲートウェイは同じ要求について評価を二回呼ぶことになる。DPoP の証明は一回しか使えず、クライアントの一つの要求には証明が一つしかないので、二回目を送れない。統合すれば、起票と判定が同じ評価の結果になり、評価と起票の間で状態が変わる TOCTOU も生じない。起票は `approval_start` を付けた評価に限り、付けない評価の作用は従来どおりにする。
- **明細は IdMagic が組み立てる。** `type` は規則の `authorization_detail_type`（条件が求めた場合は `approval_detail_type`）、`locations` は登録の `resource`、`actions` は `relation`、`identifier` は `resource_id`、`fields` は `operation_parameters` とする。Agent やリソースサーバーの自由な文言（`binding_message` など）は受け取らない。
- **トークンを差し替えない。** 承認はトークンの RAR ではなく、サーバー側の `ApprovalRequest` に置く。照合では、テナント、トークンから導いた利用者、クライアント、Agent が承認リクエストの値と一致することを求める。これは MCP の「クライアントの bearer トークンは変わらない」というモデルと、AARP の「承認それ自体はアクセスを与えず、再評価で判定する」というモデルに合う。
- **参照できる承認を限る。** `approval_request_id` で参照できるのは、評価の起票で作られ、参照する評価と同じ登録が起票した承認リクエストだけとする。CIBA で起票した承認リクエストは参照できない。
- **使用は一回限りのままにする。** AARP は有効期間内の再利用を許すが、承認を求めるのは高影響の操作なので、REQ-OAUTH2-065 の使用権の規則（同じ `operation_key` の再送だけを許す）を適用する。
- **同じ操作の起票を重ねない。** 同じテナント、利用者、クライアント、登録、操作、正規化した引数の `Pending` の承認リクエストが期限内にあれば、新たに起票せずその識別子を返し、メールを再送しない。さらに、リソースサーバーが起票した `Pending` の承認リクエストは一人の利用者につき 10 件までとし、超えた場合は `approval_request_limit_exceeded` で不許可にする。評価の流量制限は一クライアント当たり毎秒 100 回であり、メールの濫用を抑えられないためである。
- **主体が利用者のトークンだけで起票する。** MCP クライアントの前にいる本人が承認者となる。主体を導けないトークンと、主体が Agent のトークンでは起票せず、従来の `approval_required` を返す。
- **URL モードの用途の解釈。** 2026-07-28 版は「MCP サーバーが自分自身への認可に URL モードを使ってはならない」と定める。今回は、MCP クライアントのアクセスの認可は従来の OAuth のままで、トークンも変えず、操作ごとの承認を IdMagic という外部の主体で行うので、この禁止に当たらないと判断する。承認の URL は事前に認証済みの URL ではなく、開いた者がポータルのセッションで本人であることを確かめる（elicitation のフィッシング対策の MUST を満たす）。
- **`requestState` は参照だけを運ぶ。** 中身は版、`approval_request_id`、ツール名と正規化した引数の digest、承認リクエストの期限とする。AEAD で保護し、改ざん、期限切れ、digest の不一致の状態は拒否する。鍵は `idmagic.client_secret` から HKDF で導く。同じ登録の複数台のゲートウェイが設定を増やさずに同じ鍵を持てるためで、シークレットを替えると保留中の状態が無効になるが、その場合は再度の承認を求めるだけで済む。主体を状態に入れないのは、承認を待つ間にトークンが更新され得るためで、主体の一致は IdMagic が再評価で確かめる。
- **クライアントの能力で分ける。** 要求の `_meta` の `io.modelcontextprotocol/clientCapabilities` が `elicitation.url` を宣言しない場合、ゲートウェイは `approval_start` を送らない。結果が承認の不足なら、400 と `-32021`（MissingRequiredClientCapability）、`data.requiredCapabilities` を返し、起票しない。仕様が、宣言していない能力を必要とする要求にこの応答を求めているためである。
- **再送の結果。**
  - 保留中なら、同じ URL の elicitation と、新しい `requestState` を持つ `InputRequiredResult` を返す。MRTR は同じ要求への繰り返しを許す。`inputRequests` を省くとクライアントが即時に再送し続けるので、省かない。
  - 拒否なら、403 を返し、起票し直さない。
  - 期限切れか承認リクエストが残っていなければ、新たに起票する。
  - elicitation の `action` が `accept` でない再送は、評価せずに 403 を返す。

### 主要な型とシグネチャ（予定）

```go
// approval/domain
type ApprovalInitiator string // "client_backchannel" | "resource_server"
type ApprovalRequest struct {
    // 既存の項目に加えて
    Initiator          ApprovalInitiator
    ResourceServerID   *string // Initiator が resource_server のとき必須
    OperationDigest    *string // 重複起票の判定に使う、正規化した操作と引数の digest
    // AuthReqIDHash と IntervalSeconds は client_backchannel のときだけ必須にする
}

// 評価のユースケースへの入力の追加
type ResourceAccessEvaluation struct {
    // 既存の項目に加えて
    ApprovalStart     bool
    ApprovalRequestID *string
}
// 出力の追加
type StartedApproval struct {
    ID        string
    ExpiresAt time.Time
    Status    string // "pending"
    URI       string // アカウントポータルの承認リクエストの URL
}
```

時刻、ID の生成、通知、承認ストアは既存のポートを使う。
重複起票の判定と上限の数え上げは、承認ストアの一つの操作で不可分に行う。
ゲートウェイの `requestState` の封緘と開封は、時刻と鍵を入力に取る純粋な関数にする。
変更は OAuth2 モジュールの中（評価のユースケースと承認のパッケージ）、ゲートウェイ、フロントエンドに閉じる。モジュール間の依存も、テーブルの所有者も変えない。

### TypeSpec の変更（予定）

- `ResourceAccessEvaluationRequest` に `approval_start?: boolean` と `approval_request_id?: string` を加える。
- `ResourceAccessEvaluationResponse` に `approval_request?: { id, status, expires_at, approval_uri }` を加える。不許可の応答だが、トークンのクレームは含まない。
- 不許可の理由に `approval_pending`、`approval_denied`、`approval_request_limit_exceeded` を加える。

### 着手時に比べる案：MCP のステップアップ認可

MCP 2026-07-28 の認可の仕様では、サーバーが 403 と `error="insufficient_scope"`、必要なスコープを返し、クライアントが認可をやり直して新しいトークンで再送する。
必要なスコープは要求の引数から動的に決めてよい。
ゲートウェイが一回限りのスコープ（例：`urn:idmagic:approval:<id>`）を示し、IdMagic の認可画面で操作の明細を表示して、`authorization_details` を持つトークンを発行すれば、REQ-OAUTH2-065 の既存の照合をそのまま使える。
本人がブラウザーの前にいる認可コードの流れであり、トークンを取り直す点は主要な IdP の方式（RFC 9470 型のステップアップと RAR）と同じである。
T000 で、URL モードの elicitation とステップアップ認可の実クライアントの対応、スコープの和をとるクライアントの振る舞いでトークンに残る権限、承認画面の本人確認を比べ、採否を決める。
ステップアップ認可を採るなら、以下の設計と計画を書き直す。

### 採用しない案

- **汎用のクライアントに CIBA を求める。** 汎用の MCP クライアントは CIBA を呼ばない。自作の Agent のための経路は wi-34766 が扱う。
- **ツールのエラー結果で承認待ちを伝える。** LLM の解釈に依存し、利用者の UI へ確実には届かない。2026-07-28 版には MRTR という正規の経路がある。
- **`-32042`（URLElicitationRequiredError）。** 2025-11-25 版だけの符号で、2026-07-28 版では出してはならない。
- **form モードの elicitation による確認。** 確認の UI は MCP クライアントが描くので、Agent の実行環境が自動で承諾でき、本人の判断の証拠にならない。
- **独立した起票の入口（AARP の Access Request の形）。** DPoP の証明が一回限りであるため、ゲートウェイが一つの要求で評価と起票の両方を呼べない。互換の入口は wi-80811 で、評価の結果を再利用する形で検討する。
- **ゲートウェイが承認の成立を待って評価を繰り返す。** 同じ理由で証明を再利用できず、ゲートウェイが状態を持つことになる。
- **`requestState` 専用の鍵ファイル。** 設定とローテーションの手順が増える。鍵を分ける要求が出たら再検討する。

## 計画

1. T001 で REQ-OAUTH2-065 と REQ-OAUTH2-062 に起票と参照の規則を加え、REQ-OAUTH2-043 に起票者の表示と URL での表示を加える。ゲートウェイには承認の起票と再送の要件を新しい ID で追加し、`affected_spec` に加える。
2. T002 で、汎用の MCP クライアントの立場で、承認が必要な `tools/call` から本人の承認、再送、一回の転送までの受け入れ RED を作る。
3. IdMagic の評価と承認のストア、ゲートウェイ、ポータルの順に実装する。

作るものを変える未解決の問いは、T000 の方式の選択である。

## タスク

- [ ] T000 [Decision] URL モードの elicitation と MCP のステップアップ認可を、実クライアントの対応と安全性で比べ、採否を確定する。
- [ ] T001 [Spec] REQ-OAUTH2-062、065、043、ゲートウェイの新しい要件、TypeSpec、脅威モデル（承認の起票の濫用、`requestState` の改ざん、承認の URL のフィッシング）を更新し、`mise run check-spec` を通す。
- [ ] T002 [Acceptance] MCP SDK のクライアントで、承認の不足から InputRequiredResult、承認、再送、一回の転送までの RED を `mcp_gateway_approval_e2e_test.go` に作る。
- [ ] T003 [Domain] `ApprovalRequest` の起票者の区分と不変条件、`requestState` の封緘と開封を単体テストで RED から実装する。
- [ ] T004 [App] 評価の起票、重複の排除、上限、参照の照合を、メモリと PostgreSQL のアダプターとともに実装する。
- [ ] T005 [App] ゲートウェイの能力の判定、InputRequiredResult、再送の処理と監査を実装する。
- [ ] T006 [UI] ポータルで承認の URL から承認リクエストを開き、起票したリソースサーバーを表示する。
- [ ] T007 [Verify] 引数の変更、期限切れ、拒否、他人のトークン、別の登録からの参照、並行の再送、重複起票、上限、改ざんした `requestState`、能力を宣言しないクライアントを検証する。

## 検証

- `mise run test-go-package -- ./backend/shared/http/server_http`：承認の E2E を、汎用のクライアントの経路へ拡張する。
- `mise run test-go-package -- ./backend/oauth2/...` と `./backend/mcpgateway/...`：評価と承認、ゲートウェイの局所検査。
- 承認ストアの PostgreSQL のアダプターは DB を使うテストで確かめる。
- ポータルの変更は `mise run test-ui-e2e-file` で確かめる。
- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>` とする。

## リスク

- 承認の起票は、承認による許可へつながる新しい経路である。参照できる承認を同じ登録が起票したものに限り、利用者、クライアント、Agent、明細、引数の一致を IdMagic で確かめる。これらの否定の場合をすべて T007 で検証する。
- Agent は `tools/call` を送るだけで本人へメールを送らせられる。重複起票の排除と、一人の利用者当たりの保留中の上限で抑える。
- `fields` は承認画面に表示し、承認ストアに残る。引数にシークレットを入れない前提を維持し、ゲートウェイの監査へは引数を出さない。
- 承認の URL は、MCP クライアントに見せてもよい値（承認リクエストの UUID）だけを含める。URL を他人に開かせても、本人のセッションでなければ判断できない。
