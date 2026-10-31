# コネクタの管理

## 概要

この文書は、管理者が外部サービスのコネクタを定義する仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | コネクタの作成、参照、更新、無効化、削除 |
| 行為者 | テナント管理者、管理 API のクライアント |
| 扱わないもの | 本人の連携は[接続](connection.md)が、実行の評価は[代理実行](execution.md)が扱う |

## 操作

### 管理者によるコネクタの作成と更新

#### REQ-TOKENVAULT-001 管理者は検証を通ったコネクタを管理でき、クライアントシークレットは応答に現れない

- 管理者が有効な定義でコネクタを作成したとき、TokenVault は、`active` のコネクタを作り、201 とコネクタを返し、`TokenVaultConnectorCreated` を発行する。
- 管理者がコネクタを参照、一覧したとき、TokenVault は、自テナントのコネクタを 200 で返す。
- 管理者がコネクタを更新したとき、TokenVault は、200 とコネクタを返し、`TokenVaultConnectorUpdated` を発行する。
- コネクタを返すとき、TokenVault は、`client_secret` を含めず、設定の有無だけを `client_secret_configured` で示す。
- `token_endpoint_auth_method` が省略されたとき、TokenVault は、`client_secret_basic` を使う。
- 更新の `client_secret` が省略されたとき、TokenVault は、保存したシークレットを維持する。
- エンドポイントかルールのオリジンが HTTPS でないか、利用者情報、フラグメント、内部のアドレスのリテラルを含む場合、TokenVault は、400 と `invalid_request` で拒否する。
- ルールが 100 件を超えるか、`method` が `GET`、`HEAD`、`POST`、`PUT`、`PATCH`、`DELETE` でないか、`path_prefix` が `/` で始まらないか `..` を含むか、`required_scopes` が `allowed_scopes` の部分集合でない場合、TokenVault は、400 と `invalid_request` で拒否する。
- 同じ名前のコネクタがテナントにある場合、TokenVault は、409 と `connector_conflict` で拒否する。
- 別のテナントか存在しないコネクタを指定された場合、TokenVault は、404 と `connector_not_found` で拒否する。
- `admin` のロールを持たない利用者が要求した場合、TokenVault は、403 と `access_denied` で拒否する。
- `connectors:write` を持たない API アクセストークンで変更を要求された場合、TokenVault は、403 と `insufficient_scope` で拒否する。

### 管理者によるコネクタの無効化と削除

#### REQ-TOKENVAULT-002 無効化は連携と実行を次の要求から止め、削除は接続も消す

- 管理者がコネクタの `status` を `disabled` にしたとき、TokenVault は、接続を保持し、以後の連携の開始と代理実行を拒否する。
- 管理者が `disabled` のコネクタを `active` へ戻したとき、TokenVault は、保持した接続での代理実行を再び許す。
- 管理者がコネクタを削除したとき、TokenVault は、コネクタと接続と進行中の連携を消し、204 を返し、`TokenVaultConnectorDeleted` を発行する。
- **判断**：削除では上流へ失効を試みない。接続の数だけ外部呼び出しが要り、失敗を管理者の操作の結果にできないためである。残留は[リスク](../risks.md)に記す。
