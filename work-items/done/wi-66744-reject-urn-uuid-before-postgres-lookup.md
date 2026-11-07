---
status: completed
---

# urn:uuid: 接頭辞の ID を PostgreSQL の照会前に存在しない行として扱う

`spec-diff` は規範仕様の差分なし。REQ-IDMANAGEMENT-099 の「存在しない Agent は 404 と `agent_not_found`」に実装を合わせた。
`sharedpg.IsUUIDText` が `urn:uuid:` 接頭辞を大文字小文字を問わず拒否するようにした。そのため urn 形式の ID は PostgreSQL の 22P02 で失敗せず、存在しない行として扱われる。
判定を使う tokenvault、sharedsignals、user、agent、sourcing のリポジトリすべてに効く。
Agent の永続化の共有契約に urn 形式の照会を加え、判定の両側を単体テストで固定した。
urn 形式を正規形に直して受け付ける案は、受理範囲を広げ仕様の判断が要るため採らなかった。
