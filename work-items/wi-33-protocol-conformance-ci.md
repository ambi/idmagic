---
depends_on: []
status: pending
authors: ["tn"]
risk: medium
created_at: 2026-06-20
---

# OAuth / OIDC / FAPI conformance smoke を CI 検証に追加する

## Motivation
idmagic は OAuth/OIDC/FAPI 系の仕様を SCL と ADR に多く取り込んでいるが、
実装が標準 conformance から drift していないことを継続的に検証する仕組みが不足している。
production-ready IdP では、unit test だけでなく外部 conformance suite / smoke が必要になる。

## Scope
- **decision**:
  - 新規 ADR: conformance 検証を assurance evidence として扱う方針を定義する。 full certification ではなく、CI で回す smoke と release 前に回す expanded suite を分ける。
- **scl**:
  - assurance に OAuth/OIDC/FAPI conformance evidence を追加する。
  - Discovery / JWKS / Authorization Code / PKCE / PAR / DPoP / private_key_jwt の acceptance を更新する。
- **go**:
  - conformance 用 seed tenant / client / user を起動時に用意できる test mode を追加する。
  - discovery metadata の conformance profile を確認し、不足 claim を補う。
- **ci**:
  - Docker Compose で idmagic + UI + Postgres + Valkey を起動する conformance profile を追加する。
  - OpenID Foundation conformance suite または軽量 smoke harness を導入する。
  - 最初は authorization_code + PKCE、discovery、JWKS、token、userinfo、PAR の最小 suite に絞る。
  - FAPI は smoke から始め、full certification は手動 release gate とする。
- **documentation**:
  - README にローカル conformance smoke の実行手順と、full certification との差を記録する。

## Out of Scope
- OIDC/FAPI 正式認証の申請作業。
- 全ブラウザ E2E。SPA E2E は `wi-22`。
- SAML conformance。SAML WI の後続で扱う。

## Plan
- 対象を OIDC Core authorization-code、OAuth 2.0 基本 endpoint、FAPI 2.0 Security Profile smoke に分ける。外部 conformance suite の全 certification を PR ごとに回すのではなく、ローカル deterministic test、PR smoke、夜間 full suite の三段階にする。
- suite は `just` recipe から disposable stack と専用 realm/client/user を seed し、公開 issuer URL が必要な場合だけ CI service/tunnel を使う。固定 client secret や管理者 token を repository/artifact に残さない。
- expected failure は protocol、test ID、根拠 ADR/SCL、owner、expiry を持つ allowlist とし、未知 failure と解消済み allowlist の残存を CI failure にする。
- conformance が見つけた製品契約差異は本 WI 内で場当たり修正せず、先に該当 context SCL を変更してから実装する。suite wrapper と結果正規化だけを本 WI の恒久資産にする。

## Tasks
- [ ] T001 [Matrix] 現行 discovery metadata、grant、PAR/DPoP/FAPI 実装を suite test ID に対応付け、PR/nightly 対象と既知 gap を一覧化する。
- [ ] T002 [Harness] disposable stack の起動、realm/client/user seed、issuer TLS、cleanup、結果取得を行う recipe と wrapper を追加する。
- [ ] T003 [OIDC/OAuth] Core/OAuth smoke profile を設定し、machine-readable result を JUnit/JSON artifact に正規化する。
- [ ] T004 [FAPI] FAPI profile の必要 certificate/key/redirect/PAR 設定を CI secret から注入し、対象 test を夜間 workflow に組み込む。
- [ ] T005 [Exceptions] test ID・根拠・期限付き allowlist validator を実装し、未知 failure/期限切れ/予期せぬ pass を失敗させる。
- [ ] T006 [SCL Follow-up] suite で判明した外部契約差異ごとに該当 SCL 節を更新し、実装修正と regression test を追加する。
- [ ] T007 [CI/Docs] PR smoke/nightly workflow、artifact retention、failure triage とローカル再現手順を記載する。
- [ ] T008 [Verify] clean environment で連続実行し、seed 衝突・secret 漏洩・flaky test がないことを確認する。

## Verification
- `just test-go`
- `just lint-go`
- `just build-go`
- `just dev-compose`
  - reason: conformance smoke の対象環境を起動できることを確認する。
- `conformance smoke command TBD`
  - reason: 選定した harness で最小 suite が pass することを確認する。

## Risk Notes
conformance suite は重く、CI の時間と flakes が増えやすい。毎 PR は smoke、
nightly/release は expanded suite に分ける。失敗時に仕様違反か harness 設定ミスかを
切り分けられるよう、raw output を evidence に保存する。
