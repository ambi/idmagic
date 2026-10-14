---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# 全アプリケーションに適用する既定ログインポリシーを導入する

テナントデフォルトサインインポリシーを導入した (ADR-081)。デフォルトとアプリ個別の関係はレビューを経て**上書きモデル**に決定: アプリが独自の有効ルールを持てばデフォルトを完全に置換して評価し、持たなければデフォルトを適用する (合成 / floor は不採用、`exempt_from_tenant_default` フラグは廃止)。デフォルトより弱い上書き (認証強度・再認証を求めるまでの時間・許可ネットワークのいずれかを緩める) は許可するが `weaker_than_default` を返して UI で警告する。評価は OIDC / SAML / WS-Fed 共通の `EvaluateApplicationAccess` 経路で行い評価器は fail-closed。永続化 (`tenant_default_sign_in_policies` テーブル / memory リポジトリ)、取得・更新 API (`GET`/`PUT /api/admin/default-sign-in-policy`)、監査イベント `TenantDefaultSignInPolicyUpdated`、権限 `AdminTenantDefaultSignInPolicyManage` を追加。UI はサインインポリシーを総合的に扱う `/admin/sign-in-policy` 画面 (デフォルトを詳細表示→編集モードで編集 + アプリ別の上書き有無/弱い警告/実効ポリシー一覧) とし、アプリ編集画面では「テナントデフォルト」「このアプリの上書き」「最終的に適用されるポリシー」を区別表示して弱い上書きに警告する。用語は「既定」→「デフォルト」、「再認証最大経過秒数」→「再認証を求めるまでの時間」に統一し、内部ルール名を UI に露出しないようにした。既存テナントは空デフォルト (allow-all) で移行し大規模ロックアウトを回避、ロールバックは行クリア/削除で行う。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
