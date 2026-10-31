# リソースアクセスの評価

## 概要

この文書は、登録したリソースサーバーが、提示されたアクセストークンで自身の資源への操作を実行してよいかを、業務処理の直前に評価する仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | リソースサーバーの認証と登録の解決、規則の照合、トークンの検証、送信者制約の確認、関係の判定との合成 |
| 行為者 | 登録したリソースサーバー |
| 扱わないもの | 関係の判定は [AccessControl](../../authorization/features/check.md) が、Agent の発行の上限とポリシーは[ガードレール](guardrail.md)が、業務処理の成功と原子性はリソースサーバーが扱う |

## モデル

許可は次の論理積である。

1. 呼び出し元のクライアントを認証でき、そのクライアントに束縛した `Active` の `McpResourceServer` がある。
2. 要求した `resource_type` と `relation` の組が、登録の `access_rules` にある。
3. 提示されたトークンが有効で、失効しておらず、要求先のテナントのものであり、`aud` が登録の `resource` を含む。
4. 送信者制約付きのトークンなら、提示の証明が一致する。
5. 規則が `authorization_detail_type` を指定するなら、トークンの `authorization_details` に一致するものがある。
6. トークンから導いた主体と行為者について、AccessControl が規則の `scopes` を含めて許可する。
7. 規則が `condition` を持つなら、[条件の評価](#条件の評価)が拒否せず、評価でき、承認を求めるなら 8 へ進む。
8. 規則が `requires_approved_operation` を持つか、条件が承認を求めるなら、[承認した操作の評価](#承認した操作の評価)が許可する。
9. 規則が `limits` を持つなら、[実行回数の上限](#実行回数の上限)を超えない。

主体と行為者は、提示されたトークンだけから導く。

| トークン | 主体 | 行為者 |
| --- | --- | --- |
| `principal_type` が `agent` | `agent:<agent_id>` | `agent_chain` のうち主体を除く Agent |
| サブジェクトが登録されたクライアントでない | `user:<sub>` | `agent_id` と `agent_chain` の Agent |
| 上のどちらでもない | 評価しない | 評価しない |

## 操作

### リソースサーバーによる操作の評価

#### REQ-OAUTH2-062 登録したリソースサーバーは自身の規則の操作だけを提示トークンで評価できる

- 束縛した `Active` の登録を持つクライアントが、モデルの論理積を満たす操作を評価したとき、OAuth2 は、200 と `Cache-Control: no-store`、`decision=true`、スコープ、`delegation_mode`、`authorization_details` を返し、`ResourceAccessEvaluated` を発行する。
- 要求の本文が `subject`、`actor`、`tenant_id` を含むとき、OAuth2 は、その値を判定に使わない。
- 提示されたトークンが無効、失効、期限切れ、別のテナントのものであるか、`aud` が登録の `resource` を含まない場合、OAuth2 は、200 と `decision=false`、理由 `token_rejected` だけを返し、関係の判定を要求しない。
- 組が `access_rules` にない場合、OAuth2 は、200 と `decision=false`、理由 `operation_not_registered` を返す。
- 規則の `authorization_detail_type` と同じ `type` で、`locations` が登録の `resource` を含み、`actions` が `relation` を含み、`identifier` が空か `resource_id` と等しい詳細がトークンにない場合、OAuth2 は、200 と `decision=false`、理由 `authorization_detail_mismatch` を返す。
- 主体か行為者をトークンから導けない場合、OAuth2 は、200 と `decision=false`、理由 `principal_unresolved` を返す。
- AccessControl が許可しない場合、OAuth2 は、200 と `decision=false`、その理由を返す。
- 不許可を返すとき、OAuth2 は、トークンのクレームを返さない。
- クライアントを認証できない場合、OAuth2 は、401 と `invalid_client` で拒否する。
- 認証したクライアントに束縛した `Active` の登録がない場合、OAuth2 は、403 と `access_denied` で拒否する。
- 組が `access_rules` にない評価を受けたとき、OAuth2 は、トークンと提示の証明を確かめ、拒否の理由を `operation_not_registered` のままにする。
- トークンが有効で提示の証明が成り立ち、主体を導けた評価を記録するとき、OAuth2 は、`ResourceAccessEvaluated` に、利用者である主体の `userId`、主体か先頭の行為者である Agent の `agentId`、行為者の Agent の `actorChain` を含める。
- 評価を記録するとき、OAuth2 は、`ResourceAccessEvaluated` に応答の `X-Request-ID` と同じ `requestId` を含める。
- トークンが無効か提示の証明が成り立たない評価を記録するとき、OAuth2 は、`ResourceAccessEvaluated` に主体と Agent を含めない。
- 規則が `condition` を持つ操作で、ほかの論理積を満たし、`deny_when` と `require_approval_when` のどちらも偽であるとき、OAuth2 は、人の承認を求めずに許可する。
- `deny_when` が真である場合、OAuth2 は、`require_approval_when` の値によらず、200 と `decision=false`、理由 `condition_denied` を返す。
- `deny_when` が偽で `require_approval_when` が真である場合、OAuth2 は、REQ-OAUTH2-065 の判定へ進む。
- 式の参照する属性がないか、属性の型が演算に合わないか、`attributes` が JSON のオブジェクトでないか、キーが重複するか、絶対値が 2 の 53 乗以上の数か有効桁が 15 桁を超える数を含むか、実行の費用が上限を超えるか、保存済みの式をコンパイルできない場合、OAuth2 は、200 と `decision=false`、理由 `condition_indeterminate` を返す。
- 規則が `condition` を持たないとき、OAuth2 は、`attributes` を読まない。
- `attributes` が 16 KiB を超える場合、OAuth2 は、400 と `invalid_request` で拒否する。
- トークンの拒否、送信者の証明の不正、明細の不一致、主体の未解決、関係の不許可で不許可を返すとき、OAuth2 は、条件を評価しない。
- 条件を評価した評価を記録するとき、OAuth2 は、`ResourceAccessEvaluated` に `conditionVersion`、`conditionOutcome`、`declaredAttributesDigest` を含め、属性の平文を含めない。
- 管理者が条件を差し替えたとき、OAuth2 は、次の評価から新しい条件を適用し、その版を記録する。
- REQ-OAUTH2-065 の操作でないものを許可したとき、OAuth2 は、応答と `ResourceAccessEvaluated` に承認の相関を含めない。
- `token`、`resource_type`、`resource_id`、`relation` のどれかがない場合、OAuth2 は、400 と `invalid_request` で拒否する。
- 規則が `limits` を持つ操作で、ほかの論理積を満たす評価を受けたとき、OAuth2 は、適用する上限ごとに固定窓の回数を数える。
- 適用するどの上限の回数も `max_requests` 以内である評価を受けたとき、OAuth2 は、上限を理由に拒否しない。
- 同じ窓の回数が `max_requests` を超えた場合、OAuth2 は、200 と `decision=false`、理由 `limit_exceeded` を返し、超えた上限と再試行の時刻を応答に含めない。
- トークンの拒否、送信者の証明の不正、規則にない操作、明細の不一致、主体の未解決、関係の不許可、条件の拒否か評価不能、承認の不足か不一致で不許可を返すとき、OAuth2 は、上限の回数を数えない。
- 規則が `limits` を持たないとき、OAuth2 は、上限の回数を読み書きしない。
- 上限の `by` の単位の値を評価から導けない場合、OAuth2 は、その上限を適用しない。
- 回数を超えて不許可を記録するとき、OAuth2 は、`ResourceAccessEvaluated` に超えた上限の `exceededLimitBy` と `exceededLimitWindowSeconds` を含める。
- 上限の回数を読み書きできないか、レートリミッターが配線されていない場合、OAuth2 は、503 と `temporarily_unavailable` を返し、許可しない。
- 管理者が `max_requests` を変えたとき、OAuth2 は、次の評価から新しい値で今の窓の回数を判定する。
- 管理者が `by` か `window_seconds` を変えたとき、OAuth2 は、その上限を新しい回数から数える。
- クライアントの認証と入力の検査を通る評価を受けたとき、OAuth2 は、テナントと認証したクライアント ID の組ごとに、`resource_access` ポリシーの固定窓の回数を数え、上限までは評価を続ける。
- 同じ窓の上限を超えた場合、OAuth2 は、評価せずに 429 と `rate_limited`、`Cache-Control: no-store`、窓の残り秒数を切り上げた同じ値の `Retry-After` と `retry_after_seconds` を返す。
- 要求元の IP、提示トークン、資源、操作が変わっても認証したクライアントが同じ場合、OAuth2 は、同じ予算を使い、許可と不許可の両方を数える。
- 制限の設定が省略されているとき、OAuth2 は、一クライアント当たり 1 秒の固定窓で 100 回を上限にし、運用者が API プロセスの `RATE_LIMIT_RESOURCE_ACCESS_MAX_REQUESTS` と `RATE_LIMIT_RESOURCE_ACCESS_WINDOW_SECONDS` で正の整数へ変更できるようにする。
- 共有カウンターを読み書きできない場合、OAuth2 は、評価せずに 429 と `rate_limited`、`Cache-Control: no-store`、30 秒の `Retry-After` と `retry_after_seconds` を返す。
- レートリミッターが配線されていない場合、OAuth2 は、503 と `temporarily_unavailable` を返し、評価しない。
- 流量制限で評価しない場合、OAuth2 は、`ResourceAccessEvaluated` を発行せず、DPoP の証明と承認の使用権を消費しない。
- トークンの状態、失効、関係のストア、提示の証明のリプレイ記録を読めない場合、OAuth2 は、503 と `temporarily_unavailable` を返し、許可しない。
- **判断**：トークンの拒否の理由を語り分けない。リソースサーバーは拒否の種類で動作を変える必要がなく、語り分けると別の資源宛てのトークンの有効性を知らせることになる。
- **判断**：応答を再利用させない。停止の保証を「次の評価から不許可」とするためであり、評価の後に確定した業務操作は取り消さない。
- **判断**：主体を応答ではなく監査イベントへ残す。不許可の試行も記録でき、リソースサーバーへ渡す情報を増やさない。提示の証明が成り立たないトークンの持ち主を記録すると、盗んだトークンの試行が本人の操作に見えるので記録しない。
- **判断**：上限の超過を 429 にしない。429 はリソースサーバー自身の過負荷を表し、超過は操作を許さないという判定なので、他の不許可と同じ形で返す。
- **判断**：超えた上限を応答で語らない。Agent が応答から上限を探れないようにし、管理者には監査で示す。

#### REQ-OAUTH2-063 送信者制約付きのトークンの評価は IdMagic が提示の証明を確かめる

- DPoP に束縛したトークンの評価で、`dpop_proof` の鍵が束縛と一致し、`htm`、`htu`、`ath`、`iat` が正しく、`htu` が登録の `resource` の配下であるとき、OAuth2 は、証明の `jti` を記録して評価を続ける。
- mTLS に束縛したトークンの評価で、`client_certificate_thumbprint` が束縛と一致するとき、OAuth2 は、評価を続ける。
- 証明か拇印がない、一致しない、または再利用された場合、OAuth2 は、200 と `decision=false`、理由 `sender_proof_invalid` を返す。
- 送信者制約のないトークンの評価で証明を受けたとき、OAuth2 は、証明を検証せず、`jti` を記録しない。
- **判断**：検証を IdMagic に置く。リソースサーバーの実装漏れが許可へ倒れないようにするためである。mTLS の接続はリソースサーバーが終端するので、拇印は認証済みのリソースサーバーの申告として受ける。

#### REQ-OAUTH2-079 評価は規則が宣言した宛先への一回限りの交換の許可を発行できる

- `token_exchange_resource` を持つ評価が REQ-OAUTH2-062 の論理積を満たし、値が規則の `token_exchange_resources` にあるとき、OAuth2 は、`decision=true` と、有効期間が 60 秒で一回だけ使える `token_exchange_grant` を返す。
- `token_exchange_resource` がない評価を許可したとき、OAuth2 は、`token_exchange_grant` を返さない。
- `token_exchange_resource` が規則の `token_exchange_resources` にないか、人の承認を求める操作の場合、OAuth2 は、ほかの論理積を満たす評価に 200 と `decision=false`、理由 `token_exchange_not_permitted` を返し、承認の使用権を記録しない。
- ほかの理由で不許可を返すとき、OAuth2 は、`token_exchange_grant` を返さない。
- 許可を保存できない場合、OAuth2 は、503 と `temporarily_unavailable` を返し、許可しない。
- `token_exchange_grant` を伴う Token Exchange を受けたとき、OAuth2 は、許可を発行した評価のクライアント、`subject_token`、宛先のすべてが要求と一致し、期限内で未使用であることを確かめる。
- 許可を確かめられないか、使用済みか、期限切れか、一致しない場合、OAuth2 は、400 と `invalid_grant` で拒否し、トークンを発行せず、`TokenExchangeRejected` を発行する。
- 別のクライアント、`subject_token`、宛先の要求を拒否するとき、OAuth2 は、許可を消費しない。
- 同じ許可の Token Exchange が並行した場合、OAuth2 は、ちょうど一方だけにトークンを発行する。
- `token_exchange_grant` を `actor_token` か外部ワークロードの `subject_token` とともに受けた場合、OAuth2 は、400 と `invalid_request` で拒否する。
- 許可を伴う交換で発行したとき、OAuth2 は、`TokenExchanged` の `senderConstraintRebound` を真にする。
- 主体が Agent の `subject_token` を許可とともに交換したとき、OAuth2 は、発行するトークンの主体を同じ Agent のままにする。
- 主体が Agent で、`act` が `Active` の登録に束縛したクライアントだけであるトークンを評価するとき、OAuth2 は、そのクライアントを行為者に数えず、主体を導く。
- `subject_token` の Agent か利用者を失効させた後の評価と交換では、OAuth2 は、許可を発行せず、発行済みの許可でも交換を拒否する。
- **判断**：許可を評価に載せる。評価は元の鍵の証明を確かめる唯一の場所であり、登録だけを根拠にすると盗んだトークンをゲートウェイへ持ち込める。
- **判断**：宛先を規則で宣言する。トークンの権限の範囲内でも、操作ごとにどの資源へ渡してよいかは管理者の判断である。
- **判断**：承認の操作へ許可を出さない。交換の派生は承認を引き継がないので、下流で承認の照合が成り立たない。
- **判断**：有効期間を短くする。許可は呼び出しごとに得るので、再利用する理由がなく、ゲートウェイの鍵の漏洩の影響を抑える。

### 条件の評価

規則の `condition` は、[CEL](https://cel.dev/) の真偽式で操作ごとの結果を分ける。

| 変数 | 内容 | 出どころ |
| --- | --- | --- |
| `token` | 検証済みのトークンの `sub`、`client_id`、`scope`（文字列のリスト）、`aud`、`iat`、`exp`、`principal_type`、`agent_id`、`agent_chain`、`delegation_mode`。値のないクレームはキーを持たない | IdMagic が検証する |
| `now` | 評価の時刻の `timestamp` | IdMagic が決める |
| `operation` | `resource_type`、`resource_id`、`relation` | リソースサーバーの要求。論理積の他の段と同じ値である |
| `attributes` | 要求の `attributes` の JSON オブジェクト。数は `double` になる | リソースサーバーの申告 |

| `deny_when` | `require_approval_when` | 結果 |
| --- | --- | --- |
| 真 | 問わない | 拒否 |
| 評価不能 | 問わない | 評価不能 |
| 偽か省略 | 真 | 承認要求 |
| 偽か省略 | 評価不能 | 評価不能 |
| 偽か省略 | 偽か省略 | 許可 |

条件の `version` は、`deny_when`、`require_approval_when`、`approval_detail_type` の内容から導く。
同じ内容は同じ版になるので、監査イベントの版を管理の記録と突き合わせられる。

MCP ゲートウェイは `attributes` を送らないので、ゲートウェイの配下の操作では、`attributes` のキーを読む式は評価不能になる。
ツール名は `operation.resource_id` で判定できる。

### 実行回数の上限

規則の `limits` は、その規則の操作を許可する回数を固定窓で抑える。
数えるのは評価で許可した数であり、業務が実行した数ではない。
評価の後に業務が実行しなかった操作も、再送も、一回に数える。

| `by` の単位 | 分ける値 | 値がない評価 |
| --- | --- | --- |
| `agent` | 主体か先頭の行為者である Agent | Agent のいない利用者のトークン |
| `user` | 利用者である主体 | 主体が Agent のトークン |
| `resource` | 要求の `resource_id` | なし |

`by` を省いた上限は、規則の操作の全体で一つの回数を持つ。
流量と総回数は窓の長さの違いであり、たとえば 1 秒の窓で 10 回と 1 日の窓で 1000 回を併せて宣言できる。

上限は宣言の順序によらず、`by` の単位の多いものから、同じなら窓の短いものから順に数え、超えた上限で止める。
超えた要求は、止めるまでに数えた上限の回数を減らす。
狭い単位と短い窓を先に数えるので、暴走した Agent が共有の回数と長い窓の回数を使い切りにくい。

ゲートウェイの `mapping.tools` は、ツールを `resource_type` と `relation` の組へ写すので、一つの組へ写したツールは規則の上限を共有する。
宣言のないツールは同じ組へ集まり、ツール名が `resource_id` に入るので、`by` の `resource` でツールごとに数えられる。
`resource_id_argument` で写した操作の `resource_id` は呼び出す側が選ぶので、`by` に `resource` を含めると値を変えて上限を避けられる。

### 承認した操作の評価

規則の `requires_approved_operation` が真の操作と、規則の `condition` が承認を求めた操作は、人が承認した一回の操作として評価する。
対象は、承認リクエストの `authorization_details` のうち、`type` が規則の `authorization_detail_type`（条件が求めた場合は `approval_detail_type`）、`locations` が登録の `resource` を含み、`actions` が `relation` を含み、`identifier` が `resource_id` と等しい明細である。
引数は明細の `fields` と比べる。
承認の使用権はリソースサーバーと `operation_key` の組が得る。

| 状況 | 結果 |
| --- | --- |
| 未使用の承認への一致する要求 | 許可。使用権を記録する |
| 使用権を持つ組の再送 | 許可。`replayed` が真 |
| 別の `operation_key` | 不許可。`approval_already_used` |

リソースサーバーは、`operation_key` ごとに業務作用を一回に保ち、結果を応答の `approval_request_id` と `operation_digest` とともに記録する。
結果の不明な要求は同じ `operation_key` で再送する。
IdMagic は実行の結果を持たず、この冪等性のないリソースサーバーについて一回性を保証しない。

#### REQ-OAUTH2-065 承認が必要な操作は承認と同じ対象と引数で一度だけ許可する

- 規則が `requires_approved_operation` を持つか条件が承認を求めた操作で、REQ-OAUTH2-062 の論理積を満たし、トークンの承認リクエストの明細が操作と一致し、`operation_parameters` が明細の `fields` と等しく、使用権が未使用か同じ組のものであるとき、OAuth2 は、200 と `decision=true`、`approved_operation` の `approval_request_id`、`operation_digest`、`replayed` を返し、それらを載せた `ResourceAccessEvaluated` を発行する。
- `operation_parameters` と `fields` を比べるとき、OAuth2 は、キーの順序と空白を無視し、文字列を正規化せず、どちらもないものを等しいものとして扱う。
- 明細が一致しないか、引数が等しくない場合、OAuth2 は、200 と `decision=false`、理由 `approved_operation_mismatch` を返し、使用権を記録しない。
- `operation_parameters` が JSON のオブジェクトでないか、キーが重複するか、絶対値が 2 の 53 乗以上の数か有効桁が 15 桁を超える数を含む場合、OAuth2 は、200 と `decision=false`、理由 `approved_operation_mismatch` を返す。
- 規則の明細を満たすトークンが、CIBA の承認から発行されたものでないか、トークン交換で派生したものである場合、OAuth2 は、200 と `decision=false`、理由 `approval_required` を返す。
- 条件が承認を求めた操作で、トークンが CIBA の承認から発行されたものでないか、`approval_detail_type` の一致する明細を持たない場合、OAuth2 は、200 と `decision=false`、理由 `approval_required` を返す。
- 条件が承認を求めなかった操作を許可するとき、OAuth2 は、トークンが持つ承認の使用権を記録しない。
- 承認リクエストの `expires_at` を過ぎたか、承認リクエストが残っていないか、トークンの主体とクライアントが承認リクエストと異なる場合、OAuth2 は、200 と `decision=false`、理由 `approval_expired` を返す。
- 別の `operation_key` で使用権が記録済みの場合、OAuth2 は、200 と `decision=false`、理由 `approval_already_used` を返す。
- 別の `operation_key` の要求が並行した場合、OAuth2 は、ちょうど一方だけを許可する。
- トークンの拒否、送信者の証明の不正、関係の不許可で不許可を返すとき、OAuth2 は、使用権を記録しない。
- 規則の `limits` の超過で不許可を返すとき、OAuth2 は、使用権を記録しない。
- 別の `operation_key` の要求が並行して使用権を得られなかった場合、OAuth2 は、その要求が数えた上限の回数を戻さない。
- `requires_approved_operation` を持つ規則で `operation_key` がないか、`operation_key` が 256 文字を超えるか制御文字を含むか、`operation_parameters` が 64 KiB を超える場合、OAuth2 は、400 と `invalid_request` で拒否する。
- 条件が承認を求め、トークンが一致する明細の承認を持つ操作で `operation_key` がない場合、OAuth2 は、400 と `invalid_request` で拒否する。
- 使用権の記録を読み書きできない場合、OAuth2 は、503 と `temporarily_unavailable` を返し、許可しない。
- **判断**：短い予約と確定を採らない。予約の期限が切れた後の再試行は、結果の不明な一回目と二重に実行され得る。同じ鍵の再送だけを許し、重複の排除をその鍵で資源へ委ねる。
- **判断**：条件の承認の明細の種類を規則の `authorization_detail_type` と分ける。後者は常にトークンへ求めるので、流用すると閾値以下の自動許可が成り立たない。
- **判断**：使用の期限をトークンの期限で延ばさない。人が判断した有効期間を超えて操作を実行させないためである。

## 品質

評価の応答時間は [SLO-RESOURCE-ACCESS-LATENCY](../../../requirements/quality.md#サービス目標) に従う。
処理量の局所測定は [CAP-RESOURCE-ACCESS-THROUGHPUT](../../../requirements/quality.md#リソースアクセス評価の局所測定) に従い、本番のキャパシティ受入れとは区別する。

## セキュリティ上の考慮

`attributes` はリソースサーバーの申告であり、IdMagic は業務の事実を保証しない。
申告を偽るリソースサーバーは自身の資源についての条件を回避できるので、条件はリソースサーバーの実装の誤りと Agent の行動を抑えるものであり、リソースサーバー自身を抑えない。
シークレットの値は `attributes` に入れない。

上限の回数の鍵は利用者と Agent の識別子を含み、PostgreSQL にはハッシュだけを保存する。
応答とログに鍵を出さない。
メモリのアダプターは単一プロセスの中でだけ上限を守るので、複数のプロセスで動かす構成は PostgreSQL を使う。

`fields` は承認の画面に表示し、トークンにも載る。
シークレットの値は `fields` に入れず、必要なら資源と取り決めた鍵付きハッシュを置く。
監査イベントは `operation_digest` だけを持ち、`operation_key` の平文と引数を持たない。

この入口は、管理 API の診断の経路とは別である。
リソースサーバーは、提示されたトークンの主体についてだけ、自身が登録した種類の資源を尋ねられる。
任意の主体の権限を探索する経路はない。

`token_exchange_grant` は送信者制約の保証を弱める値で、ログと監査イベントへ出さず、PostgreSQL にはハッシュだけを保存する。

`Disabled` にした登録、登録の削除、クライアントの削除は、次の評価から効く。

## 実現方式

### アーキテクチャ

OAuth2 が入口と編成を持ち、関係の判定は `accesscontrol/ports.RelationChecker` を通して AccessControl へ委ねる。
失効の判定は `/introspect` と同じ `AccessTokenIsRevoked` を通す。
条件はドメインの計算で、式は評価ごとにコンパイルする。
リソースサーバーとは公開の HTTP 契約だけで結ぶ。参照の実装は `examples/resource-server-pep` にある。
