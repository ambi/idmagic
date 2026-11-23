# 操作の認可と人の承認

この文書は、MCP ツールと対象の割当、実行条件、回数制限、発行ガードレール、権限の影響表示、人の承認の設定と確認を説明する。
[ID と認証](ai-agent-demo-identity.md)で Agent を準備し、利用者の権限を使う場合は[本人の委任](ai-agent-demo-delegation.md)も先に行う。
[ガイドの入口](ai-agent-demo.md)に利用パターンの一覧がある。

## 判定の流れ

トークンを持つだけで業務操作が許されるわけではない。
Gateway または業務 API は、操作の直前に IdMagic の `/resource-access/evaluate` へ照会する。
評価用クライアントで認証し、実行者のトークンをフォームの `token` に入れる。
Agent 自身のクライアントで、他のリソースサーバーの判定を代行しない。

| 段階 | 検査するもの |
| --- | --- |
| 発行 | クライアント認証、Agent と所有者の状態、グラント、対象、スコープ、本人の委任、ガードレール |
| 操作の直前 | 評価元と接続先の対応、登録した操作、トークンの有効性と対象、送信者の証明、スコープ、主体と操作者のアクセス権 |
| 追加の制約 | 必要な認可詳細、CEL 条件、人が承認した操作との一致、実行回数の上限 |
| 業務の実行 | 許可された要求だけを上流へ渡す。業務の成功と重複排除は実行側が管理する |

判定はすべてを満たす必要がある。
人が承認しても、操作割当の不足や条件による拒否を上書きしない。
HTTP 200 の応答にも `decision=false` があるため、呼び出す側は `decision=true` を確認してから処理する。

## MCP の操作と対象の割当

画面から試す完成例は[社内文書アシスタントのデモ](ai-agent-demo-walkthrough.md)である。
自分の MCP サーバーでも次の順に設定する。

1. MCP リソースサーバーを登録し、Gateway からカタログを報告するか、公開された MCP サーバーから取得する。
2. 管理画面で公開するツールを選び、操作の表示名を設定する。
3. Agent の「利用する MCP リソースサーバー」で必要な操作だけを割り当てる。
4. 対象を絞る操作では「許可する対象をエージェントごとに限定する」を有効にし、対象 ID を読む引数を設定する。
5. Agent 側で「指定した対象だけ」を選び、許可する対象 ID を入力して保存する。

文書の閲覧なら、対象 ID を読む引数を `document_id`、許可する対象を `handbook` とする。
対象別の制約を有効にした直後は対象の許可がないため、対象を追加するまで閲覧できない。
次の組み合わせを同じトークンで試す。

| 操作と対象 | 割当 | 期待結果 |
| --- | --- | --- |
| `read_document`、`handbook` | 閲覧と対象の許可あり | 実行される |
| `read_document`、`salary-2026` | 閲覧のみ | 対象の許可がないため拒否される |
| `delete_document`、`handbook` | 削除の割当なし | 閲覧と同じスコープでも拒否される |
| `read_document`、`handbook` | 閲覧を取り消した後 | 既発行トークンでも次の呼び出しから拒否される |

公開、割当、対象の許可は別の設定である。
カタログに載っただけでは公開も割当も行われない。
`tools/list` は利用者ごとの実効許可一覧ではなく、ツールを見られることだけで実行できるとは判断しない。
従来のモデル、型、関係による ReBAC を使う場合も、ツールの操作割当と対象別の関係の両方が必要になる。

## 条件と実行回数

### CEL による条件

MCP リソースサーバーの評価規則に `condition` を設定する。
たとえば少額は自動、高額は承認、上限超過は拒否にする場合、規則の条件部分を次のようにする。
`approval_detail_type` の `expense` は、先に登録した認可詳細の型名に置き換える。

```json
{
  "condition": {
    "deny_when": "attributes.amount > 100000.0",
    "require_approval_when": "attributes.amount > 10000.0",
    "approval_detail_type": "expense"
  }
}
```

これは評価規則の一部であり、この JSON だけでは接続先や操作は作成されない。
登録画面または管理 API で、操作の型、関係、必要スコープとともに保存する。
`requires_approved_operation` による常時承認と `require_approval_when` は併用しない。

| 入力 | 期待結果 |
| --- | --- |
| `amount=5000` | 他の条件を満たせば承認なしで許可 |
| `amount=20000` | 承認が必要 |
| `amount=200000` | 拒否。承認しても許可されない |
| `amount` がない、または数値でない | 評価不能として拒否 |

CEL は検証済みの `token`、IdMagic が決める `now`、要求の `operation`、実行側が申告する `attributes` を読む。
数値は `double` のため、例の比較値には `.0` を付けている。
IdMagic は `attributes.amount` が業務上の本当の金額であることまでは保証しない。
実行側で金額を確定し、判定した値と同じ値で業務処理を行う。
セルフホスト型 Gateway ではツール対応の `attribute_arguments` に送信する引数を指定し、指定しない引数を式で読めるものとしない。
マネージド型 Gateway は MCP の `params.arguments` を属性として評価する。

### 実行回数の制限

評価規則に、たとえば次の `limits` を追加する。
これは Agent ごとに 1 分 10 回と、規則全体で 1 日 1000 回の二つの上限である。

```json
{
  "limits": [
    {"by": ["agent"], "max_requests": 10, "window_seconds": 60},
    {"max_requests": 1000, "window_seconds": 86400}
  ]
}
```

`by` は `agent`、`user`、`resource` またはそれらの組み合わせで、省略すると規則全体の上限になる。
該当する Agent や利用者がない評価では、その単位の制限は適用されないため、利用者だけの呼び出しも抑えるなら全体の上限などを併せる。
`resource` は評価要求の `resource_id` の単位であり、MCP の設定によっては文書ではなくツール名になる。
文書ごとの制限を意図する場合は、対応付けた値を確認する。

数えるのは許可した評価であり、業務の成功回数ではない。
許可後の通信失敗や同じ要求の再送も回数を使い、上限を超えると `limit_exceeded` で拒否される。
規則の `max_requests` の変更は次の評価から効き、`by` や窓の長さを変えると新しい回数で数える。

## 発行ガードレール

発行ガードレールは、Agent が取得できるトークンを制限する。
実行回数の制限や、既発行トークンの停止には使わない。
テナントのデフォルトと Agent 個別の二層をそれぞれ検査し、個別設定でデフォルトの拒否を緩めることはできない。

1. Agent の詳細からガードレールの編集を開く。
2. 許可するリソースと認可詳細の型、発行回数、期限、委任の深さ、トークン有効期間など必要な上限を設定する。
3. 超過時に拒否する項目と、所有者の承認を求める項目を選んで保存する。
4. 範囲内と範囲外の発行を試し、応答の有効期間と `AgentGuardrailEvaluated` を確認する。

API ではまず `GET /api/admin/v1/agents/{agent_id}/guardrail-policy` で二層を読み、同じパスへ PUT する。
たとえば初めて個別ポリシーを作る要求は次の形になる。
更新時には `expected_version` に読み取った `agent.version` を使う。

```json
{
  "expected_version": "",
  "allowed_resources": ["https://mcp.example.com/"],
  "issuance_rate": {"max_issuances": 10, "window_seconds": 60},
  "max_token_ttl_seconds": 300,
  "max_delegation_depth": 1
}
```

テナントのデフォルトは `/api/admin/v1/agent-guardrail-default` で管理する。
デフォルト用の設定画面はないため、この API を使う。
未設定の項目は制限せず、許可リストの空配列は何も許さない。
`max_token_ttl_seconds` は有効期間を短縮し、ほかの上限は拒否または承認を求める。
`on_violation` の省略時は拒否であり、委任の深さの超過は承認で上書きできない。
バージョン競合の 409 では再取得してから変更を適用する。

超過を `require_approval` にした項目を超えられるのは、Agent の現在の有効な所有者が承認した CIBA 発行だけである。
別の利用者の承認や、`deny` にした項目の超過は許されない。
停止したいときはポリシーを書き換えるだけで済ませず、[Agent の無効化](ai-agent-demo-operations.md)を行う。

## 権限の影響の説明

同意、Agent への本人の付与、承認の画面には、権限が何を変更し、個人情報を扱うかを判断前に表示する。
管理者は MCP リソースサーバーのスコープに `scope_hints`、認可詳細の型と公開ツールに `impact` を登録する。
影響の値は次の形である。

```json
{
  "changes": "irreversible",
  "personal_data": true
}
```

| 値 | 表示の意味 |
| --- | --- |
| `changes=none` | 変更なし |
| `changes=reversible` | 元に戻せる変更 |
| `changes=irreversible` | 元に戻せない変更 |
| `personal_data=true` / `false` | 個人情報を扱う / 扱わないという登録者の申告 |
| 未登録、未知の値、取得不能 | 影響と個人情報の扱いが不明。元に戻せない変更と同じ注意を促す |

`personal_data` は省略して「扱わない」と推定させず、登録時に明示する。
同名スコープが複数の有効な接続先にある場合、対象指定がなければ最も強い変更の影響と個人情報の論理和を表示する。
一つでも未登録なら不明として扱い、認可要求に `resource` があればその登録に絞る。
操作承認は起票時のツール名と操作 ID に対応する影響を参照する。

表示は登録者の申告であり、IdMagic が業務の事実を保証するものではない。
影響の値だけでは許可、拒否、認証水準、発行する権限は変わらないため、禁止と承認条件は前の節で設定する。
登録と変更は `PermissionImpactChanged`、判断前の画面は `permission-impact.spec.ts` で確認できる。

## 発行前の人の承認

CIBA は、ブラウザーを操作していない Agent などが本人に承認を求め、承認後にトークンを取得するフローである。
「監督下」Agent は通常の発行を拒否し、この承認を経た発行に限定する。
自律 Agent も必要な場面で CIBA を使える。

1. 認証用クライアントで CIBA のグラントと `openid` を含む必要なスコープを有効にする。Agent の登録画面では「人の承認を依頼できるようにする」を選ぶ。
2. `<issuer>/bc-authorize` にクライアント認証を付け、`login_hint` または `id_token_hint` のどちらか一つ、`openid` を含む `scope`、本人が依頼を識別できる `binding_message` をフォームで送る。明細が必要なら `authorization_details` の JSON 文字列も送る。
3. 応答の `auth_req_id`、`expires_in`、`interval` を保持する。
4. 本人がアカウント画面の承認依頼を開き、必要なステップアップ認証の後に承認または拒否する。
5. クライアントは `interval` を守り、トークンエンドポイントへ `grant_type=urn:openid:params:grant-type:ciba` と `auth_req_id` を送り、クライアント認証してポーリングする。

`authorization_pending` は待機を続け、`slow_down` では間隔を延ばす。
拒否や期限切れでは処理を終え、成功後は同じ `auth_req_id` を再利用しない。
監督下 Agent に発行するトークンには refresh token がなく、Token Exchange で承認を引き継ぐこともできない。

画面で試す場合は[デモの緊急時の手順](ai-agent-demo-walkthrough.md#4-緊急時)を使う。
汎用 CIBA クライアントの対話例は、`mise run dev-memory` を起動した別の端末で `mise run demo-ciba` を実行できる。
後者はローカルの共有シークレットを持つデモ用クライアントであり、Agent に紐付ける公開鍵認証の例とは異なる。
いずれも、トークン発行の確認だけで業務操作一回の実行を検証したことにはならない。
承認対象のリソース向けに発行する場合は、`authorization_details.locations` に登録済みの対象を含める。
契約の詳細は [CIBA と承認](../modules/oauth2/features/approval.md)を参照する。

## 一回の操作の承認

削除や送金のように、承認した対象と引数の組を一回だけ実行したい場合に使う。
認可詳細の型を登録し、規則に `authorization_detail_type` と `requires_approved_operation=true` を設定するか、CEL の承認条件と `approval_detail_type` を設定する。
RAR の明細は型に登録したスキーマを通し、`locations` は接続先、`actions` は関係、`identifier` は対象、`fields` は実行する引数と一致させる。
ここでいう対象は評価の `resource_id` であり、MCP カタログ方式では文書 ID とは限らない。
明細を手で推測せず、評価応答の認可修復情報を使うと、対象と操作の対応を保てる。

認可詳細の型は管理画面、または `/api/admin/v1/authorization-detail-types` で登録する。
スキーマは受け付けるフィールドごとに、集合を狭める `set`、数値の上限を下げる `at_most`、許可値を選ぶ `enum`、値を固定する `exact` の意味を指定する。
未登録のフィールドや型の不一致は拒否される。
たとえば先の `expense` では `locations` と `actions` を `set`、`identifier` と通貨を `exact`、金額を `at_most` として、表示テンプレートに金額と対象を含める。
操作の実行時には、金額の上限内であるだけでなく、本人が承認した実際の引数と一致することも検査する。

### 利用者のトークンから起票する手順

1. Gateway または PEP が、実行する操作に新しい `operation_key` を作り、`operation_parameters` に引数の JSON 文字列を入れて評価する。
2. 利用者を主体とするトークンで `approval_start=true` を添えると、承認が必要な場合に `decision=false`、`reason=approval_pending` と `approval_request` が返る。
3. クライアントは応答の `approval_uri` を本人へ示し、本人はログインとステップアップ認証を経て、対象と引数を確認して承認する。
4. 元のトークン、同じ対象と引数、同じ `operation_key` に `approval_request_id` を添えて再評価する。
5. `decision=true` の場合だけ一回実行し、返された承認 ID と `operation_digest` を実行記録に結び付ける。

この経路ではトークンを取り直す必要はない。
承認依頼は 600 秒で期限切れになり、同じ利用者、クライアント、Agent、接続先の要求でだけ利用できる。
主体が利用者ではない M2M トークンからは承認者を自動選択せず、`approval_required` のままになる。

セルフホスト型 Gateway の `proxy` モードでは、URL の elicitation に対応した MCP クライアントからこの流れを利用できる。
クライアントは `elicitation.url` の能力を宣言し、返された `input_required` の URL で本人に判断してもらい、同じ引数と `requestState`、`inputResponses.idmagic_approval.action=accept` を返して再開する。
Gateway は状態と引数を検証してから再評価し、Gateway 用の状態を上流へ渡さない。
この能力を宣言しないクライアントや Envoy の `ext_authz` では、この起票を自動で行わず、403 の認可修復情報を使う経路になる。

### CIBA の明細を使う手順

Agent から本人を指定して依頼する場合は、評価応答の `authorization_remediation.authorization_details` を使い、同じ操作の明細を CIBA で本人に承認してもらう。
承認後に取得したトークンを、その操作の評価に提示する。
引数を変えると `approved_operation_mismatch`、別の実行への再利用は `approval_already_used` で拒否される。
単なるスコープへの同意や、明細の一致しない CIBA トークンでは、この操作は許されない。

MCP Gateway は許可された要求の転送を最大一回に限定し、タイムアウト時に自動再送しない。
独自 PEP が同じ `operation_key` の再送を受け入れる場合は、業務結果を保存して重複実行を防ぐ。
IdMagic の許可の消費だけでは、業務の成功やちょうど一回の完了は保証されない。

## 自動の確認

| 対象 | コマンド |
| --- | --- |
| 同じスコープの操作を個別に割り当てる | `mise run test-ui-e2e-file -- tests/e2e/agent-demo-operations.spec.ts` |
| 同意、委任、承認の前に影響を表示する | `mise run test-ui-e2e-file -- tests/e2e/permission-impact.spec.ts` |
| 発行ガードレール | `mise run test-go-one -- ./backend/shared/http/server_http TestAgentGuardrailLimitsIssuanceThroughToken` |
| 利用者の操作から承認を起票し、実行する | `mise run test-go-one -- ./backend/shared/http/server_http TestMcpGatewayLeadsAGenericClientThroughHumanApproval` |
| 承認による転送を一回に限る | `mise run test-go-one -- ./backend/shared/http/server_http TestMcpGatewayForwardsAnApprovalAtMostOnce` |

これらの統合テストはテスト用の鍵、通知先、MCP サーバーを使う。
実際の業務の完了を確認する際は、上流の更新結果と重複排除の記録まで照合する。
契約の詳細は[リソースアクセス](../modules/oauth2/features/resource-access.md)、[ガードレール](../modules/oauth2/features/guardrail.md)、[同意と影響表示](../modules/oauth2/features/consent.md)を参照する。
