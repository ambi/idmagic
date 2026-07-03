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
- **decisions**: decisions/ADR-028-breached-password-checker.md
- **implementation**: internal/shared/adapters/policy/hibp_breached_password_checker.go, internal/shared/adapters/policy/hibp_breached_password_checker_test.go, .golangci.yml

# Out of Scope
- HIBP Range API の採否変更
- fail-open / cache / metric の挙動変更
- password 保存・照合用 Argon2id hasher の変更

# Verification
- [object Object]
- [object Object]

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
  - [object Object]
  - [object Object]
  - [object Object]
  - [object Object]
  - [object Object]
  - [object Object]
