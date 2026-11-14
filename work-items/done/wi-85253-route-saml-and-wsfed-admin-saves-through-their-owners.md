---
status: completed
---

# SAML の SP と WS-Fed の RP の管理 API も、所有者の登録操作を通す

SAML の SP と WS-Fed の RP の管理 API は、保存先へ直接書かず、各モジュールの管理操作（`Register`、`DeleteStandalone`、Entra の構成の `SaveEntra`）を通すようになった。
AuthnRequest の署名の証明書の確認は、カタログと管理 API が共有する一か所にまとまった。
起票時に問題にした「管理 API が証明書なしの SP を保存する」は、着手時にはすでに解消していた。そのため、この拒否は欠けていたテストを足して固定した。

着手時に、管理 API の登録が Application に属する SP と RP を作り直し、`ApplicationID` を消す欠陥を見つけた。
利用者と合意し、この上書きを 409 の `application_owned_protocol` で拒否するようにした。
`spec-diff` の差分は、REQ-SAML-005 と REQ-WSFEDERATION-001 の要件文の追加、`RegisterSamlServiceProvider` と `RegisterWsFedRelyingParty` の 409 の追加、`ApplicationOwnedProtocolError` の説明の拡大である。
Entra の構成が同じレルムの汎用の RP を置き換えることと、ハンドラーの依存の型から書き込みの操作を外すことは、対象外のまま残した。
