# Feature: 復旧コードの例

## Rule: REQ-AUTHENTICATION-036 復旧コードで成立した第二要素は MFA の要求を満たす

### Example: EX-AUTHENTICATION-036-01 通常経路

- Given 対象 Application の実効サインインポリシーは `Mfa` である
- And ユーザー "alice" は復旧コードを生成済みで、第二要素を待つ `authentication_pending` の LoginSession を持つ
- When ユーザー "alice" が正しい復旧コードを送信する
- Then 認証が成立し、`amr` に `rc` が加わって `acr` が `urn:idmagic:acr:mfa` になる
- Then 同じセッションによる次の認可要求は、第二要素を再び求めずに継続する

### Example: EX-AUTHENTICATION-036-02 誤った復旧コードを送信する

- Given 対象 Application の実効サインインポリシーは `Mfa` である
- And ユーザー "alice" は復旧コードを生成済みで、第二要素を待つ `authentication_pending` の LoginSession を持つ
- When ユーザー "alice" が正しい復旧コードを送信する
- But 誤った復旧コードを送信する
- Then エラー "UnauthorizedError"
- And LoginSession は `authentication_pending` のままで、`amr` に `rc` は加わらず `acr` も上がらない
