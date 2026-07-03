---
id: idp-wi-81-rename-authentication-ports-alias
title: "authentication ports の import alias を authnports に改名する"
created_at: 2026-06-28
authors: ["Codex"]
status: completed
risk: low
---
# Motivation
`authports` は authentication の略として使われていたが、authorization の略にも見え、
`oauthports` と並んだときに責務が読み取りづらい。認証を表す `authnports` に揃え、
authn/authz の区別を import alias に反映する。

# Scope
- **scl_sections**:
- **code**: idmagic/internal/** imports of idmagic/internal/authentication/ports

# Out of Scope
- package 名やディレクトリ名の変更
- SCL の規範振る舞い変更
- OAuth2 ports の alias 変更

# Verification
- GOCACHE=/tmp/idmagic-cache go test ./...

# Risk Notes
Go import alias と参照識別子だけの変更で、型・関数・package path は変えない。

# Completion
- **Completed At**: 2026-06-28
- **Summary**:
  `idmagic/internal/authentication/ports` の import alias を `authports` から
  `authnports` に変更した。package 名、ディレクトリ名、SCL、公開 API は変更していない。
- **Verification Results**:
  - [object Object]
