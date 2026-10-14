---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# エージェント ID / Token Exchange のセキュリティ是正

wi-49 / wi-50 のレビューで見つかった Agent kill-switch と Token Exchange の
権限境界を是正した。Agent は同一 tenant の既存 active User を owner_sub として
必須検証し、killed Agent の delete を拒否する。AgentCredentialBinding は
tenant 内 client_id 一意とし、Postgres migration 0014 と memory repository /
BindCredential use case の両方で重複束縛を拒否する。Token Exchange は
requested_token_type の未対応値を拒否し、発行前に TokenGrantTokenExchange の
Authorizer gate を通す。policy context には actor_sub / subject_sub / audience /
delegation_depth / requested scopes を載せ、local policy は grant 宣言・client scope
subset・resource 必須を fail-closed に評価する。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
