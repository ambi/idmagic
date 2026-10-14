---
status: completed
archived_from: 1bcef86ab1b9b074010de8a7d10ca00a044c63b4
---

# テナントブランディングの色を既定値へ明示的に戻せるようにする

primary / accent color ごとに、現在値または未設定（IdMagic 既定）の状態を表示し、設定済みの色を個別に既定へ戻せるようにした。
空文字列を送る既存の UpdateTenantBranding 契約を SCL に明記し、保存後も未設定状態を表示する。

全文は `git show 1bcef86ab1b9b074010de8a7d10ca00a044c63b4:<このファイルのパス>` で読む。
