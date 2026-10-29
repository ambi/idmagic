---
status: completed
---

# 束縛解除と再委譲の後も発行時のエージェントに基づいて停止を効かせる

REQ-OAUTH2-012 を、現在の束縛ではなく発行時の Agent で判定する規範に改めた。
アクセストークンは要求したクライアントの Agent と、交換の入力が持っていた Agent を `agent_chain` に載せ、失効判定はその Agent の存在、`Active` の状態、エポックを読む。
現在の束縛の Agent のエポックも引き続き読み、`agent_chain` を持たない旧トークンの保証を残す。
REQ-OAUTH2-047 は同じ判定への参照に、REQ-OAUTH2-048 は失効した `subject_token` の拒否を加え、トークン交換の入力は `/introspect` と同じ判定を通す。
REQ-SHAREDSIGNALS-001 は発行時の Agent のエポックで比べると明記した。
強制終了した Agent はエポックによらず無効になるため、EX-SHAREDSIGNALS-001-02 と EX-OAUTH2-012-02 の対照を無効化から再有効化した後の発行に改めた。
token.md の「所有者が復帰すれば発行も自動的に再開する」を、REQ-IDMANAGEMENT-081 の `Disabled` が残る規範に合わせて直した。

独立した観測として、正式な `/token`、管理 API、`/revoke`、`/introspect` を通す `agent_revocation_lifecycle_e2e_test.go` を残した。
保存表現は変えていない。
残る制約は次のとおりである。
`agent_chain` を持たない旧トークンは、配備から最大 600 秒、交換の中間の Agent の停止を見落とす。
Agent の削除は `AgentAccessRevoked` の SET を外部へ送らない。
リフレッシュトークン自体のイントロスペクションは Agent の停止を読まない。
署名だけで JWT を検証する外部の資源は、有効期限まで停止を検知できない。
