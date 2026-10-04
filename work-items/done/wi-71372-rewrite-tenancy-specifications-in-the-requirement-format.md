---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-04
priority: p2
depends_on: [wi-39141-move-tenancy-to-the-feature-and-design-layout, wi-86874-model-test-identity-management-state-machines-from-their-matrices]
change_kind: docs
evidence_policy: risk-based-v4
documentation_impact:
  level: release_note
  reason: 製品の振る舞いは変えないが、実装だけが守っていた System 管理者によるテナントの取得と設定の更新を要件として約束するので、依存してよい境界が増えたことを API の利用者へ知らせる。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-71372-rewrite-tenancy-specifications-in-the-requirement-format.md }
initial_context:
  specification:
    - SPECIFICATION_FORMAT.md
    - docs/domain/identity-management/agent/README.md
    - work-items/done/wi-83002-rewrite-remaining-identity-management-features-in-the-requirement-format.md
  typespec:
    - spec/contexts/tenancy/models.tsp
    - spec/contexts/tenancy/main.tsp
  source:
    - tools/check/src/check-unspecified-vocabulary.ts
    - tools/check/src/unspecified-vocabulary.ts
    - backend/shared/testing_statematrix/statematrix.go
    - backend/shared/http/support_http/tenant_middleware.go
    - backend/tenancy/usecases/manage_tenants.go
    - backend/tenancy/usecases/manage_notification_templates.go
    - backend/tenancy/usecases/manage_user_attribute_schema.go
    - backend/tenancy/usecases/manage_group_attribute_schema.go
    - backend/tenancy/handlers_http/admin_tenant_handler.go
    - backend/tenancy/handlers_http/admin_settings_handler.go
    - backend/tenancy/handlers_http/admin_branding_handler.go
    - backend/tenancy/handlers_http/admin_notification_template_handler.go
    - backend/tenancy/handlers_http/admin_user_attribute_schema_handler.go
    - backend/tenancy/handlers_http/admin_group_attribute_schema_handler.go
    - backend/tenancy/handlers_http/integration_endpoints_handler.go
  tests:
    - backend/idmanagement/handlers_http/user_lifecycle_model_test.go
    - backend/tenancy/handlers_http/implicit_rules_test.go
    - backend/tenancy/handlers_http/refusal_effects_test.go
  stop_before_reading: [frontend]
affected_spec:
  - { path: docs/domain/tenancy/attribute-schema/README.md, requirement: REQ-TENANCY-002 }
  - { path: docs/domain/tenancy/attribute-schema/README.md, requirement: REQ-TENANCY-020 }
  - { path: docs/domain/tenancy/attribute-schema/README.md, requirement: REQ-TENANCY-043 }
  - { path: docs/domain/tenancy/branding/README.md, requirement: REQ-TENANCY-004 }
  - { path: docs/domain/tenancy/branding/README.md, requirement: REQ-TENANCY-005 }
  - { path: docs/domain/tenancy/branding/README.md, requirement: REQ-TENANCY-032 }
  - { path: docs/domain/tenancy/branding/README.md, requirement: REQ-TENANCY-033 }
  - { path: docs/domain/tenancy/branding/README.md, requirement: REQ-TENANCY-034 }
  - { path: docs/domain/tenancy/branding/README.md, requirement: REQ-TENANCY-035 }
  - { path: docs/domain/tenancy/integration-endpoints/README.md, requirement: REQ-TENANCY-001 }
  - { path: docs/domain/tenancy/integration-endpoints/README.md, requirement: REQ-TENANCY-041 }
  - { path: docs/domain/tenancy/integration-endpoints/README.md, requirement: REQ-TENANCY-042 }
  - { path: docs/domain/tenancy/lifecycle/README.md, requirement: REQ-TENANCY-003 }
  - { path: docs/domain/tenancy/lifecycle/README.md, requirement: REQ-TENANCY-014 }
  - { path: docs/domain/tenancy/lifecycle/README.md, requirement: REQ-TENANCY-025 }
  - { path: docs/domain/tenancy/lifecycle/README.md, requirement: REQ-TENANCY-026 }
  - { path: docs/domain/tenancy/lifecycle/README.md, requirement: REQ-TENANCY-027 }
  - { path: docs/domain/tenancy/notification-template/README.md, requirement: REQ-TENANCY-015 }
  - { path: docs/domain/tenancy/notification-template/README.md, requirement: REQ-TENANCY-016 }
  - { path: docs/domain/tenancy/notification-template/README.md, requirement: REQ-TENANCY-017 }
  - { path: docs/domain/tenancy/notification-template/README.md, requirement: REQ-TENANCY-018 }
  - { path: docs/domain/tenancy/notification-template/README.md, requirement: REQ-TENANCY-038 }
  - { path: docs/domain/tenancy/notification-template/README.md, requirement: REQ-TENANCY-039 }
  - { path: docs/domain/tenancy/notification-template/README.md, requirement: REQ-TENANCY-040 }
  - { path: docs/domain/tenancy/quota/README.md, requirement: REQ-TENANCY-012 }
  - { path: docs/domain/tenancy/quota/README.md, requirement: REQ-TENANCY-013 }
  - { path: docs/domain/tenancy/quota/README.md, requirement: REQ-TENANCY-036 }
  - { path: docs/domain/tenancy/quota/README.md, requirement: REQ-TENANCY-037 }
  - { path: docs/domain/tenancy/resolution/README.md, requirement: REQ-TENANCY-006 }
  - { path: docs/domain/tenancy/resolution/README.md, requirement: REQ-TENANCY-007 }
  - { path: docs/domain/tenancy/resolution/README.md, requirement: REQ-TENANCY-008 }
  - { path: docs/domain/tenancy/resolution/README.md, requirement: REQ-TENANCY-009 }
  - { path: docs/domain/tenancy/resolution/README.md, requirement: REQ-TENANCY-010 }
  - { path: docs/domain/tenancy/resolution/README.md, requirement: REQ-TENANCY-011 }
  - { path: docs/domain/tenancy/resolution/README.md, requirement: REQ-TENANCY-022 }
  - { path: docs/domain/tenancy/resolution/README.md, requirement: REQ-TENANCY-023 }
  - { path: docs/domain/tenancy/resolution/README.md, requirement: REQ-TENANCY-024 }
  - { path: docs/domain/tenancy/settings/README.md, requirement: REQ-TENANCY-019 }
  - { path: docs/domain/tenancy/settings/README.md, requirement: REQ-TENANCY-021 }
  - { path: docs/domain/tenancy/settings/README.md, requirement: REQ-TENANCY-028 }
  - { path: docs/domain/tenancy/settings/README.md, requirement: REQ-TENANCY-029 }
  - { path: docs/domain/tenancy/settings/README.md, requirement: REQ-TENANCY-030 }
  - { path: docs/domain/tenancy/settings/README.md, requirement: REQ-TENANCY-031 }
  - { path: docs/domain/tenancy/settings/README.md, requirement: REQ-TENANCY-044 }
  - { path: docs/domain/tenancy/lifecycle/README.md, requirement: REQ-TENANCY-045 }
---

# Tenancy の機能仕様を、EARS 形式の要件だけで書く方式へ書き直す

## 動機

wi-39141 は Tenancy の文書を機能仕様と内部設計の構造へ移すが、規則の内容と書き方は変えない。
wi-17076 で定めた書き方（EARS 形式の要件、要約表と担保手段と要判断の廃止、状態遷移表（マトリクス形式））は、構成の移行の後に別に適用する必要がある。
Tenancy の文書は約 1,300 行あり、IdManagement に次いで大きい。

## 対象範囲

- wi-39141 で移した Tenancy の機能仕様を、`user` と同じ方式で書き直す。
  要件の ID は変えず、本文を EARS 形式に改める。
- テナントのライフサイクルの状態機械に状態遷移表（マトリクス形式）を加え、wi-86874 の方式でモデルベースのテストを加える。
- `check-unspecified-vocabulary` の対象に Tenancy を加え、導入時点の違反を許容リストに載せてから、書き直しで消す。
- 書き直しで見つけた未記載の振る舞いを、仕様にない振る舞いの分類に従って扱い、(c) は起票する。

## 対象外

- 構成の移行。
  wi-39141 が扱う。
- (c) に分類した振る舞いの実装の変更。

## 設計

書き方は `SPECIFICATION_FORMAT.md` と、wi-83002 による IdManagement の書き直しの結果に従う。
要件の義務を変える判断は、書き直しに紛れ込ませず、(a)(b)(c) の分類として記録する。

着手時に決めたこと：

- 要件の ID とタイトルは変えない。例の付録の `Rule` のタイトルと一致させ続けるためである。
- 本文が空だった要件（REQ-TENANCY-001 など 19 件）は、例の付録、実装、テストが固定している応答から要件文を書く。
- 担保手段の欄は消す。要判断の欄 6 件は、未決定の点がすべて wi-12979 の表に載っているので、欄だけを消す。
- 依存先の wi-86874 は未完了だが、表を実行時に読むモデルベースのテストの方式は wi-89346 が User で確立している。TenantLifecycle は非同期の処理を含まないので、wi-86874 の未決定の点（時刻とジョブの扱い、共通化）に依存しない。
- TenantLifecycle の状態遷移表の列は、無効化、再開、正規ロケーションへの要求とする。状態によって結果が変わるのは、この三つだけである。
- `TenantQuotaUpdated` は発行する経路がなく、wi-470 が発行を加える。許容リストに残す。
- 要件の見出しの例の欄に、その要件の例をすべて並べた。例の付録だけにあった約束を、要件から辿れるようにするためである。
- 例の付録が `InvalidRequestError` と書いていた拒否のうち、実装が別のコードを返す 4 件（`invalid_branding` 2 件、`invalid_notification_template` 2 件）は、テストが固定している応答に書き直し、そのコードをテストでも表明した。
- (c) に分類した振る舞いは、仕様を書き足して既存の実装を説明しない。要件には維持すべき約束だけを書き、食い違いは起票した work item が扱う。
- モデルベースのテストの予測と比較の部分は、User のテストと同じ形を繰り返す。二つ目の類似箇所なので、`testing_statematrix` への移動は wi-86874 の判断に委ねる。

### 未記載の振る舞いの分類

| # | 見つけた振る舞い | 見つけた工程 | 分類 | 扱い |
| --- | --- | --- | --- | --- |
| 1 | 属性スキーマの保存は `TenantUserAttributeSchemaUpdated`、`TenantGroupAttributeSchemaUpdated` を、通知テンプレートの上書きの保存は `NotificationTemplateUpdated` を、Hard Quota の拒否は `QuotaExceeded` を発行する | 許容リスト | (a) | REQ-TENANCY-002、020、017、013 |
| 2 | System 管理者による realm を指定したテナントの設定の更新（`UpdateTenant`）が、どの要件にもない | 書き直し | (a) | REQ-TENANCY-044 を加えた |
| 3 | System 管理者による realm を指定したテナントの取得（`GetTenant`）が、どの要件にもない | 書き直し | (a) | REQ-TENANCY-045 を加えた |
| 4 | default テナントの無効化は 400 `invalid_request` で拒否し、`Active` のテナントの再開も `TenantEnabled` を発行する | 状態遷移表 | (a) | REQ-TENANCY-003、027、TenantLifecycle |
| 5 | 例の付録が `InvalidRequestError` と書いていた拒否を、実装は 400 `invalid_branding`、400 `invalid_notification_template` で返す | 書き直し | (a) | REQ-TENANCY-032、017、038 と例の付録を直した |
| 6 | `tenant_conflict`、`invalid_branding`、`invalid_notification_template` を TypeSpec が宣言していない | 書き直し | (c) | wi-30417 |
| 7 | User と Group の CSV の取り込みは、Hard Quota で止まっても `QuotaExceeded` を発行しない | 許容リスト | (c) | wi-52861。REQ-TENANCY-013 には維持すべき約束だけを書いた |
| 8 | `TenantQuotaUpdated` を発行する経路がなく、正規ロケーションの切り替えもイベントを発行しない | 許容リスト | (c) | 既存の wi-470。許容リストに残した |
| 9 | 上限の更新はパスの引数を realm ではなく `id` として読み、存在しないテナントを 404 で拒否しない | 書き直し | (c) | 既存の wi-471 |
| 10 | Soft の上限（`audit_events_retained`、`export_artifacts_bytes`）を数えず、警告も発行しない | 書き直し | (c) | 既存の wi-38516。モデルの節は変えていない |

候補のうち漏れでないと判断したもの：

- 管理画面の色ごとの「デフォルトに戻す」表示は、画面の状態であり、仕様に書かない。空文字列で未設定に戻す約束は REQ-TENANCY-032 が持つ。
- 負の上限を保存すること（REQ-TENANCY-037）、値の変わらない項目を `changed_fields` に載せること（REQ-TENANCY-031）などの要判断の 6 件は、現在の挙動を要件に書き、決定は wi-12979 に残した。

## タスク

- [x] T001 [Spec] 機能仕様を書き直す。検査：`mise run check-spec`。
- [x] T002 [Spec] 状態遷移表（マトリクス形式）を加える。検査：`mise run check-spec`。
- [x] T003 [Test] モデルベースのテストを加える。検査：`mise run test-go-test -- ./backend/tenancy/handlers_http 'TestTenantLifecycle.*'`。
- [x] T004 [Tooling] 語彙の検査の対象に Tenancy を加え、許容リストを減らす。検査：`mise run check-unspecified-vocabulary`。
- [x] T005 [Verify] 変更を検証する。

## 検証

- `mise run check-spec`、`mise run check-unspecified-vocabulary`
- `mise run spec-diff` で、要件の差分が本文の書き直しと、状態遷移表の追加だけであること。
- `mise run verify`

## リスク

- テナントの解決とリソース上限は、セキュリティの境界に関わる。
  拒否の条件を述べる文を落とさないよう、書き直しの前後で義務を突き合わせる。

## 完了
- **Completed At**: 2026-10-04
- **Summary**:
  `mise run spec-diff -- main` の結果、規範の差分は REQ-TENANCY-001 から 043 の本文の書き直し、REQ-TENANCY-044 と 045 の追加、TenantLifecycle の状態遷移の変更（状態遷移表（マトリクス形式）と自己遷移の追加）であり、要件の削除はない。
  追加した 2 件は、実装だけが守っていた System 管理者によるテナントの設定の更新と取得を要件にしたもので、振る舞いは変えていない。
  本文が空だった 19 件の要件に要件文を書き、担保手段の欄と要判断の欄 6 件を消した。要判断の未決定の点は wi-12979 に残っている。
  9 機能の `README.md` は 535 行から 675 行に、Tenancy の設計を除く全文書は 1,531 行から 1,671 行になった。空だった要件に本文を書いたので増えた。
  `check-unspecified-vocabulary` の対象に Tenancy を加え、導入時点の 5 件のうち要件に書いた 4 件を許容リストから消した。残る `TenantQuotaUpdated` は wi-470 が扱う。
  TenantLifecycle の状態遷移表を実行時に読み、全セルと種を固定した乱数の操作列で HTTP の結果と比べるモデルベースのテストを加えた。
  未記載の振る舞いを設計の表のとおりに分類し、(c) を wi-30417、wi-52861 として起票し、既存の wi-470、wi-471、wi-38516 に対応づけた。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-unspecified-vocabulary`
  - **Requirement**: N/A: 仕様の書き方の変更であり、既存の要件の義務を変えない。
  - **Observed Failure**: 9 機能を書き直した後、許容リストを変える前の検査は、`TenantUserAttributeSchemaUpdated`、`TenantGroupAttributeSchemaUpdated`、`QuotaExceeded`、`NotificationTemplateUpdated` の 4 件を「appears in a requirement now」として失敗した。
  - **Detection Reason**: 要件に書いた語が許容リストに残ることと、書き直しで要件から語が落ちることの両方を、要件の本文と遷移の表から区別する。
- **Unit RED Evidence**:
  - **Test**: `mise run test-go-test -- ./backend/tenancy/handlers_http 'TestTenantLifecycle.*'`
  - **Requirement**: REQ-TENANCY-027
  - **Observed Failure**: 実装に合致する表では初回から成功したので、故障を注入して確かめた。`Disabled × 正規ロケーションへの要求` のセルを `何もしない` にすると `want 2xx` で、default の無効化の拒否を外すと `status=204 code="", want 400 invalid_request` で、再開の `TenantEnabled` の発行を外すと `イベント [] を発行した。表は [TenantEnabled] を予測する` で失敗した。
  - **Detection Reason**: 表のセルと、状態、応答、イベントの三つの観測結果を一つずつ突き合わせるので、表と実装のどちら側の食い違いも検出する。
- **Change-Resistance Results**:
  `mise run test-go-mutation -- backend/tenancy/handlers_http` は 152 件の変異のうち 125 件を検出し、17 件が生き残った。
  生き残りのうち、要件に書いた結果を表明するテストがないものは、ブランド設定の更新の `changed_fields`（REQ-TENANCY-004）、テナント管理者による設定の更新の `changed_fields` の `display_name` 以外の項目（REQ-TENANCY-031）、カスタム属性のないテナントの `attributes` の空の配列（REQ-TENANCY-002、020）、署名証明書のフィンガープリントの区切りの位置（REQ-TENANCY-041）である。この記録ではテストを加えていない。
  残る 2 件は、テナント管理者による設定の取得で上限と使用量を読めなかったときの省略であり、どの要件にも書いていない。リソース上限のセキュリティ上の考慮が参照できることだけを述べており、読めないときの扱いは (b) とした。
- **Verification Results**:
  - `mise run check-spec` - 成功
  - `mise run check-unspecified-vocabulary` - 成功
  - `mise run check-work-items` - 成功
  - `mise run test-go-package -- ./backend/tenancy/handlers_http` - 成功
  - `mise run lint-go` - 成功（1 回目は `thelper` で失敗し、`t.Helper()` を加えて成功）
  - `mise run verify` - 成功
