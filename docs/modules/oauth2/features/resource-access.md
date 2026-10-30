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
- `token`、`resource_type`、`resource_id`、`relation` のどれかがない場合、OAuth2 は、400 と `invalid_request` で拒否する。
- トークンの状態、失効、関係のストア、提示の証明のリプレイ記録を読めない場合、OAuth2 は、503 と `temporarily_unavailable` を返し、許可しない。
- **判断**：トークンの拒否の理由を語り分けない。リソースサーバーは拒否の種類で動作を変える必要がなく、語り分けると別の資源宛てのトークンの有効性を知らせることになる。
- **判断**：応答を再利用させない。停止の保証を「次の評価から不許可」とするためであり、評価の後に確定した業務操作は取り消さない。

#### REQ-OAUTH2-063 送信者制約付きのトークンの評価は IdMagic が提示の証明を確かめる

- DPoP に束縛したトークンの評価で、`dpop_proof` の鍵が束縛と一致し、`htm`、`htu`、`ath`、`iat` が正しく、`htu` が登録の `resource` の配下であるとき、OAuth2 は、証明の `jti` を記録して評価を続ける。
- mTLS に束縛したトークンの評価で、`client_certificate_thumbprint` が束縛と一致するとき、OAuth2 は、評価を続ける。
- 証明か拇印がない、一致しない、または再利用された場合、OAuth2 は、200 と `decision=false`、理由 `sender_proof_invalid` を返す。
- 送信者制約のないトークンの評価で証明を受けたとき、OAuth2 は、証明を検証せず、`jti` を記録しない。
- **判断**：検証を IdMagic に置く。リソースサーバーの実装漏れが許可へ倒れないようにするためである。mTLS の接続はリソースサーバーが終端するので、拇印は認証済みのリソースサーバーの申告として受ける。

## セキュリティ上の考慮

この入口は、管理 API の診断の経路とは別である。
リソースサーバーは、提示されたトークンの主体についてだけ、自身が登録した種類の資源を尋ねられる。
任意の主体の権限を探索する経路はない。

`Disabled` にした登録、登録の削除、クライアントの削除は、次の評価から効く。

## 実現方式

### アーキテクチャ

OAuth2 が入口と編成を持ち、関係の判定は `accesscontrol/ports.RelationChecker` を通して AccessControl へ委ねる。
失効の判定は `/introspect` と同じ `AccessTokenIsRevoked` を通す。
リソースサーバーとは公開の HTTP 契約だけで結ぶ。参照の実装は `examples/resource-server-pep` にある。
