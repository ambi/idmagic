---
status: completed
---

# Directory Connector で取り込んだ AD の User に、Entra のフェデレーションが使う UPN と sourceAnchor を持たせる

Directory Connector で取り込んだ AD の User に、Entra の定型設定でそのままトークンを発行できるようにした。
REQ-DIRECTORYCONNECTOR-004 で、AD の方言の `preferred_username` のデフォルトを `userPrincipalName` にし、`on_premises_immutable_id` に `objectGUID`（上書きで `mS-DS-ConsistencyGuid`）のバイト列の標準の base64 を写す。OpenLDAP の方言は写さず、上書きも拒否する。対応のキーは常に `objectGUID` で、sourceAnchor の選択と切り離した。
REQ-SOURCING-016 と `DirectoryUserItem` で、項目の `on_premises_immutable_id` を User の同名の組み込みの属性（`unique`、`editable_by_user` でない）に保存し、`unique` の属性の衝突を `failed` と `attribute_conflict`（`DirectoryItemError`）にした。SCIM はこの属性を変えない。
IdManagement の一意性の表は、組み込みの `unique` の属性にも、スキーマのないテナントとスキーマの削除の後にも効く（REQ-IDMANAGEMENT-089 に合わせた）。
REQ-WSFEDERATION-001 と `ConfigureEntraFederation` で、構成を拒否せず、UPN がフェデレーションするドメインに属さない取り込み元の User の数と先頭の 5 人を `unfederated_source_users` で返し、画面に表示する。sourceAnchor の初期値は `on_premises_immutable_id` にした（`EntraFederationProfile` の説明も同じ）。
変更した行の変異は全件検出され、配線の切断、スキーマの削除で組み込みの鍵を落とす故障、URL 安全な base64 を手で注入して検出を確かめた。sourceAnchor のバイト列の長さは確かめず、上書きした文字列の属性も base64 にする。
OpenLDAP のコンテナーを使う `backend/directoryconnector` と `ldap_goldap` のテストは、ほかのパッケージと並行に走らせたときに LDAP の接続が EOF で失敗することがあった。単独では毎回通り、今回の変更は接続に触れない。
