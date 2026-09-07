---
status: completed
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-09-03
change_kind: bugfix
priority: p1
depends_on: []
evidence_policy: risk-based-v3
documentation_impact:
  level: release_note
  reason: "Cookie セッションからのクォータ更新は CSRF トークンを伴わないと拒否されるようになる。セッション Cookie だけでこの操作を呼ぶ運用があれば、リリース読者はその経路が閉じたことを知る必要がある。"
  references:
    - { kind: release_note, path: docs/releases/changes/wi-467.md }
initial_context:
  specification:
    - docs/contexts/tenancy/scenarios.feature.md#REQ-TENANCY-012
    - docs/contexts/tenancy/decisions.md
  typespec:
    - IdMagic.Tenancy.Operations.UpdateTenantQuota
  source:
    - backend/tenancy/handlers_http/admin_tenant_handler.go
    - backend/tenancy/handlers_http/routes.go
    - backend/shared/http/support_http/csrf.go
    - backend/shared/http/server_http/routes.go
    - frontend/src/api/admin.ts
  tests:
    - backend/shared/http/support_http/csrf_test.go
    - backend/shared/http/server_http/tenant_routes_test.go
    - backend/shared/http/server_http/control_plane_boundary_test.go
  stop_before_reading: [infra, load, docs/runbooks, frontend/src/features]
affected_spec:
  - { path: docs/contexts/tenancy/scenarios.feature.md, requirement: REQ-TENANCY-012 }
  - { path: spec/contexts/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.UpdateTenantQuota }
primary_use_cases:
  - id: tenant-quota-csrf
    requirement: REQ-TENANCY-012
    observable_result: CSRF トークンを伴わない Cookie セッションからのクォータ更新は HTTP 403 で拒否され、保存済みのクォータも利用量も変わらない。
    unit_test:
      path: backend/tenancy/handlers_http/admin_tenant_handler_test.go
      name: TestUpdateTenantQuotaRefusesCookieSessionWithoutCSRF
      task: test-go-race
    e2e_test:
      path: backend/shared/http/server_http/tenant_quota_csrf_test.go
      name: TestUpdateTenantQuotaRejectsCookieSessionWithoutCSRF
      task: test-go-race
    unit_fault_model: ハンドラーが検証を副作用より後ろに置き、403 を書きながら `QuotaRepo.SetQuota` まで進む。
    e2e_fault_model: クォータ経路だけが `VerifyBrowserRequest` を通らず、ほかの制御面変更ハンドラーと合成が食い違う。
---

# テナントクォータ更新で Origin と CSRF トークンを検証する

## Motivation

Tenancy の決定は、状態を変える管理リクエストについて、セッション認証に加えて `Origin` と CSRF トークンの検証を要求している。

テナントの作成、属性更新、停止、再開、正規ロケーション切替は、ハンドラーの先頭で `VerifyBrowserRequest` を呼ぶ。

しかし `UpdateTenantQuota` だけはこの検証を行わず、認証と `system_admin` の確認後にクォータを保存する。

フロントエンドは `X-Csrf-Token` を送信しているが、サーバーが値を検証しないため、画面の実装だけが安全性を示している状態になっている。

## Scope

- `UpdateTenantQuota` の副作用より前に `VerifyBrowserRequest` を実行する。
- Cookie セッションによる要求は、正しい `Origin`、テナントに対応する CSRF Cookie、`X-Csrf-Token` がすべて一致する場合だけ許可する。
- `Authorization` ヘッダーによる正規の Bearer または DPoP 資格情報は、既存の `VerifyBrowserRequest` 契約どおり Cookie CSRF 検査の対象外とし、認証、スコープ、制御面主体の検査は維持する。
- CSRF 拒否時にクォータと利用量のいずれも変わらないことを HTTP 受け入れテストで確認する。
- Tenancy のほかの制御面変更ハンドラーを棚卸しし、同じ欠落がないことを回帰テストで固定する。

## Out of Scope

- `VerifyBrowserRequest` の double-submit Cookie 方式を別方式へ置き換えること。
- API アクセストークンから `UpdateTenantQuota` を呼べなくすること。
  [[wi-461-control-plane-credential-boundary]] はクォータ更新の自動化を維持する。
- システムコンソールに要求する認証強度または再認証時刻。
  [[wi-468-system-console-privileged-session-assurance]] が扱う。
- クォータ値の妥当性、同時更新、監査イベントを変更すること。
- 403 が名乗る本体を `invalid_origin` と `csrf_failed` へ広げること。
  [[wi-454-declared-403-body-vs-guard-error-codes]] が扱う。この作業項目は既存の宣言と同じ 403 の範囲に収まる。
- `handleUpdateTenantQuota` が兄弟の経路と食い違っているそのほかの点。
  テナント識別子の解決、`WriteAdminAccessError` を通した拒否の写像、404、内部エラー文字列の露出、`DecodeJSON` と `NoStoreJSON` は
  [[wi-471-align-the-tenant-quota-update-route-with-the-admin-api-conventions]] が扱う。同項目はこの作業項目に依存しており、
  ここで先に検査を入れておけば、規約合わせの変更が CSRF の欠落を再現しないことをテストが押さえる。

## Design

`VerifyBrowserRequest` は資格情報の種類を見て、Cookie を周囲資格情報として使う要求だけに Origin と CSRF の検査を適用する。

そのため `UpdateTenantQuota` の冒頭で同じ関数を呼んでも、`tenants:write` を持つ非ブラウザーの Bearer または DPoP クライアントは従来どおり呼び出せる。

拒否検査は `tenant_id` の解決、リクエストボディのデコード、`QuotaRepo.SetQuota` より前に置く。

認可後に置くと、CSRF 要求が対象の存在や本文の妥当性を観測できるため採らない。

ルーターへ個別の CSRF ミドルウェアを追加する案も採らない。

同じルート群には読出しと書込みが混在し、Bearer と DPoP の例外判断を既存関数と二重に実装することになるためである。

変更する論理はハンドラー冒頭の 1 つの防御呼び出しだけであり、新しいドメイン型も操作シグネチャも導入しない。

時刻、乱数、識別子生成、設定はこの経路に現れない。永続化は `QuotaRepo` ポートの背後にあり、テストはそのポートの呼び出し回数を直接観測して「拒否したのに保存した」誤実装を検出する。

## Plan

1. CSRF トークンを持たない Cookie セッションが `UpdateTenantQuota` を成功させる現在の挙動を HTTP 境界で観測し、RED を確認する。
2. `VerifyBrowserRequest` を副作用より前へ追加する。
3. CSRF 拒否で `QuotaRepo.SetQuota` が呼ばれず、保存済みクォータが変わらないことを確認する。
4. 正しい CSRF を持つシステムコンソールと、正規の Bearer または DPoP クライアントの成功を確認する。
5. 制御面変更ハンドラーの棚卸し結果をテスト名または作業項目の完了記録へ残す。

## Tasks

- [x] T001 [Acceptance] CSRF のない Cookie セッションでクォータが変更される現在の挙動を観測し、RED を確認する。
  REQ-TENANCY-012 / `TestUpdateTenantQuotaRejectsCookieSessionWithoutCSRF`
  (`backend/shared/http/server_http/tenant_quota_csrf_test.go`)。
- [x] T002 [App] `UpdateTenantQuota` の冒頭へ `VerifyBrowserRequest` を追加する。
  `backend/tenancy/handlers_http/admin_tenant_handler.go`。
- [x] T003 [Unit] 拒否時に保存ポートが呼ばれないことを確認する。
  REQ-TENANCY-012 / `TestUpdateTenantQuotaRefusesCookieSessionWithoutCSRF`
  (`backend/tenancy/handlers_http/admin_tenant_handler_test.go`)。
- [x] T004 [Acceptance] 不正 Origin、CSRF 欠落、不一致を拒否し、正しい Cookie セッションと非周囲資格情報を許可することを確認する。
  REQ-TENANCY-012 / `TestUpdateTenantQuotaAcceptsSystemConsoleSessionWithCSRF`、
  `TestUpdateTenantQuotaAcceptsMatchingOriginAndCSRFToken`、`TestUpdateTenantQuotaAcceptsNonAmbientCredentials`、
  `TestUpdateTenantQuotaRefusesBeforeDecodingTheBody`。
- [x] T005 [Inventory] 制御面変更ハンドラーの Origin と CSRF 検査を棚卸しする。
  REQ-TENANCY-012 / `TestTenancyStateChangingAdminRoutesVerifyBrowserRequests`
  (`backend/shared/http/server_http/tenant_quota_csrf_test.go`) が Tenancy の状態変更ルート 16 本を表として固定する。
- [x] T006 [Verify] 仕様とセキュリティ制御の検査を通す。

## Verification

- `mise run test-go-race`
- `mise run check-security-controls`
- `mise run report-security-test-gaps`
- `mise run check-spec`
- `mise run check-work-items`
- `mise run verify`

## Risk Notes

拒否応答だけを確認しても、応答を書いた後に保存処理が進む誤実装を検出できない。

受け入れテストは HTTP 403 に加え、既存クォータが変わらず保存ポートも呼ばれないことを確認する。

Bearer または DPoP まで一律に CSRF 必須へすると、周囲資格情報ではない正規の自動化を壊す。

資格情報別の許可側テストを同じ変更に含める。

セキュリティ上の前提は、Cookie が周囲資格情報であり `Authorization` ヘッダーはそうではない、という `VerifyBrowserRequest` の既存の線引きをそのまま使うことである。

互換性の前提は、システムコンソールが既に `X-Csrf-Token` を送っており (`frontend/src/api/admin.ts` の `updateAdminTenantQuota`)、画面側の変更を必要としないことである。移行は不要で、退避はこの 1 行の防御呼び出しを戻すだけで済む。

新しい公開記号や規範 ID を割り当てず、既存のセキュリティ決定へ実装を一致させる修正であるため、`reversibility` は reversible とする。

## Completion

- **Completed At**: 2026-09-05
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返す。規範仕様は変わっておらず、この作業は Tenancy の決定 (状態を変える管理リクエストはセッション認証に加えて `Origin` と CSRF トークンを検証する) が要求していた検査を、唯一それを欠いていた `UpdateTenantQuota` へ実装した差分である。意味の差は、Cookie セッションからのクォータ更新が CSRF トークンの一致を伴わなければ成立しなくなったことに尽きる。`Authorization` ヘッダーの Bearer と DPoP は従来どおり Cookie CSRF 検査の対象外である。
- **Primary Use Case Evidence**:
  - id: tenant-quota-csrf
    unit_red: "`TestUpdateTenantQuotaRefusesCookieSessionWithoutCSRF` は 5 つの部分ケースすべてで失敗した。`status = 200, want 403` に加え `QuotaRepo.SetQuota calls = 1, want 0`、`stored quota.users = 20000, want 100` を観測した。"
    e2e_red: "`TestUpdateTenantQuotaRejectsCookieSessionWithoutCSRF` は `status = 200, want 403`、`QuotaRepo.SetQuota calls = 1, want 0`、`stored quota.users = 20000, want 100` で失敗した。同じ変更前の状態で `TestTenancyStateChangingAdminRoutesVerifyBrowserRequests` は 16 ルート中 `PUT /realms/default/api/admin/v1/tenants/acme/quota` の 1 本だけが `status = 200, want 403` で落ち、欠落がこの経路に限られることを示した。"
    unit_fault_injection: "`VerifyBrowserRequest` の戻り値を捨てる実装 (`_ = d.VerifyBrowserRequest(c)`) にすると、応答は 403 のままだが `QuotaRepo.SetQuota calls = 1` と `stored quota.users = 20000` で検出された。検査を `SetQuota` の後ろへ移した実装も同じ 2 つの表明で検出された。"
    e2e_fault_injection: "同じ 2 つの誤実装で `TestUpdateTenantQuotaRejectsCookieSessionWithoutCSRF` が `QuotaRepo.SetQuota calls = 1, want 0` と `stored quota.users = 20000, want 100` で失敗した。検査の呼び出しを削除した状態では、加えて `TestTenancyStateChangingAdminRoutesVerifyBrowserRequests` のクォータ行が失敗した。"
- **Change-Resistance Results**:
  変更した論理は防御呼び出し 1 つなので、その自由度 (有無、戻り値の扱い、位置) を網羅する 4 つの変異を実際に適用し、テストを実行した。
  - M1 検査そのものを削除する: 検出された (`TestUpdateTenantQuotaRefusesCookieSessionWithoutCSRF` の 5 ケース、`TestUpdateTenantQuotaRejectsCookieSessionWithoutCSRF`、`TestTenancyStateChangingAdminRoutesVerifyBrowserRequests`)。これは実装前の RED そのものである。
  - M2 検査を `QuotaRepo.SetQuota` の後ろへ移す: 検出された。応答は 403 になるが `SetQuota` の呼び出し回数と保存済みの値が変わる。応答コードだけを見る表明では取り逃がす変異であり、ポートの観測を入れた理由である。
  - M3 戻り値を捨てる (`_ = d.VerifyBrowserRequest(c)`): 検出された。`echo: response already written to client` が出て応答は 403 のままなので、状態の観測がなければ通ってしまう。`mise run check-security-controls` の R2 も同じ形を静的に禁じている。
  - M4 検査を本文デコードの後ろ、`SetQuota` の前へ移す: 検出された (`TestUpdateTenantQuotaRefusesBeforeDecodingTheBody` が `status = 400, want 403`)。副作用は防がれたままなので、状態の観測だけでは検出できない変異である。
  - 等価変異 M5 検査を `RequireControlPlaneUser` の直後へ移す: 検出されなかった。このハンドラーの認可判定は対象テナントを引かないため、認可の前後で呼び出し元が観測できる情報が変わらない。唯一の差は未認証かつ CSRF なしの要求が受け取る拒否の種類だが、この経路の拒否の写像は現在 `WriteAdminAccessError` を通っておらず [[wi-471-align-the-tenant-quota-update-route-with-the-admin-api-conventions]] の主題であるため、ここでテストを固定すると同項目の変更と衝突する。
  - 手法の限界: 変異はこの 1 つの呼び出しの周囲に閉じており、`VerifyBrowserRequest` 自身の比較論理は変異させていない。そちらは `backend/shared/http/support_http/csrf_test.go` が既に所有する。
- **Verification Results**:
  - `mise run test-go-race` - passed
  - `mise run lint-go` - passed (0 issues)
  - `mise run check-security-controls` - passed (183 declared refusal(s), 17 promised by a 403 on a state change, 102 awaiting a test)
  - `mise run report-security-test-gaps` - read; 拒否の効果を読み戻して確かめていない箇所は依然 91 件あり、`backend/tenancy/handlers_http` は 3 件を占める。この作業項目が加えた 3 つのテストはいずれも読み戻しを含む。
  - `mise run check-spec` - passed
  - `mise run check-work-items` - passed
  - `mise run verify` - passed
