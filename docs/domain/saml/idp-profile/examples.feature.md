# Feature: IdP プロファイルとメタデータの例

## Rule: REQ-SAML-001 SP は署名証明書を取得できる

### Example: EX-SAML-001-01 通常経路

- Given SP が IdMagic テナントの SAML メタデータまたは証明書ダウンロード URL を参照できる
- When SP が証明書ダウンロード URL にリクエストを送る
- Then SP は現在有効な `XmlFederationSigning` 証明書を PEM 形式で取得する
- Then 取得した証明書は同じ時点の SAML メタデータで公開される証明書と一致する
- Then ローテーションの移行期間中に信頼するすべての証明書は SAML メタデータから取得する

### Example: EX-SAML-001-02 フェデレーション署名資格情報を利用できない

- Given SP が IdMagic テナントの SAML メタデータまたは証明書ダウンロード URL を参照できる
- When SP が証明書ダウンロード URL にリクエストを送る
- Then フェデレーション署名資格情報を利用できない
- Then 証明書を返さずにエラーを返す

## Rule: REQ-SAML-003 専用プロファイルは固有のメタデータを公開する

### Example: EX-SAML-003-01 通常経路

- Given テナントに `default` プロファイルと `dedicated` プロファイルが存在する
- When `dedicated` プロファイルのメタデータ URL を取得する
- Then メタデータでプロファイル固有の entityID、SSO / SLO URL、署名証明書を公開する
- Then `default` プロファイルのメタデータでは異なる署名資格情報を公開する

### Example: EX-SAML-003-02 存在しないプロファイルまたは別テナントのプロファイル ID を指定する

- Given テナントに `default` プロファイルと `dedicated` プロファイルが存在する
- When `dedicated` プロファイルのメタデータ URL を取得する
- But 存在しないプロファイルまたは別テナントのプロファイル ID を指定する
- Then メタデータや証明書を公開せず、not found を返す

## Rule: REQ-SAML-004 管理者は SAML IdP プロファイルを共有用または専用として管理できる

### Example: EX-SAML-004-01 通常経路

- Given テナントには変更できない `default` の `shared` プロファイルが存在する
- When 管理者が読み取り専用の連携エンドポイント画面から、プロファイルの管理一覧と詳細画面へ移動する
- Then 専用プロファイルの一覧と詳細が表示される
- When 管理者がプロファイル作成画面で `shared` プロファイルを作成する
- Then 複数の SP からそのプロファイルを選択できる
- When 管理者がプロファイル詳細から編集画面へ移り、追加プロファイルの名前またはモードを変更する
- Then 変更が保存される
- When 管理者が `dedicated` プロファイルを作成して 1 つの SP に割り当てる
- Then `dedicated` プロファイルと SP の関連付けが保存される
- When 管理者が未使用の追加プロファイルを削除する
- Then プロファイルが削除される

### Example: EX-SAML-004-02 `dedicated` プロファイルを別の SP にも割り当てる

- Given テナントには変更できない `default` の `shared` プロファイルが存在する
- When 管理者が読み取り専用の連携エンドポイント画面から、プロファイルの管理一覧と詳細画面へ移動する
- Then 専用プロファイルの一覧と詳細が表示される
- When 管理者がプロファイル作成画面で `shared` プロファイルを作成する
- Then 複数の SP からそのプロファイルを選択できる
- When 管理者がプロファイル詳細から編集画面へ移り、追加プロファイルの名前またはモードを変更する
- Then 変更が保存される
- When 管理者が `dedicated` プロファイルを作成して 1 つの SP に割り当てる
- But `dedicated` プロファイルを別の SP にも割り当てる
- Then 関連付けを InvalidRequestError で拒否する

### Example: EX-SAML-004-03 プロファイルが SP から参照されている、またはデフォルトプロファイルである

- Given テナントには変更できない `default` の `shared` プロファイルが存在する
- When 管理者が読み取り専用の連携エンドポイント画面から、プロファイルの管理一覧と詳細画面へ移動する
- Then 専用プロファイルの一覧と詳細が表示される
- When 管理者がプロファイル作成画面で `shared` プロファイルを作成する
- Then 複数の SP からそのプロファイルを選択できる
- When 管理者がプロファイル詳細から編集画面へ移り、追加プロファイルの名前またはモードを変更する
- Then 変更が保存される
- When 管理者が `dedicated` プロファイルを作成して 1 つの SP に割り当てる
- Then `dedicated` プロファイルと SP の関連付けが保存される
- When 管理者が未使用の追加プロファイルを削除する
- But プロファイルが SP から参照されている、またはデフォルトプロファイルである
- Then 削除を conflict で拒否する
