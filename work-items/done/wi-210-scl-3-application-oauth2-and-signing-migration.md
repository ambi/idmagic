---
status: completed
archived_from: bbc6f79eaa45561c4621bcce2e18d9e8d06a13fd
---

# Application・ClaimMapping・OAuth2・SigningKeys context を SCL 3.0 に移行する

- Application / ClaimMapping / OAuth2 / SigningKeys の4 context を SCL 3.0へ意味移行した。旧
`invariants` / `permissions` / `user_experience` を model constraints、state guards、interface
`requires` / `ensures`、authorization principal / policy / resource、scenario、flowへ再配置した。
- OAuth2 の全32 interfaceに publicまたはprotected `access`を明示した。`Authorize.requires`へ
application assignment gate・sign-in policy fail-closed、`Token.requires`へtenant locality・DPoP
freshness/replay防止・signing provider healthを置き、token grant別policyを定義した。
- latency / error rate / availability / throughputだけをboolean indicator型SLOとして残し、OAuth2の
TTL・rate limit・replay window・retentionをADR-109へ、SigningKeysのrotation/overlap/archive設定を
ADR-108へ移した。値とruntime behaviorは変更していない。
- ApplicationからOAuth2所有のclient CRUD / consent isolation scenarioを移し、旧username/IP hashを
前提としたAudit所有の陳腐化scenarioは、ADR-104と現行Audit contextに反するためOAuth2から除いた。

全文は `git show bbc6f79eaa45561c4621bcce2e18d9e8d06a13fd:<このファイルのパス>` で読む。
