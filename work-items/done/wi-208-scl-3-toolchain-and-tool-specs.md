---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# 全 SCL toolchain と tool 自身の仕様を SCL 3.0 に移行する

- RA loader、section inventory、workspace discovery と tool-only render 経路を SCL 3.0 専用へ移行した。
- HTML renderer を model constraints、interface contract/access、authorization、SLO、scenario、flow の
所有要素中心表示へ更新し、旧 invariant / permission / user experience renderer を撤去した。
- OpenAPI に public/protected security、requires/ensures、resource/policy と決定的な AuthZEN/Cedar action
metadata を追加し、JSON Schema に field/model constraint 変換と非対応 CEL の明示保持を追加した。
- 全5 tool spec と派生 HTML を SCL 3.0 self-hosting fixture として同期した。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
