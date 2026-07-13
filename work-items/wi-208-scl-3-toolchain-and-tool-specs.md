---
status: pending
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

- [ ] T001 [RA] bundle loader、section kind、型を SCL 3.0 専用形へ更新する。
- [ ] T002 [HTML] 新 section/局所契約/意味参照を表示し、旧 invariant/UX renderer を撤去する。
- [ ] T003 [OpenAPI] interface access、resource、requires/ensures と新参照規則を生成へ反映する。
- [ ] T004 [JSONSchema] field/model constraints の生成と非対応 CEL の明示的扱いを実装する。
- [ ] T005 [Authorization] interface→policy 参照から AuthZEN/Cedar action metadata を生成する純粋な
  中間表現と単体 test を追加する。
- [ ] T006 [SCL] `tools/ra/spec/scl.yaml`、`tools/yaml-check/spec/scl.yaml`、
  `tools/scl-to-html/spec/scl.yaml`、`tools/scl-to-openapi/spec/scl.yaml`、
  `tools/scl-to-jsonschema/spec/scl.yaml` を3.0へ移行する。
- [ ] T007 [Test] snapshot/fixture を新 HTML・OpenAPI・schema・参照 anchor に同期する。
- [ ] T008 [Verify] 全 tool の test/typecheck と tool spec validation を通す。

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
