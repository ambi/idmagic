---
id: idp-wi-80-rename-platform-to-infrastructure
title: "idmagic の横断アダプタ実装ディレクトリを infrastructure に改名する"
created_at: 2026-06-28
authors: ["Codex"]
status: completed
risk: low
---
# Motivation
`internal/platform/` は OS platform を連想させるが、実際には Clean Architecture /
DDD / RA の Layer 4 に属するコンテキスト横断アダプタ実装を収容している。
Go 側のディレクトリ名を Layer 4 の責務に合わせる。top-level の `infra/` は
`deploy/` へ改名し、`internal/infrastructure/` との語義衝突を避ける。
あわせて Go の `internal/` 境界の意味も文書化する。

# Scope
- **scl_sections**:
- **code**: idmagic/internal/infrastructure/, idmagic/internal/*/**.go import path
- **decisions**: idmagic/decisions/ADR-047-context-owned-adapter-layout.md, idmagic/decisions/ADR-068-infrastructure-directory-for-cross-context-adapters.md
- **docs**: idmagic/README.md

# Out of Scope
- SCL の規範振る舞い変更
- HTTP route、DB schema、公開 API の変更
- 過去 work item の監査記録に含まれる旧パス参照の書き換え

# Verification
- GOCACHE=/tmp/idmagic-cache go test ./...

# Risk Notes
import path とディレクトリ名の一括変更であり、実行時の意味変更はない。
リスクは置換漏れによる build/test failure に限定される。

# Completion
- **Completed At**: 2026-06-28
- **Summary**:
  `internal/platform/` を `internal/infrastructure/` に改名し、top-level の
  `infra/` を `deploy/` に改名した。Go import path、README、ADR、実行パスを更新した。
  SCL、HTTP route、DB schema、公開 API は変更していない。
- **Verification Results**:
  - [object Object]
