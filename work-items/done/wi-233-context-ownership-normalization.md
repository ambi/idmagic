---
status: completed
archived_from: c1cb9e272cde8ff6b575a1a203f05477f719fb60
---

# ClaimMapping・SigningKeys・domain event の物理所有を SCL context map に一致させる

ClaimMapping と SigningKeys の domain/usecase/adapter を SCL 上の ownership に対応する context package へ移し、consumer と bootstrap DI を公開面経由に再配線した。context 固有 event は owning domain へ移し、shared には envelope、wire marshal、複数 context が発行する技術的通知 primitive のみを残した。SCL 派生物と `ARCHITECTURE.md` も実装へ同期した。

全文は `git show c1cb9e272cde8ff6b575a1a203f05477f719fb60:<このファイルのパス>` で読む。
