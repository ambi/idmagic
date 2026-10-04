# Feature: パッシブサインインの例

## Rule: REQ-WSFEDERATION-002 登録済みの RP へのパッシブサインインはトークンを発行する

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

### Example: EX-WSFEDERATION-003-01 通常経路

- Given wtrealm が未登録、wreply が許可外、または対象ユーザーが未割り当てである
- When 不正な wsignin1.0 を受信する
- Then トークンを発行せず WsFedSignInRejected を発行する
