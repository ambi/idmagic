---
status: pending
authors: [tn]
risk: low
created_at: 2026-08-11
depends_on:
  - wi-355-replace-scl-architecture-ledgers-and-adrs
change_kind: tooling
spec_impact:
  kind: none
  reason: This work item evaluates and renames the development method without changing product behavior.
initial_context:
  source:
    - REGENERATIVE_ARCHITECTURE.md
    - SPECIFICATION_FORMAT.md
    - WORK_ITEM_FORMAT.md
    - AGENTS.md
    - tools/ra
    - .agents/skills
  tests:
    - tools/ra/src
  stop_before_reading:
    - backend
    - frontend
---

# RA を lightweight SDD として再定義・改称する

## Motivation

specification、architecture ledger、traceability manifest、新規 ADR 運用を廃止した後の開発方式は、
TypeSpec と requirements を先に更新し、実装・テスト・生成契約を追従させる軽量な
Spec-driven Development (SDD) である。現在の `Regenerative Architecture` という名称と
`REGENERATIVE_ARCHITECTURE.md` は、実際には存在しない自己再生機構や重い architecture
management を想起させ、方法論の理解コストを不必要に上げる。

残す価値のある原則を特定したうえで、方法論、文書、CLI、skills の名称を実態に合う最小の
語彙へ揃える必要がある。

## Scope

- 現行手順のうち SDD として維持すべき保証と、RA 固有語として削除できる概念を整理する。
- `Regenerative Architecture`、`RA`、`ra` CLI、関連ファイル名・skill 名の改称案を比較する。
- `REGENERATIVE_ARCHITECTURE.md` を lightweight SDD の短いガイドへ置換または改称する方針を決める。
- work item、specification-first workflow、Architecture current-state record の関係を簡潔に定義する。
- 選んだ名称を docs、AGENTS、tools、skills、README に一貫して反映する実装範囲を確定する。

## Out of Scope

- Product requirements or TypeSpec contract changes.
- specification、architecture ledger、traceability manifest、必須 ADR の再導入。
- 外部の包括的 SDD framework の導入。

## Plan

現行フローを「必須」「有用だが任意」「歴史的残骸」に分類する。名称候補ごとに意味の正確さ、
検索性、既存 path/command の移行コストを比較し、最小の概念集合を選ぶ。結論は現行仕様・設計
文書へ直接反映し、却下案の長期保存を目的とした新規 ADR は作らない。

## Tasks

- [ ] T001 [Analysis] 現行 RA 文書・CLI・skills が実際に提供する保証を一覧化する。
- [ ] T002 [Decision] lightweight SDD の名称、文書構成、command/skill naming を決める。
- [ ] T003 [Docs] 方法論文書、AGENTS、README、format 文書の用語を同期する。
- [ ] T004 [Tooling] 必要なら `ra` CLI と関連 package/skill 名を改称し、互換方針を定める。
- [ ] T005 [Verify] repository check、tool tests、stale terminology search を通す。

## Verification

- `just check`
- `just test-tools`
- `just typecheck-tools`
- `rg -n "Regenerative Architecture|REGENERATIVE_ARCHITECTURE|\\bra\\b" AGENTS.md README.md *.md tools .agents/skills`

## Risk Notes

主なリスクは、機械的な改称で過去の work item・ADR link を壊すことと、一般名 `SDD` に寄せすぎて
この repository 固有の最小ルールが曖昧になることである。履歴ファイル名の互換性と現在形の用語は
分けて扱う。
