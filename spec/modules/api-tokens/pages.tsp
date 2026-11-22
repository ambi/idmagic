namespace IdMagicUi;

model ApiTokenPages {
  @page(#{...adminContext, id: "admin-api-tokens", name: "管理 API トークン", entry: "管理メニュー", purpose: "管理用トークンの範囲、有効期限と失効状態を確認する", content: #["説明、スコープ、期限と利用状態", "発行への入口と失効の確認"]}) @category(Category.apiTokens)
  admin: PageData<{
    @transition(#{...update, event: "再取得", result: "現在のトークン一覧を表示する"}, ApiTokenPages.admin) @uses(IdMagic.ApiTokens.Operations.ListApiTokens) refresh: unknown;
    @transition(#{event: "発行"}, ApiTokenPages.createAdmin) create: unknown;
    @transition(#{...update, event: "失効を確認", result: "対象トークンを失効として表示する"}, ApiTokenPages.admin) @uses(IdMagic.ApiTokens.Operations.RevokeApiToken) revoke: unknown;
  }, ListStates>;
  @page(#{...adminContext, id: "admin-api-token-create", name: "管理 API トークンを発行", entry: "管理 API トークン一覧", purpose: "用途と権限を指定してトークンを発行する", content: #["説明、スコープと有効日数", "発行したトークンの一度だけの表示とコピー"]}) @category(Category.apiTokens)
  createAdmin: PageData<{
    @transition(#{...update, event: "発行", result: "生成したトークンを表示する", toState: "success"}, ApiTokenPages.createAdmin) @uses(IdMagic.ApiTokens.Operations.IssueApiToken) issue: unknown;
    @transition(#{event: "キャンセルまたは一覧へ"}, ApiTokenPages.admin) back: unknown;
  }>;
  @page(#{...accountContext, id: "account-api-tokens", name: "本人の API トークン", entry: "本人のポータル", purpose: "本人が発行したトークンと権限を確認する", content: #["説明、スコープ、有効期限と失効状態", "発行の入口と失効の確認"]}) @category(Category.apiTokens)
  personal: PageData<{
    @transition(#{...update, event: "再取得", result: "本人のトークンを更新する"}, ApiTokenPages.personal) @uses(IdMagic.ApiTokens.Operations.ListPersonalApiTokens) refresh: unknown;
    @transition(#{event: "発行"}, ApiTokenPages.createPersonal) create: unknown;
    @transition(#{...update, event: "失効を確認", result: "対象を失効として表示する"}, ApiTokenPages.personal) @uses(IdMagic.ApiTokens.Operations.RevokePersonalApiToken) revoke: unknown;
  }, ListStates>;
  @page(#{...accountContext, id: "account-api-token-create", name: "本人の API トークンを発行", entry: "本人の API トークン一覧", purpose: "許可された権限の範囲でトークンを発行する", content: #["説明、スコープと有効日数", "必要な再認証", "発行したトークンの一度だけの表示とコピー"]}) @category(Category.apiTokens)
  createPersonal: PageData<{
    @transition(#{...update, event: "発行", condition: "必要な再認証と発行が成功", result: "生成したトークンを表示する", toState: "success"}, ApiTokenPages.createPersonal) @uses(IdMagic.ApiTokens.Operations.IssuePersonalApiToken) issue: unknown;
    @transition(#{event: "キャンセルまたは一覧へ"}, ApiTokenPages.personal) back: unknown;
  }>;
}
