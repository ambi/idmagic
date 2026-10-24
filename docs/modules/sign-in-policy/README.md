# SignInPolicy

## 責務と境界

サインイン条件の定義、検証、実効規則の選択、評価と保存を扱う。
アプリごとの上書きとテナントのデフォルト、MFA、信頼済みデバイス、再認証の期限、許可するネットワークの意味をこのモジュールに閉じる。
アプリのメタデータを変更することなく、アカウント画面と各プロトコルが同じ条件の定義と評価を利用できるようにする。

| 扱わないもの | 担当 |
| --- | --- |
| アプリの実在、有効性、割り当てとフェデレーションの関門 | [Application](../application/README.md) |
| 認証方式の実行、認証事実の解釈と追加認証 | [Authentication](../authentication/README.md) |
| 管理画面の HTTP と表示用の変換 | [ApplicationCatalog](../application-catalog/README.md) |
| 資源への権限と関係に基づく認可 | [AccessControl](../authorization/README.md) |
| 同意、グラント、トークンと Assertion の発行 | 各プロトコル |

既存の公開 API、要件 ID と TypeSpec の識別子を保つため、外向きの契約は Application の[サインインポリシー](../application/features/sign-in-policy.md)を一次情報とする。
要求と応答、保存する JSON、条件の意味と更新イベントの名前は変えず、定義と実装の所有者を分ける。

## 主な流れ

| 流れ | 契機 | 実行する場所 | 詳細 |
| --- | --- | --- | --- |
| 設定の取得と更新 | 管理者がアプリごとの条件かテナントのデフォルトを変更する | ApplicationCatalog が公開ユースケースを呼び、SignInPolicy が検証、保存とイベントを担当する | [設定の仕様](../application/features/sign-in-policy.md#操作) |
| フェデレーションの評価 | OIDC、SAML、WS-Fed が発行を始める | Application の関門が割り当てを確認し、SignInPolicy の規則の選択と評価を使う | [サインインポリシーの設計](../application/features/sign-in-policy.md#実現方式) |
| IdMagic 自身の画面へのサインイン | 管理コンソールかアカウント画面へのサインイン | Authentication がデフォルトの条件と同じ評価器を使い、必要な追加認証を実行する | [IdMagic 自身の画面での評価](../application/features/sign-in-policy.md#idmagic-自身の画面へのサインインでの評価) |
| アプリの削除 | Application がアプリを削除する | Application がポリシーの削除だけの契約を呼ぶ | [保存と削除](data.md#保存と削除) |

## 設計の要点

評価器は読み取り専用の `SignInAuthentication` と時刻を入力にし、Authentication の型を import しない。
管理操作は、要求側の `ApplicationLookup` で同じテナントのアプリの実在だけを問い合わせる。
Application のルートが自分の保存先から存在の事実を返し、組み立て地点が渡すため、SignInPolicy から Application への依存はない。
Application は削除のために保存型を受け取らず、必要な削除操作だけを要求する。

## 機能一覧

| 機能 | 内容 |
| --- | --- |
| [サインインポリシー](../application/features/sign-in-policy.md) | 条件の定義、検証、デフォルトと上書きの選択、評価と更新。コードは機能スライスへ分けず、モジュール直下の公開層と非公開の保存アダプターに置く |

## 文書

| 設計領域 | 文書 | 内容 |
| --- | --- | --- |
| データ設計 | [SignInPolicy のデータ](data.md#集約) | 条件の所有、Application への参照と削除の境界 |

| 文書 | 内容 |
| --- | --- |
| [SignInPolicy の用語集](glossary.md) | 条件と評価が使う語彙 |
