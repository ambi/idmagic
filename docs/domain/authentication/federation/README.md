# 外部 IdP との連携

## 概要

この文書は、テナント単位の上流の OIDC と SAML の接続によるログインの仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | 上流の接続の管理、外部 subject とローカルの User の関連付け、関連付けと JIT のポリシー、ログインセッションへの引き渡し |
| 行為者 | EndUser、本人（明示的な関連付けと解除）、テナント管理者 |
| 扱わないもの | 下流向けの SAML の IdP と WS-Federation の発行は、各プロトコルの Context が扱う。JIT で作る User の記録は `IdManagement` が扱う |

## モデル

ブローカーは、まず `FederatedIdentity` を通じて不変な外部 subject を解決する。
検証済みのメールアドレスによる関連付けを許すのは、接続に明示的なポリシーがあり、上流のクレームが検証済みで、テナントの中で一意に一致する場合だけである。
JIT は接続ごとに個別に有効にし、メールのドメインの許可リストでさらに絞り込める。
明示的な関連付けと解除にはステップアップ認証を求め、最後に残った利用可能なサインインの手段は取り除けない。

- **判断**：アイデンティティブローカーを Authentication に置く理由は、[ログインの時点のアイデンティティブローカーを Authentication に置く](../design/decisions.md#ログインの時点のアイデンティティブローカーを-authentication-に置く)。

## 状態遷移

### IdentityProviderConnectionLifecycle

上流との接続は、利用できる `Active` と経路を止めた `Disabled` の 2 状態だけを行き来する。作成直後は `Disabled` である。メタデータの再取得の失敗や、信頼の根拠にあたらない項目の更新は状態を変えず、最後に成功した内容を保持する。

| State | Kind | Meaning |
|---|---|---|
| Disabled | initial | 経路を止めている。作成直後はこの状態である |
| Active | — | 上流との接続を利用できる |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Active | IdentityProviderConnectionDisabled | — | Disabled |  |
| Disabled | IdentityProviderConnectionActivated | — | Active |  |

## 操作

### 利用者による外部 IdP のログイン

#### REQ-AUTHENTICATION-001 外部 OIDC 認証は検証済みの subject を常に同じローカル User へ相関する

#### REQ-AUTHENTICATION-002 検証済みメールアドレスによる自動リンクは明示ポリシーと一意な一致を要求する

### 本人による外部アイデンティティの関連付けと解除

#### REQ-AUTHENTICATION-003 外部アイデンティティの明示的なリンクと解除はステップアップ認証を要求する

### 管理者による外部 IdP の接続の管理

#### REQ-AUTHENTICATION-025 外部 IdP 接続の管理は対話セッションに限る

#### REQ-AUTHENTICATION-037 外部 IdP 接続は、利用者との連携が残っている間は削除できない

## セキュリティ上の考慮

上流の IdP から受け取ったものは、すべて信頼しない。
ログインの要求が使うのは保存済みのプロバイダーとエンドポイントの設定だけであり、ブラウザーから任意の Discovery の URL やトークンの URL を指定することはできない。
クライアントシークレットはデータベースに保存せず `secret_reference` だけを持ち、外部のトークンと SAML の Assertion はログインの時点で検証したうえで保持しない。

外部 IdP の接続の管理 API は、API アクセストークンからは到達できない、対話のセッションに限る操作とする。
理由は、[外部 IdP の接続の管理を対話のセッションに限る](../design/decisions.md#外部-idp-の接続の管理を対話のセッションに限る)。
