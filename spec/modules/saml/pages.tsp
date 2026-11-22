namespace IdMagicUi;

model SamlProfilePages {
  @page(#{...adminContext, id: "admin-saml-idp-profiles", name: "SAML IdP プロファイル", entry: "テナント設定の接続情報", purpose: "共有または専用の IdP 識別子を管理する", content: #["プロファイルと利用するアプリ", "再取得、作成と詳細"]}) @category(Category.federation)
  list: PageData<{
    @transition(#{...update, event: "再取得", result: "プロファイルの一覧を更新する"}, SamlProfilePages.list) @uses(IdMagic.Saml.Operations.ListSamlIdentityProviderProfiles) refresh: unknown;
    @transition(#{event: "作成"}, SamlProfilePages.create) create: unknown;
    @transition(#{event: "詳細"}, SamlProfilePages.detail) detail: unknown;
    @transition(#{event: "戻る"}, AdminSettingsPages.settings) back: unknown;
  }, ListStates>;
  @page(#{...adminContext, id: "admin-saml-idp-profile-create", name: "SAML IdP プロファイルを作成", entry: "プロファイル一覧", purpose: "名前と共有方式を指定してプロファイルを作る", content: #["表示名と共有・専用の方式", "作成とキャンセル"]}) @category(Category.federation)
  create: PageData<{
    @transition(#{event: "作成", condition: "登録が成功", result: "作成した詳細へ進む"}, SamlProfilePages.detail) @uses(IdMagic.Saml.Operations.CreateSamlIdentityProviderProfile) save: unknown;
    @transition(#{event: "キャンセル"}, SamlProfilePages.list) cancel: unknown;
  }>;
  @page(#{...adminContext, id: "admin-saml-idp-profile-detail", name: "SAML IdP プロファイルの詳細", entry: "プロファイル一覧", purpose: "識別子、接続先と利用状態を確認する", content: #["メタデータ、証明書と利用アプリ", "編集と削除の確認"]}) @category(Category.federation)
  detail: PageData<{
    @transition(#{event: "編集"}, SamlProfilePages.edit) edit: unknown;
    @transition(#{event: "削除を確認", condition: "削除が成功", result: "一覧へ戻る"}, SamlProfilePages.list) @uses(IdMagic.Saml.Operations.DeleteSamlIdentityProviderProfile) delete: unknown;
    @transition(#{event: "戻る"}, SamlProfilePages.list) back: unknown;
  }>;
  @page(#{...adminContext, id: "admin-saml-idp-profile-edit", name: "SAML IdP プロファイルを編集", entry: "プロファイル詳細", purpose: "プロファイルの名前を変更する", content: #["現在の表示名", "保存とキャンセル"]}) @category(Category.federation)
  edit: PageData<{
    @transition(#{event: "保存", condition: "更新が成功", result: "詳細へ戻る"}, SamlProfilePages.detail) @uses(IdMagic.Saml.Operations.UpdateSamlIdentityProviderProfile) save: unknown;
    @transition(#{event: "キャンセル"}, SamlProfilePages.detail) cancel: unknown;
  }>;
}
