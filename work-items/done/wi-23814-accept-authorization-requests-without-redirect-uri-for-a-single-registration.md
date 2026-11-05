---
status: completed
---

# 登録が一件のクライアントの、`redirect_uri` を省いた認可要求の扱いを決める

REQ-OAUTH2-005 を変え、`openid` を含まない認可要求で、登録した `redirect_uri` が一件のクライアントは `redirect_uri` を省けるようにした。省いた要求は登録値へ応答を返す。`openid` を含む要求と、登録が一件でないクライアントの要求で省いたら、リダイレクトせずに `invalid_request` で拒否する。
トークン要求は全クライアントで RFC 6749 §4.1.3 どおりにした。認可要求で示した `redirect_uri` は再送を必須にし、省いた交換を `invalid_request`、異なる値を `invalid_grant` で拒否し、どちらもコードを消費しない。省いた認可要求のコードだけ、省いた交換を受け入れる。
`standards.md` に `RFC6749-REDIRECT-URI-OPTIONAL`、`RFC6749-TOKEN-REDIRECT-URI`、`OIDC-CORE-TOKEN-REDIRECT-URI` を required で加え、EX-OAUTH2-005-09 を加えた。TypeSpec では `AuthorizeParameters.redirect_uri` を任意にし、`AuthorizationRequest` と `AuthorizationCodeRecord` に `redirect_uri_omitted` を加えた。

判断：OAuth 2.1 草案 17 版 §4.1.3 はトークン要求から `redirect_uri` を除く。全クライアントに PKCE を強制しているので防御は変わらないが、RFC 6749 と OpenID Connect Core を `partial` にしないことを優先し、Keycloak、fosite と同じ扱いにした。OAuth 2.1 にだけ従い再送しないクライアントは対象外とし、OAuth 2.1 は宣言していない。草案 §10.2 どおりにクライアントごとに切り替える案は、登録モデルの変更を伴うので、必要になったときに別の項目で扱う。
`redirect_uri_omitted` は省略を表す向きにしたので、この項目を持たない保存済みの記録は再送を求める側に読まれる。交換の `redirect_uri` の検査がコードの検索の後に移ったため、引き換え済みのコードを `redirect_uri` なしで再提示しても再利用として検出し、ファミリーを失効させる。
