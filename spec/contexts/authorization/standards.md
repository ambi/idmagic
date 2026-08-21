# Authorization Standards

## OpenID AuthZEN Authorization API

https://openid.net/specs/authorization-api-1_0.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| AUTHZEN-FGA-EVALUATION | required | MUST | 関係に基づく判定は `{subject, action, resource, context}` の評価に載せ、関係の成否は判定 context の事実として渡す。 |
| AUTHZEN-FGA-ACTOR-CHAIN | required | MUST | 代行チェーンは判定 context に明示的に載せ、各段のプリンシパル種別・識別子・有効性を分離して表す。 |
| AUTHZEN-FGA-FAIL-CLOSED | required | MUST | 評価器が判定を返せない、事実が欠けている、深さ上限に達した、ストアへ到達できないいずれの場合も許可へ退避しない。 |
| AUTHZEN-FGA-SEARCH | optional | MAY | 主体を固定してリソースを列挙する探索 (Search) を提供する。上限つきの走査で、打ち切りを結果に示す。 |

## OAuth 2.0 Token Exchange

RFC 8693 — https://www.rfc-editor.org/rfc/rfc8693.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC8693-FGA-ACTOR-AND | required | MUST | 代行トークンでの判定は、`sub` の主体と `act` チェーン上のすべての actor が同じ関係を持つときにだけ許可する。エージェントは代行するユーザーの権限を超えられない。 |
