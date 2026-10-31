# リソースアクセスの評価

## 概要

この文書は、登録したリソースサーバーが、提示されたアクセストークンで自身の資源への操作を実行してよいかを、業務処理の直前に評価する仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | リソースサーバーの認証と登録の解決、規則の照合、トークンの検証、送信者制約の確認、関係の判定との合成 |
| 行為者 | 登録したリソースサーバー |
| 扱わないもの | 関係の判定は [AccessControl](../../authorization/features/check.md) が、Agent の上限とポリシーはガードレールの機能が、業務処理の成功と原子性はリソースサーバーが扱う |

## モデル

許可は次の論理積である。

1. 呼び出し元のクライアントを認証でき、そのクライアントに束縛した `Active` の `McpResourceServer` がある。
2. 要求した `resource_type` と `relation` の組が、登録の `access_rules` にある。
3. 提示されたトークンが有効で、失効しておらず、要求先のテナントのものであり、`aud` が登録の `resource` を含む。
4. 送信者制約付きのトークンなら、提示の証明が一致する。
5. 規則が `authorization_detail_type` を指定するなら、トークンの `authorization_details` に一致するものがある。
6. トークンから導いた主体と行為者について、AccessControl が規則の `scopes` を含めて許可する。
7. 規則が `requires_approved_operation` を持つなら、[承認した操作の評価](#承認した操作の評価)が許可する。

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
- REQ-OAUTH2-065 の操作でないものを許可したとき、OAuth2 は、応答と `ResourceAccessEvaluated` に承認の相関を含めない。
- `token`、`resource_type`、`resource_id`、`relation` のどれかがない場合、OAuth2 は、400 と `invalid_request` で拒否する。
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

#### REQ-OAUTH2-063 送信者制約付きのトークンの評価は IdMagic が提示の証明を確かめる

- DPoP に束縛したトークンの評価で、`dpop_proof` の鍵が束縛と一致し、`htm`、`htu`、`ath`、`iat` が正しく、`htu` が登録の `resource` の配下であるとき、OAuth2 は、証明の `jti` を記録して評価を続ける。
- mTLS に束縛したトークンの評価で、`client_certificate_thumbprint` が束縛と一致するとき、OAuth2 は、評価を続ける。
- 証明か拇印がない、一致しない、または再利用された場合、OAuth2 は、200 と `decision=false`、理由 `sender_proof_invalid` を返す。
- 送信者制約のないトークンの評価で証明を受けたとき、OAuth2 は、証明を検証せず、`jti` を記録しない。
- **判断**：検証を IdMagic に置く。リソースサーバーの実装漏れが許可へ倒れないようにするためである。mTLS の接続はリソースサーバーが終端するので、拇印は認証済みのリソースサーバーの申告として受ける。

### 承認した操作の評価

規則の `requires_approved_operation` が真の操作は、人が承認した一回の操作として評価する。
対象は、承認リクエストの `authorization_details` のうち、`type` が規則の `authorization_detail_type`、`locations` が登録の `resource` を含み、`actions` が `relation` を含み、`identifier` が `resource_id` と等しい明細である。
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

- 規則が `requires_approved_operation` を持つ操作で、REQ-OAUTH2-062 の論理積を満たし、トークンの承認リクエストの明細が操作と一致し、`operation_parameters` が明細の `fields` と等しく、使用権が未使用か同じ組のものであるとき、OAuth2 は、200 と `decision=true`、`approved_operation` の `approval_request_id`、`operation_digest`、`replayed` を返し、それらを載せた `ResourceAccessEvaluated` を発行する。
- `operation_parameters` と `fields` を比べるとき、OAuth2 は、キーの順序と空白を無視し、文字列を正規化せず、どちらもないものを等しいものとして扱う。
- 明細が一致しないか、引数が等しくない場合、OAuth2 は、200 と `decision=false`、理由 `approved_operation_mismatch` を返し、使用権を記録しない。
- `operation_parameters` が JSON のオブジェクトでないか、キーが重複するか、絶対値が 2 の 53 乗以上の数か有効桁が 15 桁を超える数を含む場合、OAuth2 は、200 と `decision=false`、理由 `approved_operation_mismatch` を返す。
- 規則の明細を満たすトークンが、CIBA の承認から発行されたものでないか、トークン交換で派生したものである場合、OAuth2 は、200 と `decision=false`、理由 `approval_required` を返す。
- 承認リクエストの `expires_at` を過ぎたか、承認リクエストが残っていないか、トークンの主体とクライアントが承認リクエストと異なる場合、OAuth2 は、200 と `decision=false`、理由 `approval_expired` を返す。
- 別の `operation_key` で使用権が記録済みの場合、OAuth2 は、200 と `decision=false`、理由 `approval_already_used` を返す。
- 別の `operation_key` の要求が並行した場合、OAuth2 は、ちょうど一方だけを許可する。
- トークンの拒否、送信者の証明の不正、関係の不許可で不許可を返すとき、OAuth2 は、使用権を記録しない。
- `operation_key` がないか 256 文字を超えるか制御文字を含むか、`operation_parameters` が 64 KiB を超える場合、OAuth2 は、400 と `invalid_request` で拒否する。
- 使用権の記録を読み書きできない場合、OAuth2 は、503 と `temporarily_unavailable` を返し、許可しない。
- **判断**：短い予約と確定を採らない。予約の期限が切れた後の再試行は、結果の不明な一回目と二重に実行され得る。同じ鍵の再送だけを許し、重複の排除をその鍵で資源へ委ねる。
- **判断**：使用の期限をトークンの期限で延ばさない。人が判断した有効期間を超えて操作を実行させないためである。

## 品質

評価の応答時間は [SLO-RESOURCE-ACCESS-LATENCY](../../../requirements/quality.md#サービス目標) に従う。
処理量の局所測定は [CAP-RESOURCE-ACCESS-THROUGHPUT](../../../requirements/quality.md#リソースアクセス評価の局所測定) に従い、本番のキャパシティ受入れとは区別する。

## セキュリティ上の考慮

`fields` は承認の画面に表示し、トークンにも載る。
シークレットの値は `fields` に入れず、必要なら資源と取り決めた鍵付きハッシュを置く。
監査イベントは `operation_digest` だけを持ち、`operation_key` の平文と引数を持たない。

この入口は、管理 API の診断の経路とは別である。
リソースサーバーは、提示されたトークンの主体についてだけ、自身が登録した種類の資源を尋ねられる。
任意の主体の権限を探索する経路はない。

`Disabled` にした登録、登録の削除、クライアントの削除は、次の評価から効く。

## 実現方式

### アーキテクチャ

OAuth2 が入口と編成を持ち、関係の判定は `accesscontrol/ports.RelationChecker` を通して AccessControl へ委ねる。
失効の判定は `/introspect` と同じ `AccessTokenIsRevoked` を通す。
リソースサーバーとは公開の HTTP 契約だけで結ぶ。参照の実装は `examples/resource-server-pep` にある。
