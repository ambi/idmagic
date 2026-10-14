---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# エージェント向け継続的アクセス評価 (CAEP/SSF) と即時失効・kill-switch 伝播

CAEP/SSF によるエージェントの継続的アクセス評価と即時失効を実装した。新規 bounded context
`docs/domain/sharedsignals/SPECIFICATION.md` と `backend/sharedsignals/` (domain / ports /
db_memory / db_postgres / usecases / handlers_http / sign_jose / verify_jose / push_http) を追加し、
以下を成立させた:
- `AgentRevocationEpoch` の単調前進を、条件付き `ON CONFLICT ... WHERE` により **DB 制約として**
fail-closed に強制する (アプリケーション層の判定に依存しない)。
- 汎用 `EventReactor` (`AgentRevocationReactor`) を既存の `Emit` 経路に合成し、
`AgentKilled` / `AgentDisabled` / `AgentCredentialUnbound` / `UserDisabled` / `UserSoftDeleted` /
`UserDeleted` を trigger に epoch を前進させる。業務ロジック本体は無変更で済み、
[[wi-323-caep-ssf-for-human-user-sessions]] が `SessionEnded` 等を足す際も reactor に
`case` を追加するだけで済む形にした。
- OAuth2 `Introspect` で epoch を評価し、失効前に発行された Agent トークンを `active=false` にする。
- SET の署名・送信 (outbox + exponential backoff + dead-letter) と、inbound SET 受理
(`POST /ssf/streams/{stream_id}/events`、署名 / replay / subject 解決の検証) を実装した。
- `objectives.RevocationPropagationLatency` (p99 < 30 秒) を SLO として記録した。
当初の Task 内訳に無かった `ReceiveSecurityEvent` の実装欠落を T005 の `just spec-render` で
検出し、T007 として追加実装した。
**意図的に残した範囲**は [[wi-324-sharedsignals-agent-revocation-followups]] へ引き継いだ:
Hard Quota (`QuotaExceededError` の宣言のみで実装が無い)、所有者オフボード後の新規 token 発行停止
(epoch は前進するが `Agent.Status` は変わらないため `client_credentials` は成功しうる)、
`support_http/auth.go` の `resolveAuthnContext` が epoch と denylist を迂回する経路、
RFC 9493 Subject Identifiers の解釈。このうち後者 2 件はセキュリティ性が高く、
wi-324 側で先行着手するよう優先度を整理した。
なお本記録の作成時点で T001〜T007 は既に完了していたが `status` が `pending` のまま
`work-items/` に残っており、バックログが「エージェント失効は未実装」と誤って表示していた。
[[wi-369-agent-capability-survey-2026-08]] の棚卸しでこれを検出し、実装が緑であることを
確認したうえでクローズした。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
