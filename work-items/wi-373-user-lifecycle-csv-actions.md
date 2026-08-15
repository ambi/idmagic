---
status: pending
authors: [tn]
risk: high
created_at: 2026-08-15
depends_on: [wi-284-improve-csv-import-export]
change_kind: feature
affected_spec:
  - { path: spec/contexts/identity-management/SPECIFICATION.md, requirement: REQ-IDMANAGEMENT-004 }
  - { path: spec/contexts/identity-management/models.tsp, symbol: IdMagic.Contract.UserImportJob }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.Contract.ApplyAdminUserImport }
---

# User CSV に明示的な lifecycle action 列を追加する

## Motivation

[[wi-284-improve-csv-import-export]] で User CSV は安全な部分 upsert になったが、扱えるのは作成と更新だけである。`status` 列は読み取り専用として受理するだけで適用せず、無効化・削除・復元・完全削除の CSV lifecycle action は当時明示的に対象外とした。そのため退職者の一括無効化や大量の削除予約は、管理 API を 1 件ずつ呼ぶか管理画面を 1 人ずつ操作するしかない。

一方でこれらは破壊的で、多くは不可逆な操作である。`UserLifecycle` 状態機械の遷移ガード、`PendingDeletion` の猶予期間 30 日、自分自身を削除できない自爆防止、外部の取り込み元が管理する `User` の保護、そして `admin:user_delete` / `admin:user_restore` / `admin:user_purge` という操作別の権限分割は、いずれも既存の安全境界である。CSV という一括経路がこれらを暗黙に迂回してはならない。CSV 経路が `admin:user_import` だけで削除まで実行できてしまうと、[[wi-372-admin-api-granular-scope-enforcement]] で確定した操作別の権限分割そのものが無意味になる。

## Scope

- User CSV スキーマへの書き込み可能列 `lifecycle_action` の追加。値は `disable` / `enable` / `soft_delete` / `restore` / `purge` の閉じた集合とし、列が無い場合と空セルはいずれも lifecycle を変更しない。
- 行計画の操作種別に `disabled` / `enabled` / `soft_deleted` / `restored` / `purged` を追加し、プレビューと適用の結果が返す操作別件数を対応させる。
- `UserLifecycle` の遷移ガードの再利用。現在の状態から到達できない遷移、猶予期間を満たさない `purge`、既に同じ終着状態にある行は、安定したエラーコードで拒否するか `unchanged` と判定する。
- 行単位の権限判定。`disable` / `enable` は `admin:user_update`、`soft_delete` は `admin:user_delete`、`restore` は `admin:user_restore`、`purge` は `admin:user_purge` を、`admin:user_import` に加えて要求する。
- 自分自身を対象とする行、および外部の取り込み元が管理する `User` を対象とする行の fail-closed な拒否。
- 不可逆な `purge` に対する明示的な確認と、プレビュー結果での破壊的操作件数の独立表示を含む管理 UI。
- 無編集のエクスポート結果を適用しても lifecycle が 1 件も変化しないことを固定する往復テスト。

## Out of Scope

- CSV に存在しない `User` を自動削除する authoritative full-sync semantics。
- `status` 列を書き込み可能にすること。理由は Design に書く。
- `Group` と `GroupMembership` の lifecycle。[[wi-350-group-csv-round-trip]] と [[wi-351-per-group-membership-csv-round-trip-ui]] が扱う。
- 猶予期間の設定化、自動 purge スケジューラーの挙動変更、`UserLifecycle` 状態機械そのものの変更。
- password、password hash、MFA secret など秘密情報の import/export。
- CSV から `User` の所有権を外部の取り込み元からローカルへ移す操作。

## Design

### 目標状態ではなく明示的な action 列にする

`status` を書き込み可能にして目標状態を書かせる案を検討したが、次の 2 点で退けた。

第 1 に、既存のエクスポートは `status` 列を出力する。目標状態方式では、無編集のエクスポート結果をそのまま適用するという [[wi-284-improve-csv-import-export]] が固定した往復不変条件が、意図しない状態遷移を発火させる経路に変わる。フィルターや分割で一部の行だけを含むファイルでも同じ危険がある。`lifecycle_action` はエクスポートで常に空を出力するため、無編集の往復は全行 `unchanged` のままである。

第 2 に、`Deleted` という目標状態は 2 つの異なる操作に対応する。猶予期間の経過による削除と、管理者が `purge=true` を指定する即時の匿名化カスケードは、可逆性がまったく異なるにもかかわらず同じ終着状態を持つ。目標状態では管理者の意図を表現できない。

[[wi-351-per-group-membership-csv-round-trip-ui]] は `membership_state=present|absent` という目標状態方式を採るが、membership の追加と削除は対称かつ可逆で終着状態が 2 つしかないため、この 2 つの理由がどちらも当てはまらない。方式の違いは意図的なものであり、両者を揃えない。

### 判定と拒否

`lifecycle_action` は他の列と同時に指定できる。1 行の中ではプロフィールとロールの更新を先に計画し、その結果に対して lifecycle 遷移を計画する。両方が成立しなければ行全体を拒否し、片方だけを保存しない。

現在の状態から見て既に終着状態にある行は `unchanged` とする。`Disabled` の `User` への `disable` は冪等な no-op であり、エラーにはしない。逆に `Deleted` は終端状態であるため、`Deleted` への `restore` や `enable` は到達不能な遷移として拒否する。

権限は行ごとに判定する。1 つのファイルが `soft_delete` と `purge` を混在させる場合、実行者が `admin:user_delete` だけを持つなら `purge` の行だけが拒否され、他の行は適用を続ける。ファイル全体を拒否しないのは、行単位の部分成功という既存の意味論を保つためである。

自己操作の拒否は、既存の `DeleteAdminUser` が持つ自爆防止と同じ境界を CSV にも引く。所有権を確認できない場合は外部管理として扱う既存の fail-closed 方針も、lifecycle action に等しく適用する。

### プレビューと適用の結合

CSV の受け取りは 1 回だけ、適用は成功済みプレビューの ID と SHA-256 だけを参照し、現在状態から再計画するという [[wi-284-improve-csv-import-export]] の境界をそのまま踏襲する。プレビュー時に許可された遷移が適用時点で不正になっていれば、その行は適用時に拒否される。したがってプレビューの件数は保証ではなく見積もりであり、UI もそう表示する。

`purge` の確認手段は 2 案ある。`ApplyAdminUserImportHttpRequest` に確認フィールドを追加してサーバー側で強制する案と、UI の二重確認だけに留める案である。前者は API を直接叩くクライアントにも効くが、他の破壊的操作にない非対称な要素を契約へ持ち込む。T001 の仕様策定で決める未解決事項として残す。

## Plan

1. specification-first で `lifecycle_action` の語彙、遷移ガード、行単位の権限要求、自己操作と source-managed の拒否を REQ-IDMANAGEMENT-004 系の normative scenario として固定する。`purge` の確認手段もここで決める。
2. ドメインの列スキーマと解析を拡張し、閉じた語彙、空セルの意味、往復不変条件を test-first で実装する。
3. 計画器に lifecycle 遷移の判定を追加する。プロフィール更新と lifecycle の組み合わせ、冪等な no-op、到達不能な遷移、猶予期間ガードを網羅する。
4. 適用を 1 行 1 トランザクションで確定し、既存の監査イベント (`UserDisabled` / `UserSoftDeleted` / `UserRestored` / `UserDeleted`) を CSV 経路からも同じ形で発行する。
5. HTTP、worker、UI を内側から外側へ接続し、破壊的操作件数の独立表示と確認を実装する。
6. 無編集の往復、権限不足の混在ファイル、プレビュー後の状態変化を統合テストで固定して全 gate を通す。

## Tasks

- [ ] T001 [Spec] `lifecycle_action` の語彙、遷移ガード、操作別権限、自己操作・source-managed 拒否、`purge` の確認手段を TypeSpec と REQ-IDMANAGEMENT-004 系 scenario へ書き、`just check-scl` を通す。
- [ ] T002 [Domain] 列スキーマへの `lifecycle_action` 追加、閉じた語彙の解析、空セルと列欠落の同値、エクスポートでの常時空出力を test-first で実装する。
- [ ] T003 [UseCase] `UserLifecycle` ガードを再利用する行計画を実装し、冪等な no-op、到達不能な遷移、猶予期間未達の `purge`、プロフィール更新との組み合わせを網羅する。
- [ ] T004 [UseCase] 行単位の権限判定と、自己操作・source-managed・所有権判定不能の fail-closed 拒否を実装する。権限不足の行だけが拒否され他行が続くことを固定する。
- [ ] T005 [Apply] lifecycle 遷移、プロフィール更新、監査を 1 行 1 transaction で確定し、行間の部分成功と適用時の再判定を実装する。
- [ ] T006 [Adapter] 操作別件数の拡張、job 結果と row error への反映、HTTP と worker への接続を実装し、payload と値が job/audit へ露出しない contract test を通す。
- [ ] T007 [UI] 破壊的操作件数の独立表示、`purge` の明示確認、プレビュー件数が見積もりであることの提示を component test 先行で実装する。
- [ ] T008 [Verify] 無編集の export→apply で lifecycle が 1 件も変化しないこと、権限混在ファイルの部分拒否、プレビュー後の状態変化に対する再判定を統合テストで固定し、全 gate を green にする。

## Verification

- `just check`
- `just spec-render`
- `just check-api-compat`
- `just verify-go`
- `just verify-ui`
- `just test-ui-e2e`
- integration: 10,000 User を全互換列で export→apply し、lifecycle が 1 件も変化せず全行 `unchanged` になる。
- integration: `admin:user_delete` のみを持つ実行者が `soft_delete` と `purge` を混在させたファイルを適用し、`purge` の行だけが拒否される。
- integration: プレビュー後に対象 User を別操作で `Deleted` にし、適用が古い計画を実行せず現在状態から拒否する。

## Risk Notes

`purge` は匿名化カスケードを伴う不可逆操作であり、CSV の 1 列で大量に発火しうる。プレビューと適用の結合、現在状態からの再計画、行単位の権限判定、明示確認を多層で維持する。1 つでも欠けると、権限の弱い管理者が一括削除できる経路になる。

`lifecycle_action` は書き込み可能列の追加であるため、既存のエクスポート出力とヘッダー検証の後方互換に影響する。未知列をファイル単位で拒否する既存の挙動があるので、新しい列を出力するようになった時点で古いクライアントとの組み合わせを確認する。エクスポートで常に空を出力するという不変条件は、fuzz ではなく明示的な往復テストで固定する。
