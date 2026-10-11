---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-04
priority: p3
depends_on: []
change_kind: maintenance
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  reason: 初回公開前であり、発生経路のない宣言の削除は既存データの移行や実利用契約の変更を伴わない。
  references: []
initial_context:
  specification:
    - docs/development/specification-first-workflow.md
    - docs/formats/work-item-format.md
    - docs/development/coding-style.md
    - docs/development/testing.md
    - docs/development/writing-language.md
    - docs/development/release.md
    - docs/formats/specification-format.md
    - docs/design/application/design-guidelines.md
    - docs/design/application/api-guidelines.md
    - docs/modules/identity-management/README.md
    - docs/modules/identity-management/user-csv/README.md
    - docs/modules/identity-management/group-csv/README.md
    - docs/modules/identity-management/data-export/README.md
    - docs/modules/audit/event-search/README.md
  typespec:
    - IdMagic.IdManagement.Operations.ImportAdminUsers
    - IdMagic.IdManagement.Operations.ImportAdminGroups
    - IdMagic.IdManagement.Operations.ImportAdminGroupMembers
  source:
    - backend/idmanagement/user/handlers_http/admin_user_import_handler.go
    - backend/idmanagement/group/handlers_http/admin_group_import_handler.go
    - backend/idmanagement/group/handlers_http/admin_group_membership_import_handler.go
    - backend/idmanagement/handlers_http/admin_data_export_handler.go
    - backend/idmanagement/domain/events.go
    - backend/audit/handlers_http/admin_audit_event_handler.go
    - backend/cmd/internal/bootstrap/memory.go
    - backend/cmd/internal/bootstrap/postgres.go
    - backend/shared/http/server_http/routes.go
    - frontend/src/features/admin-audit-events/AuditEventsBrowser.tsx
    - tools/check/unspecified-vocabulary-debt.json
    - tools/check/event-emission-debt.json
  tests:
    - backend/audit/handlers_http/admin_audit_event_handler_test.go
    - backend/audit/handlers_http/audit_search_options_characterization_test.go
    - backend/idmanagement/user/handlers_http/admin_user_import_e2e_test.go
    - backend/idmanagement/group/handlers_http/admin_group_membership_import_e2e_test.go
    - backend/idmanagement/handlers_http/export_examples_test.go
    - backend/idmanagement/handlers_http/export_refusal_effects_test.go
  stop_before_reading:
    - backend/oauth2
    - backend/authentication/federation
affected_spec:
  - { path: spec/modules/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.ImportAdminUsers, impact: modifies }
  - { path: spec/modules/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.ApplyAdminUserImport, impact: modifies }
  - { path: spec/modules/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.GetAdminUserImport, impact: modifies }
  - { path: spec/modules/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.ImportAdminGroups, impact: modifies }
  - { path: spec/modules/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.ApplyAdminGroupImport, impact: modifies }
  - { path: spec/modules/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.GetAdminGroupImport, impact: modifies }
  - { path: spec/modules/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.ImportAdminGroupMembers, impact: modifies }
  - { path: spec/modules/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.ApplyAdminGroupMemberImport, impact: modifies }
  - { path: spec/modules/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.GetAdminGroupMemberImport, impact: modifies }
  - { path: spec/modules/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.StartUserCsvExport, impact: modifies }
  - { path: spec/modules/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.StartGroupCsvExport, impact: modifies }
  - { path: spec/modules/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.StartGroupMemberCsvExport, impact: modifies }
---

# IdManagement が宣言しているが、どの経路からも起きないエラーコードとイベントを消す

## 動機

wi-83002 で IdManagement の機能仕様を書き直したとき、`check-unspecified-vocabulary` の許容リストに残った 6 件は、どの要件にも書けなかった。
どれも、宣言はあるが、製品のどの経路からも観測できない。

| 語 | 宣言 | 起きない理由 |
| --- | --- | --- |
| `invalid_target` | TypeSpec のエラー、エクスポートのハンドラーの対応 | エクスポートの対象はルートで決まり、要求から未知の対象を渡せない |
| `user_import_unavailable` | TypeSpec のエラー、503 | `JobRepo` と `CSVArtifacts` が空のときだけ返すが、どちらのブートストラップも必ず配線する |
| `group_import_unavailable` | 同上 | 同上 |
| `group_membership_import_unavailable` | 同上 | 同上 |
| `FederationLinked` | ドメインイベント、監査の検索の候補 | 発行するコードがない |
| `FederationUnlinked` | 同上 | 同上 |

宣言だけが残ると、利用者は起きない応答やイベントに備えることになり、許容リストも減らせない。

## 対象範囲

- 6 件の宣言を消すか、実際に起きる経路を持たせるかを語ごとに決める。
  起きる経路を持たせる場合は、その条件を要件に書く。
- 消す語は、TypeSpec、ハンドラーの分岐、イベントの型、監査の検索の候補から消す。
- `tools/check/unspecified-vocabulary-debt.json` から、決めた語を消す。

## 対象外

- 連携の結び付けと解除の機能そのものの設計。
  `FederationLinked` と `FederationUnlinked` を発行する経路を持たせると決めた場合は、別の work item を起票する。
- CSV の既存の応答宣言の不足は [wi-17379](../active/wi-17379-declare-the-statuses-identity-management-csv-endpoints-return.md) が扱う。
- 今回変更していない CSV の境界値や監査の検索処理で見つかった変異の生存と未被覆の解消。

## 設計

6 語はすべて削除する。
`JobRepo` と `CSVArtifacts` は memory と PostgreSQL の起動構成が配線し、ルーターがハンドラーへ渡す必須依存である。
設定欠落を製品の 503 応答として公開する経路は追加しない。
エクスポート対象は User、Group、GroupMembership の専用ルートで決まるため、HTTP の `invalid_target` と対応する TypeSpec モデルを削除する。
内部で未知の対象を拒否する `ErrInvalidExportTarget` は維持する。
OAuth2 の `invalid_target` は対象外である。

`FederationLinked` と `FederationUnlinked` は発行元がないため、Go と TypeSpec の宣言、監査の検索候補、UI の分類、二つの負債リストから削除する。
連携機能の新設は行わない。
要件 ID の追加、変更、廃止はなく、TypeSpec の応答とイベント宣言を現在の要件へ合わせる。
仕様にない分岐は分類 (b) とし、利用者が到達できない設定欠落と対象指定を外部契約として維持しない。
依存、作用の確定境界、永続化の所有者は変更しないため、現在状態の設計文書の更新は不要である。

受け入れ RED の代替は、6 語を許容リストから外した `check-unspecified-vocabulary` の失敗とする。
新しい業務規則や内部計算は追加しないため、単体 RED の代替は、監査の検索候補 API が未発行イベントを返さないことを表明する HTTP テストとする。
既存の CSV テストで、必須依存を配線した preview、apply、参照とエクスポートの結果を維持する。
監査の既存候補は先行する特性化テストで固定し、outcome の `failure` を欠落させて検出能力を確認した。

## 計画

1. 語ごとに、消すか経路を持たせるかを決める。
2. 宣言と許容リストを更新する。

## タスク

- [x] T001 [Plan] 6 語を削除し、内部の未知対象の拒否と OAuth2 の同名コードを維持する。
- [x] T002 [Contract] TypeSpec、ハンドラー、イベントの宣言と監査 UI の分類を削除した。
  `check-spec`、`generate-contract`、`check-contract-drift` と HTTP パッケージ 4 件の通常テスト（230 テスト）が成功した。
  `TestAdminAuditSearchOptionsOmitsUnemittedFederationEvents` は削除前に 2 件の未発行イベントを返して失敗し、削除後に成功した。
  この宣言整理には新しい規範 ID と業務ロジックがないため、この HTTP 検査を単体 RED の代替とする。
- [x] T003 [Tooling] 未規定語彙と未発行イベントの許容リストを更新した。
  `check-unspecified-vocabulary` は削除前に対象 6 語をすべて検出して失敗し、削除後に成功した。
  到達不能な契約を削除するため、新しい受け入れユースケースの RED は設けず、この宣言検査を代替とする。
  `check-event-contract` も成功した。
- [x] T004 [Verify] 選択した局所検査、静的検査、UI の E2E と変異検査を実行し、成功、意図した互換性の後退、解析対象外と既存の検出不足を完了欄へ記録した。

## 検証

- `mise run check-spec`、`generate-contract`、`render-docs`：応答 union の変更と派生したルートメタデータ、文書を検査する。
- `mise run check-unspecified-vocabulary`、`check-event-contract`：宣言と未発行イベントの許容リストの整合を検査する。
- `mise run check-api-compat`：保存済みのベースラインとの後退を記録する。
  未公開のため `check-published-api-compat` を必須判定とし、通常変更でベースラインは更新しない。
- `mise run test-go-test`：監査、User と Group のインポート、データエクスポートの各 HTTP パッケージ全体を検査する。
  同期、goroutine、共有状態は変更しないため通常構成を選ぶ。
- `mise run test-ui-unit-file`、`test-ui-e2e-file`：監査画面の単体テストと `ui-scenario-actions.spec.ts` で絞り込みとエクスポートを検査する。
- `mise run lint-go`、`check-contract-drift`、`check-spec-impact`、`check-work-items`、`check-links`：変更した層の静的検査、契約、作業記録を検査する。
- `mise run format-check-ui`、`lint-ui`、`typecheck-ui`：監査 UI の語彙削除後の整形、静的検査、型を検査する。
- `mise run check-status-drift`：503 の宣言と実装の分岐の削除後のステータスを検査し、解析対象外を成功と区別する。
- `mise run check-go-mutation-tool` と `test-go-mutation`：固定した canary と変更した 5 個の Go パッケージを検査し、生存した変異を今回の削除と区別して記録する。
- 削除した宣言と分岐は変異オペレーターでは復元できないため、監査候補への再挿入と許容リストから外した検査で削除漏れを検出する。

## リスク

- エラーの宣言を消すと、OpenAPI の互換性の検査が後退として報告する。
  未公開なので、後退を記録し、公開状態に応じた検査で受け入れる。

## 完了

- **完了日**: 2026-10-11
- **要約**:
  `mise run spec-diff` は、TypeSpec の 6 宣言の削除と 12 操作の応答変更を報告した。
  User、Group、GroupMembership のインポートの開始、適用、参照の 9 操作から、依存未配線時の 503 を削除した。
  エクスポートの開始の 3 操作から、ルートで固定する対象の `InvalidExportTargetError` を削除した。
  未発行の `FederationLinked` と `FederationUnlinked` を宣言、監査の候補、UI の分類、二つの負債リストから削除した。
  新しい要件、連携機能、データの移行は追加していない。
- **受け入れ RED の証拠**:
  - **テスト**: `mise run check-unspecified-vocabulary`。
  - **要件**: N/A: 製品に到達経路のない宣言を削除する保守作業であり、新しい規範ユースケースはない。
  - **観測した失敗**: 許容リストから 6 語を外した時点で、対象のエラー 4 語とイベント 2 語が要件にないという 6 件の失敗を観測した。
  - **検出できる理由**: 対象モジュールの TypeSpec と Go の語彙を要件と比較するため、宣言の削除漏れを、許容リストによって隠さず検出できる。
- **単体 RED の証拠**:
  - **テスト**: `TestAdminAuditSearchOptionsOmitsUnemittedFederationEvents`（`mise run test-go-test -- ./backend/audit/handlers_http TestAdminAuditSearchOptionsOmitsUnemittedFederationEvents`）。
  - **要件**: N/A: 業務計算を変更せず、未発行のイベントを監査候補から外すため、HTTP の検索候補を観測する代替検査を選んだ。
  - **観測した失敗**: 削除前に検索候補が `FederationLinked` と `FederationUnlinked` を返し、両方の表明が失敗した。
  - **検出できる理由**: 正式な検索候補のルートから受け取った JSON を検査し、TypeSpec の削除だけで Go の候補が残る誤実装を区別する。
- **変更耐性の結果**:
  検索候補に削除した 2 イベントを再挿入すると、同じ回帰テストが 2 件の失敗を報告した。
  再挿入を戻した後は成功した。
  先行コミットの `TestCharacterizeAdminAuditSearchOptionsSupportedChoices` は、outcome の `failure` を欠落させると `outcomes=[success], want [success failure]` で失敗した。
  固定した canary は初回とキャッシュ利用時に期待した lived と not viable を区別した。
  宣言や配列要素の復活は機械の変異オペレーターで表現できないため、これらの手動注入と宣言検査を使った。

  `mise run test-go-mutation -- <package> 1 off <report>` の結果は次のとおりである。

  | パッケージ | 検出 | 生存 | 未被覆 | コンパイル不可 |
  | --- | ---: | ---: | ---: | ---: |
  | `backend/idmanagement/handlers_http` | 24 | 0 | 0 | 2 |
  | `backend/idmanagement/domain` | 65 | 7 | 3 | 2 |
  | `backend/audit/handlers_http` | 47 | 2 | 1 | 3 |
  | `backend/idmanagement/user/handlers_http` | 84 | 18 | 28 | 8 |
  | `backend/idmanagement/group/handlers_http` | 90 | 35 | 40 | 8 |

  全生存候補を読み、今回削除した宣言と nil 依存の分岐は機械変異の対象に残っていないことを確認した。
  User と Group のインポート参照で `boundaryOrdinal > 0` を `>= 0` にする境界変異は、0 の場合も初期値と同じ `startOrdinal = 1` になり、同値である。
  ほかの候補は同値と確定せず、既存の検出不足として記録する。
  CSV の転送ポリシーの 0、フィールドの上限、エラー行番号、インポート結果の kind と mode、空の result、カーソルと Link の境界、User と Group の一覧、ライフサイクルの日時、動的グループの応答、メンバーの名前解決、restore と required action のエラー、監査の limit と単一イベントの scope が該当する。
  未被覆も成功には数えない。
  これらは今回変更していない処理の検出不足であり、修正を対象外へ残した。
  レポートは `/tmp/wi-45298-backend-<package>-mutation.json` に保存した。
- **検証結果**:
  - 仕様：`check-spec`、`generate-contract`、`render-docs`、`check-spec-impact`、`check-published-api-compat` が成功した。
    生成サイトは 1,348 ページ、OpenAPI は 19 タグを持つ。
    応答だけの変更なので、生成したルートメタデータに差分はなかった。
  - 語彙：`check-unspecified-vocabulary` と `check-event-contract` が成功した。
    対象の語は回帰テストを除いて現在の宣言、ハンドラー、UI、負債リストに残っていない。
    OAuth2 の `invalid_target` は維持した。
  - Go：変更した HTTP パッケージ 4 件の通常テストは合計 230 テストが成功し、skip と not-run は 0 件だった。
    `lint-go` は 0 issues で成功した。
    共有状態や同期を変更していないため、race 構成の追加検査は選ばなかった。
  - UI：監査画面の単体 12 テストと `ui-scenario-actions.spec.ts` の E2E 19 テストが成功した。
    `format-check-ui`、`lint-ui`、`typecheck-ui` も成功した。
  - 契約：`check-contract-drift` は verified 196、unverified 146、failed 0、`check-status-drift` は verified 94、unverified 248、failed 0 だった。
    インポートの 9 操作の本文は解析済みだが、エクスポートの共有ハンドラーとエラー writer は静的解析の対象外である。
    対象外を成功と数えず、実行した HTTP テストで対応する結果を確認した。
    既存の応答宣言の不足は対象外の wi-17379 が扱う。
  - 互換性：`check-api-compat` は、既存ベースラインとの比較でインポート 9 操作の 503 削除を breaking change と報告した。
    `phase: unpublished` のため公開済み契約のゲートは成功し、ベースラインは更新していない。
  - レビュー：七つの視点で差分を確認し、削除対象と内部の未知対象の拒否を区別した。
    新しい依存、公開境界、共通化、作用は追加していない。
  - 作業記録：`check-work-items`、`check-links`、`git diff --check` が成功した。
