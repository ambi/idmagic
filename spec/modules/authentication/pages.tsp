namespace IdMagicUi;

const signInContext = #{area: Area.authentication, actor: "認証を受ける利用者", entry: "認可フロー、メールのリンクまたは直接アクセス", context: "対象テナントの認可トランザクションとセッションの Cookie。CSRF は表示 API から取得する。", references: #["[共通のフォーム規則](user-interface.md#フォームと検証)"]};
const returnTo = #{label: "認証後の戻り先", control: "URL パラメーター", initial: "URL の return_to", condition: "指定された場合", mapping: "同じテナントの許可された戻り先"};

/** Authentication が主担当となる認証のページ。 */
model AuthenticationPages {
  // 経路は kind を認可トランザクションから決めるため、route を明示する。
  @page(#{...signInContext, id: "login", name: "ログイン", route: "/login", purpose: "資格情報または既存の認証手段で認証を開始する", content: #["ユーザー名とパスワード", "Windows 統合認証の入口", "外部 IdP の入口", "パスワード再設定の入口"]})
  @category(Category.signIn)
  login: PageData<{
    @form(#{name: "資格情報", commit: "ログイン"}) credentials: {
      @field(#{label: "ユーザー名", control: "text", condition: "資格情報を使う場合"}) @bind(IdMagic.Authentication.Operations.SubmitBrowserLoginHttpRequest.username) username: unknown;
      @field(#{label: "パスワード", control: "password", condition: "資格情報を使う場合"}) @bind(IdMagic.Authentication.Operations.SubmitBrowserLoginHttpRequest.password) password: unknown;
    };
  }, {
    @transition(#{event: "ログイン", condition: "第二要素が必要", result: "第二要素の選択と入力を表示する", data: "認可トランザクション、return_to"}, AuthenticationPages.totp) @uses(IdMagic.Authentication.Operations.SubmitBrowserLogin) signIn: unknown;
    @transition(#{event: "ログイン", condition: "認証が成立し追加の対話が不要", result: "認可の結果を返す", toState: "external", data: "認可コードと state"}, Destinations.client) @uses(IdMagic.Authentication.Operations.SubmitBrowserLogin) complete: unknown;
    @transition(#{event: "外部 IdP を選択", result: "選択した IdP の認証へ進む", toState: "external", data: "IdP の識別子、return_to"}, Destinations.idp) federate: unknown;
    @transition(#{event: "パスワードを忘れた"}, AuthenticationPages.forgotPassword) reset: unknown;
    @transition(#{event: "ログイン", condition: "認証の失敗", result: "失敗を表示して入力を保持する", toState: "error", mode: "data"}, AuthenticationPages.login) failure: unknown;
  }, {
    @field(returnTo) returnTo: string;
    @field(#{label: "Windows 統合認証の抑止", control: "URL パラメーター", initial: "URL の negotiate", condition: "Windows 統合認証を提供する場合", mapping: "skip なら統合認証を自動実行しない"}) negotiate: string;
  }>;

  @page(#{...signInContext, id: "totp", name: "二要素認証", route: "/totp", purpose: "利用可能な認証手段で第二要素を確認する", content: #["TOTP、パスキーと回復コードの切り替え", "認証コードの入力", "デバイスを記憶する選択"]})
  @category(Category.signIn)
  totp: PageData<{
    @form(#{name: "第二要素の確認", commit: "認証"}) factor: {
      @field(#{label: "認証手段", control: "選択", initial: "利用可能な最初の手段", condition: "利用可能な手段が複数ある場合", mapping: "TOTP、WebAuthn、回復コードの UI の切り替え"}) method: string;
      @field(#{label: "TOTP のコード", control: "text", condition: "TOTP を選択した場合"}) @bind(IdMagic.Contract.BrowserTotpRequest.code) code: unknown;
      @field(#{label: "回復コード", control: "text", condition: "回復コードを選択した場合"}) @bind(IdMagic.Contract.BrowserRecoveryCodeRequest.code) recovery: unknown;
      @field(#{label: "デバイスを記憶する", control: "checkbox", initial: "false", condition: "ポリシーが許し TOTP またはパスキーを選択した場合"}) @bind(IdMagic.Contract.BrowserTotpRequest.remember_device) rememberDevice: unknown;
    };
  }, {
    @transition(#{event: "認証", condition: "第二要素が成立し認可が完了", result: "認可の結果を返す", toState: "external", data: "認可トランザクション"}, Destinations.client) @uses(IdMagic.Authentication.Operations.SubmitBrowserTotp, IdMagic.Authentication.Operations.SubmitBrowserRecoveryCode, IdMagic.Authentication.Operations.SubmitBrowserWebAuthn) verify: unknown;
    @transition(#{event: "認証", condition: "コードまたはパスキーの確認に失敗", result: "選択した手段と問題を表示する", toState: "error", mode: "data"}, AuthenticationPages.totp) failure: unknown;
  }, {
    @field(returnTo) returnTo: string;
  }>;

  @page(#{...signInContext, id: "forgot-password", name: "パスワード再設定の依頼", purpose: "登録したメールアドレスへ再設定の案内を依頼する", content: #["メールアドレス", "送信後の案内", "ログインへの戻り先"]})
  @category(Category.recovery)
  forgotPassword: PageData<{
    @form(#{name: "再設定の依頼", commit: "案内を送信"}) request: {
      @field(#{label: "メールアドレス", control: "email"}) @bind(IdMagic.Contract.BrowserForgotPasswordRequest.email) email: unknown;
    };
  }, {
    @transition(#{event: "案内を送信", condition: "送信要求を受理", result: "対象の存在を明かさず案内を表示する", toState: "external", data: "再設定リンク"}, Destinations.email) @uses(IdMagic.Authentication.Operations.RequestPasswordReset) send: unknown;
    @transition(#{event: "ログインへ戻る"}, AuthenticationPages.login) back: unknown;
    @transition(#{event: "案内を送信", condition: "要求を受理できない", result: "入力を残して問題を表示する", toState: "error", mode: "data"}, AuthenticationPages.forgotPassword) failure: unknown;
  }>;

  @page(#{...signInContext, id: "reset-password", name: "パスワードの再設定", purpose: "メールのリンクから新しいパスワードを設定する", content: #["新しいパスワード", "リンクの有効性", "変更の完了とログインへの入口"]})
  @category(Category.recovery)
  resetPassword: PageData<{
    @form(#{name: "新しいパスワード", commit: "変更"}) reset: {
      @field(#{label: "新しいパスワード", control: "password", condition: "有効な再設定トークンがある場合"}) @bind(IdMagic.Contract.BrowserResetPasswordRequest.new_password) newPassword: unknown;
    };
  }, {
    @transition(#{event: "変更", condition: "変更に成功", result: "変更完了とログインの入口を表示する", toState: "success", data: "token、new_password", mode: "data"}, AuthenticationPages.resetPassword) @uses(IdMagic.Authentication.Operations.ResetPasswordWithToken) reset: unknown;
    @transition(#{event: "ログインへ戻る", from: "success", condition: "パスワードの変更後", result: "ログインの入口を表示する"}, AuthenticationPages.login) backAfterReset: unknown;
    @transition(#{event: "ログインへ戻る"}, AuthenticationPages.login) back: unknown;
    @transition(#{event: "変更", condition: "期限切れ、使用済みまたは変更の拒否", result: "問題を表示し、必要なら再設定の依頼へ案内する", toState: "error", mode: "data"}, AuthenticationPages.resetPassword) failure: unknown;
  }, {
    @field(#{label: "再設定トークン", control: "URL パラメーター", initial: "URL の token", condition: "メールのリンクから開く場合"}) @bind(IdMagic.Contract.BrowserResetPasswordRequest.token) token: unknown;
  }>;
}

@flow(#{id: "recover-password", name: "メールからパスワードを再設定する", goal: "再設定の案内を受け取り、新しいパスワードを設定してログインへ戻る。", notes: #["通知メールは IdMagic の外で利用者が読む表示として区別する。", "期限切れや使用済みリンクの扱いは、パスワード再設定ページの操作表に示す。"]})
model RecoverPasswordFlow {
  @view(#{state: "ready", visible: #["登録したメールアドレスの入力"]}, AuthenticationPages.forgotPassword)
  request: unknown;
  @view(#{state: "external", visible: #["一回限りの再設定リンク"]}, Destinations.email)
  email: unknown;
  @view(#{state: "ready", visible: #["新しいパスワードの入力"]}, AuthenticationPages.resetPassword)
  reset: unknown;
  @view(#{name: "パスワードの変更完了", state: "success", visible: #["変更完了の案内", "ログインへの入口"]}, AuthenticationPages.resetPassword)
  completed: unknown;
  @view(#{state: "ready", visible: #["ユーザー名とパスワードの入力"]}, AuthenticationPages.login)
  login: unknown;
  @flowLink(#{trigger: "再設定の案内を送信", action: "send"}, RecoverPasswordFlow.request, RecoverPasswordFlow.email)
  send: unknown;
  @flowLink(#{trigger: "メールのリンクを開く"}, RecoverPasswordFlow.email, RecoverPasswordFlow.reset)
  openLink: unknown;
  @flowLink(#{trigger: "新しいパスワードを保存", action: "reset"}, RecoverPasswordFlow.reset, RecoverPasswordFlow.completed)
  save: unknown;
  @flowLink(#{trigger: "ログインへ戻る", action: "backAfterReset"}, RecoverPasswordFlow.completed, RecoverPasswordFlow.login)
  returnToLogin: unknown;
}

model AccountAuthenticationPages {
  @page(#{...accountContext, id: "change-password", name: "パスワード変更", entry: "本人のセキュリティ設定、認可フローの必須アクション", purpose: "現在のパスワードを確認して新しいパスワードへ変更する", content: #["現在と新しいパスワード", "パスワードの表示切り替え", "ポリシー違反と変更結果"]})
  @category(Category.security)
  password: PageData<{
    @form(#{name: "パスワード変更", commit: "変更"}) change: {
      @field(#{...entered, label: "現在のパスワード", control: "password"}) @bind(IdMagic.Contract.BrowserChangePasswordRequest.current_password) currentPassword: unknown;
      @field(#{...entered, label: "新しいパスワード", control: "password"}) @bind(IdMagic.Contract.BrowserChangePasswordRequest.new_password) newPassword: unknown;
    };
  }, {
    @transition(#{...update, event: "変更", condition: "更新が成功", result: "必須アクションなら認可フローを継続し、それ以外は完了を表示する"}, AccountAuthenticationPages.password) @uses(IdMagic.Authentication.Operations.ChangePassword) save: unknown;
    @transition(#{...update, event: "変更", condition: "現在のパスワードが違う、ポリシー違反または更新失敗", result: "変更せず理由を表示する", toState: "error"}, AccountAuthenticationPages.password) failure: unknown;
  }, {
    @field(#{label: "認証後の戻り先", control: "URL", initial: "return_to", condition: "認可フローから来た場合", mapping: "同じテナントの許可された戻り先"}) returnTo: string;
  }>;

  @page(#{...accountContext, id: "account-security", name: "本人のセキュリティ設定", entry: "アカウントポータル", purpose: "認証手段、回復コード、記憶済みデバイスと通知を管理する", content: #["TOTP の登録状態と登録用 QR", "パスキー一覧", "回復コードの残数と発行直後のコード", "記憶済みデバイス", "通知設定", "リンク済み外部 ID"]})
  @category(Category.security)
  security: PageData<{
    @form(#{name: "TOTP 登録", condition: "未登録で登録を開始した場合", commit: "登録を完了"}) enrollTotp: {
      @field(#{label: "設定キー", control: "読み取り専用 text", initial: "StartTotpEnrollment の secret", condition: "登録開始後。QR を読み取れない場合に展開する", mapping: "確認要求の secret。利用者は変更しない"}) secret: string;
      @field(#{...entered, label: "確認コード", control: "text", mapping: "確認要求の code"}) code: string;
    };
    @form(#{name: "TOTP 解除", condition: "登録済み", commit: "解除"}) removeTotp: {
      @field(#{...entered, label: "解除用コード", control: "text", mapping: "解除要求の code"}) code: string;
    };
    @form(#{name: "パスキー登録", condition: "WebAuthn を利用できるブラウザー", commit: "登録"}) passkey: {
      @field(#{...entered, label: "パスキー名", control: "text", mapping: "登録要求の label。資格情報はブラウザーの認証器から取得する"}) label: string;
    };
    @form(#{name: "通知設定", condition: "通知の種別が取得できた場合", commit: "切り替え時に更新"}) notifications: {
      @field(#{label: "通知を受け取る", control: "checkbox", initial: "カテゴリごとの enabled", condition: "mandatory の種別は編集不可", mapping: "無効な任意カテゴリを disabled_categories として送信する", repeated: true}) enabled: boolean;
    };
  }, {
    @transition(#{...update, event: "TOTP の登録を開始", condition: "未登録", result: "設定キーと QR を表示する"}, AccountAuthenticationPages.security) @uses(IdMagic.Authentication.Operations.StartTotpEnrollment) startTotp: unknown;
    @transition(#{...update, event: "TOTP の登録を完了", condition: "コードの確認成功", result: "登録済みの表示へ切り替える"}, AccountAuthenticationPages.security) @uses(IdMagic.Authentication.Operations.ConfirmTotpEnrollment) confirmTotp: unknown;
    @transition(#{...update, event: "登録をキャンセル", condition: "登録中", result: "未確定の設定キーを破棄する"}, AccountAuthenticationPages.security) cancelTotp: unknown;
    @transition(#{...update, event: "TOTP を解除", condition: "再認証と解除が成功", result: "登録状態を更新する"}, AccountAuthenticationPages.security) @uses(IdMagic.Authentication.Operations.RemoveTotpFactor) removeTotp: unknown;
    @transition(#{...update, event: "パスキー登録", condition: "ブラウザーの認証器で登録が成功", result: "再読込して一覧を更新する"}, AccountAuthenticationPages.security) @uses(IdMagic.Authentication.Operations.StartWebAuthnRegistration, IdMagic.Authentication.Operations.FinishWebAuthnRegistration) registerPasskey: unknown;
    @transition(#{...update, event: "パスキー削除", condition: "再認証と削除が成功", result: "対象の資格情報を一覧から除く"}, AccountAuthenticationPages.security) @uses(IdMagic.Authentication.Operations.RemoveWebAuthnCredential) removePasskey: unknown;
    @transition(#{...update, event: "回復コードを発行・再発行", condition: "再認証が成功", result: "新しいコードを一度だけ表示し、旧コードを置き換える"}, AccountAuthenticationPages.security) @uses(IdMagic.Authentication.Operations.GenerateRecoveryCodes) recovery: unknown;
    @transition(#{...update, event: "回復コードをすべて失効", condition: "再認証が成功", result: "表示中のコードを消し、残数を更新する"}, AccountAuthenticationPages.security) @uses(IdMagic.Authentication.Operations.RevokeRecoveryCodes) revokeRecovery: unknown;
    @transition(#{...update, event: "デバイスの記憶を解除・すべて解除", condition: "再認証が成功", result: "対象を記憶済みデバイス一覧から除く"}, AccountAuthenticationPages.security) @uses(IdMagic.Authentication.Operations.RevokeMyTrustedDevice, IdMagic.Authentication.Operations.RevokeMyTrustedDevices) devices: unknown;
    @transition(#{...update, event: "通知設定を切り替える", condition: "任意の種別で再認証が成功", result: "サーバーが返した通知設定を表示する"}, AccountAuthenticationPages.security) @uses(IdMagic.Authentication.Operations.UpdateMyNotificationPreferences) notifications: unknown;
    @transition(#{...update, event: "操作", condition: "キャンセル、認証器の拒否、再認証の中止または API の拒否", result: "更新を確定せず、失敗理由を表示する", toState: "error"}, AccountAuthenticationPages.security) failure: unknown;
  }>;

  @page(#{...accountContext, id: "account-activity", name: "ログイン履歴とセッション", entry: "アカウントポータル", purpose: "本人のログイン履歴を確認し不要なセッションを失効させる", content: #["現在と他のセッション、期限、端末情報", "ログインの日時、認証手段、結果"]})
  @category(Category.security)
  activity: PageData<{}, {
    @transition(#{...update, event: "対象のセッションを失効", condition: "再認証と失効が成功", result: "対象を一覧から除く。現在のセッションなら再ログインへ進む"}, AccountAuthenticationPages.activity) @uses(IdMagic.Authentication.Operations.RevokeMySession) revoke: unknown;
    @transition(#{...update, event: "他のセッションをすべて失効", condition: "再認証が成功", result: "現在以外のセッションを一覧から除く"}, AccountAuthenticationPages.activity) @uses(IdMagic.Authentication.Operations.RevokeMyOtherSessions) revokeOthers: unknown;
  }>;

  // 経路は kind を認可トランザクションから決めるため、route を明示する。
  @page(#{...browserContext, id: "mfa-enrollment", name: "認可フロー内の MFA 登録", route: "/mfa-enrollment", entry: "MFA 登録を要求する認可フロー", purpose: "ログインを継続する前に TOTP を登録する", content: #["登録用 QR と設定キー", "確認コード", "登録結果"]})
  @category(Category.signIn)
  enrollment: PageData<{
    @form(#{name: "TOTP 登録の確認", condition: "登録開始 API の成功後", commit: "登録を完了"}) confirm: {
      @field(#{...entered, label: "確認コード", control: "text"}) @bind(IdMagic.Contract.BrowserMfaEnrollmentConfirmRequest.code) code: unknown;
    };
  }, {
    @transition(#{...update, event: "登録を開始", condition: "開始要求が成功", result: "QR と設定キーを表示する"}, AccountAuthenticationPages.enrollment) @uses(IdMagic.Authentication.Operations.StartBrowserMfaEnrollment) start: unknown;
    @transition(#{...update, event: "登録を完了", condition: "コードの確認成功", result: "認可トランザクションの次の表示へ進む", toState: "external"}, Destinations.client) @uses(IdMagic.Authentication.Operations.ConfirmBrowserMfaEnrollment) confirm: unknown;
    @transition(#{...update, event: "確認", condition: "コードや登録要求の拒否", result: "登録せずエラーを表示する", toState: "error"}, AccountAuthenticationPages.enrollment) failure: unknown;
  }>;
}
