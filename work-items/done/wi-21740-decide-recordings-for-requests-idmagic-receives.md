---
status: completed
---

# IdMagic が受け取る要求と文書について、実際の相手の記録で検証するかを相手ごとに決める

IdMagic が受け取る要求について、相手ごとに実際の相手か一次資料の記録を入力にした受け入れの検査を置くかを決め、採った相手に検査を置いた。
規範の差分はない（`mise run spec-diff`）。
着手の後、利用者の判断で Kerberos と、Directory Connector が Active Directory を読む処理も対象に含めた。

採った相手と記録は次のとおりである。

- SCIM：Entra ID と Okta の文書の要求の例を転記し、受け口へ流した（`provisioning_client_replay_test.go`）。
- SAML：wi-75603 の実物の SP の AuthnRequest を `ValidateSignIn` まで通した。Entra ID の公開の連携メタデータを記録した。
- SSF：CAEP 1.0 の 13 の例を、本物の verifier で再生した。
- WebAuthn：Chrome の仮想認証器で、画面が送った本文を記録した（`mise run record-webauthn`）。
- WS-Trust：本物の MSAL.NET 4.92 が送った MEX の取得と RST を記録した（`mise run record-msal`）。
- Kerberos：手元の Chrome が Samba AD のチケットで送った SPNEGO のトークンを記録した（`mise run record-negotiate`）。
- Active Directory の LDAP：Samba AD の DC に対する統合テストを置き、CI に足した（`mise run test-go-sambaad`）。
- OAuth の動的登録：RFC 7591 の例を転記して流した。

記録が見つけた食い違いは、次の項目に起票した。

- SCIM：wi-90508、wi-56158、wi-41091。wi-248 には Entra ID と Okta の事実を書き足した。
- SSF：wi-32902。
- SAML のメタデータ：wi-73879。
- Kerberos：wi-55734、wi-10982。
- 動的登録：wi-97659。

あわせて、既存の `TestVerifierAgainstSambaAD` が `main` で失敗していることも見つけ、wi-10982 に書いた。

採らなかった相手は次のとおりである。

- OAuth の revoke、introspect、デバイス認可、CIBA、end_session：分岐が値で決まる。
- 認可コードと token など：wi-33 の適合試験が扱う。
- 外部の OP のコールバック：`state` と `code` だけを読み、仕様が結果をその値で定める。
- Directory Connector の API：配布した版がまだない。初回公開の後に、公開した版の要求を記録する。
- OpenLDAP：実物を CI で試験している。
- MCP：実際のクライアントの記録に利用者の操作が要るので、wi-11142 に起票した。

起票時の判断の誤りとして、初めの採否は基準を当てはめずに否を多く選んでいた。
利用者の指摘で、経路とコードと CI を読み直して採否を改めた。
