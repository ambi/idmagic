# OAuth2

## 責務と境界

OAuth 2.0 と OIDC のプロトコル群の全責務を扱う。
クライアントのメタデータと Dynamic Client Registration、認可の判断（認可、同意、認可コード、PAR、Device Authorization、RP-Initiated Logout）、トークンの発行とライフサイクル（アクセストークン、リフレッシュトークン、ID トークン、イントロスペクション、失効、UserInfo、Proof of Possession）、Discovery Metadata、Authorization Server Metadata、健全性の報告をこのモジュールに集約する。

| 扱わないもの | 担当 |
| --- | --- |
| トークンに載せるクレームの決定 | `ClaimMapping` |
| 署名鍵のライフサイクル | `SigningKeys` |
| 利用者の認証とセッションの保持 | `Authentication` |
| その `client_id` にそもそも到達してよいかという関門 | `Application` |
| Agent のアイデンティティとライフサイクル | `IdManagement` |
| Agent の失効エポック | `SharedSignals` |

このモジュールが受け持つのは、それらの結果をプロトコルの語彙で組み立てて外部へ返す部分である。
ApplicationCatalog からのクライアント管理にも公開操作を提供し、保存、関連付け、クレームとシークレットの更新をこのモジュールが担当する。
IdManagement からの完全削除では保存先ごとの本人の記録の削除、Agent の束縛では同じテナントのクライアントの存在照会だけを提供する。
保存先が要求側の小さな契約を直接満たし、本人データへ含める同意の表示値は組み立て地点で変換する。

## 主な流れ

| 流れ | 契機 | 実行する場所 | 詳細 |
| --- | --- | --- | --- |
| 認可 | ブラウザーの `/authorize` | `api` が、セッション、割り当て、ポリシー、同意を確かめて認可コードを発行する | [認可の設計](features/authorization.md#実現方式) |
| トークン | クライアントの `/token`、`/introspect`、`/revoke`、`/userinfo` | `api` が、クライアントを認証してから同期的に処理する | [トークンの設計](features/token.md#実現方式) |
| 承認 | クライアントのバックチャネル認可要求と、本人の承認の画面 | `api` が、判断とポーリングをストアで直列化する | [承認リクエストの設計](features/approval.md#実現方式) |
| バックチャネルのログアウト | セッションの失効 | `worker` が配信のジョブを実行する | [ログアウトの設計](features/logout.md#実現方式) |

## 設計の要点

| 方針 | 理由 |
| --- | --- |
| 曖昧さがなく列挙できる、プロトコルの上で重要な振る舞いは、仕様で一度だけ宣言し、ユースケースやアダプターで作り直さない | 状態遷移、認可の規則、Discovery Metadata、デバイスフローの遷移が、実装のたびにずれない。詳細は[判断](decisions.md#認可とデバイスのライフサイクルを宣言的な遷移表で表す) |
| 認可の判断を AuthZEN 型のポートに通す | 詳細は[判断](decisions.md#ポリシーの境界をまたぐ認可の判断を-authzen-型のポートに通す) |
| 委譲は権限を広げない | Token Exchange は元の権限の部分集合だけを渡す。詳細は[判断](decisions.md#ユーザーの代理の行為を-token-exchange-による委任で表す) |
| 委譲モードのワイヤ語彙は仕様バインディングを参照する | 公開 `domain` の型名と定数名を保ち、`backend/shared/spec` の列挙を使う。主体と委譲チェーンから値を導く規則は OAuth2 が所有する |
| モジュールの直下の `domain`、`ports`、`usecases` を、機能スライスの上に置く互換のファサードにする | 機能スライスへ分けた後も、ほかのモジュールが使う公開パッケージのパスを変えない。起動の組み立てはルートパッケージだけが行う |

## 機能一覧

| 機能 | 内容 |
| --- | --- |
| [クライアント](features/client.md) | 登録、メタデータの解決、クライアントの認証、クライアントシークレット |
| [認可](features/authorization.md) | 認可コードフロー、PKCE、PAR |
| [同意](features/consent.md) | 同意の付与、参照、撤回 |
| [デバイス認可](features/device.md) | Device Authorization Grant |
| [トークン](features/token.md) | 発行、ローテーション、イントロスペクション、失効、UserInfo、送信者制約、委譲 |
| [承認リクエスト](features/approval.md) | Agent の操作に対する人間の承認（CIBA） |
| [ログアウト](features/logout.md) | RP-Initiated Logout とバックチャネルのログアウト |
| [プロトコルのエンドポイント](features/protocol-endpoints.md) | Discovery Metadata と流量の制限 |
| [管理 API の認可](features/admin-access.md) | 管理 API のスコープとロールポリシー |

プロトコルのエンドポイントと管理 API の認可は機能スライスを持たない。前者はルートの `handlers_http` が Discovery と流量の制限を、後者はルートの AuthZEN の規則表と `policy_tenancy` が扱う。

## 文書

このモジュールが担当する SLO とキャパシティ要件は、[品質要件の割り当て](../../design/architecture/logical.md#品質要件の割り当て)から担当機能、設計、検証へ参照する。

| 設計領域 | 文書 | 内容 |
| --- | --- | --- |
| アプリケーション設計 | [標準仕様](standards.md#the-oauth-20-authorization-framework) | プロトコルの形は標準仕様と TypeSpec が、管理 API はシステムのアプリケーション設計が扱う |
| データ設計 | [OAuth2 のデータ](data.md#集約)、[トークンの設計](features/token.md#データ) | Aggregate とその関係。トークンの有効期間と保持期間はトークンの設計が、テーブルの定義はスキーマファイルが扱う |
| 信頼性設計 | [トークン](features/token.md#概要)、[ログアウトの設計](features/logout.md#アーキテクチャ) | 署名の提供元の障害はトークンが、ログアウトの配信の再試行はログアウトの設計が扱う |
| 検証設計 | [OAuth2 の検証](verification.md#パースの境界のファズテスト) | パースの境界のファズテスト |

| 文書 | 内容 |
| --- | --- |
| [OAuth2 の用語集](glossary.md) | このモジュールでの語義 |
| [OAuth2 の標準仕様](standards.md) | 採用する外部標準仕様 |
| [設計判断](decisions.md) | 代替案を比べた重要な判断 |
