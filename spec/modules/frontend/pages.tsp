namespace IdMagicUi;

model FrontendPages {
  @page(#{...browserContext, id: "home", name: "サービスの入口", entry: "直接アクセス", purpose: "アカウントと管理コンソールの入口を示す", content: #["各ポータルへのリンク", "デモ有効時のデモログイン案内"]}) @category(Category.navigation)
  home: PageData<{
    @transition(#{...navigation, event: "アカウントを開く", result: "本人のポータルへ進む"}, FrontendPages.account) account: unknown;
    @transition(#{...navigation, event: "管理コンソールを開く", result: "テナントのダッシュボードへ進む"}, FrontendPages.admin) admin: unknown;
  }>;
  @page(#{...accountContext, id: "account-home", name: "アカウントの概要", entry: "サービスの入口、ポータルのホーム", purpose: "本人の登録状態と各セルフサービスの入口を示す", content: #["必須アクション、MFA、メールの状態", "プロフィール、セキュリティ、アプリケーション、アクティビティの入口"]}) @category(Category.navigation)
  account: PageData<{
    @transition(#{...navigation, event: "プロフィール", result: "登録情報を表示する"}, AccountProfilePages.profile) profile: unknown;
    @transition(#{...navigation, event: "セキュリティ", result: "認証手段と通知を表示する"}, AccountAuthenticationPages.security) security: unknown;
  }>;
  @page(#{...adminContext, id: "admin-dashboard", name: "管理ダッシュボード", entry: "サービスの入口、管理コンソールのホーム", purpose: "テナントの概要と管理操作への入口を示す", content: #["ユーザー、アプリケーション、同意の集計とセキュリティスコア", "推奨するセキュリティ設定への入口", "クォータの使用量", "管理メニュー"]}) @category(Category.navigation)
  admin: PageData<{
    @transition(#{...navigation, event: "管理メニューのユーザー", result: "ユーザー管理へ進む"}, AdminUserPages.users) users: unknown;
    @transition(#{...navigation, event: "サインインポリシーを設定", result: "MFA を求める規則を設定する"}, SignInPolicyPages.policy) signInPolicy: unknown;
    @transition(#{...navigation, event: "フェデレーションを設定", result: "Entra とのフェデレーションを設定する"}, EntraFederationPages.list) federation: unknown;
  }>;
  @page(#{...browserContext, id: "callback", name: "認証コールバック", entry: "OIDC の認可応答", purpose: "ポータルの認証応答を処理し、処理できない応答の結果を表示する", content: #["認可コードの交換中は遷移を継続する", "デモの成功、拒否、MFA 要件不足の表示", "ポータルと再試行の入口"]}) @category(Category.navigation)
  callback: PageData<{
    @transition(#{...navigation, event: "コールバックの処理", condition: "要求の state と交換が成立", result: "保持したポータルの戻り先へ進む"}, FrontendPages.account) resume: unknown;
    @transition(#{...navigation, event: "アカウントへ", result: "アカウントの認可を開始する"}, FrontendPages.account) account: unknown;
  }>;
  @page(#{...browserContext, id: "status", name: "認証と承認の結果", entry: "承認・拒否・ログアウトの結果", purpose: "操作が終わったことと次の入口を案内する", content: #["approved、denied、signed-out、authentication-required に応じた説明", "ログアウト後のアカウントと管理コンソールへの入口"]}) @category(Category.navigation)
  status: PageData<{
    @transition(#{...navigation, event: "アカウントへログイン", condition: "signed-out", result: "本人のポータルの認可を開始する"}, FrontendPages.account) account: unknown;
    @transition(#{...navigation, event: "管理コンソールへログイン", condition: "signed-out", result: "管理コンソールの認可を開始する"}, FrontendPages.admin) admin: unknown;
  }>;
}

/** 描画を持たない経路も、実装の経路集合と照合する。 */
model FrontendRouteNotes {
  @route(#{route: "/admin/users_/$id", kind: "layout", reason: "ユーザーの詳細と編集を束ねる Outlet。"}) user: unknown;
  @route(#{route: "/admin/workload-identity_/$trustBundleId", kind: "layout", reason: "信頼バンドルの詳細、編集と関連付けを束ねる Outlet。"}) workload: unknown;
  @route(#{route: "/system/tenants_/$realm", kind: "layout", reason: "テナントの詳細、基本情報、入口方式とクォータを束ねる Outlet。"}) tenant: unknown;
  @route(#{route: "/admin/settings_/saml-idp-profiles_/$profileId", kind: "layout", reason: "SAML IdP プロファイルの詳細と編集を束ねる Outlet。"}) samlProfile: unknown;
  @route(#{route: "/admin/identity-providers_/$id", kind: "layout", reason: "外部 IdP の詳細と編集を束ねる Outlet。"}) identityProvider: unknown;
  @route(#{route: "/admin/agents_/$agentId", kind: "layout", reason: "Agent の詳細、編集と個別の管理ページを束ねる Outlet。"}) agent: unknown;
  @route(#{route: "/admin/groups_/$groupId", kind: "layout", reason: "グループの詳細と編集を束ねる Outlet。"}) group: unknown;
  @route(#{route: "/admin/applications_/$applicationId", kind: "layout", reason: "アプリケーションの詳細、編集と同期管理を束ねる Outlet。"}) application: unknown;
  @route(#{route: "/admin/access-reviews_/$planId", kind: "layout", reason: "レビュー計画の詳細と編集を束ねる Outlet。"}) accessReview: unknown;
  @route(#{route: "__root__", kind: "layout", reason: "エラー境界と Outlet を提供するルートで、独立したページではない。"}) root: unknown;
  @route(#{route: "/account", kind: "layout", reason: "本人向けの子ページを収める Outlet。概要ページは /account/ が描画する。"}) account: unknown;
  @route(#{route: "/admin", kind: "layout", reason: "管理コンソールの子ページを収める Outlet。概要ページは /admin/ が描画する。"}) admin: unknown;
  @route(#{route: "/system", kind: "layout", reason: "システムコンソールの子ページを収める Outlet。"}) system: unknown;
  @route(#{route: "/system/", kind: "redirect", reason: "/system/tenants へ転送する入口。専用ダッシュボードはまだない。"}) systemIndex: unknown;
  @route(#{route: "/account/profile", kind: "layout", reason: "プロフィールの詳細と編集を束ねる Outlet。詳細は /account/profile_/ が描画する。"}) profile: unknown;
}
