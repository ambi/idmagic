---
status: completed
---

# SCIM の User の作成と置換で、null の phoneNumbers と addresses を未割り当てとして受け付ける

REQ-SOURCING-006 を変更し、SCIM の User の作成と置換で、null か空の配列の `phoneNumbers` と `addresses` を属性の省略と同じに扱うようにした。
null でも空の配列でもない値（要素を持つ配列、`[null]`、オブジェクト、文字列）は、従来どおり 400 の `invalidValue` で拒否する。
RFC 7643 §2.5 が未割り当て、null、空の配列を同じ状態とみなし、値のない属性は捨てるデータを持たないためである。
`CreateScimUser` と `UpdateScimUser` の説明を同じ規則に直し、Entra ID の記録の `create-user-with-nulls` を再生の列へ入れた。
PATCH は対応しないパスを値に関係なく `invalidPath` で拒否し（`title` も同じ）、それ自体で一貫しているため変えていない。
