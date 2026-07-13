---
status: completed
authors: [tn]
risk: high
created_at: 2026-07-14
depends_on: [wi-207-scl-3-core-language-and-validation]
---

# 全 SCL toolchain と tool 自身の仕様を SCL 3.0 に移行する

## Motivation

SCL 3.0 を IdMagic の規範仕様へ適用する前に、読み込み・表示・派生物生成を担う全 tool が新構造を
解釈でき、tool 自身の SCL が新形式の実例として検証できる必要がある。

## Scope

- `tools/ra` の SCL bundle loader、section inventory、workspace discovery。
- `tools/scl-to-html` の型、section renderer、cross-reference、authorization/objective/flow diagram 表示。
- `tools/scl-to-openapi` の interface contract/access 対応。
- `tools/scl-to-jsonschema` の model-level constraints 対応。
- `tools/yaml-check` を含む全5 tool の `tools/*/spec/scl.yaml` を SCL 3.0 へ移行する。
- 各 tool spec の `models`、`interfaces`、`states`、旧 `invariants`、`scenarios`、旧 `permissions`、
  `objectives`、旧 `user_experience` を ADR-103 の所有規則で再分類する。

## Out of Scope

- `spec/contexts/*.yaml` と root `spec/scl.yaml` の移行。
- IdMagic runtime の policy evaluator または UI の変更。
- SCL 2.0 document を SCL 3.0 renderer/generator へ自動変換する compatibility layer。
- IdMagic の派生 HTML/OpenAPI/model schema の最終再生成。

## Plan

- inner tool model/types から loader、generator、renderer の順で更新する。
- HTML は独立 invariant section を削除し、model constraints、interface requires/ensures/access、
  authorization、SLO、navigation flow、main-success scenario を所有要素の近くに表示する。
- policy を参照する interface 群から action group と appliesTo を決定的に組み立てられる中間表現を置く。
- OpenAPI は public/protected の security metadata と requires/ensures の記述を保持するが、runtime 固有の
  policy engine extension を必須にしない。model JSON Schema は表現可能な constraints だけを生成し、
  表現不能な CEL を黙って誤変換しない。
- tool spec は各 tool の移行と同じ change で更新し、SCL 3.0 self-hosting fixture とする。

## Tasks

- [x] T001 [RA] bundle loader、section kind、型を SCL 3.0 専用形へ更新する。
- [x] T002 [HTML] 新 section/局所契約/意味参照を表示し、旧 invariant/UX renderer を撤去する。
- [x] T003 [OpenAPI] interface access、resource、requires/ensures と新参照規則を生成へ反映する。
- [x] T004 [JSONSchema] field/model constraints の生成と非対応 CEL の明示的扱いを実装する。
- [x] T005 [Authorization] interface→policy 参照から AuthZEN/Cedar action metadata を生成する純粋な
  中間表現と単体 test を追加する。
- [x] T006 [SCL] `tools/ra/spec/scl.yaml`、`tools/yaml-check/spec/scl.yaml`、
  `tools/scl-to-html/spec/scl.yaml`、`tools/scl-to-openapi/spec/scl.yaml`、
  `tools/scl-to-jsonschema/spec/scl.yaml` を3.0へ移行する。
- [x] T007 [Test] snapshot/fixture を新 HTML・OpenAPI・schema・参照 anchor に同期する。
- [x] T008 [Verify] 全 tool の test/typecheck と tool spec validation を通す。

## Verification

- `just test-tools`
- `just typecheck-tools`
- `just yaml-check-scl`
- `just yaml-check-work-items`
- `just check-ids`

## Risk Notes

risk は high。全生成器の同時変更で snapshot 差分が大きい。生成結果の見た目だけでなく、意味参照、
security metadata、constraint preservation を小さな fixture で検証する。IdMagic bundle はまだ2.0を含むため、
本 item では IdMagic の最終 `just scl-render` を完了条件にしない。

## Completion

- **Completed At**: 2026-07-14
- **Summary**:
  - RA loader、section inventory、workspace discovery と tool-only render 経路を SCL 3.0 専用へ移行した。
  - HTML renderer を model constraints、interface contract/access、authorization、SLO、scenario、flow の
    所有要素中心表示へ更新し、旧 invariant / permission / user experience renderer を撤去した。
  - OpenAPI に public/protected security、requires/ensures、resource/policy と決定的な AuthZEN/Cedar action
    metadata を追加し、JSON Schema に field/model constraint 変換と非対応 CEL の明示保持を追加した。
  - 全5 tool spec と派生 HTML を SCL 3.0 self-hosting fixture として同期した。
- **Verification Results**:
  - `just test-tools` — passed (210 tests)
  - `just typecheck-tools` — passed
  - `just yaml-check` — passed (19 SCL files, 213 work items, 312 record IDs, Architecture cross-check)
  - `just scl-render-tools` — passed (5 tool HTML artifacts regenerated)
  - `just verify` — passed (tool tests/typecheck, Go lint/race tests, UI format/lint/typecheck/347 tests/build)
- **Affected Guarantees State**:
  - guarantee: SCL toolchain loader と section inventory は SCL 3.0 の所有構造だけを解釈する。
  - state: passed
  - guarantee: HTML、OpenAPI、JSON Schema は局所契約と認可/制約の意味を黙って失わない。
  - state: passed
  - guarantee: 5 tool specs は SCL 3.0 schema と意味検証を通る自己ホスト例である。
  - state: passed
- **Evidence**:
  - procedure: SCL-first で tool specs を移行後、各 generator/renderer の unit・conformance test、tool-only
    派生物生成、workspace 全体検証をローカル作業ツリーで実行した。
  - commands: `just yaml-check-scl`, `just test-tools`, `just typecheck-tools`, `just yaml-check`,
    `just scl-render-tools`, `just verify`
  - environment: macOS arm64 workspace; Bun 1.3.14; Go race tests と frontend build を含む。
  - actor: Codex
  - source: pre-commit working tree based on `3cff03af`
  - result: passed
  - artifacts: `tools/ra/spec/ra.html`, `tools/yaml-check/spec/yaml-check.html`,
    `tools/scl-to-html/spec/scl-to-html.html`, `tools/scl-to-openapi/spec/scl-to-openapi.html`,
    `tools/scl-to-jsonschema/spec/scl-to-jsonschema.html`
