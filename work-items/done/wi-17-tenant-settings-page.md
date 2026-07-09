---
status: completed
authors: ["tn"]
risk: low
created_at: 2026-06-16
---

# 管理画面に「設定」ページを追加してテナント単位の設定を集約する

## Motivation
現状、管理画面に "設定" の入口が無い。テナント単位で admin が触れる
設定項目は以下のように散らばっている:

- **テナント表示名 (display_name)**: `/admin/tenants` の中で
  system_admin かつ default tenant の場合のみ編集可。所属テナント
  自体の表示名を変えるのに別経路が必要。
- **パスワードポリシー上書き (password_policy_override)**: 同じく
  `/admin/tenants` 経由でしか触れない。テナント内 admin (system_admin
  でない普通の admin) は触る経路を持たない。
- **メール送信設定 / 監査保持期間 / セッションタイムアウト**: 環境
  変数や SCL 経由でのみ調整でき、UI から見えない。

admin が "うちのテナントの設定を見たい・直したい" と思ったとき、
まず開く想定の場所が無いことが UX 上の摩擦になっている。
Okta は "Account" / "Customization" / "Security" の sidebar セクション、
Google Admin は "Account settings"、Microsoft 365 admin は "Settings"
にそれぞれ集約している。

本 WI は専用の `/admin/settings` ページを新設し、テナント内 admin が
触れる設定 (まずは表示名 + パスワードポリシー override) を集約する。
将来 wi-15 の RBAC inspection、メール送信設定、監査保持期間と
同列の設定タブが増える前提の枠を作る。

## Scope
- **decision**:
  - 新規 ADR は不要。既存 ADR-032 (tenant aggregate) と ADR-026 (password policy) の範囲で、UI を 1 ページ足すだけ。
- **scl**:
  - 変更なし。Tenant model / password_policy_override / DisplayName の wire 形式はそのまま流用する。
- **go**:
  - http:
    - 既存 `/admin/tenants` (system_admin scope) が tenant CRUD を 提供している。本 WI では新規 endpoint を作らず、テナント内 admin が自身の所属テナントを **GET + PATCH** できる経路を どう開くかを設計する。選択肢:
        A. 既存 `/admin/tenants/{tenant_id}` の RBAC を緩める
           (admin が自テナントのみ触れる)。system_admin との二重
           権限になり ADR-032 の境界が緩む。
        B. 新規 `/api/admin/settings` を追加し、内部で
           `actor.tenant_id` 固定の GetTenant / UpdateTenant を呼ぶ。
           tenant の境界は守られ、UI 側からは単純な GET / PATCH。
      本 WI は **選択肢 B** を採用する。
    - 新規 endpoint:
        GET   `/api/admin/settings`  → 自テナントの display_name と
                                       password_policy_override を返す。
        PATCH `/api/admin/settings`  → display_name (任意) と
                                       password_policy_override (任意) を
                                       pointer-optional で受ける。
      既存 `requireAdmin` + CSRF + tenant scope を継承する。 `UpdateTenant` use case を内部で呼ぶ (新規 use case は作らない)。
  - usecases:
    - 既存 `tenancyusecases.UpdateTenant` を流用する。actor が system_admin かどうかではなく `actor.tenant_id == target.id` を最低要件として呼ぶための薄い wrapper を bridge する。
- **ui**:
  - pages:
    - 新規 AdminSettingsPage at `/admin/settings`。タブ風の縦並び:
        - 一般 (General): tenant.display_name の編集。
        - パスワードポリシー (Password policy): min_length / max_length /
          history_depth を pointer-optional で編集。空欄なら global
          default を継承することをラベルで示す。
        - 後続のタブ枠を 1 つ "メール送信" (disabled) として置いて
          おき、将来の wi で実装する旨を示す。
    - 保存は per-tab。各 tab に独立した form と CTA "保存"。
    - system_admin かつ default tenant の場合は、ページ上部に "テナント一覧から他テナントの設定を編集する場合は `/admin/tenants` を使ってください" の補足リンクを出す。
  - navigation:
    - `adminNavItems` の末尾 (テナント nav 項目より下、もしくは sidebar の "Settings" セクションを新設するなら最下段) に "設定" を追加する。AdminNavKey に `settings` を追加。
  - routing:
    - `/admin/settings` を router.tsx に追加し、loadPageData で `kind: 'admin-settings'` を返す。
    - loadPageData は `/api/admin/settings` を取得して初期値を埋める。
- **documentation**:
  - idmagic/README.md "範囲" 節に "テナント内 admin 向けの設定 ページ" を 1 行追加する。詳細は ADR に書かない (既存 ADR で 十分)。

## Out of Scope
- 個人 (admin 自身) 用の設定 (display name / MFA 切り替え / notification preferences)。これは end user 側 `/account/*` の 拡張で扱う別 WI に分離する。
- メール送信設定 (SMTP host / from / DKIM 等) の UI。ADR-035 で 決定済みの環境変数経由を維持し、UI からの変更は別 WI。
- 監査保持期間 / セッションタイムアウトの UI。
- tenant の作成・無効化・削除。`/admin/tenants` の system_admin 経路を 継続して使う。
- tenant 横断 / 統合 admin ダッシュボード。
- branding (logo / theme color / custom domain)。Phase 5+ のスコープ。
- i18n の追加。

## Verification
- `go test ./...` (in: idmagic)
  - reason: 新規 handler のテスト (admin が自テナント設定を GET / PATCH できる、cross-tenant への書き込みは tenancy 境界で失敗、 system_admin と admin の両方で同じレスポンスを返す)。
- `go vet ./...` (in: idmagic)
- `go build ./...` (in: idmagic)
- `bun --cwd idmagic/ui typecheck`
- `bun --cwd idmagic/ui lint`
- `bun --cwd idmagic/ui build`
- 手動 1: admin (default tenant) で `/admin/settings` を開く → display_name と password_policy_override が現在値で表示される。
- 手動 2: password_policy_override.min_length を変更して保存 → 再ロードで反映されている。
- 手動 3: 同 tenant 内の別 user 作成時に新しい min_length が 適用される。
- 手動 4: system_admin かつ default tenant 以外のテナントで `/admin/settings` を触っても自テナントだけが対象。

## Risk Notes
既存 `UpdateTenant` use case を流用するため、新規 business logic は
ほぼ無い。最大のリスクは "admin に password_policy_override を
解放することで policy を弱める方向に変えられる" 点。本 WI では
下限 (min_length の最低値・max_length の上限値) を SCL の global
default で gating し、override が global より弱い場合は use case
側で reject する制約を追加する。

別途、tenant 内 admin が誤って強すぎる policy (例:
history_depth=100) を入れると、その tenant 内のパスワード変更が
事実上できなくなる。これは UI 側の上限制約 (フォームの max 値) で
防御する。

## Completion
- **Completed At**: 2026-06-17
- **Summary**:
  テナント内 admin と system_admin が自テナントの表示名・パスワードポリシー上書きを
  閲覧・更新できる `/api/admin/settings` (GET / PATCH) と `/admin/settings` ページを
  追加した。対象は actor.tenant_id に固定し cross-tenant 操作は構造的に発生しない。
  password_policy_override が objectives.PasswordPolicy の標準値より弱い値を含む
  場合は use case 側で reject する。既存 `/admin/tenants` (system_admin) の挙動は
  不変だが、password_policy_override が UpdateTenant に到達しなかった既存バグも
  併せて修正した。
  続いて UX 整理のフォローアップで、`AdminSettingsResponse` に
  `password_policy_defaults` を埋め込み、UI に「標準 最小長 12 / 最大長 128 /
  履歴件数 5」のサマリと具体数字の hint・placeholder を表示するようにした。
  「global default」というユーザ向け文言を「標準値」に統一し、weaker policy の
  エラー文言にも具体数値 (`min_length≥12 / max_length≤128 / history_depth≥5`) を
  含めた。
- **Verification Results**:
  - `GOCACHE=/tmp/idmagic-cache go test -race ./...` (in: idmagic)
    - result: ok
  - `GOCACHE=/tmp/idmagic-cache go vet ./...` (in: idmagic)
    - result: ok
  - `GOCACHE=/tmp/idmagic-cache go build ./...` (in: idmagic)
    - result: ok
  - `bun --cwd idmagic/ui typecheck`
    - result: ok
  - `bun --cwd idmagic/ui lint`
    - result: ok (43 files)
  - `bun --cwd idmagic/ui build`
    - result: ok (6350 modules)
  - 実ブラウザでの 4 手動テストケース (admin 編集 / min_length 反映確認 / cross-tenant ガード / system_admin alert) と、パスワードポリシータブで標準値カードが具体数値で表示されることの目視確認は未実施。
- **Affected Guarantees State**:
  - tenant isolation: GET / PATCH /api/admin/settings は actor.tenant_id に固定。cross-tenant 書き込みは存在しない。
  - admin RBAC: AdminSettingsRead / AdminSettingsUpdate は admin / system_admin に限定。requireTenantAdmin が actor.tenant_id == request tenant を強制する。
  - CSRF: PATCH は verifyBrowserRequest を継承し double-submit を要求する。
  - SCL coherence: 新規 interface 2 件 / permission 2 件 / value_object 2 件 (AdminSettingsResponse, PasswordPolicyDefaults) を追加し、TenantUpdated event / TenantUpdateRequest model を流用。Tenancy component の owns_* に追加済み。SCLPermissionsCoverage テストで Go action マッピングを検証。
  - policy floor: password_policy_override は objectives.PasswordPolicy の標準値より弱い場合 ErrPolicyOverrideWeaker / HTTP 400 policy_override_weaker で reject。閾値は AdminSettingsResponse.password_policy_defaults として UI にも返却される。
  - UX: UI / エラー文言ともに「標準値」表記に統一。「global default」のような SCL 用語をユーザに露出しない。
