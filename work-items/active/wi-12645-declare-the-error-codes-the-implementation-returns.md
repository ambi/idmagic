---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-04
priority: p2
depends_on: [wi-35451-rewrite-remaining-context-specifications-in-the-requirement-format]
change_kind: bugfix
affected_spec:
  - { path: docs/modules/workloadidentity/trust-configuration/README.md, requirement: REQ-WORKLOADIDENTITY-008 }
  - { path: docs/modules/workloadidentity/trust-configuration/README.md, requirement: REQ-WORKLOADIDENTITY-009 }
  - { path: docs/modules/application/catalog/README.md, requirement: REQ-APPLICATION-008 }
  - { path: docs/modules/application/sign-in-policy/README.md, requirement: REQ-APPLICATION-009 }
  - { path: docs/modules/identity-management/data-export/README.md, requirement: REQ-IDMANAGEMENT-039 }
---

# 実装が返すエラーコードを、返す Context の TypeSpec で宣言する

## 動機

wi-35451 の書き直しで、要件に書いたエラーコードのうち、TypeSpec が宣言していないものと、返さない Context が宣言しているものが見つかった。
宣言のないエラーコードは、公開の契約（OpenAPI）に現れず、クライアントは依存してよいかを判断できない。

| エラーコード | 返す Context | 食い違い |
| --- | --- | --- |
| `workload_trust_bundle_not_found`、`agent_workload_binding_not_found`、`agent_workload_binding_pattern_conflict` | WorkloadIdentity | 宣言がない |
| `invalid_icon`、`invalid_sign_in_policy` | Application | 宣言がない |
| `jobs_unavailable` | IdManagement（データのエクスポート） | Jobs の TypeSpec が宣言している |

## 対象範囲

- 上の表のエラーコードを、返す Context の TypeSpec の操作のエラーとして宣言する。
- `jobs_unavailable` の宣言を IdManagement のデータのエクスポートの操作へ移し、語彙の検査の許容リストから外す。

## 対象外

- エラーコードの名前や HTTP の状態の変更。

## 設計

既存のエラーモデルの宣言の形に合わせ、操作の `@error` の和に加える。
OpenAPI の差分は追加だけになることを `mise run check-api-compat` で確かめる。

## タスク

- [ ] T001 [Spec] TypeSpec の宣言を加え、移す。
- [ ] T002 [Verify] `mise run check-contract-drift`、`mise run check-api-compat`、`mise run check-unspecified-vocabulary` を通す。

## 検証

- `mise run verify`

## リスク

- 宣言を移すと、生成したクライアントの型名が変わる場合がある。生成物の差分を確かめる。
