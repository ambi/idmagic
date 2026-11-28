# Saml の標準仕様

## Assertions and Protocols for the OASIS Security Assertion Markup Language (SAML) V2.0

2.0 — https://docs.oasis-open.org/security/saml/v2.0/saml-core-2.0-os.pdf

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| SAML2Core-BearerAssertion | required | MUST | AuthnRequest は Version 2.0 と許容範囲内の IssueInstant を持たなければならない。発行する Assertion の Audience / Recipient / InResponseTo は、検証済みのリクエストと一致させる。同じテナント、SP、リクエスト ID の組み合わせに対して Assertion を発行できるのは一度だけとする |
| SAML2CORE-AUTHNREQUEST-OMITTED-ATTRIBUTES | required | MUST | §3.4.1 が任意とする AuthnRequest の `Destination`、`AssertionConsumerServiceURL`、`ProtocolBinding`、`NameIDPolicy` を省いた要求を受け入れ、SP のデフォルトの ACS URL へ HTTP-POST で、SP に構成した NameID 形式の Assertion を返す。`NameIDPolicy` の `Format` が `unspecified` なら、省いた要求と同じに扱う |
| SAML2Core-EncryptedAssertion | excluded | MAY | 暗号化された Assertion |

## Bindings for the OASIS Security Assertion Markup Language (SAML) V2.0

2.0 — https://docs.oasis-open.org/security/saml/v2.0/saml-bindings-2.0-os.pdf

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| SAML2Bindings-RedirectPost | required | MUST | AuthnRequest は HTTP-Redirect または HTTP-POST バインディングで受理する。リクエストで指定された ProtocolBinding が HTTP-POST 以外なら拒否する。SAMLResponse と返信可能なプロトコルエラーは HTTP-POST で返す |
| SAML2BINDINGS-REDIRECT-SIGNATURE | required | MUST | 署名を要求する SP から HTTP-Redirect で受けた AuthnRequest と LogoutRequest は、受け取ったクエリの `SAMLRequest`、`RelayState`（クエリに現れたときだけ）、`SigAlg` の値を URL エンコードのまま、この順に `SAMLRequest=値&RelayState=値&SigAlg=値` と連結した文字列で検証する。16 進の大文字と小文字、空白の `+` と `%20`、予約されていない文字を符号化するかどうかが SP ごとに違っても、正しい署名を受け入れる。名前をデコードして `SAMLRequest`、`RelayState`、`SigAlg`、`Signature` のいずれかが二度以上現れる要求は拒否する |

## Profiles for the OASIS Security Assertion Markup Language (SAML) V2.0

2.0 — https://docs.oasis-open.org/security/saml/v2.0/saml-profiles-2.0-os.pdf

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| SAML2Profile-WebBrowserSSO | required | MUST | SP 起点と IdP 起点の Web Browser SSO を提供する。未対応の ACS インデックス、NameID 形式、レスポンスバインディングはフェイルクローズで拒否する。`IsPassive=true` でログインが必要な場合は、NoPassive プロトコルレスポンスを返す |
| SAML2Profile-ECP | excluded | MAY | Enhanced Client or Proxy Profile |

## Metadata for the OASIS Security Assertion Markup Language (SAML) V2.0

2.0 — https://docs.oasis-open.org/security/saml/v2.0/saml-metadata-2.0-os.pdf

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| SAML2Metadata-IDPSSODescriptor | required | MUST | IdP メタデータでは、SSO エンドポイント、SLO エンドポイント、署名証明書、NameID 形式を公開する |
| SAML2Metadata-WantAuthnRequestsSigned | optional | MAY | SP ごとの信頼ポリシーとして、AuthnRequest / LogoutRequest の署名検証を要求できる |
