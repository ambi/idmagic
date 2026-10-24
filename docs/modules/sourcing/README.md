# Sourcing

## 責務と境界

外部の権威ある取り込み元から IdMagic へアイデンティティを取り込む。
情報の正は外部にあり、IdMagic の内部のプリンシパルはその写しである。
取り込み元との関連付け、外部の不変の ID との相関、取り込みの処理とカーソル、外部の状態に従う削除と無効化の規則を定め、取り込み元ごとに機能を設ける。

このモジュールに入るかどうかは、通信の方向や実行時の形ではなく、永続的な関連付けを持つ外部の権威があるかどうかで決まる。

| 扱わないもの | 担当 |
| --- | --- |
| 管理者による CSV のインポート | `IdManagement` |
| ログインの時点のフェデレーション | `Authentication` |
| 下流のシステムとの台帳の照合 | `Application`、`Provisioning` |
| User と Group の記録の正 | `IdManagement` |
| SCIM の API アクセストークンの発行とスコープの語彙 | `ApiTokens` |

## 主な流れ

| 流れ | 契機 | 実行する場所 | 詳細 |
| --- | --- | --- | --- |
| SCIM の要求 | 外部の IdP の `/realms/{realm_id}/scim/v2/...` への要求 | `api` が、トークンを認証してからユースケースを同期的に呼ぶ | [SCIM による取り込みの設計](features/scim.md#実現方式) |
| Directory Connector の管理 | 管理者の `/api/admin/v1/directory-connectors/...` への要求 | `api` が、管理者を認可してからユースケースを同期的に呼ぶ | [Directory Connector による取り込み](features/directory.md#操作) |
| Directory Connector の呼び出し | 閉域の Connector の `/realms/{realm_id}/directory-connector/v1/...` への要求 | `api` が、Connector の JWT を認証してからユースケースを同期的に呼ぶ | [Directory Connector による取り込み](features/directory.md#モデル) |

## 設計の要点

| 方針 | 理由 |
| --- | --- |
| 取り込み元ごとに機能を設け、共通の機構は実在する共通点が判明してから作る | 一つの取り込み元から推測した抽象を、ほかの取り込み元に押し付けない |
| 外部の表現を正規の Aggregate へ投影し、ワイヤ表現を保存しない | 正規のユーザー像を一つに保つ |
| 外部の削除を論理削除へ統合する | 外部の同期の誤りで個人識別情報を失わない。詳細は[判断](decisions.md#scim-の-user-の削除を論理削除へ統合する) |
| User と Group の由来の表示には IdManagement の照会ポートを実装する | `source_idmanagement` が SCIM の対応の有無だけを返し、表示側へ Repository 全体と保存型を渡さない |
| User と Group の書き込みは `IdentityWrites` から所有者の操作へ渡す | 管理操作と使用量、イベント、ワークフロー捕捉と通知を共有し、作成時の対応表を同じトランザクションで確定する |

## 機能一覧

| 機能 | 内容 |
| --- | --- |
| [SCIM による取り込み](features/scim.md) | 外部の IdP からの SCIM 2.0 による User と Group の同期 |
| [Directory Connector による取り込み](features/directory.md) | 閉域の AD と LDAP を取り込み元とする Directory Connector の登録、その呼び出しの認証、差分のバッチによる User の取り込み |

## 文書

| 設計領域 | 文書 | 内容 |
| --- | --- | --- |
| アプリケーション設計 | [標準仕様](standards.md#system-for-cross-domain-identity-management-core-schema) | SCIM のエンドポイントの形は標準仕様と TypeSpec が定める |
| データ設計 | [Sourcing のデータ](data.md#集約) | Aggregate とその関係。テーブルの定義はスキーマファイルが扱う |
| セキュリティ設計 | [SCIM による取り込み](features/scim.md#セキュリティ上の考慮) | トークンによるテナントの束縛はSCIM による取り込みのセキュリティ上の考慮の節が扱う |
| 信頼性設計 | [SCIM による取り込みの設計](features/scim.md#信頼性) | 作成と対応表の確定、既存の所属変更の補償を扱う |
| 検証設計 | [Sourcing の検証](verification.md#パースの境界のファズテスト) | パースの境界のファズテスト |

| 文書 | 内容 |
| --- | --- |
| [Sourcing の用語集](glossary.md) | このモジュールでの語義 |
| [Sourcing の標準仕様](standards.md) | 採用する外部標準仕様 |
| [設計判断](decisions.md) | 代替案を比べた重要な判断 |
| [リスク](risks.md) | 既知の欠陥、技術的負債、受容したリスク |
