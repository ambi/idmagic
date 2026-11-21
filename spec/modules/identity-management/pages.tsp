namespace IdMagicUi;

const userAdminContext = #{area: Area.admin, actor: "テナント管理者", entry: "管理コンソールのユーザー一覧または対象ユーザーの詳細", context: "対象テナントの管理者として認可を受ける。変更の許可は API が判断する。", references: #["REQ-FRONTEND-002", "[ナビゲーション](user-interface.md#ナビゲーション)"]};

/** IdManagement が主担当となるユーザー管理のページ。 */
model AdminUserPages {
  @page(#{...userAdminContext, id: "admin-users", name: "ユーザー一覧", purpose: "ユーザーを検索し、対象の確認と管理の入口を示す", content: #["検索語と状態による絞り込み。条件は URL に保持する", "ユーザー名、表示名、メールアドレス、状態とロール", "総件数、ページ送りと再取得", "作成と CSV の入口"]})
  @category(Category.users)
  users: PageData<{
    @transition(#{event: "検索または絞り込み", result: "条件に合う一覧を表示する", data: "query、status、cursor", mode: "data"}, AdminUserPages.users) @uses(IdMagic.IdManagement.Operations.ListAdminUsers) search: unknown;
    @transition(#{event: "詳細", data: "ユーザー ID"}, AdminUserPages.detail) detail: unknown;
    @transition(#{event: "ユーザーを追加"}, AdminUserPages.create) create: unknown;
    @transition(#{event: "CSV インポート"}, AdminUserPages.importCsv) importCsv: unknown;
  }, ListStates>;

  @page(#{...userAdminContext, id: "admin-user-detail", name: "ユーザー詳細", purpose: "現在のプロフィール、権限、所属と認証の状態を確認する", content: #["プロフィール", "ロールとグループ", "認証情報とセッション", "必須アクション", "無効化と削除の確認。削除はユーザー名の入力で確認する", "解除する認証情報を選ぶリセットのダイアログ"]})
  @category(Category.users)
  detail: PageData<{
    @transition(#{event: "編集", data: "ユーザー ID"}, AdminUserPages.edit) edit: unknown;
    @transition(#{event: "削除を確認", condition: "確認が成立し API が削除を受理", result: "一覧を再取得する", data: "ユーザー ID"}, AdminUserPages.users) @uses(IdMagic.IdManagement.Operations.DeleteAdminUser) delete: unknown;
    @transition(#{event: "一覧へ戻る"}, AdminUserPages.users) back: unknown;
    @transition(#{event: "管理操作", condition: "API が再認証を要求", result: "再認証後に元の操作を一度だけ再試行する", toState: "dialog", data: "元の操作と再認証の要求"}, Destinations.stepUp) stepUp: unknown;
  }>;

  @page(#{...userAdminContext, id: "admin-user-create", name: "ユーザーを追加", purpose: "初期資格情報とプロフィールを指定してユーザーを作成する", content: #["ユーザー名と表示名", "メールと確認済みの指定", "初期パスワード", "初期ロール"]})
  @category(Category.users)
  create: PageData<{
    @transition(#{event: "作成", condition: "作成に成功", result: "作成したユーザーの詳細を取得する", data: "作成したユーザー ID"}, AdminUserPages.detail) @uses(IdMagic.IdManagement.Operations.CreateAdminUser) create: unknown;
    @transition(#{event: "キャンセル"}, AdminUserPages.users) cancel: unknown;
    @transition(#{event: "作成", condition: "作成を拒否または通信失敗", result: "入力を保持して問題を表示する", toState: "error", mode: "data"}, AdminUserPages.create) failure: unknown;
  }>;

  @page(#{...userAdminContext, id: "admin-user-edit", name: "ユーザーを編集", purpose: "プロフィール、ロールとカスタム属性を一回の保存で変更する", content: #["プロフィールの編集", "ロールの変更内容の確認", "属性定義に応じたカスタム属性"]})
  @category(Category.users)
  edit: PageData<{
    @transition(#{event: "保存", condition: "変更内容を確認し保存に成功", result: "更新後の詳細を再取得して一度通知する", data: "ユーザー ID"}, AdminUserPages.detail) @uses(IdMagic.IdManagement.Operations.UpdateAdminUser) save: unknown;
    @transition(#{event: "キャンセル", result: "未保存の入力を破棄して詳細へ戻る", data: "ユーザー ID"}, AdminUserPages.detail) cancel: unknown;
    @transition(#{event: "保存", condition: "更新の拒否または競合", result: "入力を保持し問題を操作の近くに残す", toState: "error", mode: "data"}, AdminUserPages.edit) failure: unknown;
  }>;

  @page(#{...userAdminContext, id: "admin-user-import", name: "ユーザーの CSV インポート", purpose: "CSV をプレビューしてから確認を経て適用し、行ごとの結果を確認する", content: #["テンプレートのダウンロード", "ファイル選択", "プレビュー", "適用の確認ダイアログ", "行と列ごとの結果と問題"]})
  @category(Category.users)
  importCsv: PageData<{
    @transition(#{event: "プレビュー", condition: "プレビューのジョブが完了", result: "適用前の内容と行ごとの問題を表示する", toState: "preview", data: "CSV 内容", mode: "data"}, AdminUserPages.importCsv) preview: unknown;
    @transition(#{event: "取り込みを適用", from: "preview", condition: "プレビューが完了", result: "適用前の確認ダイアログを表示する", toState: "confirm", data: "プレビューのジョブ ID", mode: "data"}, AdminUserPages.importCsv) confirm: unknown;
    @transition(#{event: "キャンセル", from: "confirm", condition: "適用を取りやめる", result: "確認を閉じてプレビューへ戻る", toState: "preview", mode: "data"}, AdminUserPages.importCsv) cancelApply: unknown;
    @transition(#{event: "取り込みの適用を確認", from: "confirm", condition: "確認が成立し適用が完了", result: "処理件数と失敗した行を表示する", toState: "success", data: "プレビューのジョブ ID", mode: "data"}, AdminUserPages.importCsv) apply: unknown;
    @transition(#{event: "取り込みの適用を確認", from: "confirm", condition: "一部の行の処理が失敗", result: "成功と失敗を分けて表示する", toState: "partial", mode: "data"}, AdminUserPages.importCsv) partial: unknown;
    @transition(#{event: "一覧へ戻る"}, AdminUserPages.users) back: unknown;
  }, ImportStates>;
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
  @flowLink(#{trigger: "追加", action: "create"}, ManageUsersFlow.list, ManageUsersFlow.create)
  startCreate: unknown;
  @flowLink(#{trigger: "作成", action: "create"}, ManageUsersFlow.create, ManageUsersFlow.detail)
  created: unknown;
  @flowLink(#{trigger: "ユーザーを選択", action: "detail"}, ManageUsersFlow.list, ManageUsersFlow.detail)
  selectExisting: unknown;
  @flowLink(#{trigger: "編集", action: "edit"}, ManageUsersFlow.detail, ManageUsersFlow.edit)
  startEdit: unknown;
  @flowLink(#{trigger: "保存", action: "save"}, ManageUsersFlow.edit, ManageUsersFlow.detail)
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
  @flowLink(#{trigger: "プレビュー", action: "preview"}, ImportUsersFlow.upload, ImportUsersFlow.preview)
  previewFile: unknown;
  @flowLink(#{trigger: "適用", action: "confirm"}, ImportUsersFlow.preview, ImportUsersFlow.confirmation)
  openConfirmation: unknown;
  @flowLink(#{trigger: "確認して実行", action: "apply"}, ImportUsersFlow.confirmation, ImportUsersFlow.result)
  apply: unknown;
  @flowLink(#{trigger: "キャンセル", alternate: true, action: "cancelApply"}, ImportUsersFlow.confirmation, ImportUsersFlow.preview)
  cancel: unknown;
}

model AccountProfilePages {
  @page(#{...accountContext, id: "account-profile", name: "本人のプロフィール", entry: "本人のポータル、編集完了", purpose: "本人の基本情報と公開された属性を確認する", content: #["ユーザー名、表示名、氏名、メール", "属性グループごとの登録内容", "編集の入口と更新通知"]}) @category(Category.profile)
  profile: PageData<{
    @transition(#{...navigation, event: "プロフィールを編集", result: "編集可能な項目を表示する"}, AccountProfilePages.edit) edit: unknown;
  }>;
  @page(#{...accountContext, id: "account-profile-edit", name: "本人のプロフィールを編集", entry: "本人のプロフィール", purpose: "本人に編集が許可された基本情報と属性を更新する", content: #["基本情報の入力", "editable_attributes の定義に応じた入力", "保存とキャンセル"]}) @category(Category.profile)
  edit: PageData<{
    @transition(#{...navigation, event: "保存", condition: "更新成功", result: "プロフィールへ戻り success 通知を表示する"}, AccountProfilePages.profile) save: unknown;
    @transition(#{...navigation, event: "キャンセル・戻る", result: "変更を送信せずプロフィールへ戻る"}, AccountProfilePages.profile) cancel: unknown;
    @transition(#{...update, event: "保存", condition: "更新拒否または通信失敗", result: "入力を保持して問題を表示する", toState: "error"}, AccountProfilePages.edit) failure: unknown;
  }>;
  @page(#{...accountContext, id: "account-emails", name: "本人のメールアドレス", entry: "本人のポータル", purpose: "現在の確認状態を確認し、新しいアドレスの確認を依頼する", content: #["現在のメールと確認状態", "新しいメールアドレス。確認リンクの完了前は現在のアドレスを変えない", "確認通知の送信結果"]}) @category(Category.profile)
  emails: PageData<{
    @transition(#{...update, event: "確認メールを送信", condition: "依頼の受け付け成功", result: "送信結果を表示し、確認リンクをメールへ届ける", toState: "external"}, Destinations.email) send: unknown;
    @transition(#{...update, event: "送信", condition: "入力の拒否または送信失敗", result: "入力の近くに問題を表示する", toState: "error"}, AccountProfilePages.emails) failure: unknown;
  }>;
  @page(#{...browserContext, id: "email-verify", name: "メールアドレスの確認", entry: "確認メールのリンク", purpose: "メールに含まれるトークンで変更を確認する", content: #["確認の実行ボタン", "確認成功、無効・期限切れなどの結果"]}) @category(Category.profile)
  verify: PageData<{
    @transition(#{...update, event: "メールアドレスを確認", condition: "トークンが有効で確認成功", result: "確認完了を表示する", toState: "success"}, AccountProfilePages.verify) confirm: unknown;
    @transition(#{...update, event: "確認", condition: "トークンがない、期限切れ、使用済みまたは拒否", result: "確認できない理由を表示する", toState: "error"}, AccountProfilePages.verify) failure: unknown;
  }>;
  @page(#{...accountContext, id: "account-data", name: "本人のデータ", entry: "本人のポータル", purpose: "本人のデータを JSON として取得する", content: #["取得するデータの説明", "ダウンロードの入口", "取得失敗の表示"]}) @category(Category.profile)
  data: PageData<{
    @transition(#{...update, event: "データをダウンロード", condition: "取得成功", result: "ブラウザーへ本人のデータを渡す", toState: "external"}, Destinations.download) @uses(IdMagic.IdManagement.Operations.ExportAccountData) download: unknown;
  }>;
}
