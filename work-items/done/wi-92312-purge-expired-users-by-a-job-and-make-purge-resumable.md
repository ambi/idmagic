---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-04
priority: p1
depends_on: [wi-17076-limit-specifications-to-external-contracts-written-once]
change_kind: bugfix
evidence_policy: risk-based-v4
documentation_impact:
  level: release_note
  reason: 猶予期間を過ぎた削除予約の User を、ユーザー一覧の取得ではなく Batch の `retention-sweep` が完全削除するようになる。途中で失敗した完全削除を、再実行で完了できるようになる。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-92312-purge-expired-users-by-a-job-and-make-purge-resumable.md }
initial_context:
  specification:
    - docs/domain/identity-management/user/README.md#REQ-IDMANAGEMENT-005
    - docs/domain/identity-management/user/lifecycle.md#REQ-IDMANAGEMENT-050
    - docs/domain/identity-management/user/acceptance.feature.md
    - docs/domain/identity-management/user/design.md
    - docs/domain/identity-management/csv-transfer/README.md#REQ-IDMANAGEMENT-080
    - docs/domain/jobs/design/decisions.md
    - docs/design/architecture/runtime.md
  typespec: []
  source:
    - backend/idmanagement/user/usecases/admin_users.go
    - backend/idmanagement/user/usecases/user_lifecycle_commands.go
    - backend/idmanagement/user/handlers_http/admin_user_handler.go
    - backend/idmanagement/user/domain/users.go
    - backend/idmanagement/user/ports/user_repository.go
    - backend/idmanagement/user/db_memory/users.go
    - backend/idmanagement/user/db_postgres/users.go
    - backend/idmanagement/user/db_postgres/users.sql
    - backend/cmd/internal/bootstrap/retention.go
    - backend/cmd/internal/bootstrap/user_lifecycle.go
    - backend/cmd/idmagic-batch/main.go
  tests:
    - backend/idmanagement/user/usecases/user_rules_test.go
    - backend/idmanagement/handlers_http/implicit_rules_test.go
    - backend/idmanagement/user/testing_contract/contract.go
    - backend/cmd/internal/bootstrap/retention_test.go
  stop_before_reading: [frontend, spec, backend/jobs]
affected_spec:
  - { path: docs/domain/identity-management/user/README.md, requirement: REQ-IDMANAGEMENT-044 }
  - { path: docs/domain/identity-management/user/lifecycle.md, requirement: REQ-IDMANAGEMENT-050 }
primary_use_cases:
  - id: retention-sweep-purges-expired-users
    requirement: REQ-IDMANAGEMENT-044
    observable_result: 保持期限の削除を実行すると、一覧を取得しなくても、各テナントで猶予期間を過ぎた `PendingDeletion` の User が `Deleted` になり、操作者 `system`、理由 `auto_purge` の `UserDeleted` が発行される。
    boundary: acceptance
    test: { path: backend/cmd/internal/bootstrap/retention_test.go, name: TestRetentionSweepPurgesUsersPastTheGracePeriod, task: test-go-race }
    fault_model: 保持期限の削除が User の完全削除を呼ばず、期限を過ぎた User が残る。
  - id: listing-users-does-not-purge
    requirement: REQ-IDMANAGEMENT-044
    observable_result: 管理者がユーザー一覧を取得しても、猶予期間を過ぎた `PendingDeletion` の User は `PendingDeletion` のまま残り、`UserDeleted` は発行されない。
    boundary: acceptance
    test: { path: backend/idmanagement/handlers_http/implicit_rules_test.go, name: TestListingUsersDoesNotPurgeExpiredPendingDeletions, task: test-go-race }
    fault_model: 一覧のハンドラーが、一覧を作る前に完全削除を呼び続ける。
  - id: resume-interrupted-purge
    requirement: REQ-IDMANAGEMENT-050
    observable_result: 匿名化の後に関連する記録の削除で失敗した完全削除を再び要求すると、残った記録が消え、使用量が一度だけ減る。使用量の減算の後に `UserDeleted` の発行で失敗した完全削除を再び要求すると、使用量はもう減らず、最初の要求の操作者と理由で `UserDeleted` が一度だけ発行される。
    boundary: acceptance
    test: { path: backend/idmanagement/user/usecases/user_rules_test.go, name: TestDeleteUserResumesAPurgeThatFailedAfterAnonymizing, task: test-go-race }
    fault_model: 削除済みの User の完全削除が何もせずに成功し、残りの手順を行わない。または、再実行が使用量をもう一度減らす。
  - id: purge-candidates-include-unfinished-tombstones
    requirement: REQ-IDMANAGEMENT-044
    observable_result: 永続化のアダプターが、テナントの `PendingDeletion` の User と、完全削除を終えていない `Deleted` の User を返し、完全削除を終えた User とほかのテナントの User を返さない。
    boundary: adapter
    test: { path: backend/idmanagement/user/db_memory/contract_test.go, name: TestPersistenceContract, task: test-go-race }
    fault_model: 候補の検索が、完全削除を終えていない `Deleted` の User を返さず、保持期限の削除が途中で失敗した完全削除を再開しない。
---

# 猶予期間を過ぎた User を定期ジョブで完全削除し、途中で失敗した完全削除を再実行で完了できるようにする

## 動機

User の完全削除には、二つの欠陥がある。

| 対象 | 今の挙動 | 起きる問題 |
| --- | --- | --- |
| 猶予期間を過ぎた User | 管理者がユーザー一覧を取得したときにだけ完全削除する | 一覧を取得しないテナントでは、期限を過ぎた User の個人情報が残り続ける。読み取りの操作が副作用を兼ねている |
| 途中で失敗した完全削除 | Tombstone を保存した後に関連する記録の削除や使用量の減算が失敗すると、再実行しても何もせずに成功する | 残った記録と使用量を、手作業で直すほかない |

## 対象範囲

- 猶予期間を過ぎた `PendingDeletion` の User を、定期ジョブで完全削除する。
- ユーザー一覧の取得から完全削除を外す。
- 完全削除を、途中で失敗しても再実行で残りの手順を完了できるようにする。
- REQ-IDMANAGEMENT-044 と REQ-IDMANAGEMENT-050、UserLifecycle の状態遷移表、`user/design.md` の信頼性の節を改める。

## 対象外

- Agent や Group の完全削除。
- 使用量のカウンターを User の実際の件数と照合する仕組み。完全削除の後始末は、この作業で再実行から回収できるようになる。
- 完全削除を、ポートをまたぐ一つのトランザクションにすること。

## 設計

### 定期実行の置き場所

起票時は `Jobs` Context の仕組みで実行すると書いたが、`Jobs` の判断「全テナントを対象とする定期の処理を永続ジョブに混ぜない」が、この種の処理を `idmagic-batch` に置くと定めている。
保持期限で個人情報を消す処理なので、CSV の成果物の削除（REQ-IDMANAGEMENT-080）と同じく、既存の `retention-sweep`（デフォルトの配置では毎時）に加える。
新しいサブコマンドと CronJob は作らない。

### 要件の差分

| 要件 | 変更 | 変更後の要件文 |
| --- | --- | --- |
| REQ-IDMANAGEMENT-044 | 完全削除の契機を一覧の取得から保持期限の削除へ移し、`user/README.md` の一覧の節から、状態遷移表の列と同じ名前の「保持期限の削除」の節へ移す。題名も改める | 題名：猶予期間を過ぎた削除予約の User は、保持期限の削除が完全削除する<br>保持期限の削除を実行したとき、IdManagement は、すべてのテナントで、`PendingDeletion` になってから猶予期間の 30 日を過ぎた User を完全削除する。<br>保持期限の削除を実行したとき、IdManagement は、猶予期間の終わりの時刻ちょうどの User と、`PendingDeletion` になった時刻を記録していない User を完全削除しない。<br>保持期限の削除が User を完全削除したとき、IdManagement は、`UserDeleted` の操作者に `system`、理由に `auto_purge` を記録する。<br>保持期限の削除を実行したとき、IdManagement は、匿名化の後に完全削除が失敗した User の、残りの手順を行う。<br>ある User の完全削除が失敗した場合、IdManagement は、ほかの User の完全削除を続け、保持期限の削除を失敗として終える。<br>管理者がユーザー一覧を取得したとき、IdManagement は、User を完全削除しない。<br>判断：読み取りの操作に削除を兼ねさせると、一覧を取得しないテナントでは個人情報が残り続ける。デフォルトの配置は保持期限の削除を毎時実行するので、猶予期間を過ぎた User が残るのは最長で約 1 時間である |
| REQ-IDMANAGEMENT-050 | 再実行の条項を加え、何もしない条件を完全削除を終えた User に限る | 匿名化の後に完全削除が失敗した User の完全削除を再び要求されたとき、IdManagement は、終えていない手順だけを行い、使用量を一度だけ減らし、最初の要求の操作者と理由で `UserDeleted` を発行する。<br>完全削除を終えた `Deleted` の User の完全削除を要求された場合、IdManagement は、成功を返し、`UserDeleted` を発行しない。 |

例は、EX-IDMANAGEMENT-044-01 と EX-IDMANAGEMENT-044-02 の契機を保持期限の削除へ改め、一覧に含まれないことの表明を外す。
再実行の例として EX-IDMANAGEMENT-050-02 を加える。

UserLifecycle の状態遷移表は、列「一覧の取得」を「保持期限の削除」に改める。
遷移の表のガードは、猶予期間ちょうどの User を完全削除しない要件に合わせて `>=` を `>` に直す。

### 完全削除の再開

完全削除の手順のうち、冪等でないのは使用量の減算と `UserDeleted` の発行である。
関連する記録の削除（`DeleteAllForSub`）は何度行っても同じ結果になる。
そこで、Tombstone に、まだ終えていない手順と、`UserDeleted` に記録する操作者と理由を記録し、冪等でない手順を終えるたびに更新する。

| 型と操作 | 内容 |
| --- | --- |
| `userdomain.PendingPurge{Step PurgeStep; ActorUserID string; Reason string}` | 完全削除を終えていない Tombstone に付ける。`UserLifecycle.PendingPurge *PendingPurge`（JSON の `pending_purge`）として `users.lifecycle` の JSONB に入るので、テーブルの列は増えない |
| `PurgeStepReleaseUsage`（`release_usage`） | 次に、関連する記録を消し、使用量を一つ減らす |
| `PurgeStepAnnounce`（`announce`） | 次に、`UserDeleted` を発行し、下流のプロビジョニングへ通知する |
| `DeleteUser(ctx, deps AdminUserDeps, in DeleteUserInput) error` | `Deleted` で `PendingPurge` がなければ何もしない。`PendingPurge` があれば、記録された手順から再開する。それ以外は、Agent を無効化し、`PendingPurge{Step: release_usage}` を付けた Tombstone を保存してから、残りの手順を行う |
| `finishPurge(ctx, deps, tombstone *userdomain.User, now time.Time) error`（非公開） | `release_usage` なら記録を消して使用量を減らし、`announce` にして保存する。`announce` なら `UserDeleted` を発行して通知し、`PendingPurge` を消して保存する |
| `PurgeExpiredSoftDeleted(ctx, deps, now) error` | テナントの候補を `UserRepository.ListPurgeCandidates` で引き、猶予期間を過ぎた `PendingDeletion` の User と、`PendingPurge` を持つ Tombstone を `DeleteUser` にかける。失敗した User があっても残りを続け、失敗を `errors.Join` で返す |
| `UserRepository.ListPurgeCandidates(ctx, tenantID string) ([]*userdomain.User, error)` | テナントの `PendingDeletion` の User と、`pending_purge` を持つ `Deleted` の User を返す。猶予期間の判定はユースケースに残す |
| `RunRetentionSweepOnce(ctx, deps *Dependencies, now time.Time) error` | `TenantRepo.FindAll` の各テナントで `PurgeExpiredSoftDeleted` を呼ぶ。依存は `UserLifecycleCommands` と同じ集合にする |

作用の境界は既存のまま、時刻は `now` の引数、永続化は `UserRepository` と各ストアのポート、通知は `Emit` と `ProvisioningNotifier` である。

再開できない残りは次のとおりで、`user/design.md` の信頼性の節に書く。

- 冪等でない手順が成功した直後の Tombstone の保存が失敗すると、再実行がその手順をもう一度行う。
- `UserDeleted` の発行に続く反応（所有者の削除による失効エポックの前進）が失敗すると、再実行が `UserDeleted` をもう一度発行する。

### 採用しない代替案

| 代替案 | 採用しない理由 |
| --- | --- |
| 完了の印を真偽値一つにする | 使用量の減算の後に `UserDeleted` の発行が失敗すると、再実行が使用量をもう一度減らす |
| 関連する記録の削除を Tombstone の保存より前に行う | 削除と保存の間に、まだ匿名化していない User がセッションを作れる |
| 使用量の減算と Tombstone の保存を一つのトランザクションにする | Tenancy と IdManagement のポートをまたぐトランザクションを新たに作る。照合と冪等な再実行で回収する方針に反する |
| `PendingDeletion` と `Deleted` の間に状態を足す | `UserStatus` は管理 API の絞り込みと応答に現れる。内部の後始末のために公開する状態を増やさない |
| 既存の `FindAll` で全 User を読む | 毎時、全テナントの全 User を読み込む。`FindAll` は `Deleted` の User を返さないので、再開する Tombstone も見つからない |

### 仕様の漏れを探す観点で見つけたこと

| 観点 | 見つけたこと | 分類 |
| --- | --- | --- |
| 時刻 | 遷移の表のガードは猶予期間ちょうどで完全削除を許すが、EX-IDMANAGEMENT-044-02 と実装は残す | (a) 要件と例に合わせてガードを `>` に直す |
| 後始末 | 保持期限の削除が途中で失敗した完全削除を見つけられないと、自動の完全削除の失敗が回収されない | (a) REQ-IDMANAGEMENT-044 に再開の条項を書く |
| エラー | 一人の User の失敗が、以後のテナントのすべての完全削除を止める | (a) 続けて失敗を返す条項を書く |
| 状態と操作の組 | 再開は `Deleted` の行の結果を変えるように見えるが、状態は変わらず、最初の完全削除が発行すべきだった `UserDeleted` を発行するだけである | 状態遷移表は「何もしない」のまま、REQ-IDMANAGEMENT-050 の条項で述べる |

## 計画

1. 要件、例、状態遷移表、設計の差分を仕様へ適用する。
2. 一覧の取得から完全削除を外し、保持期限の削除へ加える。
3. 完全削除を再開できるようにし、候補の検索を加える。

## タスク

- [x] T001 [Spec] 要件、状態遷移表、設計の差分を書く。`mise run check-spec` は、テストが EX-IDMANAGEMENT-050-02 を引くまで「no test names it」で失敗し、テストを書いた後に通った。
- [x] T002 [Acceptance] 一覧を取得しなくても期限を過ぎた User が完全削除されることの RED を確認する（REQ-IDMANAGEMENT-044）。`TestRetentionSweepPurgesUsersPastTheGracePeriod` は `expired-a` が `pending_deletion` のまま残って失敗し、`TestListingUsersDoesNotPurgeExpiredPendingDeletions` は一覧が削除予約の User を含まず（一覧の取得が完全削除した）失敗した。
- [x] T003 [App] 再実行で完了する完全削除と、候補の検索を実装する。RED：`TestDeleteUserResumesAPurgeThatFailedAfterAnonymizing`（REQ-IDMANAGEMENT-050）は、記録の削除で失敗した後の再実行でセッションが残り、`UserDeleted` の発行で失敗した後の再実行で `UserDeleted=0, want 1` となって失敗した。`TestPurgeExpiredSoftDeletedResumesAPurgeThatFailedAfterAnonymizing` は `UserDeleted=0`、`TestPurgeExpiredSoftDeletedContinuesPastAFailingUser` は後の User が残って失敗した（REQ-IDMANAGEMENT-044）。`TestPersistenceContract` は `ListPurgeCandidates` がなくコンパイルに失敗した。
- [x] T004 [App] 保持期限の削除へ配線し、一覧のハンドラーから外す。状態遷移表の列を改めたことで `TestUserLifecycleFollowsTheStateMatrixInEveryCell` が「操作 "保持期限の削除" を起こす要求をテストが知らない」で失敗したので、列をユースケースの呼び出しへ対応させた。
- [x] T005 [Verify] 変更を検証する。

## 検証

- 各 RED と GREEN：`mise run test-go-test -- <package> <test>`
- 振る舞いが GREEN になった後：`mise run test-go-package -- <package>`、`mise run lint-go`、`mise run test-go-changed`
- PostgreSQL のアダプター：`mise run test-go-package -- ./backend/idmanagement/user/db_postgres`（サンドボックスの外）、`mise run check-schema`
- 変更への耐性：`mise run test-go-mutation -- backend/idmanagement/user/usecases`
- `mise run check-work-items`
- `mise run verify`

## リスク

- 保持期限の削除の間隔だけ、猶予期間を過ぎた User が残る。
  デフォルトの間隔（毎時）を要件の判断に書き、利用者が観測できる上限として示す。
- 既存の配置が `retention-sweep` を動かしていなければ、期限を過ぎた User が完全削除されなくなる。
  リリースノートで知らせる。

## 完了

- **Completed At**: 2026-10-07
- **Summary**:
  `mise run spec-diff -- main` の結果、REQ-IDMANAGEMENT-044 と REQ-IDMANAGEMENT-050 の要件文と、UserLifecycle の状態遷移が変わった。
  REQ-IDMANAGEMENT-044 は、猶予期間を過ぎた削除予約の User を、ユーザー一覧の取得ではなく保持期限の削除（Batch の `retention-sweep`）がすべてのテナントで完全削除すると定める。保持期限の削除は匿名化の後に失敗した完全削除を再開し、一人の失敗で残りを止めずに失敗を返す。ユーザー一覧の取得は User を完全削除しない。
  REQ-IDMANAGEMENT-050 は、匿名化の後に失敗した完全削除の再実行が、終えていない手順だけを行い、使用量を一度だけ減らし、最初の要求の操作者と理由で `UserDeleted` を発行すると定める。
  UserLifecycle の状態遷移表は列「一覧の取得」を「保持期限の削除」に改め、遷移の表のガードを猶予期間ちょうどで完全削除しない `>` に直した。
  実装は、Tombstone の `lifecycle` に残りの手順と操作者と理由（`pending_purge`）を記録し、冪等でない使用量の減算と `UserDeleted` の発行を終えるたびに進める。候補は新しい `UserRepository.ListPurgeCandidates` と部分インデックス `users_tenant_purge_candidates_idx` で引く。
- **Primary Use Case Evidence**:
  - id: retention-sweep-purges-expired-users
    red: 実装前に TestRetentionSweepPurgesUsersPastTheGracePeriod が、`expired-a` が `pending_deletion` のまま残ることで失敗した。
    fault_injection: 保持期限の削除の `RunRetentionSweepOnce` から `purgeExpiredUsers` の呼び出しを外すと、TestRetentionSweepPurgesUsersPastTheGracePeriod が同じ表明で失敗した。
  - id: listing-users-does-not-purge
    red: 実装前に TestListingUsersDoesNotPurgeExpiredPendingDeletions が、一覧の取得が User を完全削除して一覧から消えたことで失敗した。この RED は、一覧のハンドラーが完全削除を呼ぶ故障そのものを注入した状態での観測である。
    fault_injection: 一覧のハンドラーの `PurgeExpiredSoftDeleted` の呼び出しが残っている状態（変更前のハンドラー）で、TestListingUsersDoesNotPurgeExpiredPendingDeletions が失敗した。
  - id: resume-interrupted-purge
    red: 実装前に TestDeleteUserResumesAPurgeThatFailedAfterAnonymizing が、記録の削除で失敗した後の再実行でセッションが残り、`UserDeleted` の発行で失敗した後の再実行で `UserDeleted=0, want 1` となって失敗した。
    fault_injection: 完全削除の `finishPurge` で手順を `announce` へ進めて保存する処理を消すと、TestDeleteUserResumesAPurgeThatFailedAfterAnonymizing/UserDeleted_の発行で失敗した が `users usage=0, want 1 (released only once)` で失敗した。
  - id: purge-candidates-include-unfinished-tombstones
    red: 実装前に TestPersistenceContract が、`ListPurgeCandidates` がないためコンパイルに失敗した。
    fault_injection: メモリ実装の候補の条件から `PendingPurge != nil` を外すと、db_memory の TestPersistenceContract と TestPurgeExpiredSoftDeletedResumesAPurgeThatFailedAfterAnonymizing が失敗した。PostgreSQL の生成クエリから `OR lifecycle ? 'pending_purge'` を外すと、db_postgres の TestPersistenceContract が失敗した。
- **Change-Resistance Results**:
  `mise run test-go-mutation -- backend/idmanagement/user/usecases` は 392 の変異のうち 347 を検出し、30 が生き残り、8 が未被覆だった。
  変更した `DeleteUser`、`finishPurge`、`savePendingPurge`、`PurgeExpiredSoftDeleted` に生き残った変異はない。
  `admin_users.go` で生き残ったのは、変更していない `notifyProvisioning`、`revokeTrustedDevicesOnDisable`、`changedAttributeFields`、猶予期間の定数の `*`→`/`、`cascadeDeleteForSub` の nil 判定であり、いずれもこの作業の範囲外の既存の不足である。`cascadeDeleteForSub` の nil 判定の反転は、未配線のストアを呼んで nil 参照になる変異で、テストのフィクスチャが一部のストアしか配線しないため生き残る。
  手作業の故障注入（配線の除去、手順の印を進めない、候補の条件の除去）は、Primary Use Case Evidence のとおりすべて検出された。
  あわせて、既存の EX-IDMANAGEMENT-050-01 のテストはセッションの有効期限を `userRulesNow` 基準で作っており、メモリのセッションストアが実時計で期限切れと判定するため、セッションが消えたことの表明が空振りしていた。有効期限を実時計基準に直した。
- **Verification Results**:
  - `mise run verify` - 成功（初回は「既定」の表記で `check-repository` が失敗し、「デフォルト」に直した後に成功）
  - `mise run test-go-changed` - 成功（サンドボックスの外で DB テストを含めて実行）
  - `mise run test-go-package -- ./backend/idmanagement/user/db_postgres` - 成功（埋め込み PostgreSQL に新しいインデックスを含むスキーマを投入）
  - `mise run check-schema` - 未実行。Docker のデーモンが起動しておらず接続できなかった
  - `mise run test-ui-e2e` - 未実行。ブラウザに届く変更（UI と管理 API の応答の形）はない
