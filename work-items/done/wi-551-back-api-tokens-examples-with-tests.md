---
depends_on: [wi-565-make-backing-declared-examples-cheap]
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-12
priority: p3
change_kind: maintenance
evidence_policy: risk-based-v3
documentation_impact:
  level: release_note
  reason: 外部 IdP 接続の削除が連携の残る間は 409 で拒否されること、API アクセストークンの発行と失効が CSRF を検証し system_admin にも開くことは、管理者と API 利用者に見える振る舞いの変化である。未リリースのため移行の手順は要らない。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-551.md }
affected_spec:
  - { path: docs/domain/authentication/scenarios.feature.md, requirement: REQ-AUTHENTICATION-037 }
  - { path: spec/contexts/authentication/models.tsp, symbol: IdMagic.Contract.IdentityProviderConnectionInUseError }
  - { path: spec/contexts/api-tokens/main.tsp, symbol: IdMagic.ApiTokens.Operations.IssueApiToken }
  - { path: spec/contexts/api-tokens/main.tsp, symbol: IdMagic.ApiTokens.Operations.ListApiTokens }
  - { path: spec/contexts/api-tokens/main.tsp, symbol: IdMagic.ApiTokens.Operations.RevokeApiToken }
  - { path: spec/contexts/authentication/main.tsp, symbol: IdMagic.Authentication.Operations.ListIdentityProviderConnections }
  - { path: spec/contexts/authentication/main.tsp, symbol: IdMagic.Authentication.Operations.CreateIdentityProviderConnection }
  - { path: spec/contexts/authentication/main.tsp, symbol: IdMagic.Authentication.Operations.UpdateIdentityProviderConnection }
  - { path: spec/contexts/authentication/main.tsp, symbol: IdMagic.Authentication.Operations.DeleteIdentityProviderConnection }
  - { path: spec/contexts/authentication/main.tsp, symbol: IdMagic.Authentication.Operations.ActivateIdentityProviderConnection }
  - { path: spec/contexts/authentication/main.tsp, symbol: IdMagic.Authentication.Operations.DisableIdentityProviderConnection }
  - { path: spec/contexts/authentication/main.tsp, symbol: IdMagic.Authentication.Operations.RefreshIdentityProviderMetadata }
  - { path: spec/contexts/authentication/main.tsp, symbol: IdMagic.Authentication.Operations.TestIdentityProviderConnection }
  - { path: spec/contexts/authentication/main.tsp, symbol: IdMagic.Authentication.Operations.PreviewIdentityProviderMapping }
initial_context:
  specification:
    - docs/domain/api-tokens/scenarios.feature.md
  typespec: []
  source:
    - backend/apitoken/usecases/usecases.go
    - backend/apitoken/handlers_http/routes.go
    - backend/shared/http/support_http/auth.go
    - backend/shared/http/support_http/admin_scope.go
    - backend/shared/http/support_http/csrf.go
    - backend/shared/http/testing_stack/stack.go
    - backend/shared/security/tokens_jose/jwt_signer.go
    - frontend/src/features/admin-settings/ApiTokensTab.tsx
    - frontend/src/features/admin-settings/ApiTokenScopePicker.tsx
  tests:
    - backend/apitoken/usecases/usecases_test.go
    - backend/apitoken/handlers_http/handlers_test.go
    - backend/shared/http/support_http/admin_scope_test.go
    - backend/shared/http/server_http/application_api_token_scope_test.go
    - frontend/src/features/admin-settings/ApiTokensTab.test.tsx
    - tools/check/example-coverage-debt.json
  stop_before_reading:
    - infra
    - backend/apitoken/db_postgres
    - spec/contexts/api-tokens
---

# ApiTokens が宣言する具体例 16 件にテストを対応付け、被覆台帳から外す

## Motivation

[[wi-496-burn-down-the-example-coverage-debt]] は具体例の被覆台帳の消化単位を Context と決め、測定のうえで残りを Context ごとの子 work item へ割った。本項目はそのうち `docs/domain/api-tokens/scenarios.feature.md` が宣言する 16 件を引き取る。

親項目が `claim-mapping` の 3 件と `workloadidentity` の 13 件で測った結果は、**16 件のうち注記だけで済んだのは 4 件だけ**だというものである。3 件はテストが 1 つも無く、9 件は既存テストへ新しい観測を足す必要があった。件数は作業量の目安にならない。

## Scope

- 16 件を 1 件ずつ確認し、次のいずれかに解決して `tools/check/example-coverage-debt.json` から外す。
  - 当の具体例を検証しているテストが実在する → そのテストに `// EX-APITOKENS-NNN-MM: <この具体例の何を固定しているか>` の注記を足す。
  - 当の具体例を検証しているテストが無い → 書く。具体例が拒否なら、[[wi-392-refusal-tests-assert-the-absent-effect]] が定める形（拒否応答と、拒否が防いだ効果の双方を観測する）で書く。
  - 具体例が宣言されなくなっている → 台帳から外す（検査が落ちて教える）。
- 注記は「何を固定しているか」を書く。id だけの注記は禁止する。
- 実装にバグがあって具体例のとおりに振る舞っていないことが分かった場合は、難しくない修正なら本 work item で修正する。難しい修正なら別 work-item に切り出す。

## Out of Scope

- 他の Context が宣言する具体例。Context ごとに別の work item が持つ。
- `tools/check/standards-coverage-debt.json`。[[wi-495-burn-down-the-standards-coverage-debt]] が消化済みである。
- 既に台帳に載っていない拒否テストが、防いだ効果を観測していない件。[[wi-392-refusal-tests-assert-the-absent-effect]] が持つ。本項目が新しく書く拒否テストは、その規範を満たす形で書く。既存テストに注記を足すだけの件で、そのテストが効果の不在を見ていない場合は、注記を足したうえで wi-392 の対象として残す。
- ApiTokens のシナリオと具体例の追加、削除、書き換え。具体例の記述が実装と食い違うことが判明した場合は規範の変更なので、別の work item が扱う。ただし外部 IdP 接続の削除拒否は、TypeSpec の説明にだけあって規則が無かったため、利用者の判断により本項目で `REQ-AUTHENTICATION-037` を加えた。
- 行カバレッジ率の目標または閾値。
- `check-status-drift` が、同名の定義が 2 つあると両方を黙って読まない件。本項目ではこの挙動が federation と ApiTokens の契約のずれを隠していた。検査器の変更なので別の work item が扱う。
- `report-coverage-debt` の分類（`named`、`nearby`、`none`）を根拠にした台帳からの削除。分類は読む順を決める材料であり、台帳から外す根拠にはならない。

## 作業の進め方

[[wi-565-make-backing-declared-examples-cheap]] が、この作業の 1 件あたりの費用を下げる道具を用意した。使う。

- 読む順と観測点は `mise run spec-route -- <id>` が出す。規則、当の具体例の本文、契約が宣言する候補 operation とそのメソッド・パス・スコープ、同じ規則の隣の id を名指している既存テストが返る。**出力は読む順を決める材料であり、台帳から外す根拠にはしない。**
- 新しく書くテストは `backend/shared/http/testing_stack` の上に載せる。`Register` と同じ配線が option の合成で建ち、保存先は型付きの field から読み直せる。既存 fixture の全面移行はしない。触る必要が出た範囲だけ移す。
- 変更した Go の変異は `mise run test-go-mutation -- <package-directory>` で読む。手で書く故障注入は、変異器が表現できない「配線を外す」「既定の分岐を差し替える」種類だけに残す。
- 実装が具体例と食い違って消化できない件は、台帳の当該行へ `blocked_by`（先に決着すべき work item）と `finding`（実装が実際に何を返すか）を書いて残す。散文にだけ書くと、次の読み手が同じ測定をやり直す。

## Design

変更の中心は観測の追加であり、主要なデータ型と操作は既存の `usecases.Service`（`Issue`、`Authenticate`、`List`、`Revoke`）と、管理 API の粒度スコープ判定 `requireAdminApiTokenScope` である。

### 食い違いの判定

| 食い違い | 具体例 | 具体例以外の根拠 | 判定 |
| --- | --- | --- | --- |
| 発行、一覧、失効が `admin` ロールだけを受け入れ、`system_admin` だけを持つ利用者を `access_denied` で拒否する | `EX-APITOKENS-003-03`（拒否の条件を「`admin` / `system_admin` ロールを持たない」と定める） | `docs/domain/api-tokens/decisions.md` は、発行、一覧、失効を「`admin` または `system_admin` ロールを持つ、有効かつ認証済みのユーザー」に許すと定める | 実装の欠陥。修正が小さいので本項目で直す |

### 修正の設計

`support_http.Authenticator.RequireAuditReader` は、`admin` または `system_admin` を有効ロールに持つ認証済みの利用者を要求する判定であり、監査イベントの閲覧だけが使っていた。
判定の内容は監査に固有ではないため、`RequireAdministrator` へ改名し、監査と認証イベントバケットの呼び出し側を追従させる（構造変更）。
ApiTokens の発行、一覧、失効のハンドラーは `RequireAdmin` から `RequireAdministrator` へ替える（振る舞いの変更）。
2 つは別のコミットにし、構造変更を先に置く。

### 改名で表に出た契約のずれ

`check-status-drift` は、同じ名前の関数が 2 つ定義されていると、その関数をどちらも読まない。
ApiTokens のハンドラーと federation のハンドラーがどちらも `requireAdmin` という名前の前提判定を持っていたため、双方の判定が読まれず、次のずれが隠れていた。
ApiTokens の前提判定を `requireAdministrator` へ改名したことで、両方が読まれるようになった。

| operation | 実装が返すが契約に無いもの | 契約に有るが実装が返さないもの |
| --- | --- | --- |
| `ListApiTokens`、`RevokeApiToken` | 401、403 の `insufficient_scope` | `RevokeApiToken` の 400 |
| `ListIdentityProviderConnections` | 401、403 の `insufficient_scope` | なし |
| federation の状態を変える 8 operation | 401、403 の `csrf_failed`、`insufficient_scope`、`invalid_origin` | `DeleteIdentityProviderConnection` と `TestIdentityProviderConnection` の 400 |

利用者の判断により、本項目で契約を実装へ合わせる。
401 は共有の `AuthenticationRequiredResponse` を参照し、403 の本文へ問題コードを足し、返さない 400 を外す。
外した 400 はリリースベースライン `spec/idmagic.openapi.baseline.json` からも外す（未リリースのため移行は不要）。
federation の前提判定は `requireAdmin(c, csrf bool)` の引数で CSRF の検証を切り替えていたため、一覧の GET でも `csrf_failed` を返し得るように読めた。
CSRF の検証を含む `requireBrowserAdmin(c)` と含まない `requireAdmin(c)` に分け、一覧の GET には CSRF の問題コードを宣言しない（構造変更）。

### 契約ではなく実装を直した 2 件

ずれの大半は、実装が正しく契約の宣言が漏れていたものである。
次の 2 件は宣言に合わせると欠陥を契約へ固定することになるため、利用者の判断により実装を直した。

| 欠陥 | 修正 |
| --- | --- |
| ApiTokens の発行と失効が `VerifyBrowserRequest` を呼ばず、ログインセッションで届いた状態変更に Origin と CSRF の二重送信を検証しない。ほかの管理 API の状態変更はすべて検証している | 2 つのハンドラーの冒頭で `VerifyBrowserRequest` を呼ぶ。契約の 403 へ `csrf_failed` と `invalid_origin` を宣言する |
| `DeleteIdentityProviderConnection` の説明は「連携が残っていれば削除を拒否する」と言うが、ハンドラーは連携を見ない。PostgreSQL では外部キー（`ON DELETE RESTRICT`）が 500 を返し、メモリの保存先では連携を残したまま削除していた | 新しい規則 `REQ-AUTHENTICATION-037` を定め、ユースケース `DeleteConnection(ctx, deps BrokerDeps, tenantID, providerID string) error` が `IdentityRepository.ExistsForProvider(ctx, tenantID, providerID) (bool, error)` で連携を確かめ、残っていれば `ErrConnectionInUse` を返す。ハンドラーはこれを 409 `connection_in_use`（`IdentityProviderConnectionInUseError`）へ写す。管理画面は拒否を表示言語の辞書で示す |

### テストの置き場所

| 具体例 | 置き場所 | 観測 |
| --- | --- | --- |
| `EX-APITOKENS-001-01` から `-04` | `frontend/src/features/admin-settings/ApiTokensTab.test.tsx`（既存） | 主見出しと一覧の見出しの階層、3 つの Base URL とその用途の説明、API の種類とリソースごとのまとまり、`read` / `write` の意味、選んだスコープだけが発行要求に載ること、グループが閉じた状態で始まること、変更系の無いリソースに変更の選択肢が無いこと、変更系だけのリソースが右列に置かれること |
| `EX-APITOKENS-002-01` から `-03` | `backend/apitoken/handlers_http/scenario_examples_test.go`（新設） | `testing_stack` の実署名器で JWT を作り、製品と同じ overlay の `Service.Authenticate` へ提示する。不正な部分を 1 つだけ持つ JWT と、同じ jti の正しい記録を並べ、拒否の原因がその部分だけであることを対照の成功で示す。拒否では主体が空であることを見る |
| `EX-APITOKENS-003-01` から `-04` | 同上 | 管理者のログインセッションで管理 API を呼ぶ。発行された JWT の `typ`、`sub`、`client_id`、一覧に JWT 本文が無いこと、失効後に管理 API と `/introspect` がそのトークンを拒むこと。拒否では保存先の記録が増えないこと、存在しない id の失効では既存トークンが有効なままであることを読み直す |
| `EX-APITOKENS-004-01`、`-02`、`-04`、`-05` | 同上 | API アクセストークン、ポータルの OAuth アクセストークン、ログインセッションでグループの管理 API を呼ぶ。拒否では `WWW-Authenticate` の `error` と `scope`、問題の種類、グループの保存先が変わらないことを見る |
| `EX-APITOKENS-004-03` | `backend/shared/http/support_http/admin_scope_test.go`（既存） | 契約はすべての管理 operation に宣言を持つため、製品の入口からは宣言の無い operation を作れない。契約を差し替えて、宣言が空の operation と契約に無いルートの双方が `insufficient_scope` になることを見る |

### 選んだ実行レシピ

| 触った層 | 実行するタスク |
| --- | --- |
| Go の 1 テスト | `mise run test-go-test -- <package> <test>` |
| Go のパッケージ | `mise run test-go-package -- <package>` |
| UI の 1 ファイル | `mise run test-ui-unit-file -- <file>` |
| 台帳と仕様 | `mise run check-spec` |
| frontmatter | `mise run check-work-items` |

## Tasks

- [x] T001 [Acceptance] 16 件を台帳から外し、`mise run check-spec` が未対応 id を名指しで落とすことを確認する。
- [x] T002 [UI] `EX-APITOKENS-001-01` から `-04` の観測を書く。
- [x] T003 [Authenticate] `EX-APITOKENS-002-01` から `-03` の観測を書く。
- [x] T004 [AdminAPI] `EX-APITOKENS-003-01` から `-04` の観測を書く。
- [x] T005 [Scope] `EX-APITOKENS-004-01` から `-05` の観測を書く。
- [x] T006 [Structure] `RequireAuditReader` を `RequireAdministrator` へ改名する。既存テストが通ることを確認する。
- [x] T007 [Fix] 発行、一覧、失効を `system_admin` にも開く（`EX-APITOKENS-003-03`）。Unit RED を先に観測する。
- [x] T008 [Structure] federation の前提判定を `requireAdmin(c)` と `requireBrowserAdmin(c)` に分ける。
- [x] T009 [Spec] ApiTokens と federation の 11 operation の契約を実装へ合わせ、外した 400 をリリースベースラインへ反映する。`check-status-drift` と `check-api-compat` を通す。
- [x] T010 [Spec] `REQ-AUTHENTICATION-037` と `IdentityProviderConnectionInUseError`、削除の 409、発行と失効の 403 の CSRF 問題コードを宣言する。`check-spec` が新しい 2 件を名指しで落とすことを確認する。
- [x] T011 [Fix] 連携が残る外部 IdP 接続の削除を拒否する（`EX-AUTHENTICATION-037-01`、`-02`）。Unit RED を先に観測する。
- [x] T012 [Fix] ApiTokens の発行と失効で Origin と CSRF を検証する。Unit RED を先に観測する。
- [x] T013 [UI] 外部 IdP 接続の削除拒否を表示言語で示す。
- [x] T014 [Verify] 対象パッケージ、仕様検査、総合検証を通して完了記録を作る。

## Verification

- `mise run check-spec` が、`docs/domain/api-tokens/scenarios.feature.md` の 16 件を `tools/check/example-coverage-debt.json` から外した状態で通る。台帳から外す前に同じ検査が当の id を名指しで落とすことを、消化ごとに観測する。
- 予定する Acceptance RED は、台帳から 16 件を外した `mise run check-spec` である。
- 予定する Unit RED は、`system_admin` だけを持つ利用者のセッションで発行を要求すると 201 にならないことの `backend/apitoken/handlers_http` の観測である。修正の無い具体例では、新しく書く各テストが観測する分岐を手で外し、そのテストが落ちることを確かめる。
- 消化したテストの所属パッケージに対する `mise run test-go-package -- <package>`。
- `mise run verify`

## Risk Notes

- **注記だけを足して終わる。** `checkNormativeCoverage` は文字列の一致しか見ないので、読まずに id を貼れば件数は速く減る。減った件数は何も意味しない。注記に「この具体例の何を固定しているか」を書かせることで区別する。親項目の測定では、16 件のうち注記だけで済んだのは 4 件だけだった。
- **`nearby` に分類された件を、テストがある証拠として読む。** 分類が言っているのは「近傍に他の拒否テストがある」だけである。親項目の測定では、`claim-mapping` の 3 件はすべて `nearby` でありながら 2 件はテストが無かった。分類は読む順の材料にとどめる。
- **拒否である具体例のテストが、応答の字面だけを見て書かれる。** 新しく書く拒否テストには [[wi-392-refusal-tests-assert-the-absent-effect]] の規範が効く。注記へ「効果の不在を何で観測したか」を書き、後から区別できるようにする。
- **具体例の `Then` が複数あるのに、観測が 1 つで済まされる。** 親項目の測定では、`EX-WORKLOADIDENTITY-008-01` のようにイベントと実際の効果の双方を言う具体例が複数あった。`Then` の数だけ観測が要る。

## Completion

- **Completed At**: 2026-09-23
- **Summary**:
  `mise run spec-diff` は `main` に対して、規則 `REQ-AUTHENTICATION-037` と TypeSpec の `IdentityProviderConnectionInUseError` の追加を返した。
  ApiTokens が台帳に残していた 16 件の具体例をすべて実装と照合し、`//spec:covers` を付けたテストへ結び付けて台帳から外した。
  UI の 4 件（`EX-APITOKENS-001-01` から `-04`）は `ApiTokensTab.test.tsx` へ新しく書いた。見出しの階層、3 つの Base URL と用途、API の種類とリソースごとのまとまり、閉じた状態で始まるグループ、変更系の無いリソース、変更系だけのリソースの列の位置を観測する。
  認証、発行と失効、粒度認可の 11 件は `backend/apitoken/handlers_http/scenario_examples_test.go` を新設し、`testing_stack` の実署名器と製品と同じ組み立てで観測した。認証の拒否は JWT と記録の一方の 1 か所だけを壊し、既定のままのトークンが通ることと対にしている。拒否の効果はトークンの記録とグループの保存先から読み直す。
  `EX-APITOKENS-004-03` は、契約がすべての管理 operation に宣言を持つため製品の入口から再現できず、契約を差し替えた `admin_scope_test.go` の単体テストで観測した。
  照合で分かった欠陥は 3 件で、利用者の判断によりすべて本項目で直した。発行、一覧、失効が `system_admin` を拒否していた（`decisions.md` と食い違い）。発行と失効が Origin と CSRF を検証していなかった。外部 IdP 接続の削除が連携の残りを見ず、PostgreSQL では 500、メモリでは連携を残したまま削除していた。削除の拒否は新しい規則 `REQ-AUTHENTICATION-037` として定め、409 `connection_in_use` を返し、管理画面は拒否を表示言語で示す。
  ApiTokens の前提判定を改名したことで、`check-status-drift` が同名の定義を読まないために隠れていた、ApiTokens と外部 IdP 接続の 11 operation の契約のずれが表に出た。実装が正しいものは契約へ 401 と 403 の問題コードを宣言し、どの経路も返さない 400 を外してリリースベースラインへ反映した。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`
  - **Requirement**: REQ-APITOKENS-001
  - **Observed Failure**: 台帳から 16 件を外すと、`check-spec` は `EX-APITOKENS-001-01` から `EX-APITOKENS-004-05` までのちょうど 16 件を「declared, but no test names it」で名指しして落ちた。`REQ-AUTHENTICATION-037` を加えた時点では、`EX-AUTHENTICATION-037-01` と `-02` の 2 件を同じ理由で名指しして落ちた。
  - **Detection Reason**: 対象は ApiTokens の 16 件と REQ-AUTHENTICATION-037 の 2 件である。台帳の行を消すだけ、規則を足すだけでは検査を通らず、各 id を名指しするディレクティブと、その id を実際に動かすテストの両方が要る。
- **Unit RED Evidence**:
  - **Test**: `TestIssueApiTokenRequiresTheAdminOrSystemAdminRole`、`TestApiTokenStateChangesRequireTheCSRFToken`（`backend/apitoken/handlers_http`）、`TestDeleteConnectionRefusesAConnectionWithLinkedIdentities`（`backend/authentication/federation/usecases`）
  - **Requirement**: REQ-AUTHENTICATION-037
  - **Observed Failure**: 修正前は、`system_admin` だけを持つ利用者の発行が 403 `access_denied`、CSRF トークンの無いセッションの発行が 201 でトークンを発行、連携が残る接続の削除が `err = <nil>` で成功した。
  - **Detection Reason**: 3 つは REQ-AUTHENTICATION-037 と REQ-APITOKENS-003 の双方にわたる。3 つとも応答に加えて保存先を読み直すので、拒否を書いてから効果を残す実装とも区別できる。
- **Change-Resistance Results**:
  `mise run test-go-mutation` は `backend/apitoken/handlers_http` の 10 件をすべて検出した。`backend/authentication/federation/usecases` は 64 件のうち 62 件を検出し、生存した 2 件と被覆されない 5 件はすべて既存の `broker.go` と `flow.go` にあり、新しい `connection_delete.go` には無い。
  変異器が表現できない故障を手で注入し、すべて検出された。認証では失効、空のスコープ集合、audience、記録の期限、JWT の発行者と exp の判定を外した。発行では `expiry_days` の境界を変えた。認可では空の宣言の素通し、API アクセストークンのポータル境界への差し戻し、ロール判定とスコープ判定の除去、失効の id の取り落とし、発行と一覧の経路を `RequireAdmin` へ戻す形、失効だけ CSRF の検証を外す形を試した。外部 IdP 接続では拒否の写像の除去と、PostgreSQL の問い合わせの反転を試した。UI では 5 種の配置の故障と、拒否の辞書への写像の除去を試した。
  DB を使うテストはサンドボックスの中ではスキップされるため、`TestConnectionAndIdentityRepositoriesRoundTrip` はサンドボックスの外で走らせ、問い合わせを反転した故障で落ちることを確かめた。
- **Verification Results**:
  - `mise run check-spec` - 成功
  - `mise run check-status-drift` - 成功
  - `mise run check-contract-drift` - 成功
  - `mise run check-api-compat` - 成功（削除した 3 つの 400 はリリースベースラインへ反映した）
  - `mise run test-go-changed` - 成功
  - `mise run lint-go` - 成功
  - `mise run verify` - 成功
  - `mise run test-ui-e2e` - 成功（38 件）
