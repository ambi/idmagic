# WI-552: XML フェデレーション署名鍵のローテーションと、現在の署名鍵の無効化拒否の問題コード

作業項目は `wi-552-back-signing-keys-examples-with-tests` である。

管理者は、SAML と WS-Federation が XML 署名に使う `XmlFederationSigning` 鍵を管理 API からローテートできるようになった。
`POST /api/admin/v1/keys/rotate` は任意の JSON 本文 `{"usage": "XmlFederationSigning"}` を受け取り、デフォルトスコープの XML フェデレーション鍵を新しい鍵へ切り替える。
本文が無いか `usage` が無ければ、これまでどおり JWT の `Signing` 鍵をローテートする。
`usage` が `Signing` と `XmlFederationSigning` のどちらでもない要求は、何もローテートせずに 400 の `invalid_request` で拒否される。
ローテーション前の証明書は、猶予期間のあいだ SAML と WS-Federation のメタデータに残る。
規範上の条件は [REQ-SIGNINGKEYS-006](../../domain/signing-keys/scenarios.feature.md) と、契約の `RotateTenantSigningKey` が定める。
SAML の専用 IdP プロファイルの鍵と、ライフサイクルバッチによる XML フェデレーション鍵の周期ローテーションは、まだ対象にしていない。

現在の署名鍵の無効化を拒否する 400 の問題コードは、`urn:idmagic:error:active_key_cannot_be_disabled` から、契約が宣言する `urn:idmagic:error:invalid_request` へ変わった。
拒否の条件と、鍵が変わらないことは同じである。
