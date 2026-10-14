---
status: completed
archived_from: 6fcf71b2efe191f99055cceb92329033710424c8
---

# SAML IdP が AuthnRequest の意味を fail-closed に検証し重複発行を防止する

AuthnRequest の Version / IssueInstant / binding / ACS / NameID semantics を fail-closed に検証し、
passive の未認証要求には検証済み ACS へ NoPassive response を返すようにした。tenant / SP /
request ID の TTL replay reservation を assertion 発行直前に置き、memory と Valkey の両方で
原子的に一度だけ発行できるようにした。

全文は `git show 6fcf71b2efe191f99055cceb92329033710424c8:<このファイルのパス>` で読む。
