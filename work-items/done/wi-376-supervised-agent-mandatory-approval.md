---
status: completed
archived_from: e50ccb1f62ec962e4c3e592b991b0b096589a9d2
---

# `AgentKind.Supervised` のエージェントには人間の承認なしにトークンを発行しない

`AgentKind.Supervised` が実行時の意味を持つようになった。それまで宣言でしかなかった区分が、いまはトークン発行の可否を決める。人間の承認を記録しない 3 つの発行経路 — `client_credentials`、トークン交換 (self-issued / workload / 交換クライアント自身の 3 者すべて)、ワークロード ID 連携 — は、関与する Agent が `Autonomous` であると確認できない限り `unauthorized_client` で拒否する (REQ-OAUTH2-050)。`Supervised` な Agent が通れるのは、承認が `ApprovalRequest` として記録される CIBA だけになった。判定は `Kind != Autonomous` の否定形で書いてあり、列挙に無い値も承認が必要な側へ倒れる。拒否は `AgentApprovalRequired` として、承認は `BackchannelAuthApproved.agentId` として監査に残り、どちらの向きの判断も区分から追える。
`AgentRegisterRequest.kind` は必須になった。区分が実行時の意味を持つ以上、省略時の既定値はどちらへ倒しても誤る — `supervised` なら「登録は成功したのに一切トークンを得られない Agent」が黙って生まれ、`autonomous` ならフェイルクローズを目的とする変更で省略が安全でない側へ倒れる。あわせて、不正な列挙値を黙って `supervised` へ丸めていた `normalizeAgentKind` を撤去し、`AgentKindRequiredError` と `InvalidAgentKindError` で明示的に拒否するようにした。この丸め込みが実際に何を隠していたかは既存テストが示していて、`kind: "daemon"` で登録していた 3 件のテストが今回はじめて失敗した。
仕様側では、リクエスト / レスポンス / イベントとしてしか存在しなかった `Agent` 集約を `model Agent` として宣言し ([[wi-369-agent-capability-survey-2026-08]] の再評価条件を解消)、Go では発行済みだが仕様に無かった `AccessTokenClaims.agent_id` を宣言した。`model Agent` の追加により SharedSignals 側の published-language stub `Agent` が衝突したため、実体が同じ名前空間に来たことで不要になった stub を削除した。DB では `agents.kind` の `DEFAULT 'supervised'` を取り除いた。未リリースのため既存行のデータ移行は行っていない。
意味差分 (`mise run spec-diff`): 追加シナリオ `REQ-OAUTH2-050`、変更シナリオ `REQ-IDMANAGEMENT-009`、追加 TypeSpec 宣言 `Agent` / `AgentKindRequiredError` / `InvalidAgentKindError` / `AgentApprovalRequired`、削除 `sharedsignals:Agent`。

全文は `git show e50ccb1f62ec962e4c3e592b991b0b096589a9d2:<このファイルのパス>` で読む。
