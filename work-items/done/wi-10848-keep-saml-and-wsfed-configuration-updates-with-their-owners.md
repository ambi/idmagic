---
status: completed
---

# ApplicationCatalog の SAML と WS-Fed の管理を所有者の公開操作へ通す

Saml に `ServiceProviderAdministration`、WsFederation に `RelyingPartyAdministration` の公開操作を置き、ApplicationCatalog の SP と RP の作成、設定の更新、Application への関連付け、削除をそれらへ移した。
NameID と IdP プロファイルの既定値、Assertion の署名、証明書の要否とトークンの種類の検査は所有者が持ち、Catalog は入力の変換、必須項目の 400、クレームの公開の下限の検査、`ApplicationClaimMappingUpdated` の発行だけを行う。
`mise run spec-diff` は規範の差分なしを示し、既存のテストを変えずに HTTP の応答、保存とイベントを保った。
照合で見つかった作成時の置き換え、長さの検査の欠落、イベントの記述の不一致は wi-68811 で扱う。
