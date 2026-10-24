# ApplicationCatalog

## 責務と境界

アプリケーションの管理とポータルの HTTP を提供し、Application と各プロトコルの公開操作を編成する。
認可、入力と表示のワイヤ変換、プロトコル設定と Application の関連付けを扱う。
独自のテーブル、保存型とアクセス評価器は持たない。

| 扱わないもの | 担当 |
| --- | --- |
| Application、割り当て、分類と並び順の保存と更新 | [Application](../application/README.md) |
| サインインポリシーとアクセス判定 | [Application](../application/README.md) |
| クライアント、SP と RP の保存とプロトコルの実行 | OAuth2、Saml、WsFederation |
| クレームの公開の下限 | ClaimMapping |

## 主な流れ

管理 API はテナント、認証、認可とブラウザーの要求を検証してから、Application の公開ユースケースと各プロトコルの操作を呼ぶ。
ポータルの一覧、カテゴリと並び順は同じ Application の状態を読み、アイコンの配信もここで行う。
フェデレーションの開始は Catalog を通らず、各プロトコルから Application の割り当てとポリシーの関門を呼ぶ。

## 設計の要点

Application の `usecases` は状態所有者の既存の更新操作をそのまま公開し、Catalog 用に同じ入力型と委譲関数を複製しない。
同一トランザクションの関連付けは Application の保存実装と各プロトコルの linker が担当し、Catalog の分離で確定の境界を増やさない。
OIDC 管理では OAuth2 の公開管理操作とその入力、結果とエラーを直接利用する。
管理用の表示は HTTP の非公開の型へ写し、所有者の保存型をそのままシリアライズしない。
資格情報の状態は OAuth2 の判定を使い、表示へ渡すのは識別子、作成時刻、期限、失効時刻と状態だけである。
SAML と WS-Fed の設定も、Saml の `ServiceProviderAdministration` と WsFederation の `RelyingPartyAdministration` で作成、更新、関連付け、削除する。
デフォルト値、署名の設定と値の検査は各所有者が保ち、Catalog は所有者の保存型を組み立てず、Repository と直接の SQL を持たない。

## 機能一覧

外部へ提供する契約と状態の保証は、Application の機能仕様が定める。
ここでは同じ規範を複製しない。

| 機能 | 内容 |
| --- | --- |
| [カタログ](../application/features/catalog.md) | アプリケーションとプロトコル設定の管理、シークレットとアイコン |
| [割り当て](../application/features/assignment.md) | 管理 API による割り当ての操作 |
| [サインインポリシー](../application/features/sign-in-policy.md) | 個別とテナントのデフォルトのポリシーの管理 |
| [ポータル](../application/features/portal.md) | 割り当て済みアプリケーションの一覧と並び順 |

## 文書

状態と原子性は [Application のデータ設計](../application/data.md)、依存と公開範囲は[バックエンド設計](../../design/application/backend.md#モジュール間の依存規則)を参照する。
