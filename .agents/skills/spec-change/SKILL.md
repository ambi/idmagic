---
name: spec-change
description: Specification-first workflow for feature and behavior changes. Update TypeSpec for models/APIs/authentication and requirements Markdown for requirements/scenarios/glossary/standards/state transitions before implementation.
---

# 仕様先行の変更

実装より先に、所有 context の最小仕様を更新する。

1. モデル、API、HTTP、status/error、認証方式は `spec/contexts/<context>/{models,main}.tsp` を更新する。
2. 要件、scenario、glossary、standard、状態遷移は `requirements.md` を更新する。
3. 状態遷移は `From | Event | Guard | To | Effects` の言語非依存表にする。
4. 新しい規範要件には未使用の `REQ-<CONTEXT>-NNN` を採番する。既存 ID を再利用・並べ直ししない。
5. 細粒度 authorization policy DSL は追加しない。認証方式だけ TypeSpec に置き、認可挙動はコードとテストで保つ。
6. work item の `affected_spec` を requirement ID / TypeSpec symbol に同期する。
7. `just check-spec` と `just check-api-compat` を通す。生成物は commit しない。

構造・技術・ディレクトリ規約も変わる場合だけ `new-architecture` Skill を併用する。
