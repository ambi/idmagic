# RP と Entra フェデレーションの管理

## 概要

この文書は、WS-Fed の RP の登録、参照、削除と、Entra のドメインフェデレーションの定型設定の仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | RP の登録と更新、一覧、削除。Entra のドメインフェデレーションの定型設定による RP の作成と更新 |
| 行為者 | テナント管理者、管理 API のクライアント |
| 扱わないもの | アプリケーションに属する RP の設定は `Application` のプロトコル設定からも行える。クレームのポリシーの意味は `ClaimMapping` が定める |

## モデル

`EntraFederationProfile` は、WS-Fed の RP のための定型設定である。
Entra はどのドメインの要求でも、自身の識別子 `urn:federation:MicrosoftOnline` を `wtrealm` と `AppliesTo` に使う。
そのため定型設定は、`wtrealm` と audience を `urn:federation:MicrosoftOnline` とする `WsFedRelyingParty` をテナントに一つだけ持ち、その中に構成したドメインごとの IssuerUri を持つ。

| 要素 | 内容 |
| --- | --- |
| ドメインの組 | 検証済みのドメインと、そのドメインの IssuerUri。IssuerUri はドメインの間で重ならない |
| sourceAnchor の属性 | 定型設定に一つ。属性スキーマで `unique` で、`editable_by_user` でない属性に限る。すべてのドメインの利用者の ImmutableID は、この属性の値そのものである。Directory Connector が AD から取り込む組み込みの属性 `on_premises_immutable_id` はこの条件を満たし、管理画面はこれを初期値にする |
| エンドポイント | 受動、能動、MEX の各エンドポイント |

| 必須のクレーム | 発行のしかた |
| --- | --- |
| UPN | `preferred_username` から `http://schemas.xmlsoap.org/claims/UPN` として発行する |
| ImmutableID | sourceAnchor の属性の値を変えずに（`entra_immutable_id`）、永続的な NameID、`http://schemas.xmlsoap.org/claims/nameidentifier`、`http://schemas.microsoft.com/LiveID/Federation/2008/05/ImmutableID` に含める |

Assertion の発行者は、利用者の UPN のドメインに最も長く一致する構成済みのドメインの IssuerUri とする。
サブドメインの利用者は、Entra が親のドメインの設定を継ぐので、親のドメインの IssuerUri で発行する。

必須のクレームは定型設定で固定し、フェイルクローズで扱う。
`entra_immutable_id` は、sourceAnchor の属性を公開できるときだけ公開できる。
sourceAnchor の値の重なりは、属性が `unique` なので IdManagement が書き込みの時点で拒否する。
空の値と `@` を含む値は ImmutableID として使えない。
設定の時点では、フェデレーションするドメインの利用者の欠けた値と使えない値を拒否し、発行の時点では、属性の定義が守られていないか対象の利用者の値が使えなければクレームの発行を拒否する。

- **判断**：値を変換しないのは、Entra Connect が sourceAnchor を immutableId と同じものとし、文字列でない属性（objectGUID）は同期の側で base64 に符号化し、文字列の属性はそのまま使うからである。値をそのまま使えば、保存した値の一意性が ImmutableID の一意性になり、同じ ImmutableID の別の表記が重なりを逃れることがない。

- **判断**：定型設定にする理由は、[Entra のドメインフェデレーションを専用の定型設定として扱う](../decisions.md#entra-のドメインフェデレーションを専用の定型設定として扱う)。

## 操作

### 管理 API のクライアントによる RP と Entra フェデレーションの操作

#### REQ-WSFEDERATION-001 管理 API クライアントは WS-Fed スコープの信頼設定だけを操作できる

- 同じ `wtrealm` の RP がない間、管理者が RP を登録したとき、WsFederation は、RP を作り、201 と保存した RP を返す。
- 同じ `wtrealm` の RP がある間、管理者が RP を登録したとき、WsFederation は、作成時刻を保って RP を置き換え、200 と保存した RP を返す。
- 管理者が RP を一覧したとき、WsFederation は、テナントのすべての RP を 200 で返す。
- 管理者が `wtrealm` を指定して RP を削除したとき、WsFederation は、その RP を消して 204 を返す。
- 管理者が Entra のドメインフェデレーションを構成したとき、WsFederation は、`wtrealm` と audience を `urn:federation:MicrosoftOnline` とするテナントに一つの SAML 1.1 の RP に、ドメインと IssuerUri（省略時は `urn:idmagic:entra:<ドメイン>`）の組を追加または置き換え、`EntraFederationConfigured` を発行し、PowerShell に渡す設定値とともに返す。
- 管理者が Entra のドメインフェデレーションを構成したとき、WsFederation は、Microsoft Graph PowerShell の名前を持つ設定値、有効な XML 署名証明書の公開部分の base64、値を埋めた `New-MgDomainFederationConfiguration` と `Update-MgDomainFederationConfiguration` のコマンドを返す。
- 準備済みの XML 鍵がある間、管理者が Entra のドメインフェデレーションを構成したとき、WsFederation は、その証明書の公開部分の base64 を `NextSigningCertificate` として設定値とコマンドに含める。
- 管理者が Entra のドメインフェデレーションを構成したとき、WsFederation は、`SignOutUri` に `wsignout1.0` を受ける受動的なエンドポイントを、`PreferredAuthenticationProtocol` に `wsFed` を、`FederatedIdpMfaBehavior` に `acceptIfMfaDoneByFederatedIdp` を返す。
- 取り込み元の管理下の User のうち、構成した後の定型設定で UPN がどの構成済みのドメインにも属さない削除されていない User がいる間、管理者が Entra のドメインフェデレーションを構成したとき、WsFederation は、構成を拒否せず、該当する User の数（`unfederated_source_users.user_count`）と、ユーザー名の順で先頭の 5 人までのユーザー名（`unfederated_source_users.sample_usernames`）を返す。
- 該当する取り込み元の管理下の User がいない間、管理者が Entra のドメインフェデレーションを構成したとき、WsFederation は、`unfederated_source_users` を返さない。
- 管理者が構成の結果を表示したとき、WsFederation は、作成と更新のコマンドをそれぞれコピーできる形で表示し、制限のコードを選択した言語の辞書の文言で表示する。
- 構成の結果が `unfederated_source_users` を持つ間、管理者が構成の結果を表示したとき、WsFederation は、該当する User の数とユーザー名を表示する。
- **判断**：UPN がフェデレーションしない取り込み元の User を拒否の理由にしないのは、フェデレーションしないドメインの User が AD に正当に存在するからである。取り込み元の User に限るのは、管理者が手で作った User より、取り込みの対応付けの誤り（`userPrincipalName` の UPN サフィックスや `ldap.attributes` の上書き）で UPN を欠く方が起きやすく、サインインの時点まで気付かないからである。
- 指定したドメインを構成していない間、管理者が Entra のドメインフェデレーションを構成したとき、WsFederation は、201 を返す。
- 指定したドメインを構成している間、管理者が Entra のドメインフェデレーションを構成したとき、WsFederation は、200 を返す。
- `wsfed:read` のスコープの API アクセストークンで呼び出されたとき、WsFederation は、RP の一覧だけを許可する。
- `wsfed:write` のスコープの API アクセストークンで呼び出されたとき、WsFederation は、RP の登録と削除と Entra のドメインフェデレーションの構成だけを許可する。
- `wsfed:write` を持たないトークンで変更の操作を要求された場合、WsFederation は、403 と `insufficient_scope` で拒否し、RP を変えない。
- 別のテナントで発行したトークンを提示された場合、WsFederation は、401 と `invalid_token` で拒否し、RP を変えない。
- `wtrealm`、返信先、`claim_policy.name_id` の `format` か `source_attribute` のない RP の登録を要求された場合、WsFederation は、400 と `invalid_request` で拒否し、RP を変えない。
- 2,048 文字または 2,048 バイトを超える `wtrealm` の RP の登録を要求された場合、WsFederation は、422 と `field_length_exceeded` で拒否し、RP を変えない。
- `wtrealm` を指定しない削除を要求された場合、WsFederation は、400 と `invalid_request` で拒否する。
- Application に属する RP の削除を要求された場合、WsFederation は、409 と `application_owned_protocol` で拒否し、RP を残す。
- ドメインがないか URL の形であるか、sourceAnchor の属性がないか、返信先が絶対 URL でない Entra の構成を要求された場合、WsFederation は、400 と `invalid_request` で拒否し、RP を変えない。
- sourceAnchor の属性が属性スキーマで `unique` でない場合、WsFederation は、Entra の構成を 422 と `source_anchor_not_unique` で拒否し、RP を変えない。
- sourceAnchor の属性が属性スキーマで `editable_by_user` である場合、WsFederation は、Entra の構成を 422 と `source_anchor_editable` で拒否し、RP を変えない。
- 構成した後の定型設定で UPN がフェデレーションするドメインに属する既存の User のどれかが sourceAnchor の値を持たない場合、WsFederation は、Entra の構成を 422 と `source_anchor_missing` で拒否し、該当する User の数（`user_count`）と先頭の 5 人までのユーザー名（`sample_usernames`）を返し、RP を変えない。
- 構成した後の定型設定で UPN がフェデレーションするドメインに属する既存の User のどれかが、空か `@` を含む sourceAnchor の値を持つ場合、WsFederation は、Entra の構成を 422 と `source_anchor_invalid` で拒否し、`user_count` と `sample_usernames` を返し、RP を変えない。
- 管理者が Entra のドメインフェデレーションを構成したとき、WsFederation は、UPN がフェデレーションするドメインに属さない User の sourceAnchor の値を確かめない。
- Entra の定型設定を持つ RP と同じ `wtrealm` の RP の登録を要求された場合、WsFederation は、409 と `entra_relying_party` で拒否し、RP を変えない。
- IssuerUri がほかの構成済みのドメインの IssuerUri と同じか、ほかのドメインを構成している間に構成済みのものと異なる sourceAnchor の属性を指定された場合、WsFederation は、Entra の構成を 400 と `invalid_request` で拒否し、RP を変えない。
- `admin` のロールを持たない利用者が RP か Entra のドメインフェデレーションの管理 API を要求した場合、WsFederation は、403 と `access_denied` で拒否し、RP を変えない。
- **例**：EX-WSFEDERATION-001-01、EX-WSFEDERATION-001-02、EX-WSFEDERATION-001-03、EX-WSFEDERATION-001-04

## セキュリティ上の考慮

RP の登録、参照、削除と Entra のドメインフェデレーションの設定は、`AdminFederationTrustsManage` の権限（AuthZEN の action `admin:federation_trusts_manage`）を要する。
対話のセッションでは、`admin` ロールを持つ、有効かつ認証済みのユーザーが、所属テナントに対して行える。

## 実現方式

### アーキテクチャ

sourceAnchor の属性には、Entra の利用者の ImmutableID（`onPremisesImmutableId`）と同じ文字列を保存する。
AD の objectGUID を sourceAnchor にする場合は、Entra Connect と同じく、取り込みの側が .NET の `Guid.ToByteArray()` のバイト順で base64 に符号化した値を保存する。

### 信頼性

保存した値が Entra の ImmutableID と食い違うと、Entra は Assertion を社内の同じユーザーへ関連付けられず、アカウントの重複やサインインの失敗を招く。
GUID の文字列のまま保存した値は、base64 に符号化した ImmutableID と一致しない。
Entra 側では原因を特定しにくいので、サインインの失敗が続く場合は、まず発行した ImmutableID と、Entra の利用者の `onPremisesImmutableId` を突き合わせる。

IdMagic は、Microsoft Entra hybrid join の即時の登録に要る `windowstransport` を提供しない（[判断](../decisions.md#ws-trust-の能動的な対応を-usernamemixed-の-issue-だけに絞る)）。
フェデレーションしたドメインの端末は、Entra Connect による同期の後の登録（sync join）で Hybrid Join を終える。
sync join には、Windows 10 1803 以降の端末であることと、Entra Connect が対象の OU のコンピューターとデフォルトの端末の属性を同期することが要る（[Microsoft Entra hybrid join の手動構成](https://learn.microsoft.com/en-us/entra/identity/devices/hybrid-join-manual)）。
端末の最初の登録の試みは失敗し、Entra Connect がコンピューターの `userCertificate` を同期した後の試みで登録が終わる（[Okta の Hybrid Join の説明](https://help.okta.com/en-us/content/topics/provisioning/azure/haad-join/about-haad.htm)が同じ流れを示す）ので、構成の直後に登録が終わらなくても障害ではない。
登録が終わらない場合は、Entra Connect の同期の範囲と、端末の `dsregcmd /status` の結果を確かめる。

### 証明書の交代

共有 XML 鍵の準備を管理 API で行うと、メタデータは現在と次の証明書を公開し、Entra の構成結果と Graph PowerShell のコマンドは `SigningCertificate` と `NextSigningCertificate` を返す。
管理者はこの共有鍵を使うすべてのフェデレーション済みドメインへ次の証明書を登録する。
Graph の登録値を読み直して一致を確認してから、SigningKeys の確認付きの切り替えを行う。
登録が終わるまで現在の署名鍵は変わらず、切り替えの時刻から旧証明書を 7 日の間メタデータに残す。
Entra の自動更新の時期に依存せず、管理者が登録を確認する。
