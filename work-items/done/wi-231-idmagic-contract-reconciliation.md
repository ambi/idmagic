---
status: completed
authors: ["tn"]
risk: medium
created_at: 2026-07-16
depends_on: [wi-230-ra-traceability-graph]
---

# IdMagic の runtime route と SCL/OpenAPI 契約を完全に一致させる

## Motivation
監査時点で runtime には204 HTTP operation、派生 OpenAPI には198 operation があり、`just yaml-check` は差分を検出しない。account/password/email/step-up/role policy/SAML 更新の仕様漏れ、admin consent path の誤記、同一 SCIM path の GET/POST を失う generator 不備を解消し、再発を contract test で防ぐ。

## Scope
- `spec/contexts/authentication.yaml` の account context、password reset context、step-up WebAuthn challenge interface/binding/scenario。
- `spec/contexts/identity-management.yaml` の email verification context interface/binding/scenario。
- `spec/contexts/oauth2.yaml` の role policy interface と admin consent `/api/admin` path 修正。
- `spec/contexts/application.yaml` の SAML configuration update binding。
- `spec/contexts/system.yaml` の liveness/readiness/startup probe 契約。
- `spec/contexts/scim.yaml` の Users/Groups collection list interface/binding/scenario。
- `spec/contexts/ws-federation.yaml` の単一 passive endpoint binding ownership。
- `tools/scl-to-openapi` の同一 path 複数 method merge。
- 組み立て済み Echo router と生成 OpenAPI の method/path 集合 contract test。

## Out of Scope
- endpoint の新規製品機能追加。
- wire response の互換性変更。
- protocol conformance suite 全体。
- UI route と browser flow の全面棚卸し。

## Plan
- 実装済み挙動を正として無条件に SCL へ転記せず、既存 SCL の ownership、access、scenario と照合して規範契約を確定する。
- contract test は tenant prefix と path parameter 名を正規化するが、手書き allowlist は持たない。
- operational probe も外部 HTTP 契約として System context に含める。
- OpenAPI generator は path item を上書きせず method 単位でマージし、重複 method は error にする。

## Tasks
- [x] T001 [Inventory] runtime-only、spec-only、generator-loss の差分を固定 fixture にする。
- [x] T002 [SCL] 欠落 interface/binding/scenario と誤 path を修正する。
- [x] T003 [Generator] SCIM GET/POST を含む path merge を修正する。
- [x] T004 [Contract] assembled router と OpenAPI の双方向集合検査を追加する。
- [x] T005 [Derived] SCL 派生物を再生成する。
- [x] T006 [Verify] method/path 差分0件と既存 wire test の成功を確認する。

## Verification
- `just yaml-check-scl`
- `just test-tools`
- `just test-go`
- `just scl-render`
- `just verify`

## Risk Notes
仕様漏れの補完に見えても認可・CSRF・公開範囲を規範化する変更を含む。各 interface の `access`、失敗 scenario、既存 handler test を同時に確認し、実装の偶然を仕様へ固定しない。

## Completion

- **Completed At**: 2026-07-17
- **Summary**:
  runtime-only 14 operation と誤った admin consent 3 path を SCL の models、interfaces、
  access、scenarios へ正規化し、SCIM Users/Groups collection GET と SAML 設定更新、
  operational probe を含む生成 OpenAPI を runtime route と一致させた。同一 path の異なる
  method は一つの path item に保持し、同一 method/path の重複所有は生成時に拒否する。
  assembled Echo router と生成 OpenAPI を realm prefix と path parameter 名だけ正規化して
  双方向比較する contract test を追加し、手書き allowlist なしで差分 0 件を固定した。
- **Verification Results**:
  - `just yaml-check-scl` - passed (19 SCL files)
  - `just test-tools` - passed (228 tests)
  - `just test-go` - passed (assembled route/OpenAPI contract difference 0)
  - `just scl-render` - passed (OpenAPI, JSON Schema, app/tool HTML synchronized)
  - `just lint-go` - passed (0 issues)
  - `just verify` - passed (SCL/Work Item/Architecture/traceability, tool typecheck/tests, Go lint/race tests)
- **Affected Guarantees State**:
  公開 HTTP operation の method/path 集合は SCL 派生 OpenAPI と assembled runtime router で
  同一になった。tenant prefix と parameter 名以外は正規化せず、新しい欠落・余剰 route を
  CI で fail closed に検出する。既存 endpoint の wire response、認可、CSRF 挙動は変更していない。
- **Evidence**:
  Codex が macOS arm64 の local workspace にて、commit 前 working tree と source revision
  `f794dfa0c44c2885932b11fe431d2a632403214d` を対象に上記 `just` recipe を実行した。
  結果は terminal output で確認し、派生 artifact は各 `spec/` 配下へ保存した。
