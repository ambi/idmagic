---
status: completed
archived_from: 88ee974b21001938ad03ca972800698a2d768296
---

# トップレベル `internal/` を `backend/` に、`ui/` を `frontend/` に改名し、`cmd/` を `backend/cmd/` へ移す

Go 実装と entry point を `backend/`、React UI を `frontend/` へ履歴を保って移動し、Go import path、ビルド・Docker・CI・sqlc・検証 manifest・運用文書を同期した。ADR-092 を追加し、ADR-068/070 の関連判断を supersede として記録した。SCL の規範振る舞いは不変。

全文は `git show 88ee974b21001938ad03ca972800698a2d768296:<このファイルのパス>` で読む。
