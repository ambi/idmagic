---
status: in_progress
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-10-04
priority: p2
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/sourcing/features/scim.md, requirement: REQ-SOURCING-002, impact: modifies }
  - { path: docs/modules/sourcing/features/scim.md, requirement: REQ-SOURCING-005, impact: modifies }
  - { path: spec/modules/sourcing/main.tsp, symbol: IdMagic.Sourcing.Operations.CreateScimUser }
  - { path: spec/modules/sourcing/main.tsp, symbol: IdMagic.Sourcing.Operations.CreateScimGroup }
  - { path: spec/modules/sourcing/main.tsp, symbol: IdMagic.Sourcing.Operations.ScimQuotaExceededResponse }
primary_use_cases:
  - id: user-owner-effects
    requirement: REQ-SOURCING-002
    observable_result: User の作成と属性変更が所有者のイベントと下流通知を伴い、作成が使用量を増やす。
    boundary: acceptance
    test: { path: backend/sourcing/internal/scim/usecases/owner_writes_test.go, name: TestSCIMUserWritesUseIdentityCommands, task: test-go-race }
    fault_model: SCIM が UserRepository.Save を直接呼び、所有者の作用を飛ばす。
  - id: group-owner-effects
    requirement: REQ-SOURCING-005
    observable_result: Group と所属の作成と変更が所有者のイベントと下流通知を伴う。
    boundary: acceptance
    test: { path: backend/sourcing/internal/scim/usecases/owner_writes_test.go, name: TestSCIMGroupWritesUseIdentityCommands, task: test-go-race }
    fault_model: SCIM が GroupRepository の書き込みを直接呼び、所属のイベントを飛ばす。
  - id: creation-rollback
    requirement: REQ-SOURCING-005
    observable_result: 対応表の保存に失敗すると User、ワークフロー、Group、初期所属と使用量が残らず、イベントと通知も出ない。
    boundary: adapter
    test: { path: backend/shared/http/server_http/sourcingdeps/writes_test.go, name: TestSCIMCreationRollsBackOnReferenceFailure, task: test-go-race }
    fault_model: 対応表を所有者の作成と別にコミットする。
  - id: quota-http-wiring
    requirement: REQ-SOURCING-002
    observable_result: 正式な SCIM の入口が User と Group の上限を 422 の SCIM エラーで拒否し、記録と使用量を増やさない。
    boundary: e2e
    reason: 正式なルーターから所有者の操作と使用量を接続する配線の欠落は、個々の操作のテストでは検出できない。
    test: { path: backend/shared/http/server_http/scim_writes_e2e_test.go, name: TestSCIMCreationEnforcesQuotaThroughTheRouter, task: test-go-race }
    fault_model: 正式な配線で QuotaRepository を渡し忘れる。
---

# SCIM による User と Group の作成と更新を IdManagement のユースケースへ通す

## 動機

SCIM の取り込みのうち、User の無効化と削除の予約だけは `UserLifecycle` のポートを通り、管理 API で同じ変更をしたときと同じイベントと通知が伴う。
一方、User と Group の作成と更新、Group のメンバーシップは、IdManagement の Repository を直接保存する。
Sourcing はドメインイベントを発行せず、リソース上限も確かめない。
SCIM で作った User と Group には、監査の記録、ライフサイクルワークフローの捕捉、下流への通知が伴わず、上限を超えて作れる。

また、Group の作成の途中で失敗したときの後始末（作った Group と対応の記録の削除）はエラーを捨てるので、外部の IdP から見えない孤立した Group が残りうる。

wi-26063 で Sourcing の内部設計をコードと照合して見つけ、`docs/modules/sourcing/risks.md` に載せた。

## 対象範囲

- SCIM の User と Group の作成、更新、メンバーシップの変更を、IdManagement のユースケースまたは公開する操作へ通す。
- 作成と対応の記録の保存を、片方だけが残らないように確定する。
- `docs/modules/sourcing/risks.md` の該当の行を消し、`scim/design.md` を直す。

## 対象外

- SCIM の属性の対応付けの変更。

## 設計

### 仕様の差分

REQ-SOURCING-002 に、User の作成と属性変更を IdManagement へ通すこと、作成の User と対応表を一緒に確定すること、作成上限を 422 の SCIM エラーで拒否することを書く。
REQ-SOURCING-005 に、Group の作成と更新、所属変更と削除を IdManagement へ通すこと、作成の Group と初期所属と対応表を一緒に確定すること、作成上限を同じ形式で拒否することを書く。
作成の TypeSpec 操作に 422 と `application/scim+json` の `ScimProtocolError` を加える。
SCIM に固有の上限用 `scimType` は新設せず、省略する。
根拠は REQ-TENANCY-013 の状態コードと [RFC 7644 の 3.12 節](https://www.rfc-editor.org/rfc/rfc7644.html#section-3.12)の省略可能な `scimType` である。

### 境界の比較

入力は REQ-SOURCING-002、005、REQ-IDMANAGEMENT-089 と REQ-TENANCY-013 である。
D1、D2、D4、D6、D7 を適用する。
SCIM の属性の解析と投影は Sourcing、User と Group の規則、上限、イベントと通知は IdManagement に残す。
Source の作成用操作は既存の User 作成の共通核を使い、パスワードを作らない。
属性の更新と Group の変更は既存の管理用ユースケースへ渡す。
既存の由来表示用の対応表と IdManagement の表の所有者は変えない。

| 案 | 保証と波及 | 採否 |
| --- | --- | --- |
| SCIM でイベントとクォータを実装する | 作成や所属の規則が二つになり、変更のたびに両方を直す | 不採用 |
| 書き込みを所有者へ渡し、対応表の保存を補償する | 補償自体の失敗とプロセス停止で片方だけが残る | 不採用 |
| 所有者の操作を使い、作成を同じ Tx へ束縛する | 規則を複製せず、User の保存とワークフロー捕捉の既存の確定も維持できる | 採用 |
| ジョブによる照合で回収する | 対応表なしの記録が残る時間と、再試行の識別を新たに規定する必要がある | 対象外 |

Sourcing が要求する書き込みは `IdentityWrites` に限定する。
作成は `CreateUser(ctx, scimID, user)` と `CreateGroup(ctx, scimID, group, members)`、変更は `UpdateUser`、`UpdateGroup`、`SetGroupMembership`、`DeleteGroup` とする。
組み立て地点の実装が、所有者の操作、対応表と既存のイベント出力先を結ぶ。
PostgreSQL の作成は外側の Tx を持ち、その Tx に束縛した所有者の保存先、クォータ、ワークフロー捕捉、対応表を使う。
イベントと下流への通知はコミット後に発行し、ロールバックした作成からは発行しない。
メモリ構成では、既存のメモリ保存先の失敗しない保存を利用し、入力の検証を保存より先に終える。
汎用のトランザクション基盤や、イベントバスは追加しない。

SCIM の属性対応、PATCH の対象の選択、PUT の省略値、active の既存のライフサイクル経路は維持する。
同一テナントの本人と Group の確認は所有者にも残す。
新しい DB 列やデータ移行は不要で、差分の取り消しで戻せる。
製品結果に関わる未決事項はない。
仕様にない上限時の形式と作成の確定は、今回の対象範囲を具体化する要件として分類 (a) にする。
SCIM に `UserCreated` などの発行を再実装する案は分類 (c) として採らない。

既存の `active=false` による Disabled の作成は維持し、REQ-SOURCING-002 に書き起こす（分類 (b)）。
既存 Group の複数所属の更新は今回の原子性の対象に含めず、補償の限界をリスクへ残す。

## 計画

1. SCIM の作成で、イベントが発行されないことと上限が効かないことを RED として確かめる。
2. 書き込みをユースケースへ移す。
3. 作成の原子性を直す。

## タスク

- [x] T001 [App] SCIM の書き込みをユースケースへ通す。
  - `TestSCIMUserWritesUseIdentityCommands` と `TestSCIMGroupWritesUseIdentityCommands` で REQ-SOURCING-002、005 のイベント、通知と使用量を観測する。直接保存する実装はイベントの欠落で失敗する。
  - `TestSCIMCreationEnforcesQuotaThroughTheRouter` で正式なトークンとルーターから 201 と 422 の SCIM 表現を確かめる。上限エラーの変換前は 500 で失敗する。
- [x] T002 [App] 作成の原子性を直す。
  - `TestSCIMCreationRollsBackOnReferenceFailure` で PostgreSQL の対応表だけを失敗させ、資源、使用量、ワークフロー、イベントと通知が残らないことと、再試行の成功を確かめる。
- [x] T003 [Verify] 検証する。
  - 局所検査、変更行の変異検査、Go 全体の race 検査と UI の既定ゲートを通す。Application の既存のファズシードのスキップは今回の対象外であり、SCIM の DB 検査の代替に数えない。
  - 集約検査で見つけた追加の TypeSpec 型の影響宣言と、既存のデータ設計の用語表記を直し、`check` と文書品質のテストを再実行する。Go の実装は変わらないため、成功した Go 全体の検査を繰り返さない。

## 検証

- `mise run verify`
- `mise run report-contract-evidence -- --changed --base-revision main --target backend/sourcing/module.go --target backend/shared/http/server_http/routes.go` で構成入口の参照不足を着手時に照合する。
- RED と GREEN は `mise run test-go-test`、所有者と Sourcing の局所検査は `mise run test-go-package` で選ぶ。逆依存の検査は `verify` の Go 全体の race 検査に含め、同じテストを `test-go-changed` では繰り返さない。
- TypeSpec 操作の変更後に `mise run generate-contract` と `mise run check-operation-references` を行う。
- 変更行のミューテーション検査の基準は `9a30fa37f` とし、SCIM のユースケースと組み立ての実装を調べる。
- 作成時の Tx の配線とワークフロー捕捉の切断は、構文の変異で表現できないため手で注入する。
- Tx の開始を外す故障では使用量の残存、ワークフロー捕捉の切断では再試行後の実行の欠落、クォータの切断では拒否するはずの作成の 201 を、それぞれ対応する主要ユースケースで検出する。
- 生存変異から、複数所属の削除を一件で打ち切る故障と、Group 削除後に対応表を残す故障の観測を補う。User 通知の失敗を記録するログ条件の反転は、通知の実行と操作結果を変えず、診断ログだけの観測不足として扱う。
- Sourcing の root の三つの条件は、ツールが実行する同一パッケージの検査では未実行になるため、条件を一つずつ反転して PostgreSQL の主要ユースケースから検査する。開始の成功を打ち切る故障は、誤った成功応答に加えて接続を解放しなくなるため、失敗の観測後にその検証プロセスを終了する。
- HTTP の quota 判定は選択した変異ツールが変異を生成しないため、422 への変換前の 500 の失敗と、クォータ配線を外した 201 の故障注入で補う。SCIM と User の変更行の分岐、組み立てとルーターはそれぞれのパッケージで検査する。
- `TestSCIMCommittedGroupEventsSurviveNotificationFailure` で、Group の確定後の通知が失敗しても初期所属のイベントと通知を続け、対応表を保つことを確かめる。
- 複数モジュールの保存と正式な構成入口を変更するため、最後に `verify` で静的検査と Go 全体の race 検査を行う。UI の振る舞いは変えないためブラウザーテストは追加しない。

## リスク

上限が効くようになると、既存の外部の IdP の同期が上限で止まりうる。
公開状態は `unpublished` なのでリリース文書は追加せず、現在の要件に上限時の応答を記す。

## 要約

REQ-SOURCING-002、005 に、SCIM の書き込みを IdManagement の操作へ通し、使用量、イベント、ワークフロー捕捉と下流通知を共有することを定めた。
User と対応表、または Group と初期所属と対応表を同じ PostgreSQL トランザクションで確定し、イベントと通知を確定後へ遅らせる。
User と Group の作成上限は、TypeSpec と要件に定めた 422 の SCIM エラーで拒否する。
既存の `active=false` による Disabled の作成は要件へ書き起こして維持した。
既存 Group の複数所属の更新は原子性の対象外とし、補償の限界を現在のリスクへ残した。
