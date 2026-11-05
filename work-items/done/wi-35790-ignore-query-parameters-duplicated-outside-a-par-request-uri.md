---
status: completed
---

# `request_uri` の外に重ねた認可パラメーターを、拒否せずに無視する

PAR の `request_uri` を使う `/authorize` は、クエリに重ねた `request_uri` と `client_id` 以外の認可パラメーターを拒否せずに無視し、PAR で保存した値だけで認可を進める。
`client_id` が PAR と一致しない要求は、これまでどおり `invalid_request` で拒否する。
`standards.md` に RFC 9101 の節と RFC9101-QUERY-PARAMETERS-IGNORED を加え、REQ-OAUTH2-009 から混在の拒否を除いた。
拒否を固定していた EX-OAUTH2-005-04 は REQ-OAUTH2-005 から取り除き、重ねた値が結果に影響しない例を EX-OAUTH2-009-03 として PAR の要件の下に置いた。
FAPI 2.0 Security Profile は認可サーバーに重ねた値の拒否を求めないため、プロファイルによらず無視する。
一つのクエリで同じ名前を重複させた要求は、`request_uri` があってもこれまでどおり拒否する。
RFC 9101 が必須とするクエリの `client_id` を省いた要求も、これまでどおり受け入れる。
