namespace IdMagicUi;

model PortalApplicationPages {
  @page(#{...accountContext, id: "account-apps", name: "本人のアプリケーションランチャー", entry: "本人のポータル", purpose: "利用可能なアプリケーションを起動し表示順を変更する", content: #["カテゴリ別のアプリケーションカード", "起動の入口", "ドラッグ操作またはキーボードによる並べ替え", "保存中と保存失敗"]}) @category(Category.applications)
  apps: PageData<{
    @transition(#{...update, event: "表示順を変更", condition: "並べ替えが確定", result: "並べ替えた順序を保存する"}, PortalApplicationPages.apps) @uses(IdMagic.Application.Operations.ReorderMyApplications) reorder: unknown;
    @transition(#{...navigation, event: "アプリケーションを選択", condition: "launch_url がありドラッグ操作の直後でない", result: "別タブでアプリケーションを開く", toState: "external"}, Destinations.client) launch: unknown;
  }>;
}
