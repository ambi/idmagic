# WS-Trust の能動的 STS

## 概要

この文書は、`/trust/usernamemixed` で WS-Trust 1.3 の `Issue` を受け付け、署名した SAML Assertion を返す仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | UsernameToken による認証、WS-Addressing と WS-Security の必須要素の検証、`MessageID` の再利用の拒否、RSTR の返却 |
| 行為者 | SecurityTokenRequester（UsernameToken を提示して `Issue` を呼ぶ能動的なクライアント） |
| 扱わないもの | `Validate`、`Renew`、`Cancel` は実装しない。Kerberos と IWA の `windowstransport` は範囲の外である |

## モデル

能動的な WS-Trust への対応は、汎用的な相互運用ではなく、Microsoft 365 型のリッチクライアントによるサインインを対象とする。

| 要素 | 検証 |
| --- | --- |
| UsernameToken | 既存のユーザー、パスワードのハッシュ、ログインの試行の制限で検証する |
| `MessageID` | Assertion の有効期間のあいだ記録し、再利用を拒否する |
| Timestamp | 期限切れの値と遠い未来の値を拒否する |
| `To`、`Action`、`RequestType`、`KeyType` | フェイルクローズで検証する |
| `AppliesTo` | 登録した RP に解決できなければ拒否する |

発行する Assertion の audience と recipient は、`AppliesTo` の RP に限る。
クレームは、その RP の `ClaimMappingPolicy` を通じて発行する。
RSTR は SOAP 1.2 で署名した SAML Assertion を返す。
トークンの型は、RST が SAML 1.1 または SAML 2.0 を明示すればその型、明示しなければ RP の設定の型とし、それ以外の型を要求されたら拒否する。

- **判断**：対応する範囲を絞る理由は、[WS-Trust の能動的な対応を usernamemixed の Issue だけに絞る](../design/decisions.md#ws-trust-の能動的な対応を-usernamemixed-の-issue-だけに絞る)。

## 操作

### クライアントによる WS-Trust の Issue

#### REQ-WSFEDERATION-004 妥当な WS-Trust Issue は RSTR を返す

- クライアントが `Active` の User の正しい UsernameToken と登録済みの RP の `AppliesTo` を持つ妥当な RST を `/trust/usernamemixed` へ送ったとき、WsFederation は、署名した SAML Assertion を含み、`RelatesTo` が RST の `MessageID` を指す RSTR を、200 と `application/soap+xml` で返し、`WsTrustTokenIssued` を発行する。
- RST にトークンを発行するとき、WsFederation は、Assertion の audience を RP の audience（設定がない RP では `wtrealm`）に、recipient を `AppliesTo` に、有効期間を発行の 1 分前から 5 分後までにし、RP の `ClaimMappingPolicy` でクレームを発行し、応答に `Cache-Control: no-store` を付ける。
- RST が SAML 1.1 または SAML 2.0 のトークンの型を指定したとき、WsFederation は、その型の Assertion を返す。
- RST がトークンの型を指定しないとき、WsFederation は、RP に設定した型（設定がない RP では SAML 1.1）の Assertion を返す。
- SAML 1.1 と SAML 2.0 以外のトークンの型を指定された場合、WsFederation は、400 で拒否し、`WsTrustTokenRejected` を発行し、トークンを発行しない。
- Assertion の有効期間（5 分）の間に同じ `MessageID` の RST を再び受けた場合、WsFederation は、400 で拒否し、`WsTrustTokenRejected` を発行し、トークンを発行しない。
- UsernameToken のユーザー名が未知か、パスワードが誤っているか、User が `Active` でないか、ログインの試行の制限に達している場合、WsFederation は、401 で拒否し、`WsTrustTokenRejected` を発行し、トークンを発行しない。
- クレームを発行できないか、Entra の定型設定の ImmutableID を導けない場合、WsFederation は、500 で拒否し、`WsTrustTokenRejected` を発行し、トークンを発行しない。
- **例**：EX-WSFEDERATION-004-01、EX-WSFEDERATION-004-02、EX-WSFEDERATION-004-03

#### REQ-WSFEDERATION-005 不正なエンベロープの WS-Trust Issue は拒否する

- SOAP 1.2 のエンベロープとして読めないか、`RequestSecurityToken`、`MessageID`、`To`、`Action`、UsernameToken のユーザー名かパスワード、`AppliesTo` のどれかがない RST を受けた場合、WsFederation は、400 で拒否し、`WsTrustTokenRejected` を発行し、トークンを発行しない。
- `Action` または `RequestType` が WS-Trust 1.3 の `Issue` でないか、`KeyType` が `Bearer` でない RST を受けた場合、WsFederation は、400 で拒否し、`WsTrustTokenRejected` を発行し、トークンを発行しない。
- Timestamp の `Expires` が `Created` より後でないか、`Created` が現在より 5 分を超えて未来か、`Expires` を過ぎている RST を受けた場合、WsFederation は、400 で拒否し、`WsTrustTokenRejected` を発行し、トークンを発行しない。
- `To` がテナントの能動的な STS の URL と一致しないか、`AppliesTo` がテナントに登録した RP の `wtrealm` でない RST を受けた場合、WsFederation は、400 で拒否し、`WsTrustTokenRejected` を発行し、トークンを発行しない。
- **例**：EX-WSFEDERATION-005-01

## セキュリティ上の考慮

能動的な STS は管理者の認可を通らず、UsernameToken で認証する。
`AppliesTo` が登録した RP に解決できることを求め、未登録の宛先にトークンを発行しない。
`MessageID` の記録と audience の限定により、リプレイや audience の取り違えが RP の境界を越えることを防ぐ。
