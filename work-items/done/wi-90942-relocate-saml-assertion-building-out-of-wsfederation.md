---
status: completed
---

# SAML のアサーションの生成を WsFederation の非公開パッケージから出す

`mise run spec-diff` は、作業範囲の起点に対して規範の変更なしと報告した。
SAML のアサーションの組み立てと XML 署名を、WsFederation の非公開パッケージから共有ライブラリ `backend/shared/security/tokens_saml` へ移した。
入力は SAML の語彙（認証コンテキスト、NameID、属性）にし、発行結果と認証方式からの写像は Saml と WsFederation がそれぞれ持つ。
XML 連携署名の鍵から署名器を作る処理は SigningKeys のルートの `FederationSigner` にし、組み立て地点が各モジュールへ渡す。
Saml と Tenancy から WsFederation への `private-import` 2 件と、それに伴う `module-cycle` 3 件が台帳から消えた。
ハンドラーが写像の結果をアサーションの入力へ渡すことを確かめる段のテストは、移す前からなく、今回も足していない。

全文は `git log -1 -- work-items/active/wi-90942-relocate-saml-assertion-building-out-of-wsfederation.md` が示すコミットの親、またはそのコミットで読む。
