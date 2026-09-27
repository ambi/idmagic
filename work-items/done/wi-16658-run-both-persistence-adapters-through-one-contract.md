---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-26
priority: p1
depends_on: []
change_kind: tooling
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: "テストの配置と開発文書の 1 節だけを変える。利用者が観測できる差分が無く、リリースの読者へ知らせるものが無い。"
  references: []
spec_impact: { kind: none, reason: "永続化アダプターの既存の振る舞いを、2 実装に共通のテストで固定する作業である。シナリオも公開契約も変えない。既存の PostgreSQL 制約へ合わせられる局所的な食い違いは本項目で直し、規範判断を要する食い違いだけを bugfix work item に切り出す。" }
initial_context:
  specification: []
  typespec: []
  source:
    - backend/authorization/testing_contract/contract.go
    - backend/idmanagement/user/ports/user_repository.go
    - backend/idmanagement/user/ports/email_change_token_store.go
    - backend/idmanagement/user/ports/user_import.go
    - backend/tenancy/ports/tenant_user_attribute_schema_repository.go
    - backend/idmanagement/user/db_memory/users.go
    - backend/idmanagement/user/db_memory/email_change_token.go
    - backend/idmanagement/user/db_memory/tenant_user_attribute_schema.go
    - backend/idmanagement/user/db_memory/user_import_committer.go
    - backend/idmanagement/user/db_postgres/users.go
    - backend/idmanagement/user/db_postgres/users.sql
    - backend/idmanagement/user/db_postgres/email_change_token.go
    - backend/idmanagement/user/db_postgres/email_change_tokens.sql
    - backend/idmanagement/user/db_postgres/tenant_user_attribute_schema.go
    - backend/idmanagement/user/db_postgres/tenant_user_attribute_schemas.sql
    - backend/idmanagement/user/db_postgres/user_import_committer.go
    - infra/schema/postgres.sql
  tests:
    - backend/authorization/db_memory/repository_test.go
    - backend/authorization/db_postgres/repository_test.go
    - backend/idmanagement/user/db_memory/users_test.go
    - backend/idmanagement/user/db_memory/email_change_token_test.go
    - backend/idmanagement/user/db_postgres/users_test.go
    - backend/idmanagement/user/db_postgres/email_change_token_test.go
    - backend/idmanagement/user/db_postgres/user_import_committer_test.go
    - backend/idmanagement/user/db_postgres/fixtures_test.go
  stop_before_reading:
    - backend/idmanagement/user/usecases
    - backend/idmanagement/user/handlers_http
    - spec
---

# メモリと PostgreSQL の永続化アダプターを、1 つの契約テストへ通す

## 動機

永続化を持つパッケージは `db_memory` と `db_postgres` の 2 実装を対で持つ。
2026-09-26 時点で、この対は 35 組ある。
`docs/development/testing.md` は、両者が同じ振る舞いをすることを保証する共有の契約テストが無く、これを「未着手の負債」と明記している。

この負債は、2 つの経路で製品に届く。

- **上位のテストがメモリ実装をフェイクとして使う。** 他パッケージのテストがメモリ実装を読み込む箇所は 35 組で延べ 639 ファイルある。多い順に `idmanagement/user` 153、`oauth2` 84、`tenancy` 71、`idmanagement/group` 39、`authentication/session` 35 である。メモリ実装が PostgreSQL 実装と違う振る舞いをすれば、これらのテストは本番で起きないことを確かめ、本番で起きることを見逃す。
- **メモリ実装は本番の構成でも選べる。** `PERSISTENCE` は `memory` と `postgres` を受け付ける（`backend/cmd/internal/bootstrap/memory.go`）。メモリ実装の食い違いは、テストの誤りでは済まない。

両者のテストは独立に書かれており、対象も件数も対応していない。
たとえば `tenancy` はメモリ側 1 件、PostgreSQL 側 12 件、`datakeys` はメモリ側 11 件、PostgreSQL 側 4 件、`authentication/recovery` と `wsfederation` は PostgreSQL 側に 1 件も無い。
トランザクション、一意制約、NULL と空文字列の区別、時刻の精度、並び順といった PostgreSQL 固有の意味について、メモリ側のテストは何も言わない。
2026-09-26 に DB テストを実際に動かして測ると、`db_postgres` 全体の文のカバレッジは 59.0 %（6,743 文中 3,978 文）で、`authentication/recovery`、`wsfederation`、`oauth2/token` の 3 組は 0 % だった。

前例は 1 組だけある。
`backend/authorization/testing_contract` は関係タプルと認可モデルの Repository の契約を 1 か所に置き、`authorization/db_memory` と `authorization/db_postgres` の両方のテストが同じ本体を実行する。
`testing.md` の「存在しない」はこの前例を反映していない。

## 対象範囲

- `db_memory` と `db_postgres` を対で持つ 35 組のうち、`authorization` を除く 34 組について、ポートごとの契約テストを 1 か所に置き、両実装のテストから同じ本体を実行する。
- 契約が扱う性質を、少なくとも次の観点で揃える。保存したものの読み戻し、存在しないものの扱い（`not found` の型）、一意制約の違反、テナント境界、並び順とページング、時刻の精度、NULL と空文字列の区別、楽観ロックまたは revision の衝突。ポートがその性質を持たない場合は対象外とする。
- 契約を通したことで不要になった、片方の実装だけの重複テストを削る。片方にしか無い性質（PostgreSQL のトランザクション境界、メモリ側の容量上限など）を確かめるテストは残す。
- `docs/development/testing.md` の「メモリアダプターと PostgreSQL アダプターの同値性は保証されていない」節を、完了後の状態へ書き換える。
- 2 実装の食い違いが見つかった場合、既存の PostgreSQL 制約へ合わせられる局所的な修正は本項目に含める。規範判断を要する食い違いは bugfix work item に切り出し、契約テストでは `t.Skip` ではなく、切り出した work item を名指す理由付きで一時的に外す。

## 対象外

- 上位のユースケースやハンドラーのテストを、両方のアダプターで実行すること。設計で却下した。
- メモリ実装の廃止。本番の構成で選べる以上、残す理由がある。廃止するなら構成の変更として別に扱う。
- 行カバレッジ率の目標または閾値。
- 規範判断または広い設計変更を要する食い違いの修正。切り出した先で扱う。
- `shared/storage` の汎用基盤（`testing_postgres` など）そのものの変更。契約テストが必要とする補助を足すことは含める。

## 設計

契約の置き場所は、`authorization/testing_contract` と同じく、Context ごとの `testing_contract` パッケージとする。
テスト補助を非テストパッケージに置き、`db_memory` と `db_postgres` の `_test.go` から `Fixture` を渡して同じ本体を呼ぶ。
`Fixture` は、契約が使う記号名から実在の識別子への対応（PostgreSQL ではテナント行などを事前に用意する）を持つ。
この形は既に 1 組で動いており、新しい仕組みを足さずに広げられる。

時刻は `testing_postgres.Now()` と同じく、PostgreSQL の分解能（マイクロ秒）に切り捨てた値を契約の入力にする。
精度の違いを契約から隠すのではなく、「保存した時刻はマイクロ秒で読み戻る」ことを契約として固定し、メモリ実装がナノ秒を返すなら食い違いとして扱う。

`idmanagement/user` の基準実装では、User repository、email-change token、tenant attribute schema、import row committer の 4 境界を一つの `testing_contract.Run` から実行した。
メモリと PostgreSQL の package 検査を並列に実行したキャッシュ済み所要は 8.31 秒（各 package の報告は 3.23 秒と 8.10 秒）で、Context ごとの fixture factory は反復可能な大きさだった。
検出した食い違いは、メモリ User repository が同一テナント内の active `preferred_username` 重複を受理する 1 件だった。
これは PostgreSQL の既存の部分一意索引へ合わせる局所修正として本項目に含め、別 ID の active user だけを拒否し、同じ user の更新と削除済み username の再利用は妨げない。
この基準結果から、残る 33 組も本項目内で Context ごとの `testing_contract` と fixture factory を追加する。
独立した仕様成果へ分ける根拠はなく、同じ負債の完了判定を複数記録へ複製しない。

継続実装では、`apitoken`、`audit`、`authentication/recovery`、`authentication/securitynotification`、
`authentication/password`、`authentication/session`、`authentication/totp`、`authentication/trusteddevice`、
`authentication/webauthn`、`datakeys`、`idmanagement/agent`、`idmanagement/group`、`jobs`、
`oauth2/client`、`oauth2/consent`、`oauth2/logout`、`oauth2/token`、`shared/ratelimit`、
`sourcing/scim`、`tenancy`、`wsfederation` を追加で同じ契約本体へ接続した。
各契約は保存・読み戻しだけでなく、その境界に固有のテナント分離、期限、状態遷移、重複抑止、
一意性またはページングの代表的な性質を含む。PostgreSQL 側の UUID/FK と時刻の前提は各 fixture factory に閉じ込め、契約本体は論理 ID を使う。

最後に `application`、`authentication`、`authentication/federation`、`authentication/mfa`、
`idgovernance`、`idmanagement`、`oauth2`、`provisioning`、`saml`、`sharedsignals`、
`workloadidentity` も接続した。
単に同名ディレクトリが並ぶ `shared/storage` は例外であり、`db_memory` は keyset pagination の純粋関数、
`db_postgres` は接続、回路遮断、SQL 実行の基盤を所有し、両者が実装する共通ポートがない。
したがって比較可能な永続化アダプターの組は、既存の `authorization` を含む 34 Context だった。
ディレクトリ名だけを数えた着手時の 35 組から `shared/storage` を除き、34 組すべてを共有契約へ接続した。

契約はさらに、`workloadidentity` のメモリ実装が同一テナント内の trust bundle issuer 重複を受理する食い違いを検出した。
これは PostgreSQL の既存一意制約へ合わせる局所修正として本項目に含め、別 ID の bundle だけを拒否し、
同じ bundle の更新と別テナントでの同一 issuer は妨げない。

`authentication/webauthn` の契約は、メモリ実装が WebAuthn セッションキーをテナントで分離していない食い違いも検出した。
PostgreSQL 実装が既に複合主キーで固定している境界へ合わせ、メモリ実装の内部キーへテナント ID を含める局所修正を本項目に含めた。
同じテナント内の保存と消費の振る舞いは変えない。

重複テストの整理では、共有契約が同じ公開ポートを通して置き換えた PostgreSQL 片側だけの
`idgovernance` workflow 往復テストと `workloadidentity` trust bundle 往復・テナント分離・issuer 一意性テストを削除した。
SQL のチャンク境界、CAS、複数 replica、cascade、GC、暗号化された保存表現など、片方の実装だけが持つ性質のテストは残した。

却下した案は次のとおり。

| 案 | 却下した理由 |
|---|---|
| 上位のテストを両方のアダプターで実行する | テスト数は数千件に及び、PostgreSQL での実行時間が支配的になる。失敗したときに、どのポートのどの性質が食い違ったかを読めない |
| メモリ実装を捨て、すべてのテストで PostgreSQL を使う | `testing.md` は、読み戻せる依存を契約を満たすフェイクで置き換える方針を採っている。フェイクが契約を満たすことを確かめるのが本項目であり、フェイクをやめることではない。本番の構成からも外れる |
| ジェネリクスで全ポート共通の契約を 1 つ書く | ポートの形がそろっていない。共通化できるのは「保存して読み戻す」程度で、並び順、一意制約、revision など食い違いが起きやすい性質はポートごとに違う |

進め方は wi-496 に倣う。
最初に上位のテストからの利用が最も多い `idmanagement/user` を通しで行い、1 組あたりの所要と、見つかった食い違いの件数を記録する。
その結果で残りの進め方を決める。
件数を理由に項目を割らない。
割るのは、組ごとに独立して受け入れられる成果があると測定で分かった場合だけとする。

## 計画

1. `idmanagement/user` の契約を書き、両実装で実行する。所要、契約が扱った性質、見つかった食い違いを設計へ記録する。
2. 記録をもとに、残る 33 組をこの項目で続けるか、Context ごとに割るかを決めて設計へ書く。
3. 上位のテストからの利用が多い順（`oauth2`、`tenancy`、`idmanagement/group`、`authentication/session`、`application` …）に契約を広げる。
4. 食い違いを bugfix work item へ切り出す。
5. 重複テストを削り、`testing.md` を書き換える。

未解決の問いは 1 つある。
契約の本体を PostgreSQL で実行すると、組ごとにテナント行などの前提データの用意が要る。
`shared/storage/fixtures_postgres` で賄えるかは、計画 1 で確かめる。

着手時の readiness pass で、`idmanagement/user/db_postgres` は自身の User repository を所有するため `shared/storage/fixtures_postgres` を import すると循環依存になることを確認した。
同パッケージの既存 `fixtures_test.go` が tenant と user を作る最小の前提を既に持つため、共有 fixture は変更せず、各アダプターの `_test.go` から `testing_contract` へ package-local な fixture factory を渡す。

この作業は製品の規範的振る舞いを変えない tooling 変更なので、Acceptance と Unit の製品境界は該当しない。
Acceptance RED の代替検査は、両アダプターのテストからまだ存在しない同じ `testing_contract` 関数を呼び、`mise run test-go-package` のコンパイル失敗を観測することとする。
Unit RED の代替検査は、各契約へ既存テストにない代表的な不変条件を先に加え、一方のアダプターで期待どおり失敗することを同じ package task で観測することとする。
T001 では `mise run test-go-package backend/idmanagement/user/db_memory` と `mise run test-go-package backend/idmanagement/user/db_postgres`、複数 package に広がった後は `mise run test-go-changed` を使う。

## タスク

- [x] T001 [Baseline] `idmanagement/user` の契約を書いて両実装で実行し、所要と食い違いを記録する。
- [x] T002 [Plan] 残る 33 組の進め方を決めて設計へ書く。
- [x] T003 [Test] 残る組へ契約を広げる。
- [x] T004 [Triage] 見つかった食い違いを分類する。規範判断を要する差はなく、PostgreSQL の既存制約へ合わせる局所修正 3 件を本項目へ含めた。
- [x] T005 [Cleanup] 契約と重複する片側だけのテストを削る。
- [x] T006 [Docs] `docs/development/testing.md` の同値性の節を書き換える。
- [x] T007 [Verify] 変更を検証する。

## 検証

- 各組の `db_memory` と `db_postgres` のテストが、同じ `testing_contract` の本体を実行する。
- メモリ実装に食い違いを 1 つ注入すると、契約テストのメモリ側だけが落ちる。
- `mise run test-go-race` を embedded-postgres が起動できる環境で実行し、契約テストがスキップされずに通る。
- `mise run verify`

2026-09-26 の中間検証では、変更した全 Go パッケージのテストは永続化契約を含めて通過した。
`backend/shared/http/server_http` の 2 テストだけは、作業ツリーに未生成の `spec/generated/openapi/idmagic.openapi.json` を読むため、生成済み成果物を用意していない環境では失敗する。
これは今回の契約テスト変更とは無関係である。
`mise run lint-go` と `mise run check-work-items` は通過した。
`mise run test-go-mutation -- backend/idmanagement/user/db_memory` は efficacy 90.62 %（Killed 29、Lived 3、Not covered 3）だった。

## リスク

- **契約が読み戻しだけで終わる。** 保存して読み戻すだけなら、2 実装はたいてい一致する。食い違いが起きやすいのは、一意制約、並び順、NULL、時刻、revision である。設計の観点一覧を組ごとに当て、該当しない観点は理由を書いて外す。
- **PostgreSQL 側のテストが黙ってスキップされる。** embedded-postgres を起動できない環境（サンドボックスを含む）では `testing_postgres.Require` がスキップする。2026-09-26 のカバレッジ計測でも、サンドボックス内では `db_postgres` がほぼ 0 % と出た。検証は起動できる環境で行い、スキップされていないことをログで確かめる。
- **食い違いを契約テストの側で吸収する。** 片方に合わせて契約を緩めると、負債が見えなくなるだけである。食い違いは切り出し、契約は規範の側に置く。

## 完了

- **Completed At**: 2026-09-27
- **Summary**:
  `mise run spec-diff` は `main` に対する規範仕様の差分がないと報告した。
  比較可能な 34 Context すべてで、メモリ実装と PostgreSQL 実装が同じ `testing_contract` 本体を実行するようにした。
  共有契約が検出した `preferred_username` の一意性、WebAuthn セッションのテナント分離、trust bundle issuer の一意性という 3 件の局所的不整合を、既存の PostgreSQL 制約へ合わせて修正した。
- **Acceptance RED Evidence**:
  - **Test**: 各 Context の `db_memory` と `db_postgres` に契約 runner を先に追加し、未実装の `testing_contract.Run` を `mise run test-go-package -- ./backend/<context>/db_memory` と `mise run test-go-package -- ./backend/<context>/db_postgres` から実行した。
  - **Requirement**: N/A: 製品の規範的振る舞いを変えない tooling 変更であり、対応する REQ はない。
  - **Observed Failure**: 両アダプターの runner が同じ未実装契約へ到達し、`contract not implemented` の意図した失敗を報告した。
  - **Detection Reason**: 片方だけに独立したテストを書く実装では両 runner が同じ失敗点へ到達しないため、共有契約への接続を識別できる。
- **Unit RED Evidence**:
  - **Test**: `TestPersistenceContract/user_repository/same_tenant_rejects_duplicate_username`、`TestWebAuthnSessionStoreContract`、`TestPersistenceContract` の trust bundle issuer 重複検査を、メモリ実装の修正前に各 package task で実行した。
  - **Requirement**: N/A: 既存の PostgreSQL 制約との同値性を固定する内部テストであり、対応する REQ はない。
  - **Observed Failure**: メモリ実装だけが、同一テナントの active `preferred_username` 重複、別テナントからの同一 WebAuthn セッションキーの取得、同一テナントの trust bundle issuer 重複を受理した。
  - **Detection Reason**: それぞれ PostgreSQL の既存一意制約または複合キーが拒否する入力を同じ契約から与えるため、メモリ実装の意味の差を検出できる。
- **Change-Resistance Results**:
  `backend/idmanagement/user/db_memory` の体系的変異は 34 件中 Killed 29、Lived 2、Not covered 3 であり、生き残りと未被覆は既存の import committer と `ListPageBefore` に限られ、変更した username 一意性ロジックには残らなかった。
  `backend/workloadidentity/db_memory` の体系的変異は 10 件中 Killed 7、Lived 0、Not covered 3 であり、issuer 一意性ロジックに生き残りはなかった。
  `backend/authentication/webauthn/db_memory` では変異器がテナント配線の除去を表現できなかったため、`sessionKey` からテナント ID を外す手動フォールトを注入したところ、`TestWebAuthnSessionStoreContract` が別テナントからセッションを取得して失敗した。
- **Verification Results**:
  - `mise run lint-go` - 成功
  - `mise run check-work-items` - 成功
  - `mise run check` - 成功
  - `mise run test-go-race` - `mise run verify` 内で成功し、PostgreSQL 契約のスキップなし
  - `mise run verify` - 成功
  - `mise run spec-diff` - `main` に対する規範仕様の差分なし
