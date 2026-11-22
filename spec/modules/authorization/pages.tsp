namespace IdMagicUi;

model AuthorizationPages {
  @page(#{...adminContext, id: "admin-agent-access", route: "/admin/authorization/access", name: "Agent のアクセス割り当て", entry: "管理メニュー、Agent のリソース", purpose: "Agent とリソースサーバーの関係を管理する", content: #["Agent とサーバーの絞り込み", "リソースと関係の割り当て", "解除の確認", "認可モデル、関係タプルと試行の入口"]}) @category(Category.authorization)
  access: PageData<{
    @transition(#{event: "追加"}, AuthorizationPages.createAccess) create: unknown;
    @transition(#{...update, event: "解除を確認", result: "関係を削除して一覧を再取得する"}, AuthorizationPages.access) @uses(IdMagic.Authorization.Operations.WriteRelationTuples) delete: unknown;
    @transition(#{...update, event: "対象を絞り込む", result: "選択した Agent とサーバーの関係を表示する"}, AuthorizationPages.access) filter: unknown;
    @transition(#{event: "認可モデル"}, AuthorizationPages.authorizationModel) authorizationModel: unknown;
    @transition(#{event: "関係タプル"}, AuthorizationPages.tuples) tuples: unknown;
    @transition(#{event: "アクセスを試す"}, AuthorizationPages.check) check: unknown;
  }, ListStates>;
  @page(#{...adminContext, id: "admin-agent-access-create", name: "アクセス割り当てを追加", entry: "アクセス割り当ての一覧", purpose: "Agent とリソースの関係を登録する", content: #["Agent、サーバー、リソースと関係の選択", "成功後の一覧と追加通知"]}) @category(Category.authorization)
  createAccess: PageData<{
    @transition(#{event: "追加", condition: "保存が成功", result: "割り当て一覧へ戻る"}, AuthorizationPages.access) @uses(IdMagic.Authorization.Operations.WriteRelationTuples) save: unknown;
    @transition(#{event: "キャンセル"}, AuthorizationPages.access) cancel: unknown;
  }>;
  @page(#{...adminContext, id: "admin-authorization-model", name: "認可モデル", entry: "アクセス管理", purpose: "公開中または指定バージョンの型と関係を確認する", content: #["リソース型、関係と書き換え規則", "バージョンの選択と再取得", "モデルがない場合の作成案内"]}) @category(Category.authorization)
  authorizationModel: PageData<{
    @transition(#{...update, event: "バージョンを選択または再取得", result: "指定した認可モデルを表示する"}, AuthorizationPages.authorizationModel) @uses(IdMagic.Authorization.Operations.GetAuthorizationModel) refresh: unknown;
    @transition(#{event: "作成", condition: "モデルがない"}, AuthorizationPages.createModel) create: unknown;
    @transition(#{event: "編集"}, AuthorizationPages.editModel) edit: unknown;
    @transition(#{event: "アクセス管理へ"}, AuthorizationPages.access) back: unknown;
  }, ReadStates>;
  @page(#{...adminContext, id: "admin-authorization-model-create", name: "認可モデルを作成", entry: "モデルがないときの作成案内", purpose: "最初の認可モデルを確認して公開する", content: #["フォームと JSON の編集", "文書閲覧のテンプレート", "公開前の差分確認と競合時の再確認"]}) @category(Category.authorization)
  createModel: PageData<{
    @transition(#{...update, event: "公開前に確認", result: "現行のバージョンと変更差分を表示する", toState: "review"}, AuthorizationPages.createModel) review: unknown;
    @transition(#{event: "公開", from: "review", condition: "競合なく公開が成功", result: "公開したモデルへ進む"}, AuthorizationPages.authorizationModel) @uses(IdMagic.Authorization.Operations.PutAuthorizationModel) publish: unknown;
    @transition(#{...update, event: "編集へ戻る", from: "review", result: "編集内容を保持して確認を閉じる"}, AuthorizationPages.createModel) edit: unknown;
    @transition(#{event: "キャンセル"}, AuthorizationPages.authorizationModel) cancel: unknown;
  }, { ...FormStates; review: "公開する差分と現行バージョンを確認する。"; }>;
  @page(#{...adminContext, id: "admin-authorization-model-edit", name: "認可モデルを編集", entry: "認可モデル", purpose: "型と関係の変更を差分で確認して公開する", content: #["フォームと JSON の切り替え", "現行との差分、公開の確認とバージョン競合"]}) @category(Category.authorization)
  editModel: PageData<{
    @transition(#{...update, event: "公開前に確認", result: "現行モデルと差分を表示する", toState: "review"}, AuthorizationPages.editModel) review: unknown;
    @transition(#{event: "公開", from: "review", condition: "バージョンが一致し公開が成功", result: "更新したモデルへ戻る"}, AuthorizationPages.authorizationModel) @uses(IdMagic.Authorization.Operations.PutAuthorizationModel) publish: unknown;
    @transition(#{...update, event: "編集へ戻る", from: "review", result: "入力を保持して確認を閉じる"}, AuthorizationPages.editModel) edit: unknown;
    @transition(#{event: "キャンセル"}, AuthorizationPages.authorizationModel) cancel: unknown;
  }, { ...FormStates; review: "公開する差分と競合の有無を確認する。"; }>;
  @page(#{...adminContext, id: "admin-authorization-tuples", name: "関係タプル", entry: "アクセス管理", purpose: "主体とオブジェクトの関係を検索して削除する", content: #["主体、型、識別子と関係の絞り込み", "追加の入口", "一件または対象オブジェクトの関係を削除する確認"]}) @category(Category.authorization)
  tuples: PageData<{
    @transition(#{...update, event: "検索または再取得", result: "条件に合う関係を表示する"}, AuthorizationPages.tuples) @uses(IdMagic.Authorization.Operations.ListRelationTuples) filter: unknown;
    @transition(#{event: "追加"}, AuthorizationPages.createTuple) create: unknown;
    @transition(#{...update, event: "削除を確認", result: "対象の関係を削除して再取得する"}, AuthorizationPages.tuples) @uses(IdMagic.Authorization.Operations.WriteRelationTuples) delete: unknown;
    @transition(#{event: "アクセス管理へ"}, AuthorizationPages.access) back: unknown;
  }, ListStates>;
  @page(#{...adminContext, id: "admin-authorization-tuple-create", name: "関係タプルを追加", entry: "関係タプルの一覧", purpose: "主体、関係とオブジェクトを指定して登録する", content: #["認可モデルに応じた型と関係の選択", "主体と対象の識別子"]}) @category(Category.authorization)
  createTuple: PageData<{
    @transition(#{event: "追加", condition: "登録が成功", result: "関係タプルの一覧へ戻る"}, AuthorizationPages.tuples) @uses(IdMagic.Authorization.Operations.WriteRelationTuples) save: unknown;
    @transition(#{event: "キャンセル"}, AuthorizationPages.tuples) cancel: unknown;
  }>;
  @page(#{...adminContext, id: "admin-authorization-check", name: "アクセス判定の試行", entry: "アクセス管理", purpose: "指定した主体と委任の文脈でアクセスを試す", content: #["主体、関係、対象と委任の連鎖", "モデルのバージョンと一貫性", "許可判定とアクセス可能なリソースの列挙"]}) @category(Category.authorization)
  check: PageData<{
    @transition(#{...update, event: "アクセスを判定", result: "許可または拒否の結果を表示する"}, AuthorizationPages.check) @uses(IdMagic.Authorization.Operations.CheckAccess) check: unknown;
    @transition(#{...update, event: "リソースを列挙", result: "アクセス可能な対象を表示する"}, AuthorizationPages.check) @uses(IdMagic.Authorization.Operations.ListAccessibleResources) list: unknown;
    @transition(#{event: "アクセス管理へ"}, AuthorizationPages.access) back: unknown;
  }>;
}
