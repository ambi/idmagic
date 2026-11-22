namespace IdMagicUi;

model SigningKeyPages {
  @page(#{...adminContext, id: "admin-keys", name: "署名鍵", entry: "管理メニュー", purpose: "テナントの JWT 署名鍵と更新を管理する", content: #["公開鍵、状態と有効期間", "選択した鍵の詳細", "ローテーションと無効化の確認", "XML 署名鍵への入口"]}) @category(Category.keys)
  keys: PageData<{
    @transition(#{...update, event: "再取得またはローテーションを確認", result: "現在の鍵と新しい鍵を表示する"}, SigningKeyPages.keys) @uses(IdMagic.SigningKeys.Operations.ListAdminKeys, IdMagic.SigningKeys.Operations.RotateTenantSigningKey) rotate: unknown;
    @transition(#{...update, event: "無効化を確認", result: "鍵の状態を更新する"}, SigningKeyPages.keys) @uses(IdMagic.SigningKeys.Operations.DisableTenantKey) disable: unknown;
    @transition(#{event: "XML 署名鍵へ"}, XmlFederationKeyPages.keys) xml: unknown;
  }, ListStates>;
  @page(#{...systemContext, id: "system-key-health", name: "署名鍵の稼働状態", entry: "システム管理メニュー", purpose: "全テナントの署名鍵の状態を確認する", content: #["テナント別の有効鍵と更新状態", "再取得と失敗の表示"]}) @category(Category.keys)
  health: PageData<{
    @transition(#{...update, event: "再取得", result: "テナントごとの署名鍵の状態を更新する"}, SigningKeyPages.health) @uses(IdMagic.SigningKeys.Operations.ListTenantKeyHealth) refresh: unknown;
  }, ReadStates>;
}

model XmlFederationKeyPages {
  @page(#{...adminContext, id: "admin-xml-federation-key", name: "XML フェデレーションの署名鍵", entry: "署名鍵または Entra フェデレーション", purpose: "信頼先の更新を確認して共有 XML 署名鍵を切り替える", content: #["現行、準備中と検証用の鍵", "Entra、WS-Federation と SAML の信頼先", "証明書と手順のコピー", "信頼先ごとと全体の更新確認", "切り替え直前の再照合と準備の取消確認"]}) @category(Category.keys)
  keys: PageData<{
    @transition(#{...update, event: "次の鍵を準備", result: "準備した鍵と信頼先を再取得する", toState: "prepared"}, XmlFederationKeyPages.keys) @uses(IdMagic.SigningKeys.Operations.RotateTenantSigningKey) prepare: unknown;
    @transition(#{...update, event: "切り替えを確認", from: "prepared", condition: "全信頼先の更新確認があり再取得した内容が一致", result: "準備した鍵を署名に使い、表示を再取得する"}, XmlFederationKeyPages.keys) @uses(IdMagic.SigningKeys.Operations.RotateTenantSigningKey) activate: unknown;
    @transition(#{...update, event: "準備の取消を確認", from: "prepared", result: "準備した鍵を無効にして再取得する"}, XmlFederationKeyPages.keys) @uses(IdMagic.SigningKeys.Operations.DisableTenantKey) cancel: unknown;
    @transition(#{event: "Entra の設定"}, EntraFederationPages.list) entra: unknown;
    @transition(#{event: "信頼先のドメインを開く", data: "ドメイン"}, EntraFederationPages.detail) domain: unknown;
    @transition(#{event: "署名鍵へ戻る"}, SigningKeyPages.keys) back: unknown;
  }, { ...FormStates; prepared: "次の鍵と各信頼先の更新確認を表示する。"; }>;
}
