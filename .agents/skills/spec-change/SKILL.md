---
name: spec-change
description: Specification-first workflow for feature and behavior changes. Update TypeSpec for models, APIs, and authentication, and the owning SPECIFICATION.md for normative scenarios, glossary, standards, state transitions, authorization boundaries, and design before implementation.
---

# 仕様先行の変更

実装より先に、所有 context の最小仕様を更新する。

1. モデル、API、HTTP、status/error、認証方式は `spec/contexts/<context>/{models,main}.tsp` を更新する。
2. 規範的な振る舞い、scenario、glossary、standard、状態遷移、認可境界は owning `SPECIFICATION.md` の意味を所有する section を更新する。
3. 状態遷移は `From | Event | Guard | To | Effects` の言語非依存表にする。
4. 新しい観測可能な規範的振る舞いには未使用の `REQ-<CONTEXT>-NNN` を採番する。既存 ID を再利用・並べ直ししない。
5. 細粒度 authorization policy DSL は追加しない。認証方式だけ TypeSpec に置き、認可挙動はコードとテストで保つ。
6. work item の `affected_spec` を normative scenario / standard ID または TypeSpec symbol に同期する。
7. `just check-spec` と `just check-api-compat` を通す。生成した OpenAPI / HTML は commit しない。

構造・技術・ディレクトリ規約も変わる場合だけ `update-design` Skill を併用する。
