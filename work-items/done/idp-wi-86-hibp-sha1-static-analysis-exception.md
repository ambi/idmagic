---
id: idp-wi-86-hibp-sha1-static-analysis-exception
title: "HIBP SHA-1 fingerprint の静的解析例外を明示する"
created_at: 2026-07-01
authors: ["Codex"]
status: completed
risk: low
---

# Motivation
CodeQL `go/weak-sensitive-data-hashing` が HIBP Range API 用 SHA-1 fingerprint を
password hashing として検出した。HIBP は SHA-1 prefix lookup をプロトコルとして
要求するためアルゴリズム置換はできないが、inline `nolint:gosec` だけで抑止すると
例外範囲と根拠が監査しにくい。password 保存・照合用 hash ではないこと、例外を
HIBP adapter の専用関数 1 箇所に限定すること、gosec/CodeQL で同じ判断になることを
記録可能にする。

# Scope
- **scl_sections**:
- **decisions**:
  - decisions/ADR-028-breached-password-checker.md
- **implementation**:
  - internal/shared/adapters/policy/hibp_breached_password_checker.go
  - internal/shared/adapters/policy/hibp_breached_password_checker_test.go
  - .golangci.yml

# Out of Scope
- HIBP Range API の採否変更
- fail-open / cache / metric の挙動変更
- password 保存・照合用 Argon2id hasher の変更

# Verification
- `GOCACHE=/tmp/idmagic-cache GOLANGCI_LINT_CACHE=/tmp/idmagic-golangci-cache golangci-lint run ./internal/shared/adapters/policy`
- `GOCACHE=/tmp/idmagic-cache go test ./internal/shared/adapters/policy`

# Risk Notes
HIBP prefix/suffix 計算を専用関数に移すだけで、外部 API への入力・比較・fail-open
挙動は変えない。主な残リスクは CodeQL の inline suppression コメント形式が GitHub
側で将来変更されること。その場合も ADR と golangci-lint の限定例外により、例外範囲は
HIBP adapter の 1 箇所として監査できる。

# Completion
- **Completed At**: 2026-07-01
- **Summary**:
  HIBP Range API 用 SHA-1 計算を `hibpRangePrefixSuffix` に閉じ、テストは同じ関数を
  使うようにしてテスト側の SHA-1 再実装を削除した。inline `nolint:gosec` は消し、
  `.golangci.yml` に HIBP adapter ファイルかつ G401/G505 だけの限定例外を追加した。
  ADR-028 には、この SHA-1 が password 保存・照合用 hash ではなく HIBP lookup
  fingerprint であること、通常の password hash は Argon2id のままであること、
  静的解析例外を HIBP adapter の専用関数 1 箇所に限定することを追記した。
- **Verification Results**:
  - `GOCACHE=/tmp/idmagic-cache GOLANGCI_LINT_CACHE=/tmp/idmagic-golangci-cache golangci-lint run ./internal/shared/adapters/policy` - passed
    - environment: local macOS, Go 1.26.4, golangci-lint 2.12.2
    - result: 0 issues。
  - `GOCACHE=/tmp/idmagic-cache go test ./internal/shared/adapters/policy` - passed
    - environment: local macOS, Go 1.26.4
    - result: 対象 package 成功。
  - `GOCACHE=/tmp/idmagic-cache GOLANGCI_LINT_CACHE=/tmp/idmagic-golangci-cache golangci-lint run ./...` - passed
    - environment: local macOS, Go 1.26.4, golangci-lint 2.12.2
    - result: 0 issues。
  - `GOCACHE=/tmp/idmagic-cache go test ./...` - passed
    - environment: local macOS, Go 1.26.4, outside Codex filesystem sandbox for httptest port binding
    - result: 全 Go package 成功。
  - `bun run yaml-check:work-items` - passed
    - environment: tools
    - result: 94 file OK。
  - `codeql database analyze /tmp/idmagic-codeql-go codeql/go-queries:Security/CWE-327/WeakSensitiveDataHashing.ql` - passed
    - environment: local CodeQL CLI 2.25.6, codeql/go-queries 1.6.4
    - result: SARIF results length 0 for the target query after rebuilding a Go database for internal/shared/adapters/policy.
- **Affected Guarantees State**:
  - guarantee: privacy (生 password 非送信 / prefix 5 文字のみ)
  - state: passed
  - guarantee: security_static_analysis (HIBP SHA-1 例外の限定)
  - state: passed
  - guarantee: traceability (ADR と実装の相互参照)
  - state: passed
