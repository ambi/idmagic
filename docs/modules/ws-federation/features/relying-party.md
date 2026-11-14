# RP と Entra フェデレーションの管理

## 概要

この文書は、WS-Fed の RP の登録、参照、削除と、Entra のドメインフェデレーションの定型設定の仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | RP の登録と更新、一覧、削除。Entra のドメインフェデレーションの定型設定による RP の作成と更新、Graph PowerShell の手順の案内、取り込んだ登録値と期待値の照合、切り戻しを確かめた後のドメインごとの解除 |
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
- 管理者が `wtrealm` を指定して Entra の定型設定を持たない RP を削除したとき、WsFederation は、その RP を消して 204 を返す。
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
- 管理者が Entra のドメインを解除したとき、WsFederation は、Microsoft のレルム探索にそのドメインを問い合わせ、次の表に従って解除するか拒否する。

| レルム探索の結果 | 観測記録 | 結果 | 確認の根拠（`confirmed_by`） |
| --- | --- | --- | --- |
| `NameSpaceType` が `Federated` で、`AuthURL` のスキームとパスが受動のエンドポイントと一致し、ホストが大文字と小文字を区別せずに一致する | 問わない | 422 と `entra_federation_still_active` で拒否する | — |
| `NameSpaceType` が `Managed` | 問わない | 解除する | `realm_discovery` |
| `NameSpaceType` が `Federated` で、`AuthURL` が受動のエンドポイントと一致しない | 問わない | 解除する | `realm_discovery` |
| 探索の失敗か無効化、または `NameSpaceType` がそのほかの値 | 切り戻しを示す | 解除する | `observation` |
| 探索の失敗か無効化、または `NameSpaceType` がそのほかの値 | ないか、切り戻しを示さない | 422 と `entra_rollback_unconfirmed` で拒否する | — |

- `NameSpaceType` が `Federated` で `AuthURL` がホストを持つ HTTP または HTTPS の絶対 URL でない場合、WsFederation は、探索で確かめられないものとして、探索の失敗と同じく観測記録で判定する。
- 観測記録を解除の根拠にするとき、WsFederation は、確認の状態が `stale` でなく、ドメインの組が Entra のテナント ID を持てば観測したテナント ID が大文字と小文字を区別せずに一致し、観測した構成のどれも IssuerUri、`passive_sign_in_uri`、`active_sign_in_uri`、`metadata_exchange_uri` のいずれでも期待値と一致せず、`authentication_type` が大文字と小文字を区別せずに `Managed` で構成が 0 件か、`Federated` で構成がちょうど 1 件の記録だけを、切り戻しを示すものとして扱う。
- **判断**：レルム探索は、Microsoft のサインインの画面が利用者のドメインから送り先を決める公開のエンドポイントであり、解除の時点の実際の送り先を返す。探索が IdMagic を指すと答えたら、取り込んだ観測より新しい事実なので観測記録を問わずに拒否する。`Unknown` は Entra に登録のないドメインでも、Microsoft が情報を絞った場合にも返り得るので、解除の根拠にしない。
- 管理者が Entra のドメインを解除したとき、WsFederation は、行を固定して読み直した RP からそのドメインの組だけを外し、ほかのドメインの組、sourceAnchor の属性、エンドポイント、返信先、クレームのポリシー、作成時刻を変えずに保存し、そのドメインの観測記録を消し、`EntraFederationDomainRemoved` を発行し、ドメイン、残るドメイン、`relying_party_deleted` に false、`confirmed_by` を 200 で返す。
- 最後のドメインを解除したとき、WsFederation は、Entra の RP を削除し、観測記録を消し、`EntraFederationDomainRemoved` を発行し、空の残るドメインと `relying_party_deleted` に true を 200 で返す。
- Entra のドメインを解除したとき、WsFederation は、署名鍵を破棄せず、残るドメインと、その後に構成するドメインへ同じ鍵で発行する。
- 解除したドメインの UPN の利用者が Entra の発行を要求した場合、WsFederation は、構成していないドメインの利用者と同じく発行を拒否する。
- **判断**：外部の切り戻しとローカルの解除は一つの原子的な操作にならない。切り戻しだけが成功した間はドメインが構成されたまま残るので、管理者は解除を再実行して完了できる。解除の後に Entra が IdMagic へ戻された場合も、外したドメインへ発行しないので失敗はサインインの拒否に倒れる。
- 管理者が Entra のドメインを一覧で表示したとき、WsFederation は、ドメインごとの確認の状態を表示し、保存しただけのドメインを反映済みとして表示しない。
- 管理者が Entra のドメインの詳細を表示したとき、WsFederation は、変更の操作なしに保存した設定、期待する Entra の設定、証明書、手順を表示し、編集の画面への入口を置く。
- 管理者が Entra のドメインの詳細を表示したとき、WsFederation は、作成と更新のコマンドと PowerShell の手順をそれぞれコピーできる形で表示し、制限のコードを選択した言語の辞書の文言で表示する。
- ドメインの詳細が `unfederated_source_users` を持つ間、管理者がドメインの詳細を表示したとき、WsFederation は、該当する User の数とユーザー名を表示する。
- 管理者が Entra のドメインの詳細を表示したとき、WsFederation は、未確認、一致、不一致、再確認が必要を区別し、取得時刻、確認時刻、観測したテナントと、手動で取り込んだ観測であることを表示する。
- 管理者が Entra のドメインの詳細を表示したとき、WsFederation は、導入の前提（AD への認証の委譲、UPN と ImmutableID、Entra Connect のコンピューターの同期、SCP、SYSTEM の通信）と、構成の後の確認（sync join の同期待ち、端末の参加、PRT、アプリの SSO）を別の項目として表示する。
- 管理者が画面で Entra のドメインフェデレーションの構成を保存したとき、WsFederation は、そのドメインの詳細の画面へ移る。
- 定型設定がない間、管理者が画面で Entra の構成を追加したとき、WsFederation は、`on_premises_immutable_id` を使い、内部の属性名の入力なしに保存できるようにする。
- 定型設定がある間、管理者が画面で Entra の構成を追加または編集したとき、WsFederation は、保存済みの属性を初期値に使い、属性名の入力なしにその値を保って保存できるようにする。
- 管理者が Entra の構成を追加または編集したとき、WsFederation は、ImmutableID の保存先を読み取り専用で示し、明示的な詳細設定でだけ属性名を変更できるようにする。
- 管理者が Entra の構成または詳細を表示したとき、WsFederation は、Entra Connect の属性の選択確認、Connector の対応付け、再同期、Graph の実値との比較を案内し、GUID の生バイトの Base64 と表示文字列や二重 Base64 を区別する。
- 画面の保存が `source_anchor_missing` または `source_anchor_invalid` で拒否されたとき、WsFederation は、件数と対象のユーザー名に加えて、保存先の属性と Connector の対応付けと再同期を確認する案内を表示する。
- 管理者が画面で取り込んだ観測文書が拒否されたとき、WsFederation は、入力した観測文書を残して失敗を表示する。
- 管理者が Entra のドメインの詳細を表示したとき、WsFederation は、解除の項目に、解除で残るドメイン、Managed への切り替えの `Connect-MgGraph` と `Update-MgDomain` のコマンド、Microsoft の移行の手順への入口、変更前の構成の保存から以前の IdP の構成を戻す説明を表示する。
- **判断**：切り戻しのスクリプトを生成しないのは、Managed への切り替えが一つのコマンドで足り、以前の構成の復元は管理者が保存した構成と Entra の状態に依存し、IdMagic が実行を確かめられないからである。
- 管理者が画面でドメインの解除を確定したとき、WsFederation は、対象のドメインと残るドメインを確認の画面に示してから解除を要求する。
- 画面での解除が成功したとき、WsFederation は、一覧の画面へ移り、解除したドメインを通知に示す。
- 画面での解除が `entra_federation_still_active` で拒否されたとき、WsFederation は、ドメインの詳細に留まり、Entra がまだ IdMagic へ利用者を送っていることと、Managed への変換の反映に最大 60 分かかることを表示する。
- 画面での解除が `entra_rollback_unconfirmed` で拒否されたとき、WsFederation は、ドメインの詳細に留まり、Microsoft に確かめられなかったことと、読み戻しを取り込めばその結果で解除できることを表示する。
- 管理者が Entra のドメインを一覧で表示したとき、WsFederation は、Entra の RP 全体を削除する操作を置かない。
- 管理者が Entra のドメインの一覧か詳細を表示したとき、WsFederation は、共有 XML 鍵の画面への入口を置く。
- **判断**：UPN がフェデレーションしない取り込み元の User を拒否の理由にしないのは、フェデレーションしないドメインの User が AD に正当に存在するからである。取り込み元の User に限るのは、管理者が手で作った User より、取り込みの対応付けの誤り（`userPrincipalName` の UPN サフィックスや `ldap.attributes` の上書き）で UPN を欠く方が起きやすく、サインインの時点まで気付かないからである。
- 指定したドメインを構成していない間、管理者が Entra のドメインフェデレーションを構成したとき、WsFederation は、201 を返す。
- 指定したドメインを構成している間、管理者が Entra のドメインフェデレーションを構成したとき、WsFederation は、200 を返す。
- `wsfed:read` のスコープの API アクセストークンで呼び出されたとき、WsFederation は、RP の一覧と Entra のドメインの一覧と詳細の取得だけを許可する。
- `wsfed:write` のスコープの API アクセストークンで呼び出されたとき、WsFederation は、RP の登録と削除と Entra のドメインフェデレーションの構成と観測文書の取り込みとドメインの解除だけを許可する。
- `wsfed:write` を持たないトークンで変更の操作を要求された場合、WsFederation は、403 と `insufficient_scope` で拒否し、RP と観測記録を変えない。
- 別のテナントで発行したトークンを提示された場合、WsFederation は、401 と `invalid_token` で拒否し、RP と観測記録を変えない。
- `wtrealm`、返信先、`claim_policy.name_id` の `format` か `source_attribute` のない RP の登録を要求された場合、WsFederation は、400 と `invalid_request` で拒否し、RP を変えない。
- 2,048 文字または 2,048 バイトを超える `wtrealm` の RP の登録を要求された場合、WsFederation は、422 と `field_length_exceeded` で拒否し、RP を変えない。
- `wtrealm` を指定しない削除を要求された場合、WsFederation は、400 と `invalid_request` で拒否する。
- Application に属する RP の削除を要求された場合、WsFederation は、409 と `application_owned_protocol` で拒否し、RP を残す。
- Entra の定型設定を持つ RP の削除を要求された場合、WsFederation は、409 と `entra_relying_party` で拒否し、RP と観測記録を残す。
- **判断**：Entra の RP はすべてのドメインの信頼を持つので、汎用の削除は Entra 側の設定を残したまま全ドメインのサインインを止める。ドメインごとの解除だけが外せる。
- 構成していないドメインか、解除を済ませたドメインの解除を要求された場合、WsFederation は、レルム探索を呼ばずに 404 と `not_found` で拒否する。
- 解除を `entra_federation_still_active` か `entra_rollback_unconfirmed` で拒否した場合、WsFederation は、RP、観測記録を変えず、`EntraFederationDomainRemoved` を発行しない。
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
- Application に属する RP と同じ `wtrealm` の RP の登録を要求された場合、WsFederation は、409 と `application_owned_protocol` で拒否し、RP を変えない。
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

### Graph PowerShell の手順

IdMagic は Entra へ書き込まず、Graph の資格情報を持たない。
Entra を変える操作と読む操作は、管理者が自分の権限で Microsoft Graph PowerShell（`Microsoft.Graph.Identity.DirectoryManagement`）から行い、IdMagic はその手順の文字列を返すか表示する。
Graph のクライアントを IdMagic に置く自動の方式は、[Microsoft Graph による自動化の作業項目](../../../../work-items/active/wi-33604-automate-entra-domain-federation-through-microsoft-graph.md)で検討する。
手順の値は、ドメイン、IssuerUri、エンドポイント、証明書など IdMagic が保存した値と期待値だけから作り、PowerShell の単一引用符のリテラルに埋める（曲がった引用符も二重にする）。

| 手順 | 返し方 | 接続の権限 | 作用と出力 | 止まる条件 |
| --- | --- | --- | --- | --- |
| `connect` | 詳細の `procedure` | `Domain.Read.All`、`Domain-InternalFederation.ReadWrite.All` | モジュールがなければ現在の利用者に入れ、接続し、ドメインの `Id`、`AuthenticationType`、`IsVerified` を表示する | ドメインの組がテナント ID を持ち、接続先のテナントが異なる |
| `connect_read_only` | 詳細の `procedure` | `Domain.Read.All`、`Domain-InternalFederation.Read.All` | `connect` と同じ。変更の権限を求めない | `connect` と同じ |
| `backup` | 詳細の `procedure` | `connect` の権限 | `Get-MgDomainFederationConfiguration` の結果だけを `idmagic-entra-<ドメイン>-before-<UTC の時刻>.json` に書き出す。利用者の情報と資格情報を含めない | 構成の取得の失敗 |
| `compare` | 詳細の `procedure` | `connect` の権限 | 期待値と現在の構成の、文字列として異なる項目を表示する。構成がなければその旨を表示する | 構成の取得の失敗 |
| `apply` | 詳細の `procedure` | `connect` の権限 | 構成があれば `Update-MgDomainFederationConfiguration` で、なければ `New-MgDomainFederationConfiguration` で期待値を書く | 構成の取得の失敗 |
| `read_back` | 詳細の `procedure` | `connect_read_only` の権限で足りる | `Get-MgDomain` と `Get-MgDomainFederationConfiguration` の結果を観測文書の形式で `idmagic-entra-<ドメイン>-observation.json` に BOM なしで書き出す | ドメインか構成の取得の失敗。ファイルを作らない |
| Managed への切り替え | 画面が表示する固定のコマンド | `Domain.ReadWrite.All`、`Directory.AccessAsUser.All` | `Update-MgDomain -AuthenticationType 'Managed'` | Graph の拒否 |

Managed への切り替えの権限と cmdlet は、Microsoft の [federation から cloud authentication への移行](https://learn.microsoft.com/en-us/entra/identity/hybrid/connect/migrate-from-federation-to-cloud-authentication)の「Convert domains from federated to managed」に合わせる。
同じ資料は、変換に最大 60 分かかることと、PHS か PTA の準備を先に済ませることを求める。
以前の IdP の構成の復元は、`backup` のファイルの値で `Update-MgDomainFederationConfiguration` を実行する手順を運用手順書で説明し、IdMagic はスクリプトを生成しない。
戻す値は管理者が保存した構成と Entra の状態に依存し、IdMagic は実行の結果を読み戻しでしか確かめられないからである。

読み戻しは、構成の後に期待値と照合する（確認の状態）ために使う。
解除の前の確認はレルム探索が担い、読み戻しの観測記録は探索で確かめられないときの代わりの根拠になる。

### レルム探索

解除の前に、WsFederation は Microsoft のサインインの画面が使う公開のエンドポイントに、ドメインの送り先を問い合わせる。

| 項目 | 内容 |
| --- | --- |
| 要求 | `GET <ENTRA_REALM_DISCOVERY_URL>?login=idmagic-realm-check%40<ドメイン>&json=1`。ローカル部は存在しない利用者でよく、送るのはドメイン名だけである |
| 設定 | `ENTRA_REALM_DISCOVERY`（`microsoft` か `disabled`、デフォルトは `microsoft`）。`disabled` では探索を呼ばず、観測記録だけで判定する。`ENTRA_REALM_DISCOVERY_URL` のデフォルトは `https://login.microsoftonline.com/getuserrealm.srf` で、https に限り、試験のために loopback の http だけを許す |
| 通信の上限 | 接続と TLS は各 2 秒、要求は 3 秒。リダイレクトを許さず、本文は 64 KiB まで |
| 読む項目 | `NameSpaceType` と `AuthURL`。WS-Fed の `AuthURL` は登録した `PassiveSignInUri` に問い合わせ文字列を付けた形なので、スキーム、ホスト、パスで比べる |
| 失敗 | 通信の失敗、200 でない応答、上限を超える本文、JSON として読めない本文 |

探索は RP の行の固定の外で行い、判定は固定の中で読み直した RP の期待値に対して行う。
外部の取得をトランザクションの中に置かず、探索の後に変わった構成でも最新の値で判定するためである。

このエンドポイントは Microsoft の公式な API の文書にない（[判断](../decisions.md#entra-のドメインの解除をレルム探索で確かめる)）。
応答が変わるか止まると、判定は「確かめられない」へ倒れ、管理者は観測の取り込みで解除できる。
Managed への変換は反映に最大 60 分かかり、その間の探索は IdMagic を指すと答え得るので、管理者は時間をおいて解除を再実行する。

### 証明書の交代

共有 XML 鍵の準備を管理 API で行うと、メタデータは現在と次の証明書を公開し、Entra の構成結果と Graph PowerShell のコマンドは `SigningCertificate` と `NextSigningCertificate` を返す。
管理者はこの共有鍵を使うすべてのフェデレーション済みドメインへ次の証明書を登録する。
Graph の登録値を読み直して一致を確認してから、SigningKeys の確認付きの切り替えを行う。
登録が終わるまで現在の署名鍵は変わらず、切り替えの時刻から旧証明書を 7 日の間メタデータに残す。
Entra の自動更新の時期に依存せず、管理者が登録を確認する。
