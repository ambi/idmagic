---
status: completed
---

# SAML の HTTP-Redirect の署名を、受け取ったままの URL エンコードで検証する

SAML IdP は、署名を要求する SP から HTTP-Redirect で受けた AuthnRequest と LogoutRequest の署名を、受け取ったクエリの `SAMLRequest`、`RelayState`、`SigAlg` の値を URL エンコードのまま連結した文字列で検証するようになった。これまではデコードして Go の `url.QueryEscape` で符号化し直していたので、ほかの規則で符号化する SP の正しい署名を拒否し、署名した後に綴りだけ変えた要求を受理していた。
標準要件 `SAML2BINDINGS-REDIRECT-SIGNATURE` を追加した（spec-diff：standards の要件の追加）。既存の `SAML2Bindings-RedirectPost` に足さず別の行にしたのは、その行を引く既存のテストが新しい句を被覆しないためである。
Bindings 3.4.4.1 が定めない点は次のとおり決めた。`SAMLRequest`、`RelayState`、`SigAlg`、`Signature` は名前をデコードして数え、二度現れたら拒否する。値の空な `RelayState=` も現れた引数として連結に含める。署名の対象でない引数は無視する。
証拠は、SimpleSAMLphp 2.5.5、Sustainsys.Saml2 2.11.2、Shibboleth SP 3.5.0 から 2026-11-28 に記録した五つの要求の再生である。取得のスクリプトは `backend/saml/domain/testdata/redirect_signatures/record/` にある。直す前の実装は、SimpleSAMLphp の二つの要求を拒否した。SimpleSAMLphp が RelayState の `~` を `%7E` と書くためである。
Shibboleth SP の LogoutRequest は記録していない。小文字の 16 進と `%20` の規則は実物の記録に現れなかったので、テストが書いた符号化で HTTP の `/saml/sso` と `/saml/slo` から確かめた。
