namespace IdMagicUi;

model SignInPolicyPages {
  @page(#{...adminContext, id: "admin-sign-in-policy", name: "サインインポリシー", entry: "管理メニュー", purpose: "テナントのデフォルトの規則とアプリごとの上書きを確認する", content: #["デフォルトの規則の閲覧とページ内での編集", "条件と要求する認証の強さ", "アプリケーション別の規則と編集への入口"]}) @category(Category.security)
  policy: PageData<{
    @transition(#{...update, event: "編集して保存", result: "デフォルトのサインイン規則を更新し閲覧へ戻す"}, SignInPolicyPages.policy) @uses(IdMagic.Application.Operations.UpdateTenantDefaultSignInPolicy) save: unknown;
    @transition(#{...update, event: "キャンセル", result: "編集を破棄して現在の規則を表示する"}, SignInPolicyPages.policy) cancel: unknown;
    @transition(#{event: "アプリの規則を編集", data: "アプリケーション ID"}, AdminApplicationPages.edit) application: unknown;
  }>;
}

model AdminApplicationPages {
  @page(#{...adminContext, id: "admin-applications", name: "アプリケーション一覧", entry: "管理メニュー", purpose: "登録アプリケーションを選び、設定と割り当てを確認する", content: #["ページ送り、再取得と選択対象の詳細", "プロトコルと状態", "割り当て済みのカテゴリ", "削除の確認"]}) @category(Category.applications)
  list: PageData<{
    @transition(#{...update, event: "選択または再取得", result: "一覧と選択対象を更新する"}, AdminApplicationPages.list) @uses(IdMagic.Application.Operations.ListAdminApplications) refresh: unknown;
    @transition(#{event: "作成"}, AdminApplicationPages.create) create: unknown;
    @transition(#{event: "編集"}, AdminApplicationPages.edit) edit: unknown;
    @transition(#{...update, event: "削除を確認", result: "対象を削除して一覧を再取得する"}, AdminApplicationPages.list) @uses(IdMagic.Application.Operations.DeleteAdminApplication) delete: unknown;
  }, ListStates>;
  @page(#{...adminContext, id: "admin-application-detail", name: "アプリケーション詳細", entry: "対象アプリケーションへの直接アクセス", purpose: "プロトコル、利用者への割り当てと資格情報の状態を確認する", content: #["基本情報、接続情報とセットアップ案内", "ユーザーとグループへの割り当ての一覧", "割り当て済みのカテゴリ", "編集、プロビジョニング、起動と削除の入口"]}) @category(Category.applications)
  detail: PageData<{
    @transition(#{event: "編集"}, AdminApplicationPages.edit) edit: unknown;
    @transition(#{event: "プロビジョニング"}, ApplicationProvisioningPages.detail) provisioning: unknown;
    @transition(#{event: "アプリを開く", condition: "起動 URL がある", result: "別タブで開く", toState: "external"}, Destinations.client) launch: unknown;
    @transition(#{event: "削除を確認", condition: "削除が成功", result: "一覧へ戻る"}, AdminApplicationPages.list) @uses(IdMagic.Application.Operations.DeleteAdminApplication) delete: unknown;
    @transition(#{event: "戻る"}, AdminApplicationPages.list) back: unknown;
  }, FormStates>;
  @page(#{...adminContext, id: "admin-application-create", name: "アプリケーションを作成", entry: "アプリケーション一覧", purpose: "接続方式に応じたアプリケーションを登録する", content: #["OIDC、SAML、WS-Federation、Web リンクとサービスの選択", "方式に応じた接続設定", "生成したシークレットの一度だけの表示"]}) @category(Category.applications)
  create: PageData<{
    @transition(#{...update, event: "作成", condition: "資格情報が発行された", result: "シークレットを一度だけ表示する", toState: "secret"}, AdminApplicationPages.create) @uses(IdMagic.Application.Operations.CreateAdminApplication) createSecret: unknown;
    @transition(#{event: "作成", condition: "シークレットの表示が不要", result: "作成した対象の詳細へ進む"}, AdminApplicationPages.detail) @uses(IdMagic.Application.Operations.CreateAdminApplication) create: unknown;
    @transition(#{event: "控えたことを確認", from: "secret", result: "作成した詳細へ進む"}, AdminApplicationPages.detail) continue: unknown;
    @transition(#{event: "キャンセルまたは一覧へ戻る"}, AdminApplicationPages.list) cancel: unknown;
  }, { ...FormStates; secret: "発行した資格情報を表示し、控えたことを確認する。"; }>;
  @page(#{...adminContext, id: "admin-application-edit", name: "アプリケーションを編集", entry: "一覧または詳細", purpose: "アプリケーションの表示情報、接続・認証設定、資格情報、割り当てとカテゴリを管理する", content: #["基本情報とアイコンの追加・削除", "方式別の接続設定と SAML 専用 IdP プロファイルの作成", "サインイン規則", "複数の保存先の検査と失敗時の入力保持", "ローテーションできる OIDC シークレットの発行、初回だけの表示と失効確認", "ユーザーとグループへの割り当ての追加と解除", "カテゴリの作成・削除と割り当て"]}) @category(Category.applications)
  edit: PageData<{
    @transition(#{...update, event: "専用の IdP プロファイルを作成", condition: "SAML アプリケーション", result: "作成したプロファイルを選択肢に加えて選ぶ"}, AdminApplicationPages.edit) @uses(IdMagic.Saml.Operations.CreateSamlIdentityProviderProfile) samlProfile: unknown;
    @transition(#{...update, event: "シークレットを発行", condition: "シークレットをローテーションできる", result: "発行したシークレットを一度だけ表示する", toState: "secret"}, AdminApplicationPages.edit) @uses(IdMagic.Application.Operations.IssueApplicationClientSecret) issue: unknown;
    @transition(#{...update, event: "控えたシークレットを閉じる", from: "secret", result: "シークレットの表示を消す"}, AdminApplicationPages.edit) hide: unknown;
    @transition(#{...update, event: "失効を確認", result: "資格情報の状態を更新する"}, AdminApplicationPages.edit) @uses(IdMagic.Application.Operations.RevokeApplicationClientSecret) revoke: unknown;
    @transition(#{...update, event: "割り当てを追加または解除", condition: "サービス以外のアプリケーション", result: "割り当てを更新する"}, AdminApplicationPages.edit) @uses(IdMagic.Application.Operations.AssignApplication, IdMagic.Application.Operations.UnassignApplication) assignments: unknown;
    @transition(#{...update, event: "カテゴリを管理", condition: "サービス以外のアプリケーション", result: "カテゴリと割り当てを更新する"}, AdminApplicationPages.edit) @uses(IdMagic.Application.Operations.CreateApplicationCategory, IdMagic.Application.Operations.DeleteApplicationCategory, IdMagic.Application.Operations.SetApplicationCategories) category: unknown;
    @transition(#{event: "保存", condition: "すべての変更の保存が成功", result: "更新後の詳細へ戻る"}, AdminApplicationPages.detail) @uses(IdMagic.Application.Operations.UpdateAdminApplication, IdMagic.Application.Operations.UploadApplicationIcon, IdMagic.Application.Operations.DeleteApplicationIcon, IdMagic.Application.Operations.UpdateApplicationOidcConfig, IdMagic.Application.Operations.UpdateApplicationWsFedConfig, IdMagic.Application.Operations.UpdateApplicationSamlConfig, IdMagic.Application.Operations.UpdateAppSignInPolicy) save: unknown;
    @transition(#{...update, event: "保存", condition: "一部の保存先が失敗", result: "入力を保持して失敗した操作を表示する", toState: "error"}, AdminApplicationPages.edit) failure: unknown;
    @transition(#{event: "キャンセル"}, AdminApplicationPages.detail) cancel: unknown;
  }, { ...FormStates; secret: "新しいシークレットと控えた後に閉じる操作を表示する。"; }>;
}

model PortalApplicationPages {
  @page(#{...accountContext, id: "account-apps", name: "本人のアプリケーションランチャー", entry: "本人のポータル", purpose: "利用可能なアプリケーションを起動し表示順を変更する", content: #["カテゴリ別のアプリケーションカード", "起動の入口", "ドラッグ操作またはキーボードによる並べ替え", "保存中と保存失敗"]}) @category(Category.applications)
  apps: PageData<{
    @transition(#{...update, event: "表示順を変更", condition: "並べ替えが確定", result: "並べ替えた順序を保存する"}, PortalApplicationPages.apps) @uses(IdMagic.Application.Operations.ReorderMyApplications) reorder: unknown;
    @transition(#{...navigation, event: "アプリケーションを選択", condition: "launch_url がありドラッグ操作の直後でない", result: "別タブでアプリケーションを開く", toState: "external"}, Destinations.client) launch: unknown;
  }>;
}
