---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-15
priority: p3
depends_on: [wi-13438-publish-oauth2-consent-revocation-and-client-administration]
change_kind: bugfix
affected_spec:
  - { path: docs/modules/oauth2/features/consent.md, requirement: REQ-OAUTH2-032 }
  - { path: docs/modules/tenancy/features/quota.md, requirement: REQ-TENANCY-036 }
---

# 本人による同意の撤回でも consents のクォータの枠を空ける

## 動機

[[wi-13438-publish-oauth2-consent-revocation-and-client-administration]] の着手時の調査で、同意の撤回が経路によってクォータの扱いを変えていることが分かった。

| 経路 | 撤回の操作へ渡すクォータの保存先 | `Granted` の同意を撤回したときの `consents` の使用量 |
| --- | --- | --- |
| 管理者の撤回（OAuth2 の管理 API） | 渡す | 1 減る |
| 本人の撤回（アカウントポータル） | 渡さない | 変わらない |

同意を付与するときは、どちらの経路で撤回されるかにかかわらず使用量へ 1 を足す。
本人が撤回するたびに使用量が実際の `Granted` の件数より大きくなり、Hard Quota の 10,000 件に実際より早く達する。
[REQ-TENANCY-036](../../docs/modules/tenancy/features/quota.md#req-tenancy-036-hard-quota-の実効値は上書きがあればその値なければデフォルト値である) は、リソースを削除したときに使用量から減算すると定める。
一方で、本人の撤回を定める [REQ-OAUTH2-032](../../docs/modules/oauth2/features/consent.md#req-oauth2-032-ユーザーは接続済みアプリの同意を自分で撤回できる) は、クォータについて何も述べていない。

## 対象範囲

- 本人の撤回でも、`Granted` から `Revoked` へ遷移したときに `consents` の使用量を 1 減らす。
- 撤回で使用量が減ることを REQ-OAUTH2-032 へ書く。
- 既に撤回済みの同意への再度の撤回では使用量を変えない。

## 対象外

- 既存のテナントでずれた使用量の補正。製品は未公開なので、移行は行わない。
- 使用量を実際の件数から再集計する仕組み。

## 設計

wi-13438 が公開する本人用の同意の操作は、OAuth2 のルートパッケージがクォータの保存先なしで組み立てる。
組み立てにクォータの保存先を渡し、管理者の撤回と同じ `RevokeConsent` の分岐を通す。

## タスク

- [ ] T001 [Spec] REQ-OAUTH2-032 に、撤回で `consents` の使用量を減らすことを書く。
- [ ] T002 [App] 本人用の同意の操作にクォータの保存先を渡す。
- [ ] T003 [Verify] 本人の撤回の前後で使用量が 1 減ることを、アカウントポータルの撤回の境界で確かめる。

## 検証

- `mise run check-spec`
- `mise run test-go-package -- ./backend/authentication/internal/handlers_http`

## リスク

- 撤回の再実行で二重に減算すると、使用量が実際の件数を下回る。
  `Granted` からの遷移のときだけ減らす既存の分岐を共有する。
