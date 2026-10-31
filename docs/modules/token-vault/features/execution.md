# 代理実行

## 概要

この文書は、Agent が利用者の接続を使って外部 API を呼ぶ仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | 呼び出しの認可、外向きの要求の制限、トークンの更新と再認可への遷移 |
| 行為者 | 代行トークンを持つ Agent |
| 扱わないもの | トークンの検証と失効の判定は [OAuth2](../../oauth2/features/token.md) が、呼び出しごとの人の承認は対象としない |

## モデル

許可は次の論理積である。呼び出しのたびに評価し、結果を再利用しない。

1. トークンが有効で失効しておらず、`aud` が Vault のリソース識別子を含み、DPoP に束縛されていれば証明が正しい。
2. トークンが Token Exchange で発行され（`act` を持ち）、`principal_type` が `agent` でなく、`agent_id` か `agent_chain` に Agent がある。
3. トークンの `authorization_details` に、`type` が `token_vault_connection`、`identifier` がコネクタの ID、`actions` が要求のメソッド、`locations` が要求のオリジンを含むものがある。
4. コネクタが `active` で、要求がコネクタの実行の規則に一致する。
5. トークンの `sub` の接続があり `active` で、付与スコープが規則の `required_scopes` を含む。

Vault のリソース識別子は `<発行者>/api/token-vault/v1` である。
`act` を求めるのは、本人の代行付与の撤回が次の呼び出しから効くのがこの形のトークンだけだからである（REQ-OAUTH2-061）。

テナントは、次を登録して利用を始める。

| 登録 | 用途 |
| --- | --- |
| `McpResourceServer`（`resource` が Vault のリソース識別子） | Agent が Token Exchange の `resource` に指定する |
| `AuthorizationDetailType`（`type` が `token_vault_connection`） | 利用者の認可で `authorization_details` を要求する |

## 操作

### Agent による代理実行

#### REQ-TOKENVAULT-008 モデルの論理積を満たす要求だけを、保管したトークンで上流へ送る

- モデルの論理積を満たす要求を受けたとき、TokenVault は、`Authorization: Bearer` に保管したアクセストークンを付けて上流へ送り、200 と上流の結果を返し、`Cache-Control: no-store` を付け、`last_used_at` を更新し、`TokenVaultExecutionPerformed` を発行する。
- トークンが無効か失効しているか、証明が正しくない場合、TokenVault は、401 と `invalid_token` で拒否する。
- 主体か Agent がモデルの 2 を満たさない場合、TokenVault は、403 と `connector_access_denied` で拒否する。
- `authorization_details` がモデルの 3 を満たさない場合、TokenVault は、403 と `connector_access_denied` で拒否する。
- コネクタが存在しないか `disabled` の場合、TokenVault は、404 と `connector_not_found` で拒否する。
- 要求が実行の規則に一致しないか、付与スコープが足りない場合、TokenVault は、403 と `connector_access_denied` で拒否する。
- 接続がない場合、TokenVault は、404 と `connection_not_found` で拒否する。
- 接続が `reauthorization_required` の場合、TokenVault は、409 と `connection_reauthorization_required` で拒否する。
- 失効やトークンの状態を確かめられない場合、TokenVault は、503 と `temporarily_unavailable` を返し、上流へ送らない。
- トークンを検証した後に拒否するとき、TokenVault は、`TokenVaultExecutionDenied` を理由とともに発行し、上流へ送らない。
- 上流へ到達できないとき、TokenVault は、`upstreamStatus` を 0 とした `TokenVaultExecutionPerformed` を発行する。
- **判断**：拒否の理由を応答で語り分けない。理由はイベントで管理者が読む。

#### REQ-TOKENVAULT-009 要求の形を厳密に検証し、解釈の揺れる URL を拒否する

- `url` が絶対 URL の `https` でないか、利用者情報、フラグメントを含むか、パスに `.`、`..` のセグメント、`%2f`、`%5c`、バックスラッシュを含む場合、TokenVault は、400 と `invalid_request` で拒否する。
- 規則を照合するとき、TokenVault は、スキーム、ホスト、ポートの一致と、パスの接頭辞のセグメント境界での一致を求める。
- `headers` が `Accept`、`Accept-Language`、`Content-Type`、`If-Match`、`If-None-Match`、`X-Request-Id` 以外を含む場合、TokenVault は、400 と `invalid_request` で拒否する。
- デコードした本文が 1 MiB を超える場合、TokenVault は、413 と `request_too_large` で拒否する。

#### REQ-TOKENVAULT-010 外向きの要求は公開アドレスだけへ送り、リダイレクトを追わない

- 宛先かトークンエンドポイントのホストへ接続するとき、TokenVault は、解決したすべてのアドレスが公開アドレスであることを確かめる。
- 解決したアドレスに公開アドレスでないものがある場合、TokenVault は、送らずに 502 と `upstream_unreachable` を返す。
- 上流がリダイレクトを返したとき、TokenVault は、追わず、3xx を上流の結果として返す。
- 環境に HTTP プロキシの設定があるとき、TokenVault は、それを使わない。
- 上流が 30 秒以内に応答を終えない場合、TokenVault は、502 と `upstream_unreachable` を返す。

#### REQ-TOKENVAULT-011 上流の応答は上限の範囲で必要な項目だけを返す

- 上流の結果を返すとき、TokenVault は、`status`、`content_type`、`body_base64` だけを返し、`Set-Cookie` を含むほかのヘッダーを返さない。
- 上流の応答本文が 5 MiB を超える場合、TokenVault は、本文を返さず 502 と `upstream_response_too_large` で拒否する。

#### REQ-TOKENVAULT-014 上流の 401 では一度だけ更新して再送する

- 更新トークンを持つ接続で上流が 401 を返したとき、TokenVault は、更新して同じ要求を一度だけ再送する。
- 再送も 401 の場合、TokenVault は、その結果を返し、接続の状態を変えない。
- 更新トークンを持たない接続で上流が 401 を返したとき、TokenVault は、その結果を返し、接続の状態を変えない。

### トークンの更新

#### REQ-TOKENVAULT-012 更新は単一飛行で、回転した更新トークンを保存する

- アクセストークンの期限まで 60 秒未満の接続を使うとき、TokenVault は、上流へ送る前に更新する。
- 同じ接続の更新が重なったとき、TokenVault は、全プロセスを通じて上流の更新を一度に一つだけ行い、ほかの要求は結果を待って使う。
- 応答が新しい更新トークンを持つとき、TokenVault は、置き換えて保存する。
- 応答が更新トークンを持たないとき、TokenVault は、保存した更新トークンを維持する。
- 待機が 10 秒を超えた場合、TokenVault は、503 と `temporarily_unavailable` を返す。
- 更新中に接続が解除か再連携されていたとき、TokenVault は、更新の結果を保存しない。

#### REQ-TOKENVAULT-013 上流の拒否だけが接続を再認可必須へ落とす

- 更新で上流が `invalid_grant` を返したとき、TokenVault は、接続を `reauthorization_required` にし、保管したトークンを消し、`TokenVaultConnectionReauthorizationRequired` を発行し、409 と `connection_reauthorization_required` を返す。
- 期限切れの接続が更新トークンを持たないとき、TokenVault は、同じ遷移と応答を行う。
- 更新で上流が 5xx、`temporarily_unavailable`、`invalid_client`、`unauthorized_client`、不正な応答を返すか、通信に失敗した場合、TokenVault は、接続を変えず、503 と `temporarily_unavailable` を返す。
- **判断**：`invalid_client` を接続へ波及させない。コネクタの設定の誤りであり、利用者の再認可では直らないためである。

### 監査

#### REQ-TOKENVAULT-015 イベントにシークレットと本文を残さない

- 代理実行のイベントを発行するとき、TokenVault は、コネクタ、接続の本人、Agent、規則の ID、メソッド、オリジン、上流の状態コードだけを載せる。
- イベント、ログ、エラーの応答を出力するとき、TokenVault は、トークン、認可コード、`state`、要求と応答の本文、URL のパスとクエリを含めない。

## セキュリティ上の考慮

規則の照合は管理者が許したオリジンとパスの範囲にトークンの送信先を閉じる。
応答本文は検査しないので、トークンを反射する上流の API を規則に含めないことは管理者の責任である（[リスク](../risks.md)）。

## 実現方式

更新のリースは 30 秒で、取得に失敗した要求は 200 ミリ秒ごとに接続を読み直す。
リースの保持者は `token_version` の一致を条件に結果を保存する。
