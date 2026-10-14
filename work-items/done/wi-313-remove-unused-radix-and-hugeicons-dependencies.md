---
status: completed
archived_from: a1f8325c82098f24bf74ab5bebeb40bf8e64bc41
---

# 未使用になった Radix UI / hugeicons パッケージを package.json から除去する

`frontend/package.json` に残っていた未使用の `@radix-ui/react-dropdown-menu` /
`@radix-ui/react-label` / `@radix-ui/react-slot`（[[wi-312-migrate-radix-to-base-ui]]
で Base UI へ移行済みだが削除し忘れていた）と `@hugeicons/core-free-icons` /
`@hugeicons/react`（shadcn init 直後の初期 `iconLibrary` の名残で一度も使われて
いなかった）を `bun remove` で除去した。これにより frontend の Radix UI 依存は
package.json レベルでも完全に無くなった。ソースコードの変更は無し。

全文は `git show a1f8325c82098f24bf74ab5bebeb40bf8e64bc41:<このファイルのパス>` で読む。
