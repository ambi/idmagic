---
id: idp-wi-115-tenant-default-application-login-policy
title: "全アプリケーションに適用する既定ログインポリシーを導入する"
created_at: 2026-07-04
authors: [tn]
status: completed
risk: high
---

# Motivation
アプリケーションごとのログインポリシーだけでは、テナント全体の最低認証要件を一箇所で保証できない。
管理者は「全アプリで MFA を既定にし、一部の低リスクアプリだけ緩和する」「全アプリで一定時間を超えた再認証を要求する」のような横断的な方針を持つ。
既定ポリシーがないと、アプリ作成時の設定漏れがそのまま弱いログイン要件になり、運用上の安全性と説明可能性が下がる。

# Scope
- `spec/contexts/application.yaml`
  - テナント既定の Application ログインポリシーを表す model / interface / event を追加する。
  - 既定ポリシーとアプリ個別ポリシーの合成規則を定義する。
  - 合成後の評価が OIDC / SAML / WS-Fed の federation 開始経路で一貫して fail-closed になる invariant を追加または更新する。
- ADR
  - 既定ポリシーとアプリ個別ポリシーの優先順位を決める。例: 既定に個別を追加で重ねる、個別が明示的に上書きする、緩和には明示的な例外設定を要求する、など。
- Go / HTTP
  - テナント既定ポリシーの永続化、取得、更新、監査イベントを実装する。
  - アプリ個別ポリシー評価前に既定ポリシーを合成し、設定漏れでも最低要件が適用されるようにする。
- UI
  - 管理者設定またはアプリケーション設定配下に、全アプリ向け既定ログインポリシーの編集画面を追加する。
  - アプリ詳細では「テナント既定」「このアプリの上書き」「最終的に適用されるポリシー」を区別して表示する。

# Out of Scope
- ログインポリシー語彙と条件モデルの全面再設計。
- アプリごとの申請/承認ワークフロー。
- 動的リスクスコアや外部デバイス管理連携。

# Verification
- `just yaml-check-scl`
- `just verify-go`
- `just verify-ui`
- 手動: 既定ポリシーだけを設定した状態で、新規アプリと既存アプリの federation 開始に同じ最低要件が適用されること。
- 手動: アプリ個別ポリシーを設定した場合、UI に合成結果が表示され、評価器も同じ合成結果で判定すること。
- 手動: 既定ポリシーを削除または緩和した場合、監査イベントと画面表示が変更内容を追跡できること。

# Risk Notes
既定ポリシーは全アプリケーションのログイン可否に影響するため、誤った合成規則や移行で大規模なログイン不能または認証強度低下を起こす可能性がある。
実装前に ADR で優先順位、例外、既存テナントの初期値、ロールバック方針を固定する。

# Completion
- **Completed At**: 2026-07-04
- **Summary**:
  テナントデフォルトサインインポリシーを導入した (ADR-081)。デフォルトとアプリ個別の関係はレビューを経て**上書きモデル**に決定: アプリが独自の有効ルールを持てばデフォルトを完全に置換して評価し、持たなければデフォルトを適用する (合成 / floor は不採用、`exempt_from_tenant_default` フラグは廃止)。デフォルトより弱い上書き (認証強度・再認証を求めるまでの時間・許可ネットワークのいずれかを緩める) は許可するが `weaker_than_default` を返して UI で警告する。評価は OIDC / SAML / WS-Fed 共通の `EvaluateApplicationAccess` 経路で行い評価器は fail-closed。永続化 (`tenant_default_sign_in_policies` テーブル / memory リポジトリ)、取得・更新 API (`GET`/`PUT /api/admin/default-sign-in-policy`)、監査イベント `TenantDefaultSignInPolicyUpdated`、権限 `AdminTenantDefaultSignInPolicyManage` を追加。UI はサインインポリシーを総合的に扱う `/admin/sign-in-policy` 画面 (デフォルトを詳細表示→編集モードで編集 + アプリ別の上書き有無/弱い警告/実効ポリシー一覧) とし、アプリ編集画面では「テナントデフォルト」「このアプリの上書き」「最終的に適用されるポリシー」を区別表示して弱い上書きに警告する。用語は「既定」→「デフォルト」、「再認証最大経過秒数」→「再認証を求めるまでの時間」に統一し、内部ルール名を UI に露出しないようにした。既存テナントは空デフォルト (allow-all) で移行し大規模ロックアウトを回避、ロールバックは行クリア/削除で行う。
- **Verification Results**:
  - `just yaml-check` - passed
  - `just verify-go` - passed
  - `just verify-ui` - passed
