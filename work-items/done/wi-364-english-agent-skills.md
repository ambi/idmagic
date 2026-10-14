---
status: completed
archived_from: 8dfb463c25614d200080700ca267794e1c4b478e
---

# agent skill の本文を英語に統一する

`.agents/skills/**/SKILL.md` 全 7 件（commit / new-work-item / implement-work-item / spec-change /
spec-render / update-design / parallel-work-items）の本文を英語に統一した。手順の順序と粒度は
変えていない。push 禁止、生成物を commit しない、baseline を通常変更で更新しない、ADR を作らない
といった禁止事項は、英語でも明示的な否定形（Do not …）で残した。
`commit` skill の frontmatter `description` にある「コミットして」「これをコミット」は、ユーザーが
入力する起動フレーズとして一致対象なのでそのまま残した。

全文は `git show 8dfb463c25614d200080700ca267794e1c4b478e:<このファイルのパス>` で読む。
