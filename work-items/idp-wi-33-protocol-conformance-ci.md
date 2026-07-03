---
id: idp-wi-33-protocol-conformance-ci
title: "OAuth / OIDC / FAPI conformance smoke を CI 検証に追加する"
created_at: 2026-06-20
authors: ["tn"]
status: pending
risk: medium
---
# Motivation
idmagic は OAuth/OIDC/FAPI 系の仕様を SCL と ADR に多く取り込んでいるが、
実装が標準 conformance から drift していないことを継続的に検証する仕組みが不足している。
production-ready IdP では、unit test だけでなく外部 conformance suite / smoke が必要になる。

# Scope
- **decision**: 新規 ADR: conformance 検証を assurance evidence として扱う方針を定義する。 full certification ではなく、CI で回す smoke と release 前に回す expanded suite を分ける。
- **scl**: assurance に OAuth/OIDC/FAPI conformance evidence を追加する。, Discovery / JWKS / Authorization Code / PKCE / PAR / DPoP / private_key_jwt の acceptance を更新する。
- **go**: conformance 用 seed tenant / client / user を起動時に用意できる test mode を追加する。, discovery metadata の conformance profile を確認し、不足 claim を補う。
- **ci**: Docker Compose で idmagic + UI + Postgres + Valkey を起動する conformance profile を追加する。, OpenID Foundation conformance suite または軽量 smoke harness を導入する。, 最初は authorization_code + PKCE、discovery、JWKS、token、userinfo、PAR の最小 suite に絞る。, FAPI は smoke から始め、full certification は手動 release gate とする。
- **documentation**: README にローカル conformance smoke の実行手順と、full certification との差を記録する。

# Out of Scope
- OIDC/FAPI 正式認証の申請作業。
- 全ブラウザ E2E。SPA E2E は `wi-22`。
- SAML conformance。SAML WI の後続で扱う。

# Verification
- [object Object]
- [object Object]
- [object Object]
- [object Object]
- [object Object]

# Risk Notes
conformance suite は重く、CI の時間と flakes が増えやすい。毎 PR は smoke、
nightly/release は expanded suite に分ける。失敗時に仕様違反か harness 設定ミスかを
切り分けられるよう、raw output を evidence に保存する。
