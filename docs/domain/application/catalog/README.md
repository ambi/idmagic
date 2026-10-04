# アプリケーションのカタログ

## 概要

この文書は、管理者が Application とそのプロトコル設定を作成、参照、更新する仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | Application とプロトコル設定の作成、IdMagic の側の連携の設定の提示、クライアントシークレットの管理、SAML のプロトコル設定の更新、公開するクレームの制限、アイコンの管理 |
| 行為者 | テナント管理者、管理 API のクライアント |
| 扱わないもの | 割り当ては[割り当て](../assignment/README.md)が、サインインポリシーは[サインインポリシー](../sign-in-policy/README.md)が扱う |

## モデル

Application に関連付けるプロトコル設定は最大一つとし、作成時に固定する。
`weblink` のアプリケーションにはプロトコル設定を関連付けず、`federated` と `service` のアプリケーションには、OAuth2 のクライアント、SAML の SP、WS-Federation の RP のいずれか一つだけを関連付ける。
作成の後の再接続、切り離し、プロトコルの種別の変更には対応しない。

プロトコル設定のうち `application_id` を持たないものは、Dynamic Client Registration や信頼の管理 API で作成され、カタログには表示しない正当なレコードである。
すべてのプロトコル設定に Application を必須とはしない。

Application を削除すると、それに紐づくプロトコル設定も連鎖して削除する。
Application に属するプロトコル設定を、各プロトコルの管理 API から直接削除しようとした場合は、競合として拒否する。

- **判断**：一対一の関係をデータベースで保証する理由は、[Application とプロトコル設定の関係を複合外部キーで保証する](../design/decisions.md#application-とプロトコル設定の関係を複合外部キーで保証する)。

## 操作

### 管理者によるアプリケーションの作成

#### REQ-APPLICATION-007 管理者は管理画面でアプリケーションと 1 つのプロトコルを構成できる

#### REQ-APPLICATION-013 admin ロールを持たない利用者は Application を操作できない

### 管理者によるアプリケーションの参照

#### REQ-APPLICATION-001 管理者はアプリケーション詳細で IdMagic 側の連携設定を確認できる

### 管理者によるクライアントシークレットの管理

#### REQ-APPLICATION-002 管理者は通常設定とは独立したセクションでクライアントシークレットを管理できる

### 管理者によるプロトコル設定の更新

#### REQ-APPLICATION-005 管理者は Application の SAML プロトコル設定を更新できる

#### REQ-APPLICATION-006 管理者は Application ごとに公開するクレームを制限できる

### 管理者によるアイコンの管理

#### REQ-APPLICATION-008 管理者は Application のアイコンをアップロード・削除できる

### 管理 API のクライアントによる操作

#### REQ-APPLICATION-004 管理 API クライアントは Application スコープで許可された操作だけを実行できる

## セキュリティ上の考慮

Application、プロトコル設定、カテゴリ、割り当て、サインインポリシーの管理は、いずれも `admin` ロールを持つ、有効かつ認証済みのユーザーに限る。
AuthZEN の action は対象ごとに分かれ、`admin:applications_manage`、`admin:application_assignments_manage`、`admin:application_policies_manage`、`admin:application_categories_manage`、`admin:tenant_default_sign_in_policy_manage` を要求する。
いずれも、操作の対象が呼び出し元と同じテナントに属することを条件とする。
