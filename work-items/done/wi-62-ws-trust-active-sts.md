---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# WS-Trust 1.3 Active Requestor STS (RST/RSTR・MEX) で能動クライアント認証に対応する

ADR-063 を追加し、WS-Trust 1.3 active STS の初期対応を Issue binding /
`usernamemixed` に限定して実装した。`/{realm}/trust/usernamemixed` は SOAP 1.2 RST を受理し、
WS-Addressing `MessageID` / `To` / `Action`、WS-Security UsernameToken、Timestamp、
`AppliesTo` を fail-closed に検証する。`MessageID` は短期 replay store に記録し、
`AppliesTo` は登録済み WS-Fed RP に解決する。username/password は既存 UserRepository /
PasswordHasher / LoginAttemptThrottle を使って検証し、RP の ClaimMappingPolicy と SAML
署名器を再利用して署名済み assertion を SOAP RSTR で返す。MEX は wi-63 の `trust/mex` が
`usernamemixed` endpoint を広告する。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
