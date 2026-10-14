---
status: completed
archived_from: dbcb32aeb1f35640f2b02bfcb0460f1339e920f6
---

# 横断シナリオが宣言する具体例 4 件にテストを対応付け、被覆台帳から外す

`mise run spec-diff` は `main` に対して `no normative specification change against main` を返した。
横断シナリオの 4 件を実装と照合した。
`EX-PLATFORM-001-01` と `EX-PLATFORM-002-01` は実装が具体例どおりに振る舞っていたが、具体例の `When` である管理者の操作を入口とするテストが無かった。管理 API を入口に、参加するすべての Context の結果を観測するテストを書き、台帳から外した。
`EX-PLATFORM-001-01` のテストは、1 回の無効化で、ステータス、既存セッション、新規ログイン、所有 Agent 2 つの失効エポックと `/introspect`、Agent ごとの `AgentAccessRevoked` の 5 つを観測する。
`EX-PLATFORM-002-01` のテストは、削除の予約でログインが閉じ、猶予期間内の復元で開き直すことを、同じ利用者の往復で観測する。
`EX-PLATFORM-003-01` と `EX-PLATFORM-003-02` は実装が食い違っていた。User と割り当ての変更は、コミットの後で best-effort に capture を呼んでおり、配信行は変更と同じトランザクションで作られない。修正は複数の Context のコミットを 1 つのトランザクションへ合成する設計判断を伴うため、wi-22350 へ切り出し、台帳の 2 行へ `blocked_by` と `finding` を書いて残した。
新しいテストのために `testing_stack` へ `WithAgentRevocation`、`Browser.Get`、`EventLog.All` を足した。製品コードは変えていない。

全文は `git show dbcb32aeb1f35640f2b02bfcb0460f1339e920f6:<このファイルのパス>` で読む。
