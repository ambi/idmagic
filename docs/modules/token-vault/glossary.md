# TokenVault の用語集

| 用語 | 定義 | 別名 |
|---|---|---|
| Connector | 管理者が定義する外部サービスの設定。認可とトークンのエンドポイント、クライアントの資格情報、許可するスコープ、実行の規則を持つ。 | コネクタ |
| Connection | 一人の User と一つの Connector の組に対して最大 1 件だけ存在する、上流のトークン一式と付与スコープの記録。 | 接続 |
| Execution Rule | Connector が代理実行を許す、HTTP メソッド、オリジン、パスの接頭辞と、接続に求めるスコープの組。 | 実行の規則 |
| Delegated Token | Token Exchange で発行され、サブジェクトが User で、`act` を持ち、`agent_id` か `agent_chain` に Agent を持つ IdMagic のアクセストークン。 | 代行トークン |
