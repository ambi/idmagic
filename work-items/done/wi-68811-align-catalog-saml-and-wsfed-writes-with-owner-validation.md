---
status: completed
---

# Application の作成と更新で、SAML の SP と WS-Fed の RP を所有者と同じ規則で拒否する

`wsfed` と `saml` の Application の作成は、同じ `wtrealm` または entityID の RP と SP がテナントにあれば、Entra の RP、別の Application に属するもの、どの Application にも属さないものを区別せず、409 と新しい `ProtocolAlreadyRegisteredError`（`protocol_already_registered`）で拒否するようになった（REQ-APPLICATION-007、`CreateAdminApplication`）。
長さの上限を超えるキーは、RP と SP を保存する前に 422 と `field_length_exceeded` で拒否する。
これまでは 400 で拒否しながら、RP と SP を先に保存していた。
拒否は `ServiceProviderAdministration` と `RelyingPartyAdministration` の `CreateForApplication` が行い、Catalog はエラーを HTTP へ写すだけである。
プロトコル設定の更新は、指定した項目の JSON の名前を `changedFields` に持つ `ApplicationUpdated` を発行し、何も指定しない更新では発行しない（REQ-APPLICATION-005、`ApplicationUpdated` の説明）。
項目名は要求の型ごとに明示的に並べ、リフレクションを使わない。
同じキーの並行した作成の排他と、作成の途中の失敗の補償は対象外のままである。
