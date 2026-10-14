---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# `brief` に、参照する TypeSpec 記号の宣言と実装の充足状況を出させる

`mise run spec-diff` reports no normative specification change against `main`, which is the intended
result for `spec_impact: none`. What changed is the terminal answer: every symbol and requirement a
work item references now carries one `Coverage` line saying whether it is declared and whether anything
outside `docs/` and `spec/` names it, with the reading that follows from those two answers — start from a
specification change, add the missing code, or read the files that already name it. The verdict counts
tests as naming, because `REQ-*` identifiers are named by tests rather than by implementation, and a
verdict that ignored them would report implemented behavior as unimplemented. Symbols are asked for by
their declared name, not their fully qualified one, because no Go or TypeScript file carries
`IdMagic.Contract.*` — the same name `declarationOf` already resolves the declaration by.

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
