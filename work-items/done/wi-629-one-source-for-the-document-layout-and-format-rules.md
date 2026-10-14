---
status: completed
archived_from: 459ff025bb4a5fc2492a68030f8405212c4cbd52
---

# 文書配置と文書フォーマットの正本を一つにし、実態との乖離を消す

`mise run spec-diff` は `main` に対する規範仕様の変更がないことを報告した。
文書配置と形式の一次情報を `SPECIFICATION_FORMAT.md` と `WORK_ITEM_FORMAT.md` へ集約し、`DOCUMENTATION_GUIDE.md` は体系の理由と参照だけを持つ形にした。
`SPECIFICATION_FORMAT.md` の配置図と `tools/workspace/src/document-layout.ts` の不一致は、新しい登録済み検査が拒否する。

全文は `git show 459ff025bb4a5fc2492a68030f8405212c4cbd52:<このファイルのパス>` で読む。
