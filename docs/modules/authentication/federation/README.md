# 外部 IdP との連携

## 概要

この文書は、テナント単位の上流の OIDC と SAML の接続によるログインの仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | 上流の接続の管理、外部 subject とローカルの User の関連付け、関連付けと JIT のポリシー、ログインセッションへの引き渡し |
| 行為者 | EndUser、本人（明示的な関連付けと解除）、テナント管理者 |
| 扱わないもの | 下流向けの SAML の IdP と WS-Federation の発行は、各プロトコルのモジュールが扱う。JIT で作る User の記録は `IdManagement` が扱う |

## モデル

ブローカーは、まず `FederatedIdentity` を通じて不変な外部 subject を解決する。
検証済みのメールアドレスによる関連付けを許すのは、接続に明示的なポリシーがあり、上流のクレームが検証済みで、テナントの中で一意に一致する場合だけである。
JIT は接続ごとに個別に有効にし、メールのドメインの許可リストでさらに絞り込める。
明示的な関連付けと解除にはステップアップ認証を求め、最後に残った利用可能なサインインの手段は取り除けない。

- **判断**：アイデンティティブローカーを Authentication に置く理由は、[ログインの時点のアイデンティティブローカーを Authentication に置く](../design/decisions.md#ログインの時点のアイデンティティブローカーを-authentication-に置く)。

## 状態遷移

### IdentityProviderConnectionLifecycle

上流との接続は、利用できる `Active` と経路を止めた `Disabled` を行き来し、削除で `Deleted` になる。作成直後は `Disabled` である。`Active` の接続で信頼の根拠にあたる項目（プロトコル、issuer、クライアント ID、エンドポイント、署名の証明書など）を更新すると、接続は `Disabled` に戻る。メタデータの再取得の失敗や、信頼の根拠にあたらない項目の更新は状態を変えず、最後に成功した内容を保持する。遷移の表の `IdentityProviderConnectionActivated` と `IdentityProviderConnectionDisabled` は遷移の契機の名前であり、ドメインイベントとしては発行しない。

| State | Kind | Meaning |
|---|---|---|
| Disabled | initial | 経路を止めている。作成直後はこの状態である |
| Active | — | 上流との接続を利用できる |
| Deleted | terminal | 接続を削除した。どの操作からも見えない |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Active | IdentityProviderConnectionDisabled | — | Disabled |  |
| Active | 信頼の根拠の更新 | — | Disabled |  |
| Disabled | IdentityProviderConnectionActivated | 接続の設定が検証を通る | Active |  |
| Active | 管理者による削除 | 連携した外部アイデンティティがない | Deleted |  |
| Disabled | 管理者による削除 | 連携した外部アイデンティティがない | Deleted |  |

| State | 外部 IdP のログイン | 接続の有効化 | 接続の無効化 | 接続の更新 | 接続の削除 |
|---|---|---|---|---|---|
| Disabled | 拒否：400 invalid_request | → Active（設定が検証を通る）<br>拒否：400 invalid_state（設定が検証を通らない） | 何もしない | 何もしない | → Deleted（連携が残っていない）<br>拒否：409 connection_in_use（連携が残っている） |
| Active | 何もしない | 拒否：400 invalid_state | → Disabled | → Disabled（信頼の根拠を変える）<br>何もしない（信頼の根拠を変えない） | → Deleted（連携が残っていない）<br>拒否：409 connection_in_use（連携が残っている） |
| Deleted | 拒否：400 invalid_request | 拒否：404 not_found | 拒否：404 not_found | 拒否：404 not_found | 何もしない |

## 操作

### 利用者による外部 IdP のログイン

#### REQ-AUTHENTICATION-001 外部 OIDC 認証は検証済みの subject を常に同じローカル User へ相関する

- 利用者が `Active` の接続を指定して外部 IdP のログインを始めたとき、Authentication は、`state`、`nonce`、PKCE を一度だけ使えるログインの試行として保存し、保存済みの設定の上流のエンドポイントへ 303 でリダイレクトする。
- 利用者がログインの一覧を要求したとき、Authentication は、`Active` の接続だけの識別子、表示名、プロトコルを返す。
- 上流のコールバックを受けたとき、Authentication は、認可コード、ID Token の署名、issuer、audience、時刻、nonce、または SAML の応答を、保存済みの設定で検証する。
- 検証済みの外部 subject に既存の関連付けがあるとき、Authentication は、同じテナント、プロバイダー、外部 subject の関連付けから同じローカルの User を解決する。
- 未連携の外部 subject で、接続の JIT が有効でメールのドメインが許可リストに合うとき、Authentication は、クレームの対応付けに従ってローカルの User と `FederatedIdentity` を作り、`FederatedIdentityLinked` を発行する。
- ローカルの User を解決したとき、Authentication は、`amr` に `federated` を持つログインセッションを作ってセッションの Cookie を発行し、`FederatedAuthenticated` を発行し、元の認可のトランザクションへ 303 で戻す。
- 存在しないか `Active` でない接続のログインを要求されたか、`return_to` が不正な場合、Authentication は、400 と `invalid_request` で拒否する。
- 上流へのログインを始められない場合、Authentication は、400 と `federation_failed` で拒否する。
- `state` か応答のないコールバックを受けた場合、Authentication は、400 と `invalid_request` で拒否する。
- `state`、`nonce`、issuer、audience、署名、時刻のどれかが一致しないか、同じ `state` かトークンの応答が再利用された場合、Authentication は、401 と `federation_failed` で拒否し、ログインセッションも関連付けも作らず、`FederatedLoginRejected` を発行する。
- 解決した User が `Active` でない場合、Authentication は、401 と `federation_failed` で拒否し、ログインセッションを作らない。
- **例**：EX-AUTHENTICATION-001-01、EX-AUTHENTICATION-001-02、EX-AUTHENTICATION-001-03

#### REQ-AUTHENTICATION-002 検証済みメールアドレスによる自動リンクは明示ポリシーと一意な一致を要求する

- 接続の `linking_policy` が `VerifiedEmail` の間、上流の `email_verified` が true のメールアドレスがテナントの検証済みのメールアドレスと一意に一致する未連携の外部 subject でログインを完了したとき、Authentication は、既存の User に `FederatedIdentity` を作り、`FederatedIdentityLinked` を発行する。
- ポリシーが `None` か、メールアドレスが未検証か、一致が一意でなく、JIT でも User を作れない場合、Authentication は、401 と `federation_failed` で拒否し、関連付けもログインセッションも作らない。
- **例**：EX-AUTHENTICATION-002-01、EX-AUTHENTICATION-002-02

### 本人による外部アイデンティティの関連付けと解除

#### REQ-AUTHENTICATION-003 外部アイデンティティの明示的なリンクと解除はステップアップ認証を要求する

- 本人がステップアップ認証を経たセッションで関連付けを始め、未使用の外部 subject で上流の認証を完了したとき、Authentication は、その外部 subject を本人へ関連付け、`FederatedIdentityLinked` を発行し、アカウントのセキュリティの画面へ 303 で戻す。
- 本人が関連付けを一覧したとき、Authentication は、外部 subject を含めずに本人の関連付けを返す。
- 本人がステップアップ認証を経たセッションで関連付けの解除を要求したとき、Authentication は、204 を返し、その関連付けを消し、`FederatedIdentityUnlinked` を発行する。
- 本人が関連付けていないプロバイダーの解除を要求したとき、Authentication は、204 を返し、何も変えない。
- ステップアップ認証を経ていないセッションで関連付けか解除を受けた場合、Authentication は、403 と `step_up_required` で拒否する。
- 別の User に関連付けた外部 subject で関連付けを完了した場合、Authentication は、401 と `federation_failed` で拒否し、関連付けを変えない。
- 解除するとパスワードも他の外部アイデンティティも残らない場合、Authentication は、403 と `unlink_denied` で拒否し、関連付けを残す。
- **例**：EX-AUTHENTICATION-003-01、EX-AUTHENTICATION-003-02、EX-AUTHENTICATION-003-03

### 管理者による外部 IdP の接続の管理

#### REQ-AUTHENTICATION-025 外部 IdP 接続の管理は対話セッションに限る

- 管理者がブラウザーのログインセッションか管理ポータルのアクセストークンで接続を作成したとき、Authentication は、`Disabled` の接続を作り、201 と、クライアントシークレットの代わりに設定の有無を示す接続を返す。
- 管理者が接続を一覧したとき、Authentication は、200 と、クライアントシークレットの代わりに設定の有無を示す接続の一覧を返す。
- 管理者が信頼の根拠を変えずに接続を更新したとき、Authentication は、200 と接続を返し、状態を変えない。
- 管理者が `Active` の接続の信頼の根拠を更新したとき、Authentication は、200 と接続を返し、接続を `Disabled` にする。
- 管理者が接続を有効化か無効化したとき、Authentication は、204 を返し、状態遷移表のとおりに状態を変える。
- 設定が検証を通らない接続か `Active` の接続の有効化を受けた場合、Authentication は、400 と `invalid_state` で拒否する。
- 管理者が OIDC の接続のメタデータの再取得を要求したとき、Authentication は、Discovery の文書を取得し直し、200 と接続を返す。
- 管理者が接続の検査を要求したとき、Authentication は、200 と、設定と上流の検査の失敗の一覧を返す。
- 管理者がクレームの対応付けの試行を要求したとき、Authentication は、200 と正規化したクレームを返す。
- API アクセストークンで接続の参照か変更を要求された場合、Authentication は、トークンのスコープにかかわらず 403 と `insufficient_scope` で拒否する。
- 管理者が API アクセストークンでセッションと認証の情報の管理 API を要求したとき、Authentication は、`sessions:read` でセッションとサインイン履歴の参照を、`sessions:write` でセッションの失効を、`users:write` で MFA の登録の許可と認証器のリセットを許可する。
- 存在しない接続の更新、有効化、無効化、再取得、検査、試行を受けた場合、Authentication は、404 と `not_found` で拒否する。
- JSON として読めない本文か保存できない設定を受けた場合、Authentication は、400 と `invalid_request` で拒否する。
- SAML の接続のメタデータの再取得を受けた場合、Authentication は、400 と `unsupported` で拒否する。
- 上流のメタデータを検証できない場合、Authentication は、400 と `metadata_invalid` で拒否し、最後に成功した内容を保持する。
- クレームの対応付けを適用できない場合、Authentication は、400 と `invalid_mapping` で拒否する。
- **例**：EX-AUTHENTICATION-025-01、EX-AUTHENTICATION-025-02

#### REQ-AUTHENTICATION-037 外部 IdP 接続は、利用者との連携が残っている間は削除できない

- 管理者が連携の残っていない接続を削除したとき、Authentication は、204 を返し、接続を `Deleted` にし、以後の一覧に含めない。
- 管理者が存在しない接続の削除を要求したとき、Authentication は、204 を返し、何も変えない。
- 連携した外部アイデンティティが残る接続の削除を受けた場合、Authentication は、409 と `connection_in_use` で拒否し、接続も連携も残す。
- **例**：EX-AUTHENTICATION-037-01、EX-AUTHENTICATION-037-02

## セキュリティ上の考慮

上流の IdP から受け取ったものは、すべて信頼しない。
ログインの要求が使うのは保存済みのプロバイダーとエンドポイントの設定だけであり、ブラウザーから任意の Discovery の URL やトークンの URL を指定することはできない。
クライアントシークレットはデータベースに保存せず `secret_reference` だけを持ち、外部のトークンと SAML の Assertion はログインの時点で検証したうえで保持しない。

外部 IdP の接続の管理 API は、API アクセストークンからは到達できない、対話のセッションに限る操作とする。
理由は、[外部 IdP の接続の管理を対話のセッションに限る](../design/decisions.md#外部-idp-の接続の管理を対話のセッションに限る)。
