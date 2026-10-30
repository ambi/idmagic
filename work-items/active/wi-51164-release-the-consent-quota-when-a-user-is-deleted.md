---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-30
priority: p3
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/tenancy/features/quota.md, requirement: REQ-TENANCY-036 }
---

# 利用者の削除でも consents のクォータの枠を空ける

## 動機

[[wi-23517-release-the-consent-quota-when-the-user-revokes-a-consent]] の調査で、利用者の削除が同意を消しても `consents` の使用量を減らさないことが分かった。
`cascadeDeleteForSub` は `ConsentRepo.DeleteAllForSub` を呼ぶだけで、`ResourceConsents` の減算を行わない。
[REQ-TENANCY-036](../../docs/modules/tenancy/features/quota.md#req-tenancy-036-hard-quota-の実効値は上書きがあればその値なければデフォルト値である) は、リソースを削除したときに使用量から減算すると定める。

## 対象範囲

- 利用者の削除で消した `Granted` の同意の件数だけ、`consents` の使用量を減らす。

## 対象外

- 既存のテナントでずれた使用量の補正。

## 設計

未定である。
`DeleteAllForSub` が消した `Granted` の件数を返すか、削除の前に数えるかを、着手時に比べる。

## タスク

- [ ] T001 [App] 利用者の削除で `consents` の使用量を減らす。

## 検証

- `mise run test-go-package -- ./backend/idmanagement/internal/user/usecases`

## リスク

- `Revoked` の同意まで数えると、使用量が実際の件数を下回る。
