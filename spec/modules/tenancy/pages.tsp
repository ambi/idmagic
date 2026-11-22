namespace IdMagicUi;

model SystemTenantPages {
  @page(#{...systemContext, id: "system-tenants", name: "テナント一覧", entry: "システムコンソール", purpose: "テナントの状態と管理の入口を確認する", content: #["テナントの一覧と再取得", "詳細、編集と新規作成", "有効化と無効化の確認、再認証"]}) @category(Category.tenants)
  list: PageData<{
    @transition(#{...update, event: "再取得", result: "現在のテナント一覧を表示する"}, SystemTenantPages.list) @uses(IdMagic.Tenancy.Operations.ListTenants) refresh: unknown;
    @transition(#{event: "作成"}, SystemTenantPages.create) create: unknown;
    @transition(#{event: "詳細"}, SystemTenantPages.detail) detail: unknown;
    @transition(#{event: "編集"}, SystemTenantPages.edit) edit: unknown;
    @transition(#{...update, event: "有効化または無効化を確認", result: "状態を変更し一覧を再取得する"}, SystemTenantPages.list) @uses(IdMagic.Tenancy.Operations.EnableTenant, IdMagic.Tenancy.Operations.DisableTenant) status: unknown;
  }, ListStates>;
  @page(#{...systemContext, id: "system-tenant-create", name: "テナントを作成", entry: "テナント一覧", purpose: "テナントの識別子と初期設定を登録する", content: #["識別子、表示名と初期設定", "再認証、作成とキャンセル"]}) @category(Category.tenants)
  create: PageData<{
    @transition(#{event: "作成", condition: "必要な再認証と登録が成功", result: "作成したテナントの詳細へ進む"}, SystemTenantPages.detail) @uses(IdMagic.Tenancy.Operations.CreateTenant) save: unknown;
    @transition(#{event: "キャンセル"}, SystemTenantPages.list) cancel: unknown;
  }>;
  @page(#{...systemContext, id: "system-tenant-detail", name: "テナント詳細", entry: "テナント一覧または保存後", purpose: "状態、入口の方式とクォータを確認する", content: #["基本情報と状態", "有効化と無効化の確認", "基本情報、入口方式とクォータへの編集の入口"]}) @category(Category.tenants)
  detail: PageData<{
    @transition(#{event: "基本情報を編集"}, SystemTenantPages.edit) edit: unknown;
    @transition(#{event: "入口方式を変更"}, SystemTenantPages.endpoint) endpoint: unknown;
    @transition(#{event: "クォータを編集"}, SystemTenantPages.quota) quota: unknown;
    @transition(#{...update, event: "有効化または無効化を確認", result: "状態を変更し詳細を再取得する"}, SystemTenantPages.detail) @uses(IdMagic.Tenancy.Operations.EnableTenant, IdMagic.Tenancy.Operations.DisableTenant) status: unknown;
    @transition(#{event: "戻る"}, SystemTenantPages.list) back: unknown;
  }>;
  @page(#{...systemContext, id: "system-tenant-edit", name: "テナントの基本情報を編集", entry: "テナント詳細または一覧", purpose: "表示情報と基本設定を変更する", content: #["既存設定と未保存の変更", "必要な再認証、保存とキャンセル"]}) @category(Category.tenants)
  edit: PageData<{
    @transition(#{event: "保存", condition: "更新が成功", result: "再取得した詳細へ戻る"}, SystemTenantPages.detail) @uses(IdMagic.Tenancy.Operations.UpdateTenant) save: unknown;
    @transition(#{event: "キャンセル"}, SystemTenantPages.detail) cancel: unknown;
  }>;
  @page(#{...systemContext, id: "system-tenant-endpoint-style", name: "テナントの入口方式", entry: "テナント詳細", purpose: "テナントの URL 方式を影響を確認して変更する", content: #["現在と変更後の URL 方式", "既存連携への影響と変更確認", "再認証とキャンセル"]}) @category(Category.tenants)
  endpoint: PageData<{
    @transition(#{event: "変更を確認して保存", condition: "更新が成功", result: "変更後の詳細へ戻る"}, SystemTenantPages.detail) @uses(IdMagic.Tenancy.Operations.SetTenantEndpointStyle) save: unknown;
    @transition(#{event: "キャンセル"}, SystemTenantPages.detail) cancel: unknown;
  }>;
  @page(#{...systemContext, id: "system-tenant-quota", name: "テナントのクォータ", entry: "テナント詳細", purpose: "テナントの上限と使用量を確認し上限を変更する", content: #["現在の使用量と上限", "再認証、保存とキャンセル"]}) @category(Category.tenants)
  quota: PageData<{
    @transition(#{event: "保存", condition: "更新が成功", result: "変更後の詳細へ戻る"}, SystemTenantPages.detail) @uses(IdMagic.Tenancy.Operations.UpdateTenantQuota) save: unknown;
    @transition(#{event: "キャンセル"}, SystemTenantPages.detail) cancel: unknown;
  }>;
}

model AdminSettingsPages {
  @page(#{...adminContext, id: "admin-settings", name: "テナント設定", entry: "管理メニュー", purpose: "テナントの共通設定と通知・表示を管理する", content: #["URL の tab に応じたタブの選択", "一般、パスワード、委任と信頼済みデバイスのポリシー", "ブランドの表示名、配色と画像", "接続 URL と SAML IdP プロファイルの入口", "通知テンプレートの言語別編集、初期化、プレビューとテスト送信", "各設定は対応するタブ内で確定する"]}) @category(Category.settings)
  settings: PageData<{
    @transition(#{...update, event: "共通設定を保存", result: "対象タブの設定を更新する"}, AdminSettingsPages.settings) @uses(IdMagic.Tenancy.Operations.UpdateAdminSettings) save: unknown;
    @transition(#{...update, event: "ブランドを保存または画像を追加・削除", result: "表示設定と画像を更新する"}, AdminSettingsPages.settings) @uses(IdMagic.Tenancy.Operations.UpdateTenantBranding, IdMagic.Tenancy.Operations.UploadTenantBrandingAsset, IdMagic.Tenancy.Operations.DeleteTenantBrandingAsset) branding: unknown;
    @transition(#{...update, event: "通知を保存または初期化", result: "選択したテンプレートと言語の内容を更新する"}, AdminSettingsPages.settings) @uses(IdMagic.Tenancy.Operations.UpdateNotificationTemplate, IdMagic.Tenancy.Operations.ResetNotificationTemplate) template: unknown;
    @transition(#{...update, event: "通知をプレビューまたはテスト送信", result: "描画結果または送信結果を表示する"}, AdminSettingsPages.settings) @uses(IdMagic.Tenancy.Operations.PreviewNotificationTemplate, IdMagic.Tenancy.Operations.SendTestNotification) preview: unknown;
    @transition(#{event: "SAML IdP プロファイル"}, SamlProfilePages.list) profiles: unknown;
    @transition(#{event: "テナント管理", condition: "システム管理権限がある"}, SystemTenantPages.list) tenants: unknown;
  }>;
}

model TenantAttributePages {
  @page(#{...adminContext, id: "admin-tenant-attributes", name: "ユーザー属性の定義", entry: "管理メニュー", purpose: "ユーザーのカスタム属性の型と公開範囲を管理する", content: #["型、表示名、編集・閲覧範囲と並び順", "属性を編集するダイアログと削除の確認", "新規定義への入口"]}) @category(Category.settings)
  users: PageData<{
    @transition(#{event: "追加"}, TenantAttributePages.createUser) create: unknown;
    @transition(#{...update, event: "編集を保存または削除を確認", result: "属性の定義を更新する"}, TenantAttributePages.users) @uses(IdMagic.Tenancy.Operations.UpdateTenantUserAttributeSchema) save: unknown;
    @transition(#{...update, event: "再取得", result: "現在の定義を表示する"}, TenantAttributePages.users) @uses(IdMagic.Tenancy.Operations.GetTenantUserAttributeSchema) refresh: unknown;
  }, ListStates>;
  @page(#{...adminContext, id: "admin-tenant-attribute-create", name: "ユーザー属性を追加", entry: "ユーザー属性の定義", purpose: "新しいカスタム属性を登録する", content: #["型、名前、公開範囲と入力の選択肢", "保存とキャンセル"]}) @category(Category.settings)
  createUser: PageData<{
    @transition(#{event: "保存", condition: "定義の更新が成功", result: "ユーザー属性の一覧へ戻る"}, TenantAttributePages.users) @uses(IdMagic.Tenancy.Operations.UpdateTenantUserAttributeSchema) save: unknown;
    @transition(#{event: "キャンセル"}, TenantAttributePages.users) cancel: unknown;
  }>;
  @page(#{...adminContext, id: "admin-tenant-group-attributes", name: "グループ属性の定義", entry: "管理メニュー", purpose: "グループのカスタム属性を管理する", content: #["型、表示と編集の設定", "編集ダイアログ、削除の確認と新規定義"]}) @category(Category.settings)
  groups: PageData<{
    @transition(#{event: "追加"}, TenantAttributePages.createGroup) create: unknown;
    @transition(#{...update, event: "編集を保存または削除を確認", result: "グループ属性の定義を更新する"}, TenantAttributePages.groups) @uses(IdMagic.Tenancy.Operations.UpdateTenantGroupAttributeSchema) save: unknown;
    @transition(#{...update, event: "再取得", result: "現在の定義を表示する"}, TenantAttributePages.groups) @uses(IdMagic.Tenancy.Operations.GetTenantGroupAttributeSchema) refresh: unknown;
  }, ListStates>;
  @page(#{...adminContext, id: "admin-group-attribute-create", name: "グループ属性を追加", entry: "グループ属性の定義", purpose: "新しいグループ属性を登録する", content: #["属性の型と表示・編集の設定", "保存とキャンセル"]}) @category(Category.settings)
  createGroup: PageData<{
    @transition(#{event: "保存", condition: "定義の更新が成功", result: "グループ属性の一覧へ戻る"}, TenantAttributePages.groups) @uses(IdMagic.Tenancy.Operations.GetTenantGroupAttributeSchema, IdMagic.Tenancy.Operations.UpdateTenantGroupAttributeSchema) save: unknown;
    @transition(#{event: "キャンセル"}, TenantAttributePages.groups) cancel: unknown;
  }>;
}
