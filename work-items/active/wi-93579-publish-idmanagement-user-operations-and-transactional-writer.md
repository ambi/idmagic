---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-10
priority: p2
depends_on: []
change_kind: refactor
spec_impact: { kind: none, reason: "ユーザーのプロファイルの取得と、呼び出し側のトランザクションに参加するユーザーの保存を、IdManagement の公開契約の経由へ変えるだけである。ユーザーの不変条件、HTTP の応答、永続状態、ドメインイベントは変えない。" }
---

# IdManagement のユーザー操作とトランザクションに参加する保存を公開する

## 動機

`tools/check/boundary-debt.json` の `private-import` のうち 6 件は、Authentication と IdGovernance から IdManagement の非公開パッケージへの依存である。

| 利用側 | 参照先 | 使っているもの |
| --- | --- | --- |
| Authentication の `deps_http` と handler | `idmanagement/usecases`、`idmanagement/user/usecases` | アカウントのプロファイルの取得（`GetUserProfile`、`AccountProfileDeps`、`ErrUserNotFound`） |
| Authentication の `password/db_postgres` | `idmanagement/user/db_postgres` | パスワードの変更と同じトランザクションでのユーザーの保存（`SaveUserTx`） |
| IdGovernance の `db_postgres` と `db_memory` | `idmanagement/user/db_postgres`、`idmanagement/user/db_memory` | ライフサイクルの手順と同じトランザクションでのユーザーの保存 |

## 対象範囲

- プロファイルの取得を IdManagement の公開操作にする。
- 呼び出し側のトランザクションに参加するユーザーの保存を、IdManagement が公開し、組み立て地点で結ぶ。
- 解消した違反 ID を台帳から消す。

## 対象外

- ユーザーの不変条件と保存表現の変更。

## 設計

[境界の負債の順位付け](../done/wi-33994-reinventory-the-remaining-boundary-debt.md)の判断を引き継ぐ。

- D2：ユーザーの不変条件の検証と更新は IdManagement に残す。
- D7：パスワードの変更とライフサイクルの手順は、ユーザーの保存と同じトランザクションで確定する必要がある。別のトランザクションで呼ぶ案は、片方だけが確定する窓を作るので採らない。
- [テナントの公開契約](../done/wi-39119-publish-the-resolved-tenant-as-tenancy-public-language.md)で Tenancy の使用量に使った形（呼び出し側のトランザクションを受け取る構築関数を組み立て地点が渡す）を、ユーザーの保存にも使う。

| 案 | 判断 |
| --- | --- |
| トランザクションに参加する保存を公開し、組み立て地点で結ぶ | 採る |
| `SaveUserTx` をそのまま公開パッケージへ移す | 採らない。保存表現の詳細と sqlc の型が利用側へ漏れる |

## タスク

- [ ] T001 [Characterize] 変更する保存の経路が、トランザクションの巻き戻しをテストで固定しているかを確かめ、固定していなければ特性化テストを置く。
- [ ] T002 [App] 公開操作とトランザクションに参加する保存を置き、利用側と組み立て地点を書き換える。
- [ ] T003 [Tooling] 解消した違反 ID を台帳から消す。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `mise run check-boundaries`
- `mise run check-boundary-debt-ratchet`
- `mise run verify`

## リスク

- 保存を呼び出し側のトランザクションの外で行う誤りは、通常の経路のテストでは見えない。
  書き込みの失敗で全体が巻き戻ることを特性化テストで固定し、トランザクションの外で保存する変異を手で入れて検出を確かめる。
