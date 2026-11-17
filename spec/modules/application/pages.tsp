namespace IdMagicUi;

model PortalApplicationPages {
  @page(#{...accountContext, id: "account-apps", name: "本人のアプリケーションランチャー", route: "/account/apps", path: "/account/apps", entry: "本人のポータル", purpose: "利用可能なアプリケーションを起動し表示順を変更する", content: #["カテゴリ別のアプリケーションカード", "起動の入口", "ドラッグ操作またはキーボードによる並べ替え", "保存中と保存失敗"]}) @category(Category.applications)
  apps: PageData<{
    @form(#{name: "表示順", condition: "アプリケーションが存在する場合", commit: "並べ替え完了時に保存"}) order: {
      @field(#{label: "アプリケーションの順序", control: "ドラッグ・キーボードで並べ替え", initial: "本人の表示順", condition: "並べ替え可能なカード", mapping: "アプリケーション ID の順序付き配列", repeated: true}) applications: string[];
    };
  }, {
    @transition(#{...update, event: "表示順を変更", condition: "並べ替えが確定", result: "並べ替えた順序を保存する"}, PortalApplicationPages.apps) @uses(IdMagic.Application.Operations.ReorderMyApplications) reorder: unknown;
    @transition(#{...navigation, event: "アプリケーションを選択", condition: "launch_url がありドラッグ操作の直後でない", result: "別タブでアプリケーションを開く", toState: "external"}, Destinations.client) launch: unknown;
  }>;
}
