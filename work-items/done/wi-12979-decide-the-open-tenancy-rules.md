---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-01
priority: p2
depends_on: [wi-78471-transcribe-implicit-specifications-of-tenancy]
change_kind: bugfix
evidence_policy: risk-based-v4
documentation_impact:
  level: release_note
  reason: テナントの再無効化、フッターリンクのラベルの上限、負のクォータ上限の扱いが変わり、System 管理者とテナント管理者が観測する結果が変わる。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-12979-decide-the-open-tenancy-rules.md }
initial_context:
  specification:
    - docs/domain/tenancy/lifecycle/README.md#REQ-TENANCY-026
    - docs/domain/tenancy/lifecycle/README.md#REQ-TENANCY-027
    - docs/domain/tenancy/settings/README.md#REQ-TENANCY-031
    - docs/domain/tenancy/branding/README.md#REQ-TENANCY-032
    - docs/domain/tenancy/quota/README.md#REQ-TENANCY-037
    - docs/domain/tenancy/notification-template/README.md#REQ-TENANCY-039
  typespec: [IdMagic.Contract.TenantQuotaUpdateRequest]
  source:
    - backend/tenancy/usecases/manage_tenants.go
    - backend/tenancy/domain/tenancy.go
    - backend/tenancy/handlers_http/admin_tenant_handler.go
  tests: [backend/tenancy/handlers_http/implicit_rules_test.go]
  stop_before_reading: [frontend]
affected_spec:
  - { path: docs/domain/tenancy/lifecycle/README.md, requirement: REQ-TENANCY-026 }
  - { path: docs/domain/tenancy/lifecycle/README.md, requirement: REQ-TENANCY-027 }
  - { path: docs/domain/tenancy/settings/README.md, requirement: REQ-TENANCY-031 }
  - { path: docs/domain/tenancy/branding/README.md, requirement: REQ-TENANCY-032 }
  - { path: docs/domain/tenancy/quota/README.md, requirement: REQ-TENANCY-037 }
  - { path: docs/domain/tenancy/notification-template/README.md, requirement: REQ-TENANCY-039 }
  - { path: spec/contexts/tenancy/models.tsp, symbol: IdMagic.Contract.TenantQuotaUpdateRequest }
primary_use_cases:
  - id: redisable-keeps-disabled-at
    requirement: REQ-TENANCY-027
    observable_result: 無効なテナントをもう一度無効化しても 204 と TenantDisabled が返り、disabled_at は一度目の無効化の時刻のまま残る。
    boundary: acceptance
    test: { path: backend/tenancy/handlers_http/implicit_rules_test.go, name: TestDisablingADisabledTenantSucceedsAgain, task: test-go-race }
    fault_model: 状態を確かめずに disabled_at を要求の時刻で上書きし、無効化が始まった時刻を失う。
  - id: footer-label-counts-characters
    requirement: REQ-TENANCY-032
    observable_result: 80 文字の日本語のラベルは保存され、81 文字のラベルは 400 invalid_branding で拒否される。
    boundary: acceptance
    test: { path: backend/tenancy/handlers_http/implicit_rules_test.go, name: TestBrandingRefusesLongMultibyteLabelsAndUppercaseSchemes, task: test-go-race }
    fault_model: ラベルの長さを UTF-8 のバイト数で数え、27 文字以上の日本語のラベルを拒否する。
  - id: negative-quota-refused
    requirement: REQ-TENANCY-037
    observable_result: 負の上限を含むクォータ更新は 400 invalid_request で拒否され、保存済みの上書きは変わらない。
    boundary: acceptance
    test: { path: backend/tenancy/handlers_http/implicit_rules_test.go, name: TestQuotaUpdateRefusesNegativeLimits, task: test-go-race }
    fault_model: 負の上限を検証せずに保存し、上書きの全体を置き換えてしまう。
---

# Tenancy の規則に残した要判断を決める

## 動機

Tenancy の既存コードを書き起こしたとき、意図が疑わしいが外部から依存され得る挙動を、現在の挙動のまま規則に書き、要判断の欄に未決定の点を残した。
要判断は、決着したら消す欄である。
残したままでは、規則を読む人は現在の挙動を維持すべきか是正すべきかを判断できない。

| 規則 | 未決定の点 |
| --- | --- |
| REQ-TENANCY-026 | 上限と使用量を読み取れないテナントを、項目を省いて一覧へ返す |
| REQ-TENANCY-027 | すでにその状態にあるテナントの無効化と再開が、`disabled_at` を動かしイベントを重ねて発行する |
| REQ-TENANCY-031 | 値が変わらない項目も `TenantUpdated` の `changed_fields` に載る |
| REQ-TENANCY-032 | フッターリンクのラベルを UTF-8 で 80 バイト以下に制限し、日本語のラベルを 26 文字までにしている |
| REQ-TENANCY-037 | 負の上限を検証せずに保存する |
| REQ-TENANCY-039 | 何も削除しなかったリセットも `NotificationTemplateReset` を発行する |

## 対象範囲

- 表の各点について、現在の挙動を維持するか是正するかを決める。
- 維持する点は、理由をその要件の **判断** の欄に書く。要判断の欄は wi-71372 の書き直しで消し、未決定の点はこの表だけに残っている。代替案を比べた判断であれば、`design/decisions.md` に書いてリンクする。
- 是正する点は、規則文を改め、実装とテストを合わせる。

## 対象外

- 表にない挙動の変更。
- クォータ更新の経路の規約合わせ。`wi-471` が扱う。経路の対象を realm で解決しないこと、更新がイベントを発行しないことも、この規約合わせに含める。

## 設計

### 決定

利用者の確認を経て、次のように決めた。

| 規則 | 決定 | 理由 |
| --- | --- | --- |
| REQ-TENANCY-026 | 維持 | 上限と使用量は一覧の補助情報である。その読み取りの障害で、テナントの一覧と、一覧から辿る無効化や再開まで失わせない |
| REQ-TENANCY-027 | 是正（`disabled_at` だけ） | 204 とイベントの発行は維持する。イベントの購読者は監査ログだけであり、操作者の操作を毎回記録する。`disabled_at` は無効化が始まった時刻を表すので、再操作で動かさない |
| REQ-TENANCY-031 | 維持 | `changed_fields` は操作者が要求した項目の記録であり、値の差分ではない。同じ値の再送という操作者の意図も監査に残す |
| REQ-TENANCY-032 | 是正 | 要件、ドメインのスキーマ（`spec.CharsAtMost`）、データベースの `char_length` 制約、管理 UI はどれも文字数で数えている。`validTenantFooterLink` の `len()` だけがバイト数で数えている |
| REQ-TENANCY-037 | 是正 | 上限 0 ですでに作成を拒否できるので、負の上限に固有の意味がない。400 と `invalid_request` で拒否する |
| REQ-TENANCY-039 | 維持 | リセットの記録は、操作者が組み込みの文面へ戻すことを求めた記録であり、上書きの有無に依存させない |

いずれも一つの要件だけを正当化する判断なので、`design/decisions.md` ではなく各要件の **判断** の欄に書く。

### 要件の差分

- REQ-TENANCY-026：**判断** を追加する。
- REQ-TENANCY-027：無効化の文を「テナントが `Active` の間」に限り、`Disabled` の間の無効化は「`disabled_at` を変えず、204 を返し、`TenantDisabled` を発行する」に改める。**判断** を追加する。EX-TENANCY-027-01 の期待値を「`disabled_at` は一度目の要求の時刻のまま」に改める。
- REQ-TENANCY-031：**判断** を追加する。
- REQ-TENANCY-032：「UTF-8 で 81 バイト以上のフッターリンクのラベル」の文を削る。EX-TENANCY-032-02 を「81 文字の日本語のラベル」に改める。
- REQ-TENANCY-037：「負の上限を指定されたとき、その上限を保存する」と「上限が負のリソースでは作成を拒否する」の 2 文を、「負の上限を指定された場合、400 と `invalid_request` で拒否し、どの上書きも保存しない」に置き換える。TypeSpec の `TenantQuotaUpdateRequest` の各項目に `@minValue(0)` を付ける。
- REQ-TENANCY-039：**判断** を追加する。

### 実装

- `usecases.SetDisabled(ctx, repo, id string, disabled bool, now time.Time) (*domain.Tenant, error)`：保存済みのテナントが `Disabled` で無効化を求められたときは、`DisabledAt` を保ったまま保存する。時刻は引数 `now` から入る。
- `domain.validTenantFooterLink(link TenantFooterLink) bool`：長さの検査を削り、完全性と HTTPS だけを検査する。長さは `tenantFooterLinkSchema` が文字数で検査する。
- `handlers_http.Deps.handleUpdateTenantQuota`：本文のデコードの後、負の値を含む要求を 400 と `invalid_request` で拒否し、`SetQuota` を呼ばない。

### 仕様にない振る舞いの分類

| 見つけた挙動 | 分類 | 反映先 |
| --- | --- | --- |
| フッターリンクの URL の長さも `len()` でバイト数を数えていた | (c) 実装を直す。要件は「2,049 文字以上」と文字数で定めている | ラベルと同じ修正で、スキーマの文字数検査に任せる |
| クォータ更新が対象を realm で解決せず、イベントも発行しない | (b) この記録では書かない | 対象外。`wi-471` の規約合わせが扱う |

## 計画

1. 各点の利用者への影響と、是正した場合の互換性を確かめる。
2. 維持と是正を決め、規則、判断、実装、テストを更新する。

## タスク

- [x] T001 [Plan] 各点を維持するか是正するかを決める。
- [x] T002 [Spec] 要件と判断の記述を更新する。`mise run check-spec`、`mise run check-api-compat`。
- [x] T003 [App] 是正する点の実装とテストを更新する。RED、GREEN は `mise run test-go-test -- ./backend/tenancy/handlers_http <test>`、まとまったら `mise run test-go-package -- ./backend/tenancy/...`、`mise run lint-go`。
- [x] T004 [Verify] 変更を検証する。`mise run test-go-mutation`、`mise run verify`。

## 検証

- `mise run check`
- `mise run verify`

## リスク

- イベントの発行を減らすと、そのイベントを数える利用者の集計が変わる。
  この記録ではイベントの発行を減らさない。
- 負の上限を保存していた運用は 400 で拒否されるようになる。製品は未リリースなので、保存済みの負の上限の移行は要らない。リリースノートで知らせる。

## 完了

- **完了日**: 2026-10-08
- **要約**:
  `mise run spec-diff` は REQ-TENANCY-026、027、031、032、037、039 と TypeSpec の `TenantQuotaUpdateRequest` の変更を示す。
  REQ-TENANCY-027 では、すでに無効なテナントの再度の無効化で `disabled_at` を変えないようにした。204 と `TenantDisabled` の発行は維持する。
  REQ-TENANCY-032 では、フッターリンクのラベルを UTF-8 のバイト数で制限する文を削り、ラベルと URL の長さを要件どおり文字数で数えるようにした。
  REQ-TENANCY-037 では、負の上限を保存する文を、400 と `invalid_request` で拒否しどの上書きも保存しない文に置き換え、`TenantQuotaUpdateRequest` の各項目に `@minValue(0)` を付けた。
  REQ-TENANCY-026、031、039 は振る舞いを維持し、理由を **判断** の欄に記した。
- **主要ユースケースの証拠**:
  - id: redisable-keeps-disabled-at
    red: 二度目の無効化で disabled_at が要求の時刻に上書きされ、TestDisablingADisabledTenantSucceedsAgain が「want the first request's time kept」で失敗した。
    fault_injection: SetDisabled の状態の確認を `if true` に置き換えると、TestDisablingADisabledTenantSucceedsAgain が同じ表明で失敗した。
  - id: footer-label-counts-characters
    red: 80 文字（240 バイト）の日本語のラベルが 400 invalid_branding で拒否され、TestBrandingRefusesLongMultibyteLabelsAndUppercaseSchemes が失敗した。
    fault_injection: validTenantFooterLink に `len(link.Label) <= 80` を戻すと、TestBrandingRefusesLongMultibyteLabelsAndUppercaseSchemes が同じ 400 で失敗した。
  - id: negative-quota-refused
    red: groups を -1 にする更新が 200 で保存され、TestQuotaUpdateRefusesNegativeLimits が「status = 200」で失敗した。
    fault_injection: ハンドラーの負の上限の検査を `if false && ...` で外すと、TestQuotaUpdateRefusesNegativeLimits が 200 で失敗した。`*limit < 0` を `<= 0` にすると、上限 0 の保存が 400 になり同じテストが失敗した。
- **変更耐性の結果**:
  `mise run test-go-mutation` を `backend/tenancy/usecases`、`backend/tenancy/handlers_http`、`backend/tenancy/domain` に実行した。
  `admin_tenant_handler.go` の変更行に生存した変異はない。
  `HasNegativeLimit` と `validTenantFooterLink` は `domain` パッケージのテストからは実行されず未被覆と報告されるが、`handlers_http` のテストが上の手書きの故障をすべて検出した。
  ほかの生存変異（`brandingChangedFields` と `adminSettingsChangedFields` の条件の否定など）は既存のコードにあり、この記録の変更行ではない。
- **検証結果**:
  - `mise run check-spec` - 成功
  - `mise run check-api-compat` - 成功
  - `mise run test-go-changed` - 成功
  - `mise run lint-go` - 成功
  - `mise run verify` - 成功（データベースのテストを通すためサンドボックス外で実行）
  - `mise run test-ui-e2e` - 未実行。変更はバックエンドの検証と仕様に閉じ、管理 UI はすでにラベルを文字数で検査しているため、ブラウザーへ届く変更がない
