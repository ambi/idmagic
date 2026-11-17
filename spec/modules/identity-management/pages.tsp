namespace IdMagicUi;

/** IdManagement が主担当となるユーザー管理のページ。 */
model AdminUserPages {
  @page(#{id: "admin-users", name: "ユーザー一覧", path: "/admin/users", route: "/admin/users", actor: "テナント管理者", entry: "管理コンソールのユーザー一覧または対象ユーザーの詳細", context: "対象テナントの管理者として認可を受ける。変更の許可は API が判断する。", purpose: "ユーザーを検索し、対象の確認と管理の入口を示す", content: #["ユーザー名", "表示名", "メールアドレス", "状態とロール", "総件数と再取得", "作成と CSV の入口"], references: #["REQ-FRONTEND-002", "[ナビゲーション](user-interface.md#ナビゲーション)"], area: Area.admin})
  @category(Category.users)
  users: {
    parameters: {
      @field(#{label: "検索語", control: "text", initial: "URL の query", condition: "常時", mapping: "一覧の query"})
      query: string;
      @field(#{label: "状態", control: "選択", initial: "URL の status または active", condition: "常時", mapping: "一覧の status"})
      status: string;
      @field(#{label: "次ページのカーソル", control: "URL パラメーター", initial: "URL の cursor", condition: "ページ送りの場合", mapping: "API の cursor"})
      cursor: string;
    };
    forms: {
      @form(#{name: "検索と絞り込み", condition: "常時", commit: "URL の条件を更新して再取得"})
      search: {
      @field(#{label: "検索語", control: "text", initial: "URL の query", condition: "常時", mapping: "一覧の query"})
      query: string;
      @field(#{label: "状態", control: "選択", initial: "URL の status または active", condition: "常時", mapping: "一覧の status"})
      status: string;
      };
    };
    states: ListStates;
    actions: {
      @transition(#{event: "検索または絞り込み", from: "ready", condition: "常時", result: "条件に合う一覧を表示する", toState: "ready", data: "query、status、cursor", mode: "data"}, AdminUserPages.users)
      @uses(IdMagic.IdManagement.Operations.ListAdminUsers)
      search: unknown;
      @transition(#{event: "詳細", from: "ready", condition: "常時", result: "移動先を表示する", toState: "ready", data: "ユーザー ID", mode: "navigation"}, AdminUserPages.detail)
      detail: unknown;
      @transition(#{event: "ユーザーを追加", from: "ready", condition: "常時", result: "移動先を表示する", toState: "ready", data: "なし", mode: "navigation"}, AdminUserPages.create)
      create: unknown;
      @transition(#{event: "CSV インポート", from: "ready", condition: "常時", result: "移動先を表示する", toState: "ready", data: "なし", mode: "navigation"}, AdminUserPages.importCsv)
      importCsv: unknown;
    };
  };

  @page(#{id: "admin-user-detail", name: "ユーザー詳細", path: "/admin/users/$id", route: "/admin/users_/$id/", actor: "テナント管理者", entry: "管理コンソールのユーザー一覧または対象ユーザーの詳細", context: "対象テナントの管理者として認可を受ける。変更の許可は API が判断する。", purpose: "現在のプロフィール、権限、所属と認証の状態を確認する", content: #["プロフィール", "ロールとグループ", "認証情報とセッション", "必須アクション", "無効化と削除の確認"], references: #["REQ-FRONTEND-002", "[ナビゲーション](user-interface.md#ナビゲーション)"], area: Area.admin})
  @category(Category.users)
  detail: {
    parameters: {
      @field(#{label: "ユーザー ID", control: "パスパラメーター", initial: "URL の id", condition: "常時", mapping: "取得と管理操作の対象 ID"})
      id: string;
    };
    forms: {
      @form(#{name: "ユーザーの削除確認", condition: "削除のダイアログを開いた場合", commit: "削除"})
      deleteConfirmation: {
      @field(#{label: "確認用ユーザー名", control: "text", initial: "空", condition: "削除のダイアログを開いた場合", mapping: "UI 内での確認。API へ送信しない"})
      confirmName: string;
      };
      @form(#{name: "認証情報のリセット", condition: "認証情報のリセットを選択した場合", commit: "選択した認証情報を解除"})
      authenticatorReset: {
      @field(#{label: "TOTP", control: "checkbox", initial: "false", condition: "リセットのダイアログを開いた場合", mapping: "解除する認証情報の種類"})
      totp: boolean;
      @field(#{label: "パスキー", control: "checkbox", initial: "false", condition: "リセットのダイアログを開いた場合", mapping: "解除する認証情報の種類"})
      webauthn: boolean;
      @field(#{label: "回復コード", control: "checkbox", initial: "false", condition: "リセットのダイアログを開いた場合", mapping: "解除する認証情報の種類"})
      recoveryCode: boolean;
      };
    };
    states: FormStates;
    actions: {
      @transition(#{event: "編集", from: "ready", condition: "常時", result: "移動先を表示する", toState: "ready", data: "ユーザー ID", mode: "navigation"}, AdminUserPages.edit)
      edit: unknown;
      @transition(#{event: "削除を確認", from: "ready", condition: "確認が成立し API が削除を受理", result: "一覧を再取得する", toState: "ready", data: "ユーザー ID", mode: "navigation"}, AdminUserPages.users)
      @uses(IdMagic.IdManagement.Operations.DeleteAdminUser)
      delete: unknown;
      @transition(#{event: "一覧へ戻る", from: "ready", condition: "常時", result: "移動先を表示する", toState: "ready", data: "なし", mode: "navigation"}, AdminUserPages.users)
      back: unknown;
      @transition(#{event: "管理操作", from: "ready", condition: "API が再認証を要求", result: "再認証後に元の操作を一度だけ再試行する", toState: "dialog", data: "元の操作と再認証の要求", mode: "navigation"}, Destinations.stepUp)
      stepUp: unknown;
    };
  };

  @page(#{id: "admin-user-create", name: "ユーザーを追加", path: "/admin/users/new", route: "/admin/users_/new", actor: "テナント管理者", entry: "管理コンソールのユーザー一覧または対象ユーザーの詳細", context: "対象テナントの管理者として認可を受ける。変更の許可は API が判断する。", purpose: "初期資格情報とプロフィールを指定してユーザーを作成する", content: #["ユーザー名と表示名", "メールと確認済みの指定", "初期パスワード", "初期ロール"], references: #["REQ-FRONTEND-002", "[ナビゲーション](user-interface.md#ナビゲーション)"], area: Area.admin})
  @category(Category.users)
  create: {
    parameters: {

    };
    forms: {
      @form(#{name: "ユーザーの作成", condition: "常時", commit: "作成"})
      create: {
      @field(#{label: "ユーザー名", control: "text", initial: "空", condition: "常時", mapping: "preferred_username"})
      @bind(IdMagic.Contract.AdminUserCreateRequest.preferred_username)
      preferredUsername: IdMagic.Contract.AdminUserCreateRequest.preferred_username::type;
      @field(#{label: "表示名", control: "text", initial: "空", condition: "常時", mapping: "name"})
      @bind(IdMagic.Contract.AdminUserCreateRequest.name)
      name: IdMagic.Contract.AdminUserCreateRequest.name::type;
      @field(#{label: "メールアドレス", control: "email", initial: "空", condition: "常時", mapping: "email"})
      @bind(IdMagic.Contract.AdminUserCreateRequest.email)
      email: IdMagic.Contract.AdminUserCreateRequest.email::type;
      @field(#{label: "メールを確認済みにする", control: "checkbox", initial: "false", condition: "常時", mapping: "email_verified"})
      @bind(IdMagic.Contract.AdminUserCreateRequest.email_verified)
      emailVerified: IdMagic.Contract.AdminUserCreateRequest.email_verified::type;
      @field(#{label: "初期パスワード", control: "password", initial: "空", condition: "常時", mapping: "password"})
      @bind(IdMagic.Contract.AdminUserCreateRequest.password)
      password: IdMagic.Contract.AdminUserCreateRequest.password::type;
      @field(#{label: "初期ロール", control: "text", initial: "空", condition: "常時", mapping: "カンマ区切りを roles 配列へ変換"})
      @bind(IdMagic.Contract.AdminUserCreateRequest.roles)
      roles: string;
      };
    };
    states: FormStates;
    actions: {
      @transition(#{event: "作成", from: "ready", condition: "作成に成功", result: "作成したユーザーの詳細を取得する", toState: "ready", data: "作成したユーザー ID", mode: "navigation"}, AdminUserPages.detail)
      @uses(IdMagic.IdManagement.Operations.CreateAdminUser)
      create: unknown;
      @transition(#{event: "キャンセル", from: "ready", condition: "常時", result: "移動先を表示する", toState: "ready", data: "なし", mode: "navigation"}, AdminUserPages.users)
      cancel: unknown;
      @transition(#{event: "作成", from: "ready", condition: "作成を拒否または通信失敗", result: "入力を保持して問題を表示する", toState: "error", data: "なし", mode: "data"}, AdminUserPages.create)
      failure: unknown;
    };
  };

  @page(#{id: "admin-user-edit", name: "ユーザーを編集", path: "/admin/users/$id/edit", route: "/admin/users_/$id/edit", actor: "テナント管理者", entry: "管理コンソールのユーザー一覧または対象ユーザーの詳細", context: "対象テナントの管理者として認可を受ける。変更の許可は API が判断する。", purpose: "プロフィール、ロールとカスタム属性を一回の保存で変更する", content: #["プロフィールの編集", "ロールの変更内容の確認", "属性定義に応じたカスタム属性"], references: #["REQ-FRONTEND-002", "[ナビゲーション](user-interface.md#ナビゲーション)"], area: Area.admin})
  @category(Category.users)
  edit: {
    parameters: {
      @field(#{label: "ユーザー ID", control: "パスパラメーター", initial: "URL の id", condition: "常時", mapping: "更新するユーザー ID"})
      id: string;
    };
    forms: {
      @form(#{name: "ユーザーの変更", condition: "常時", commit: "変更内容を確認して保存"})
      profile: {
      @field(#{label: "ユーザー名", control: "text", initial: "現在のユーザーの値", condition: "常時", mapping: "preferred_username"})
      @bind(IdMagic.Contract.AdminUserUpdateRequest.preferred_username)
      preferred_username: IdMagic.Contract.AdminUserUpdateRequest.preferred_username::type;
      @field(#{label: "表示名", control: "text", initial: "現在のユーザーの値", condition: "常時", mapping: "name"})
      @bind(IdMagic.Contract.AdminUserUpdateRequest.name)
      name: IdMagic.Contract.AdminUserUpdateRequest.name::type;
      @field(#{label: "名", control: "text", initial: "現在のユーザーの値", condition: "常時", mapping: "given_name"})
      @bind(IdMagic.Contract.AdminUserUpdateRequest.given_name)
      given_name: IdMagic.Contract.AdminUserUpdateRequest.given_name::type;
      @field(#{label: "姓", control: "text", initial: "現在のユーザーの値", condition: "常時", mapping: "family_name"})
      @bind(IdMagic.Contract.AdminUserUpdateRequest.family_name)
      family_name: IdMagic.Contract.AdminUserUpdateRequest.family_name::type;
      @field(#{label: "メールアドレス", control: "email", initial: "現在のユーザーの値", condition: "常時", mapping: "email"})
      @bind(IdMagic.Contract.AdminUserUpdateRequest.email)
      email: IdMagic.Contract.AdminUserUpdateRequest.email::type;
      @field(#{label: "メールの確認状態", control: "checkbox", initial: "現在のユーザーの値", condition: "常時", mapping: "email_verified"})
      @bind(IdMagic.Contract.AdminUserUpdateRequest.email_verified)
      email_verified: IdMagic.Contract.AdminUserUpdateRequest.email_verified::type;
      @field(#{label: "ロール", control: "text", initial: "現在のユーザーの値", condition: "常時", mapping: "カンマ区切りを roles 配列へ変換する"})
      @bind(IdMagic.Contract.AdminUserUpdateRequest.roles)
      roles: string;
      @field(#{label: "カスタム属性", control: "属性の型に応じた入力", initial: "現在のユーザーの値", condition: "テナントに属性定義がある場合", mapping: "定義ごとの入力を属性マップへ変換する"})
      @bind(IdMagic.Contract.AdminUserUpdateRequest.attributes)
      attributes: Record<unknown>;
      };
    };
    states: FormStates;
    actions: {
      @transition(#{event: "保存", from: "ready", condition: "変更内容を確認し保存に成功", result: "更新後の詳細を再取得して一度通知する", toState: "ready", data: "ユーザー ID", mode: "navigation"}, AdminUserPages.detail)
      @uses(IdMagic.IdManagement.Operations.UpdateAdminUser)
      save: unknown;
      @transition(#{event: "キャンセル", from: "ready", condition: "常時", result: "未保存の入力を破棄して詳細へ戻る", toState: "ready", data: "ユーザー ID", mode: "navigation"}, AdminUserPages.detail)
      cancel: unknown;
      @transition(#{event: "保存", from: "ready", condition: "更新の拒否または競合", result: "入力を保持し問題を操作の近くに残す", toState: "error", data: "なし", mode: "data"}, AdminUserPages.edit)
      failure: unknown;
    };
  };

  @page(#{id: "admin-user-import", name: "ユーザーの CSV インポート", path: "/admin/users/import", route: "/admin/users_/import", actor: "テナント管理者", entry: "管理コンソールのユーザー一覧または対象ユーザーの詳細", context: "対象テナントの管理者として認可を受ける。変更の許可は API が判断する。", purpose: "CSV をプレビューしてから確認を経て適用し、行ごとの結果を確認する", content: #["テンプレートのダウンロード", "ファイル選択", "プレビュー", "適用の確認ダイアログ", "行と列ごとの結果と問題"], references: #["REQ-FRONTEND-002", "[ナビゲーション](user-interface.md#ナビゲーション)"], area: Area.admin})
  @category(Category.users)
  importCsv: {
    parameters: {

    };
    forms: {
      @form(#{name: "CSV ファイルの取り込み", condition: "常時", commit: "プレビュー"})
      upload: {
      @field(#{label: "CSV ファイル", control: "file", initial: "未選択", condition: "取り込みの開始時", mapping: "選択した CSV の内容をプレビューの要求へ渡す"})
      file: string;
      };
    };
    states: ImportStates;
    actions: {
      @transition(#{event: "プレビュー", from: "ready", condition: "プレビューのジョブが完了", result: "適用前の内容と行ごとの問題を表示する", toState: "preview", data: "CSV 内容", mode: "data"}, AdminUserPages.importCsv)
      preview: unknown;
      @transition(#{event: "取り込みを適用", from: "preview", condition: "プレビューが完了", result: "適用前の確認ダイアログを表示する", toState: "confirm", data: "プレビューのジョブ ID", mode: "data"}, AdminUserPages.importCsv)
      confirm: unknown;
      @transition(#{event: "キャンセル", from: "confirm", condition: "適用を取りやめる", result: "確認を閉じてプレビューへ戻る", toState: "preview", data: "なし", mode: "data"}, AdminUserPages.importCsv)
      cancelApply: unknown;
      @transition(#{event: "取り込みの適用を確認", from: "confirm", condition: "確認が成立し適用が完了", result: "処理件数と失敗した行を表示する", toState: "success", data: "プレビューのジョブ ID", mode: "data"}, AdminUserPages.importCsv)
      apply: unknown;
      @transition(#{event: "取り込みの適用を確認", from: "confirm", condition: "一部の行の処理が失敗", result: "成功と失敗を分けて表示する", toState: "partial", data: "なし", mode: "data"}, AdminUserPages.importCsv)
      partial: unknown;
      @transition(#{event: "一覧へ戻る", from: "ready", condition: "常時", result: "移動先を表示する", toState: "ready", data: "なし", mode: "navigation"}, AdminUserPages.users)
      back: unknown;
    };
  };
}

@flow(#{id: "manage-users", name: "ユーザーを作成して編集する", goal: "ユーザーを作成し、プロフィールとロールを確認して変更する。", notes: #["既存のユーザーは一覧から詳細へ進み、同じ編集の手順を使う。", "入力不備、権限不足、保存の失敗は各ページの操作表に示す。"]})
model ManageUsersFlow {
  @view(#{state: "ready", visible: #["ユーザー名・状態・ロール", "検索と絞り込み"]}, AdminUserPages.users)
  list: unknown;
  @view(#{state: "ready", visible: #["プロフィール", "初期パスワード・ロール"]}, AdminUserPages.create)
  create: unknown;
  @view(#{state: "ready", visible: #["プロフィール・ロール", "所属・認証状態"]}, AdminUserPages.detail)
  detail: unknown;
  @view(#{state: "ready", visible: #["プロフィール・ロール", "変更内容"]}, AdminUserPages.edit)
  edit: unknown;
  @flowLink(#{trigger: "追加"}, ManageUsersFlow.list, ManageUsersFlow.create, AdminUserPages.users::type.actions::type.create)
  startCreate: unknown;
  @flowLink(#{trigger: "作成"}, ManageUsersFlow.create, ManageUsersFlow.detail, AdminUserPages.create::type.actions::type.create)
  created: unknown;
  @flowLink(#{trigger: "ユーザーを選択"}, ManageUsersFlow.list, ManageUsersFlow.detail, AdminUserPages.users::type.actions::type.detail)
  selectExisting: unknown;
  @flowLink(#{trigger: "編集"}, ManageUsersFlow.detail, ManageUsersFlow.edit, AdminUserPages.detail::type.actions::type.edit)
  startEdit: unknown;
  @flowLink(#{trigger: "保存"}, ManageUsersFlow.edit, ManageUsersFlow.detail, AdminUserPages.edit::type.actions::type.save)
  saved: unknown;
}

@flow(#{id: "import-users", name: "CSV を確認して取り込む", goal: "取り込み前に内容を確認し、確認後にだけ変更を適用する。", notes: #["四つの表示は同じ CSV インポートページ内で切り替わる。", "結果では成功と失敗の件数、失敗した行と項目を確認できる。"]})
model ImportUsersFlow {
  @view(#{name: "ファイルの選択", state: "ready", visible: #["CSV ファイル", "テンプレートの取得"]}, AdminUserPages.importCsv)
  upload: unknown;
  @view(#{name: "内容のプレビュー", state: "preview", visible: #["適用前の変更内容", "行ごとの問題"]}, AdminUserPages.importCsv)
  preview: unknown;
  @view(#{name: "適用の確認", state: "confirm", visible: #["対象・影響の確認", "確認ダイアログ"]}, AdminUserPages.importCsv)
  confirmation: unknown;
  @view(#{name: "取り込み結果", state: "success", visible: #["成功と失敗の件数", "失敗した行と項目"]}, AdminUserPages.importCsv)
  result: unknown;
  @flowLink(#{trigger: "プレビュー"}, ImportUsersFlow.upload, ImportUsersFlow.preview, AdminUserPages.importCsv::type.actions::type.preview)
  previewFile: unknown;
  @flowLink(#{trigger: "適用"}, ImportUsersFlow.preview, ImportUsersFlow.confirmation, AdminUserPages.importCsv::type.actions::type.confirm)
  openConfirmation: unknown;
  @flowLink(#{trigger: "確認して実行"}, ImportUsersFlow.confirmation, ImportUsersFlow.result, AdminUserPages.importCsv::type.actions::type.apply)
  apply: unknown;
  @flowLink(#{trigger: "キャンセル", alternate: true}, ImportUsersFlow.confirmation, ImportUsersFlow.preview, AdminUserPages.importCsv::type.actions::type.cancelApply)
  cancel: unknown;
}

model AccountProfilePages {
  @page(#{...accountContext, id: "account-profile", name: "本人のプロフィール", route: "/account/profile_/", path: "/account/profile", entry: "本人のポータル、編集完了", purpose: "本人の基本情報と公開された属性を確認する", content: #["ユーザー名、表示名、氏名、メール", "属性グループごとの登録内容", "編集の入口と更新通知"]}) @category(Category.profile)
  profile: PageData<{}, {
    @transition(#{...navigation, event: "プロフィールを編集", result: "編集可能な項目を表示する"}, AccountProfilePages.edit) edit: unknown;
  }, {
    @field(#{label: "更新通知", control: "URL", initial: "notice", condition: "編集完了から来た場合", mapping: "success のとき更新完了を通知する。永続化しない"}) notice: string;
  }>;
  @page(#{...accountContext, id: "account-profile-edit", name: "本人のプロフィールを編集", route: "/account/profile_/edit", path: "/account/profile/edit", entry: "本人のプロフィール", purpose: "本人に編集が許可された基本情報と属性を更新する", content: #["基本情報の入力", "editable_attributes の定義に応じた入力", "保存とキャンセル"]}) @category(Category.profile)
  edit: PageData<{
    @form(#{name: "プロフィール", condition: "常時", commit: "保存"}) profile: {
      @field(#{label: "表示名", control: "text", initial: "profile.name", condition: "常時", mapping: "name"}) @bind(IdMagic.Contract.AccountProfileUpdateRequest.name) name: string;
      @field(#{label: "名", control: "text", initial: "profile.given_name", condition: "常時", mapping: "given_name"}) @bind(IdMagic.Contract.AccountProfileUpdateRequest.given_name) givenName: string;
      @field(#{label: "姓", control: "text", initial: "profile.family_name", condition: "常時", mapping: "family_name"}) @bind(IdMagic.Contract.AccountProfileUpdateRequest.family_name) familyName: string;
      @field(#{label: "編集可能なカスタム属性", control: "定義の型に応じた text、number、date、checkbox。配列は区切り入力", initial: "profile.attributes の同じキー", condition: "profile.editable_attributes に含まれる定義ごとに表示", mapping: "attributes[key]。文字列表現を定義の型の AttributeValue へ変換する", repeated: true}) @bind(IdMagic.Contract.AccountProfileUpdateRequest.attributes) attributes: Record<unknown>;
    };
  }, {
    @transition(#{...navigation, event: "保存", condition: "更新成功", result: "プロフィールへ戻り success 通知を表示する"}, AccountProfilePages.profile) save: unknown;
    @transition(#{...navigation, event: "キャンセル・戻る", result: "変更を送信せずプロフィールへ戻る"}, AccountProfilePages.profile) cancel: unknown;
    @transition(#{...update, event: "保存", condition: "更新拒否または通信失敗", result: "入力を保持して問題を表示する", toState: "error"}, AccountProfilePages.edit) failure: unknown;
  }>;
  @page(#{...accountContext, id: "account-emails", name: "本人のメールアドレス", route: "/account/emails", path: "/account/emails", entry: "本人のポータル", purpose: "現在の確認状態を確認し、新しいアドレスの確認を依頼する", content: #["現在のメールと確認状態", "新しいメールアドレス", "確認通知の送信結果"]}) @category(Category.profile)
  emails: PageData<{
    @form(#{name: "メールの変更依頼", condition: "常時", commit: "確認メールを送信"}) change: {
      @field(#{...entered, label: "新しいメールアドレス", control: "email", mapping: "new_email。確認リンクの完了前は現在のアドレスを変えない"}) @bind(IdMagic.Contract.AccountEmailChangeRequest.new_email) email: string;
    };
  }, {
    @transition(#{...update, event: "確認メールを送信", condition: "依頼の受け付け成功", result: "送信結果を表示し、確認リンクをメールへ届ける", toState: "external"}, Destinations.email) send: unknown;
    @transition(#{...update, event: "送信", condition: "入力の拒否または送信失敗", result: "入力の近くに問題を表示する", toState: "error"}, AccountProfilePages.emails) failure: unknown;
  }>;
  @page(#{...browserContext, id: "email-verify", name: "メールアドレスの確認", route: "/account/email/verify", path: "/account/email/verify", entry: "確認メールのリンク", purpose: "メールに含まれるトークンで変更を確認する", content: #["確認の実行ボタン", "確認成功、無効・期限切れなどの結果"]}) @category(Category.profile)
  verify: PageData<{}, {
    @transition(#{...update, event: "メールアドレスを確認", condition: "トークンが有効で確認成功", result: "確認完了を表示する", toState: "success"}, AccountProfilePages.verify) confirm: unknown;
    @transition(#{...update, event: "確認", condition: "トークンがない、期限切れ、使用済みまたは拒否", result: "確認できない理由を表示する", toState: "error"}, AccountProfilePages.verify) failure: unknown;
  }, {
    @field(#{label: "確認トークン", control: "URL", initial: "token", condition: "メールのリンクから受け取る", mapping: "AccountEmailChangeConfirm.token。利用者は編集しない"}) @bind(IdMagic.Contract.AccountEmailChangeConfirm.token) token: string;
  }>;
  @page(#{...accountContext, id: "account-data", name: "本人のデータ", route: "/account/data", path: "/account/data", entry: "本人のポータル", purpose: "本人のデータを JSON として取得する", content: #["取得するデータの説明", "ダウンロードの入口", "取得失敗の表示"]}) @category(Category.profile)
  data: PageData<{}, {
    @transition(#{...update, event: "データをダウンロード", condition: "取得成功", result: "ブラウザーへ本人のデータを渡す", toState: "external"}, Destinations.download) @uses(IdMagic.IdManagement.Operations.ExportAccountData) download: unknown;
  }>;
}
