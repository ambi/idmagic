# WsFederation Scenarios

### REQ-WSFEDERATION-001: 管理 API クライアントは WS-Fed スコープの信頼設定だけを操作できる
- ACTOR ManagementApiClient
- GIVEN クライアントは対象テナントの有効な API アクセストークンを提示している
- WHEN クライアントが RP または Entra フェデレーションの操作をリクエストする
  - ALT wsfed:read だけで変更操作を要求する → 操作は AccessDeniedError で拒否される
  - ALT トークンのテナントとリクエスト先のテナントが一致しない → 操作を AccessDeniedError で拒否する
- THEN `wsfed:read` スコープでは RP の参照だけを許可する
- THEN `wsfed:write` スコープでは RP と Entra フェデレーションの変更だけを許可する

### REQ-WSFEDERATION-002: 登録済みの RP へのパッシブサインインはトークンを発行する
- ACTOR EndUser
- GIVEN wtrealm と wreply は登録済みで、対象ユーザーは対象 Application に割り当てられている
- WHEN 登録済み RP の wsignin1.0 を受信する
- THEN wtrealm、wreply、wfresh、Application の割り当てを検証する
  - ALT wfresh より認証が古い → トークンを発行せず再認証へ誘導する
  - ALT wtrealm、wreply、wauth、対象者の割り当てのいずれかが不正である → WsFedSignInRejected を発行してフェイルクローズで拒否する
- THEN 署名済み Assertion を RSTR フォームで返し、wctx を同じ値で返す

### REQ-WSFEDERATION-003: 信頼していない宛先へのパッシブサインインは拒否する
- ACTOR EndUser
- GIVEN wtrealm が未登録、wreply が許可外、または対象ユーザーが未割り当てである
- WHEN 不正な wsignin1.0 を受信する
- THEN トークンを発行せず WsFedSignInRejected を発行する

### REQ-WSFEDERATION-004: 妥当な WS-Trust Issue は RSTR を返す
- ACTOR SecurityTokenRequester
- GIVEN UsernameToken、MessageID、Timestamp、To、Action、RequestType、KeyType、AppliesTo が有効である
- WHEN WS-Trust Issue の RST を受信する
- THEN UsernameToken と RST の必須要素をすべて検証する
  - ALT MessageID が Assertion の有効期間内に再利用されている → WsTrustTokenRejected を発行してプロトコルエラーを返す
  - ALT UsernameToken の資格情報が不正である → AccessDeniedError を返しトークンを発行しない
- THEN RSTR を返す

### REQ-WSFEDERATION-005: 不正なエンベロープの WS-Trust Issue は拒否する
- ACTOR SecurityTokenRequester
- GIVEN RST の To、MessageID、AppliesTo、Action、RequestType、KeyType のいずれかが不正である
- WHEN 不正な RST を受信する
- THEN WsTrustTokenRejected を発行し、400 または 401 を返す
