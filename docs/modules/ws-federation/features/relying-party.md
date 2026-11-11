# RP と Entra フェデレーションの管理

## 概要

この文書は、WS-Fed の RP の登録、参照、削除と、Entra のドメインフェデレーションの定型設定の仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | RP の登録と更新、一覧、削除。Entra のドメインフェデレーションの定型設定による RP の作成と更新、Graph PowerShell の手順の案内、取り込んだ登録値と期待値の照合 |
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

IdMagic は Entra へ書き込まない。
管理者が Graph PowerShell で構成を適用し、読み戻した登録値の観測文書を取り込むと、WsFederation は期待値と比べた結果を観測記録として残す。
観測記録は、テナントとドメインごとに最新の一件であり、観測値、比べた期待値の指紋、比べた証明書の拇印、差分、Graph から取得した時刻と取り込んだ時刻を持つ。
観測記録は手動で取り込んだ時点の事実であり、その後の Entra の変更を検出する継続的な監視ではない。

| 確認の状態 | 意味 |
| --- | --- |
| `unverified` | 観測記録がない。保存しただけで、Entra への反映は確かめていない |
| `matched` | 現在の期待値と比べた観測記録が、すべての項目で一致した |
| `mismatched` | 現在の期待値と比べた観測記録に、異なる項目がある |
| `stale` | 観測記録の後に構成か署名鍵が変わり、現在の期待値と比べていない |

Graph の権限の不足、手順の未実行、取得の失敗は観測文書を作らないので、確認の状態を変えない。
形式の誤った文書は拒否するので、これも状態を変えない。
どれも、Entra に構成がないことを観測した `mismatched` とは区別する。

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
- 管理者が Entra のテナント ID を指定して Entra のドメインフェデレーションを構成したとき、WsFederation は、そのドメインの組にテナント ID を小文字で保存する。
- 管理者が Entra のテナント ID を指定せずに Entra のドメインフェデレーションを構成したとき、WsFederation は、そのドメインの組をテナント ID のない組で置き換える。
- 管理者が Entra のドメインを一覧したとき、WsFederation は、構成済みのドメインごとに、IssuerUri、Entra のテナント ID、確認の状態をドメインの順で 200 で返す。
- 管理者が Entra のドメインの詳細を取得したとき、WsFederation は、RP と観測記録を変えずに、保存した組、sourceAnchor の属性、返信先、構成と同じ Graph の設定値とコマンド、現在と次の署名証明書の SHA-256 の拇印と有効期限、PowerShell の手順、確認の状態、制限のコード、`unfederated_source_users` を 200 で返す。
- 管理者が Entra のドメインの詳細を取得したとき、WsFederation は、Graph への接続、確認だけの接続、変更前の構成の保存、期待値との差分の表示、構成の作成または更新、観測文書の書き出しの PowerShell の手順を返す。
- 管理者が Entra のドメインの詳細を取得したとき、WsFederation は、接続の手順で `Domain.Read.All` と `Domain-InternalFederation.ReadWrite.All` を、確認だけの接続の手順で `Domain.Read.All` と `Domain-InternalFederation.Read.All` を要求する。
- 管理者が Entra のドメインの詳細を取得したとき、WsFederation は、変更前の構成の保存の手順でフェデレーションの構成だけを書き出し、利用者の情報と資格情報を書き出さない。
- ドメインの組が Entra のテナント ID を持つ間、管理者が Entra のドメインの詳細を取得したとき、WsFederation は、そのテナントを指定して接続し、接続先のテナントが異なると止まる接続の手順を返す。
- 管理者が観測文書を取り込んだとき、WsFederation は、観測値を現在の期待値と比べ、テナントとドメインごとに最新の一件の観測記録として保存し、確認の状態と差分を 200 で返し、RP を変えない。
- 管理者が観測文書を取り込んだとき、WsFederation は、次の表の項目を比べ、異なる項目ごとに項目の名前、期待値、観測値を差分として返す。

| 項目 | 期待値 | 比べ方 |
| --- | --- | --- |
| `entra_tenant_id` | ドメインの組の Entra のテナント ID。持たない組では比べない | 大文字と小文字を区別しない |
| `authentication_type` | `Federated` | 大文字と小文字を区別しない |
| `federation_configurations` | ちょうど一つの構成。一つでなければ、構成の各項目を比べない | 件数 |
| `issuer_uri`、`passive_sign_in_uri`、`active_sign_in_uri`、`metadata_exchange_uri`、`sign_out_uri` | Graph の設定値 | 完全一致 |
| `preferred_authentication_protocol`、`federated_idp_mfa_behavior` | `wsFed`、`acceptIfMfaDoneByFederatedIdp` | 大文字と小文字を区別しない |
| `signing_certificate` | 有効な XML 署名証明書 | X.509 の DER |
| `next_signing_certificate` | 準備済みの XML 鍵の証明書。なければ空 | X.509 の DER。両方が空なら一致 |

- 管理者が観測文書を取り込んだとき、WsFederation は、証明書の値から空白を除いて base64 の X.509 として読み、DER が同じ証明書を一致とし、読めない観測値を不一致として返す。
- 観測記録がない間、WsFederation は、ドメインの確認の状態に `unverified` を返す。
- 観測記録の期待値の指紋が現在の期待値の指紋と異なる間、WsFederation は、確認の状態に `stale` を返し、記録の差分を返さない。
- 観測記録の期待値の指紋が現在の期待値の指紋と一致している間、WsFederation は、記録した比較の結果に従って `matched` か `mismatched` を、記録の差分とともに返す。
- 観測記録がある間、WsFederation は、確認の状態とともに、取得時刻、確認時刻、観測した Entra のテナント ID、比べた現在と次の署名証明書の拇印を返す。
- **判断**：期待値の指紋は、Entra のテナント ID、ドメイン、Graph の設定値、現在と次の署名証明書から求めるので、構成の変更と鍵の準備や交代のどちらでも変わる。時刻の経過だけでは失効させないのは、観測が取得の時点の事実であり、期待値が同じなら比べた結果も同じだからである。
- 指定したドメインを構成していない間、管理者が Entra のドメインフェデレーションを構成したとき、WsFederation は、そのドメインの観測記録を消す。
- **判断**：RP を削除した後に同じ設定で再び構成したドメインに、削除の前の一致を示さないためである。
- 管理者が Entra のドメインを一覧で表示したとき、WsFederation は、ドメインごとの確認の状態を表示し、保存しただけのドメインを反映済みとして表示しない。
- 管理者が Entra のドメインの詳細を表示したとき、WsFederation は、変更の操作なしに保存した設定、期待する Entra の設定、証明書、手順を表示し、編集の画面への入口を置く。
- 管理者が Entra のドメインの詳細を表示したとき、WsFederation は、作成と更新のコマンドと PowerShell の手順をそれぞれコピーできる形で表示し、制限のコードを選択した言語の辞書の文言で表示する。
- ドメインの詳細が `unfederated_source_users` を持つ間、管理者がドメインの詳細を表示したとき、WsFederation は、該当する User の数とユーザー名を表示する。
- 管理者が Entra のドメインの詳細を表示したとき、WsFederation は、未確認、一致、不一致、再確認が必要を区別し、取得時刻、確認時刻、観測したテナントと、手動で取り込んだ観測であることを表示する。
- 管理者が Entra のドメインの詳細を表示したとき、WsFederation は、導入の前提（AD への認証の委譲、UPN と ImmutableID、Entra Connect のコンピューターの同期、SCP、SYSTEM の通信）と、構成の後の確認（sync join の同期待ち、端末の参加、PRT、アプリの SSO）を別の項目として表示する。
- 管理者が画面で Entra のドメインフェデレーションの構成を保存したとき、WsFederation は、そのドメインの詳細の画面へ移る。
- 管理者が画面で取り込んだ観測文書が拒否されたとき、WsFederation は、入力した観測文書を残して失敗を表示する。
- **判断**：UPN がフェデレーションしない取り込み元の User を拒否の理由にしないのは、フェデレーションしないドメインの User が AD に正当に存在するからである。取り込み元の User に限るのは、管理者が手で作った User より、取り込みの対応付けの誤り（`userPrincipalName` の UPN サフィックスや `ldap.attributes` の上書き）で UPN を欠く方が起きやすく、サインインの時点まで気付かないからである。
- 指定したドメインを構成していない間、管理者が Entra のドメインフェデレーションを構成したとき、WsFederation は、201 を返す。
- 指定したドメインを構成している間、管理者が Entra のドメインフェデレーションを構成したとき、WsFederation は、200 を返す。
- `wsfed:read` のスコープの API アクセストークンで呼び出されたとき、WsFederation は、RP の一覧と Entra のドメインの一覧と詳細の取得だけを許可する。
- `wsfed:write` のスコープの API アクセストークンで呼び出されたとき、WsFederation は、RP の登録と削除と Entra のドメインフェデレーションの構成と観測文書の取り込みだけを許可する。
- `wsfed:write` を持たないトークンで変更の操作を要求された場合、WsFederation は、403 と `insufficient_scope` で拒否し、RP と観測記録を変えない。
- 別のテナントで発行したトークンを提示された場合、WsFederation は、401 と `invalid_token` で拒否し、RP と観測記録を変えない。
- `wtrealm`、返信先、`claim_policy.name_id` の `format` か `source_attribute` のない RP の登録を要求された場合、WsFederation は、400 と `invalid_request` で拒否し、RP を変えない。
- 2,048 文字または 2,048 バイトを超える `wtrealm` の RP の登録を要求された場合、WsFederation は、422 と `field_length_exceeded` で拒否し、RP を変えない。
- `wtrealm` を指定しない削除を要求された場合、WsFederation は、400 と `invalid_request` で拒否する。
- Application に属する RP の削除を要求された場合、WsFederation は、409 と `application_owned_protocol` で拒否し、RP を残す。
- ドメインがないか URL の形であるか、sourceAnchor の属性がないか、返信先が絶対 URL でない Entra の構成を要求された場合、WsFederation は、400 と `invalid_request` で拒否し、RP を変えない。
- GUID の形でない Entra のテナント ID の構成を要求された場合、WsFederation は、400 と `invalid_request` で拒否し、RP を変えない。
- 構成していないドメインの詳細の取得か観測文書の取り込みを要求された場合、WsFederation は、404 と `not_found` で拒否し、観測記録を変えない。
- 64 KiB を超えるか、`format` が `idmagic.entra-federation-observation.v1` でないか、未知の項目を持つか、必須の項目を欠くか、ドメインが指定したドメインと異なるか、構成が 4 件を超えるか、取得時刻が確認時刻より 5 分を超えて後の観測文書の取り込みを要求された場合、WsFederation は、400 と `invalid_request` で拒否し、観測記録を変えない。
- 保存済みの観測記録より前の取得時刻の観測文書の取り込みを要求された場合、WsFederation は、409 と `observation_outdated` で拒否し、観測記録を変えない。
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

導入、適用と反映の確認の手順は [Entra のドメインフェデレーションの導入と反映の確認](../../../runbooks/entra-domain-federation.md)に従う。

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
