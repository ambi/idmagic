namespace IdMagicUi;

model DirectoryConnectorPages {
  @page(#{...adminContext, id: "admin-directory-connectors", name: "ディレクトリ接続", entry: "管理メニュー", purpose: "ディレクトリとの接続状態を確認する", content: #["接続名、状態と最終接続", "作成、詳細と再取得"]}) @category(Category.directories)
  list: PageData<{
    @transition(#{...update, event: "再取得", result: "接続状態を更新する"}, DirectoryConnectorPages.list) @uses(IdMagic.Sourcing.Operations.ListDirectoryConnectors) refresh: unknown;
    @transition(#{event: "作成"}, DirectoryConnectorPages.create) create: unknown;
    @transition(#{event: "詳細"}, DirectoryConnectorPages.detail) detail: unknown;
  }, ListStates>;
  @page(#{...adminContext, id: "admin-directory-connector-create", name: "ディレクトリ接続を作成", entry: "ディレクトリ接続の一覧", purpose: "接続を登録し、登録トークンを取得する", content: #["接続名の登録", "登録トークンの一度だけの表示", "登録後のセットアップ案内"]}) @category(Category.directories)
  create: PageData<{
    @transition(#{...update, event: "作成", result: "登録トークンを表示する", toState: "success"}, DirectoryConnectorPages.create) @uses(IdMagic.Sourcing.Operations.CreateDirectoryConnector) save: unknown;
    @transition(#{event: "詳細へ", from: "success"}, DirectoryConnectorPages.detail) detail: unknown;
    @transition(#{event: "キャンセルまたは戻る"}, DirectoryConnectorPages.list) cancel: unknown;
  }>;
  @page(#{...adminContext, id: "admin-directory-connector-detail", name: "ディレクトリ接続の詳細", entry: "接続の一覧または登録後", purpose: "接続の登録状態と取り込み結果を確認する", content: #["接続日時、鍵と登録期限", "登録トークン再発行と無効化の確認", "取り込み履歴と展開した失敗の詳細", "委任認証への入口"]}) @category(Category.directories)
  detail: PageData<{
    @transition(#{...update, event: "登録トークンの再発行を確認", result: "新しいトークンを一度だけ表示する"}, DirectoryConnectorPages.detail) @uses(IdMagic.Sourcing.Operations.ReissueDirectoryConnectorEnrollment) reissue: unknown;
    @transition(#{...update, event: "無効化を確認", result: "接続の無効状態を表示する"}, DirectoryConnectorPages.detail) @uses(IdMagic.Sourcing.Operations.DisableDirectoryConnector) disable: unknown;
    @transition(#{event: "委任認証"}, DirectoryConnectorPages.delegation) delegation: unknown;
    @transition(#{event: "戻る"}, DirectoryConnectorPages.list) back: unknown;
  }>;
  @page(#{...adminContext, id: "admin-directory-delegated-authentication", name: "ディレクトリの委任認証", entry: "ディレクトリ接続の詳細", purpose: "認証委任の設定と疎通を確認する", content: #["有効状態、ドメインと失敗時の扱い", "ユーザーとパスワードによる認証テスト", "テスト後にパスワードを消し結果を表示する"]}) @category(Category.directories)
  delegation: PageData<{
    @transition(#{...update, event: "保存", result: "委任認証の設定を更新する"}, DirectoryConnectorPages.delegation) @uses(IdMagic.Sourcing.Operations.ConfigureDirectoryDelegation) save: unknown;
    @transition(#{...update, event: "認証をテスト", result: "認証結果を表示し送信したパスワードを消す"}, DirectoryConnectorPages.delegation) @uses(IdMagic.Sourcing.Operations.TestDirectoryDelegation) test: unknown;
    @transition(#{event: "戻る"}, DirectoryConnectorPages.detail) back: unknown;
  }>;
}
