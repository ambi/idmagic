---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# 監査の検索軸にエージェントと委譲チェーンを加え、代行操作を本人の操作と区別できるようにする

監査の検索軸に `actor.type` / `agent.id` / `delegation.actor` / `delegation.depth` / `delegation.mode` の 5 つを加え、エージェントが利用者を代行した操作を、利用者本人の操作と区別して引けるようにした。規範シナリオは REQ-AUDIT-005 (代行の区別) と REQ-AUDIT-006 (チェーン参加者からの横断検索) の 2 つを追加した。
意味の差は 3 つある。第一に、検索属性の副表が多値になった。主キーを `(event_id, attr_name)` から `(event_id, attr_name, attr_value)` へ広げ、`delegation.actor` が 1 イベントにつきチェーンの参加者ぶんの値を持てるようにした。既存の `EXISTS` 照合は「いずれかの行が一致すれば真」の意味を既に持つため、単値の軸の意味論は変わらない。第二に、`TokenExchanged` が委譲チェーンの参加者 (`actorChain`) と交換を行った Agent (`agentId`) を残すようになった。これまでは深さの整数と両端しか無く、中間の行為者を後から辿れなかった。第三に、`actor.id` のフォールバックが Agent を行為者とするイベントで止まり、代行された利用者は `target.id` に移った。行為者と対象の区別を委譲へ延長したものである。
`agentId` の解釈には注意を要する判断があった。`AgentRegistered` や `BackchannelAuthApproved` のように、`agentId` が行為者ではなく操作の対象を指すイベントが多数あるため、種別を `agentId` の有無から推測すると管理者の操作が Agent の操作として記録される。Agent が自ら振る舞う経路のイベント型を列挙して決める形にした。
Motivation が挙げていた `FgaCheckEvaluated` への参加者の追加は行っていない。Authorization には「タプルの内容と主体識別子は監査へ複製しない」という既存の判断があり、参加者を載せることはその撤回にあたる。識別子を含まない `actorChainDepth` を `delegation.depth` へ供給するにとどめた。詳細は `## Design` に記した。
仕様が得たものは規範シナリオ 2 件 (REQ-AUDIT-005 / REQ-AUDIT-006)、用語 2 件 (`AuditDelegationChain` / `AuditActorType`)、TypeSpec のフィールド 5 件 (`AuditEventSearchAttribute.multi_valued`、`AuditEventSearchOptionsResponse.actor_types` / `delegation_modes`、`TokenExchanged.agentId` / `actorChain`)、`internals.md` の節 1 件である。失ったものはない。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
