---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-24
priority: p2
depends_on: [wi-92312-purge-expired-users-by-a-job-and-make-purge-resumable]
change_kind: bugfix
affected_spec:
  - { path: docs/modules/tenancy/quota/README.md, requirement: REQ-TENANCY-013 }
---

# テナントの User の使用量を、実際の件数から再集計して補正する

## 動機

[[wi-22350-capture-provisioning-deliveries-in-the-mutation-transaction]] の調査で、User の使用量（`tenant_usages`）を User の保存と別々に確定している箇所が見つかった。
途中で失敗すると使用量が実際の件数からずれ、直す手段がない。
トランザクションで囲めば防げるが、次の理由でその方式は採らず、照合で回収する方針とした。

- Valkey に置くセッションなど、同一 DB の外の状態は囲めない。
- 囲む範囲を経路ごとに正しく保つ前提は、経路が増えるたびに破れる。

| 箇所 | 現状 | 失敗したときに残る状態 |
| --- | --- | --- |
| 作成での加算 | `CreateUser` と `ProvisionFederatedUser` が、User の保存より先に使用量を加算して確定する | User が存在しないのに使用量だけが増えたまま残り、テナントが上限に早く達する |
| 完全削除での減算 | `DeleteUser` は、減算を終えると Tombstone の `pending_purge` を進めて保存する（[[wi-92312-purge-expired-users-by-a-job-and-make-purge-resumable]]） | 減算の直後に Tombstone の保存が失敗すると、再実行が減算をもう一度行い、使用量が実際の件数より少なくなる |

完全削除を再実行で完了させる部分は wi-92312 が実装した。
この項目は、残った使用量のずれを再集計で回収する部分だけを扱う。

## 対象範囲

- テナントの User の使用量を、削除されていない User の実際の件数から数え直して補正する再集計を実装し、定期的に走らせる。
- `docs/modules/identity-management/user/design.md` の信頼性の節と `docs/modules/identity-management/design/risks.md` から、使用量を手作業で直す手順を、再集計で直る記述へ改める。
- `docs/design/application/design-guidelines.md` が、User の削除のカスケードを「1 つのトランザクションで変更してよい類型」として挙げている記述を、実装（ポートごとに確定し、再実行と照合で回収する）に合わせる。

## 対象外

- 完全削除の再実行。wi-92312 で実装済みである。
- User 以外のリソース（Group、Application など）の使用量の再集計。同じ形で広げられるが、まず User で確かめる。

## 設計

使用量 `tenant_usages` は、リソースを作るたびに全件を数えないための集計である（`docs/design/data/database.md`）。
集計が実数からずれても、定期的に実数から数え直して補正すれば、ずれは次の周期で消える。
作成時の加算と完全削除時の減算は、上限の判定のための近道として残す。

### 採用しない代替案

- **使用量の更新を User の保存と同じトランザクションに入れる。** Tenancy と IdManagement のポートをまたぐトランザクションを新たに作る。照合と冪等な再実行で回収する方針に反する。
- **User の保存に失敗したら使用量を減らし戻す補償処理。** 補償の書き込み自体が失敗し得るうえ、プロセスの異常終了では補償が走らない。

## 計画

着手時に次の問いを解決する。どれも作るものを変える。

1. 再集計を走らせる場所。全テナントを対象とする定期の処理なので、Jobs の判断に従えば `idmagic-batch` になる。既存の `retention-sweep` に加えるか、別のサブコマンドと CronJob にするか。
2. 再集計中に並行して作成または完全削除があったときの扱い。数え直しと上書きの間の加算と減算を失わない書き方（差分での補正、行ロックなど）を選ぶ。
3. 使用量の補正を、要件として REQ-TENANCY-013 か別の要件に書くか。

## タスク

- [ ] T001 [Spec] 計画の問い 3 で決めた要件を書き、`mise run check-spec` を通す。
- [ ] T002 [Acceptance] User の保存に失敗した作成で使用量だけが増え、再集計がないので戻らないことを RED として観測する。
- [ ] T003 [App] 使用量の再集計を実装し、定期実行に組み立てる。
- [ ] T004 [Docs] ユーザーの設計、IdManagement のリスク、設計ガイドラインのトランザクションの類型を現状に合わせる。
- [ ] T005 [Verify] 変更を検証する。

## 検証

- `mise run check-spec`
- `mise run test-go-changed`
- `mise run verify`

## リスク

- **再集計が並行する作成の加算や完全削除の減算を上書きで失う。** 計画の問い 2 で書き方を決め、並行する作成と完全削除を挟んだテストで確かめる。
