---
id: idp-wi-82-federation-scl-and-request-validation
title: "フェデレーション仕様範囲と受信要求検証を整合させる"
created_at: 2026-06-28
authors: ["Codex"]
status: completed
risk: medium
---
# Motivation
WS-Federation / WS-Trust / SAML の SCL が外部標準の採用範囲、除外範囲、
request validation の不変条件、管理 API の契約を十分に表現していなかった。
実装側でも SAML ForceAuthn / Destination、AuthnRequest / LogoutRequest 署名検証、
標準 SLO、WS-Trust To / KeyType が fail-closed に閉じておらず、SCL と実装の対応が弱かった。

# Scope
- **scl_sections**: standards, models, interfaces, invariants, permissions, scenarios
- **contexts**: Saml, WsFederation
- **code**: idmagic/internal/saml/, idmagic/internal/wsfederation/, idmagic/internal/spec/policy.go, idmagic/ui/src/features/admin-applications/AdminApplicationsPage.tsx, idmagic/ui/src/api/admin.ts, idmagic/ui/src/types.ts
- **docs**: idmagic/README.md

# Out of Scope
- encrypted assertion / SAML ECP / inbound federation
- WS-Trust WindowsTransport / Kerberos / silent sign-in

# Verification
- GOCACHE=/tmp/idmagic-cache go test ./internal/saml/... ./internal/wsfederation/... ./internal/spec/...
- GOCACHE=/tmp/idmagic-cache go test ./...
- bun run yaml-check:scl
- bun run yaml-check:work-items
- bun run typecheck

# Risk Notes
SAML 署名必須 trust は X.509 証明書 PEM が必要になる。証明書が欠けた既存データは
fail-closed で拒否される。SLO は LogoutResponse の返送先を登録済み SLO URL に限定するため、
未登録・不一致の SP 設定は明示的に修正が必要になる。

# Completion
- **Completed At**: 2026-06-28
- **Summary**:
  SAML / WS-Federation SCL に standards、invariants、scenarios、permissions、
  WS-Fed RP CRUD interface、WS-Trust RST の検証対象フィールドを追加した。
  実装では SAML ForceAuthn / Destination / AuthnRequest 署名 / LogoutRequest 署名 /
  LogoutResponse、WS-Trust To / KeyType を fail-closed に検証し、README の対応範囲表現を更新した。
  Application 管理 UI でも SAML AuthnRequest / LogoutRequest 署名必須設定と検証用証明書 PEM を
  作成・詳細・編集画面に反映した。
- **Verification Results**:
  - [object Object]
  - [object Object]
  - [object Object]
  - [object Object]
  - [object Object]
