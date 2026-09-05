# Feature: WsFederation Scenarios

## Rule: REQ-WSFEDERATION-001 管理 API クライアントは WS-Fed スコープの信頼設定だけを操作できる

Primary actor: `ManagementApiClient`

### Example: EX-WSFEDERATION-001-01 通常経路

- Given クライアントは対象テナントの有効な API アクセストークンを提示している
- When クライアントが RP または Entra フェデレーションの操作をリクエストする
- Then `wsfed:read` スコープでは RP の参照だけを許可する
- Then `wsfed:write` スコープでは RP と Entra フェデレーションの変更だけを許可する

### Example: EX-WSFEDERATION-001-02 wsfed:read だけで変更操作を要求する

- Given クライアントは対象テナントの有効な API アクセストークンを提示している
- When クライアントが RP または Entra フェデレーションの操作をリクエストする
- But wsfed:read だけで変更操作を要求する
- Then 操作は AccessDeniedError で拒否される

### Example: EX-WSFEDERATION-001-03 トークンのテナントとリクエスト先のテナントが一致しない

- Given クライアントは対象テナントの有効な API アクセストークンを提示している
- When クライアントが RP または Entra フェデレーションの操作をリクエストする
- But トークンのテナントとリクエスト先のテナントが一致しない
- Then 操作を AccessDeniedError で拒否する

## Rule: REQ-WSFEDERATION-002 登録済みの RP へのパッシブサインインはトークンを発行する

Primary actor: `EndUser`

### Example: EX-WSFEDERATION-002-01 通常経路

- Given wtrealm と wreply は登録済みで、対象ユーザーは対象 Application に割り当てられている
- When 登録済み RP の wsignin1.0 を受信する
- Then wtrealm、wreply、wfresh、Application の割り当てを検証する
- Then 署名済み Assertion を RSTR フォームで返し、wctx を同じ値で返す

### Example: EX-WSFEDERATION-002-02 wfresh より認証が古い

- Given wtrealm と wreply は登録済みで、対象ユーザーは対象 Application に割り当てられている
- When 登録済み RP の wsignin1.0 を受信する
- Then wfresh より認証が古い
- Then トークンを発行せず再認証へ誘導する

### Example: EX-WSFEDERATION-002-03 wtrealm、wreply、wauth、対象者の割り当てのいずれかが不正である

- Given wtrealm と wreply は登録済みで、対象ユーザーは対象 Application に割り当てられている
- When 登録済み RP の wsignin1.0 を受信する
- Then wtrealm、wreply、wauth、対象者の割り当てのいずれかが不正である
- Then WsFedSignInRejected を発行してフェイルクローズで拒否する

## Rule: REQ-WSFEDERATION-003 信頼していない宛先へのパッシブサインインは拒否する

Primary actor: `EndUser`

### Example: EX-WSFEDERATION-003-01 通常経路

- Given wtrealm が未登録、wreply が許可外、または対象ユーザーが未割り当てである
- When 不正な wsignin1.0 を受信する
- Then トークンを発行せず WsFedSignInRejected を発行する

## Rule: REQ-WSFEDERATION-004 妥当な WS-Trust Issue は RSTR を返す

Primary actor: `SecurityTokenRequester`

### Example: EX-WSFEDERATION-004-01 通常経路

- Given UsernameToken、MessageID、Timestamp、To、Action、RequestType、KeyType、AppliesTo が有効である
- When WS-Trust Issue の RST を受信する
- Then UsernameToken と RST の必須要素をすべて検証する
- Then RSTR を返す

### Example: EX-WSFEDERATION-004-02 MessageID が Assertion の有効期間内に再利用されている

- Given UsernameToken、MessageID、Timestamp、To、Action、RequestType、KeyType、AppliesTo が有効である
- When WS-Trust Issue の RST を受信する
- Then MessageID が Assertion の有効期間内に再利用されている
- Then WsTrustTokenRejected を発行してプロトコルエラーを返す

### Example: EX-WSFEDERATION-004-03 UsernameToken の資格情報が不正である

- Given UsernameToken、MessageID、Timestamp、To、Action、RequestType、KeyType、AppliesTo が有効である
- When WS-Trust Issue の RST を受信する
- Then UsernameToken の資格情報が不正である
- Then AccessDeniedError を返しトークンを発行しない

## Rule: REQ-WSFEDERATION-005 不正なエンベロープの WS-Trust Issue は拒否する

Primary actor: `SecurityTokenRequester`

### Example: EX-WSFEDERATION-005-01 通常経路

- Given RST の To、MessageID、AppliesTo、Action、RequestType、KeyType のいずれかが不正である
- When 不正な RST を受信する
- Then WsTrustTokenRejected を発行し、400 または 401 を返す
