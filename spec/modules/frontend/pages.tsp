namespace IdMagicUi;

model FrontendPages {
  @page(#{...browserContext, id: "home", name: "サービスの入口", entry: "直接アクセス", purpose: "アカウントと管理コンソールの入口を示す", content: #["各ポータルへのリンク", "デモ有効時のデモログイン案内"]}) @category(Category.navigation)
  home: PageData<{}, {
    @transition(#{...navigation, event: "アカウントを開く", result: "本人のポータルへ進む"}, FrontendPages.account) account: unknown;
    @transition(#{...navigation, event: "管理コンソールを開く", result: "テナントのダッシュボードへ進む"}, FrontendPages.admin) admin: unknown;
  }>;
  @page(#{...accountContext, id: "account-home", name: "アカウントの概要", entry: "サービスの入口、ポータルのホーム", purpose: "本人の登録状態と各セルフサービスの入口を示す", content: #["必須アクション、MFA、メールの状態", "プロフィール、セキュリティ、アプリケーション、アクティビティの入口"]}) @category(Category.navigation)
  account: PageData<{}, {
    @transition(#{...navigation, event: "プロフィール", result: "登録情報を表示する"}, AccountProfilePages.profile) profile: unknown;
    @transition(#{...navigation, event: "セキュリティ", result: "認証手段と通知を表示する"}, AccountAuthenticationPages.security) security: unknown;
  }>;
  @page(#{...adminContext, id: "admin-dashboard", name: "管理ダッシュボード", entry: "サービスの入口、管理コンソールのホーム", purpose: "テナントの概要と管理操作への入口を示す", content: #["ユーザー、グループ、アプリケーションの集計", "最近の活動と管理メニュー"]}) @category(Category.navigation)
  admin: PageData<{}, {
    @transition(#{...navigation, event: "ユーザー", result: "ユーザー管理へ進む"}, AdminUserPages.users) users: unknown;
  }>;
  @page(#{...browserContext, id: "callback", name: "認証コールバック", entry: "OIDC の認可応答", purpose: "ポータルの認証応答を処理し、処理できない応答の結果を表示する", content: #["認可コードの交換中は遷移を継続する", "デモの成功、拒否、MFA 要件不足の表示", "ポータルと再試行の入口"]}) @category(Category.navigation)
  callback: PageData<{}, {
    @transition(#{...navigation, event: "コールバックの処理", condition: "要求の state と交換が成立", result: "保持したポータルの戻り先へ進む"}, FrontendPages.account) resume: unknown;
    @transition(#{...navigation, event: "アカウントへ", result: "アカウントの認可を開始する"}, FrontendPages.account) account: unknown;
  }, {
    @field(#{label: "認可応答", control: "URL", initial: "code、state、error、error_description", condition: "プロトコルの応答として受領する", mapping: "OIDC クライアントの応答検証。利用者は編集しない"}) response: Record<string>;
  }>;
  @page(#{...browserContext, id: "status", name: "認証と承認の結果", entry: "承認・拒否・ログアウトの結果", purpose: "操作が終わったことと次の入口を案内する", content: #["approved、denied、signed-out、authentication-required に応じた説明", "ログアウト後のアカウントと管理コンソールへの入口"]}) @category(Category.navigation)
  status: PageData<{}, {
    @transition(#{...navigation, event: "アカウントへログイン", condition: "signed-out", result: "本人のポータルの認可を開始する"}, FrontendPages.account) account: unknown;
    @transition(#{...navigation, event: "管理コンソールへログイン", condition: "signed-out", result: "管理コンソールの認可を開始する"}, FrontendPages.admin) admin: unknown;
  }, {
    @field(#{label: "結果の種別", control: "URL", initial: "state。未知の値は authentication-required", mapping: "表示内容の選択だけに使う"}) state: string;
  }>;
}

/** 描画を持たない経路も、実装の経路集合と照合する。 */
model FrontendRouteNotes {
  @route(#{route: "__root__", kind: "layout", reason: "エラー境界と Outlet を提供するルートで、独立したページではない。"}) root: unknown;
  @route(#{route: "/account", kind: "layout", reason: "本人向けの子ページを収める Outlet。概要ページは /account/ が描画する。"}) account: unknown;
  @route(#{route: "/admin", kind: "layout", reason: "管理コンソールの子ページを収める Outlet。概要ページは /admin/ が描画する。"}) admin: unknown;
  @route(#{route: "/system", kind: "layout", reason: "システムコンソールの子ページを収める Outlet。"}) system: unknown;
  @route(#{route: "/system/", kind: "redirect", reason: "/system/tenants へ転送する入口。専用ダッシュボードはまだない。"}) systemIndex: unknown;
  @route(#{route: "/account/profile", kind: "layout", reason: "プロフィールの詳細と編集を束ねる Outlet。詳細は /account/profile_/ が描画する。"}) profile: unknown;
}
