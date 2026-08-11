---
name: spec-render
description: Compile TypeSpec and regenerate ignored OpenAPI and browsable specification HTML after specification or design changes.
---

# 仕様派生物の再生成

1. `just spec-render` を実行する。
2. `just check-api-compat` で release baseline に対する破壊を確認する。
3. context別tagのOpenAPIと `spec/generated/docs/index.html` が生成されることを確認する。
4. `spec/generated/` は untracked のため commit しない。
5. release 時だけ、明示的な release 手順として baseline を更新する。通常の feature change で baseline を更新して検査を回避しない。
