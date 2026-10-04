# プロトコルのエンドポイント

## 概要

この文書は、Discovery Metadata と Authorization Server Metadata の公開と、プロトコルのエンドポイントの流量の制限の仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | 宣言したエンドポイントの広告、RFC 8414 のメタデータ、realm の接頭辞付きの発行者、閾値を超えた要求の拒否 |
| 行為者 | 登録済みのクライアント |
| 扱わないもの | 各エンドポイントの処理は、それぞれの機能が扱う |

## モデル

Discovery Metadata は、手作業でもビルドの時でもなく、実行時に契約から組み立てる。

- **判断**：実行時に組み立てる理由は、[Discovery Metadata を実行時に契約から組み立てる](../design/decisions.md#discovery-metadata-を実行時に契約から組み立てる)。

## 操作

### クライアントによるメタデータの取得

#### REQ-OAUTH2-014 Discovery Metadata は宣言された全エンドポイントを広告する

#### REQ-OAUTH2-030 RFC 8414 メタデータ文書は OIDC Discovery と同等の内容を返す

#### REQ-OAUTH2-033 realm 接頭辞付きの Discovery Metadata は同じ接頭辞を持つ発行者を返す

### クライアントによるプロトコルのエンドポイントの呼び出し

#### REQ-OAUTH2-040 プロトコルエンドポイントは閾値を超えたリクエストをレート制限で拒否する
