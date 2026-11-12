# Entra のドメインフェデレーションの導入と反映の確認

## 着手条件

Microsoft Entra の検証済みドメインを IdMagic へフェデレーションする場合と、構成や XML 署名鍵を変えた後に Entra の登録値を確かめ直す場合に使う。
IdMagic の管理者と、Entra のドメインの構成を変えられる管理者が協力して行う。
定型設定と照合の規則は [RP と Entra フェデレーションの管理](../modules/ws-federation/features/relying-party.md)が定める。
IdMagic は Entra へ書き込まず、Graph の資格情報も保存しない。

## 最初に確認すること

- Directory Connector が AD のパスワードの照合を AD に委ね、利用者を取り込んでいる。
- 利用者の UPN がフェデレーションするドメインに属し、sourceAnchor の属性（通常は `on_premises_immutable_id`）が Entra の `onPremisesImmutableId` と同じ値を持つ。
- Entra Connect が、対象の OU のコンピューターとデバイスの属性を同期している。
- AD のサービス接続ポイント（SCP）が Entra のテナントを指している。
- 端末の SYSTEM アカウントが、プロキシを通して Microsoft Entra の登録とサインインの URL に届く。
- Entra の操作者が、適用には Domain Name Administrator、External Identity Provider Administrator、Hybrid Identity Administrator、Security Administrator のどれかのロールを持つ。読み戻しだけなら Global Reader か Security Reader でもよい（[一覧の API](https://learn.microsoft.com/en-us/graph/api/domain-list-federationconfiguration?view=graph-rest-1.0)）。
- Entra のテナント ID（GUID）を確認しておく。

## 実施

### IdMagic に保存する

先に[ImmutableID を同期側と合わせる](#immutableid-を同期側と合わせる)を実施する。
管理画面の Entra 連携から「ドメインフェデレーションを追加」を開き、ドメインと Entra のテナント ID を入力して保存する。
通常は組み込みの保存先 `on_premises_immutable_id` を使うため、属性名の入力は要らない。
既存の別属性を使う構成は、その保存先が読み取り専用で表示され、編集や別ドメインの追加でも保持される。
変更するときだけ「ImmutableID の詳細設定」を開く。
この欄に入れるのは IdMagic の属性名であり、AD の属性名でもユーザーごとの識別子の値でもない。
保存するとドメインの詳細へ移る。
詳細の状態は「未確認」であり、この時点では Entra は何も変わっていない。
取り込み元の User のうち UPN がどのドメインにも属さない人数が表示されたら、取り込みの対応付けを先に確かめる。

### ImmutableID を同期側と合わせる

AD の同期元の属性、IdMagic の保存先の属性、Entra のユーザーごとの実値を順に確認する。
IdMagic は Entra Connect の構成を検出せず、Graph から利用者の値を自動取得しない。

1. Entra Connect の担当者が「現在の構成を表示」（View current configuration）を開き、Review Your Solution の SOURCE ANCHOR と USER PRINCIPAL NAME を確認して導入記録と照合する。独自の同期規則がある場合は Synchronization Rules Editor でユーザーの sourceAnchor への対応付けも確認する。値を変更する操作は実行しない。確認できなければ担当者に確認してから進む。
2. Connector の `ldap.attributes.on_premises_immutable_id` を同じ AD 属性に合わせる。`objectGUID` ならデフォルトのままでよく、`mS-DS-ConsistencyGuid` なら明示的に上書きする。設定例と再同期は [Connector の手順](directory-connector.md#entra-のドメインフェデレーションと併用する)に従う。
3. 完全な再同期の完了後、IdMagic の対象ユーザーの詳細で保存先の実値を確認する。既存の別属性を使う場合は、詳細に表示された保存先を確認する。
4. 同じユーザーについて、次の Graph の読み取りで取得した値と大文字と小文字を含めて一致することを確認する。UPN も同じユーザーを指すことを確認する。

新しい Connect でも以前の sourceAnchor を引き継ぐことがある。
確認画面の手順は [Microsoft のサインイン障害の調査手順](https://learn.microsoft.com/en-us/troubleshoot/windows-server/active-directory/troubleshoot-ad-fs-sso-issue#check-immutableid-and-upn-in-issuance-claim-rule)にもある。
また、Connect が objectGUID を ConsistencyGuid に書き込む場合があるので、二つの属性が同じ値でも選択の根拠にはしない。
同期側とフェデレーション側の値を一致させる条件と Connect の選択規則は [Microsoft の設計概念](https://learn.microsoft.com/en-us/entra/identity/hybrid/connect/plan-connect-design-concepts#impact-on-ad-fs-or-third-party-federation-configuration)で確認できる。

Graph PowerShell の例は次のとおりである。
`$tenantId` と `$upn` を対象に置き換え、`User.Read.All` の読み取りに同意できる管理者が実行する。
ドメイン設定の読み戻しとは別の権限であり、IdMagic に Graph の資格情報を渡す必要はない。

```powershell
$tenantId = '00000000-0000-0000-0000-000000000000'
$upn = 'alice@contoso.com'
Connect-MgGraph -TenantId $tenantId -Scopes 'User.Read.All'
Get-MgUser -UserId $upn -Property UserPrincipalName,OnPremisesImmutableId |
  Select-Object UserPrincipalName,OnPremisesImmutableId
Disconnect-MgGraph
```

通常の `Get-MgUser` は一部のプロパティだけを返すため、`-Property` を指定する（[Get-MgUser の資料](https://learn.microsoft.com/en-us/powershell/module/microsoft.graph.users/get-mguser?view=graph-powershell-1.0)）。
GUID の生バイトを一度だけ Base64 にした文字列を比較し、GUID の表示文字列、その文字列の Base64、Base64 の二重変換を使わない。
Connector は AD のバイト列を Base64 にし、WsFederation は保存した値をそのまま発行する。
比較に使ったユーザーの値をドメイン設定の観測文書に混ぜず、公開の試験結果や監査ログへ貼り付けない。

欠損または不正値で保存が拒否されたら、表示された件数とユーザー名を起点に、IdMagic の保存先、Connector の対応付け、AD の元の値を確認する。
取り込んだユーザーの値を管理画面から手で補うと、次の同期で戻されるため、取り込み元を直して完全な再同期を行う。
値が入っていても Entra と同じであることは保存時には検証されないので、Graph との比較まで行う。

### Graph PowerShell で適用する

詳細の「Graph PowerShell の手順」を、Windows PowerShell 5.1 か PowerShell 7 で上から順に実行する。

1. 接続：`Domain.Read.All` と `Domain-InternalFederation.ReadWrite.All` を求める。テナント ID を保存したドメインでは、別のテナントに接続すると止まる。
2. 変更前の構成を保存：現在のフェデレーションの構成だけを JSON に書き出す。利用者の情報や資格情報は含まない。
3. 差分を表示：現在の構成と期待値の異なる項目を表示する。構成がなければその旨を表示する。
4. 作成または更新：既存の構成があれば更新し、なければ作成する。管理されたドメインはこの時点でフェデレーションへ切り替わり、そのドメインの利用者は IdMagic でサインインすることになる。
5. 読み戻して保存：`idmagic-entra-<ドメイン>-observation.json` を BOM なしで書き出す。

確認だけを行うときは、「確認だけの接続」で `Domain.Read.All` と `Domain-InternalFederation.Read.All` だけを求め、手順 5 だけを実行する。
各コマンドの根拠は [Graph の作成の API](https://learn.microsoft.com/en-us/graph/api/domain-post-federationconfiguration?view=graph-rest-1.0) と [ドメインの取得の API](https://learn.microsoft.com/en-us/graph/api/domain-get?view=graph-rest-1.0) にある。

### 読み戻した値を取り込む

手順 5 のファイルを詳細の「読み戻しの結果」に読み込むか貼り付け、「取り込んで照合」を押す。
状態は次のどれかになる。

| 状態 | 次の作業 |
| --- | --- |
| 一致 | 構成の後の確認へ進む |
| 不一致 | 差分の表の項目を見て、手順 3 と 4 をやり直し、もう一度読み戻す |
| 再確認が必要 | 前回の照合の後に IdMagic の構成か XML 署名鍵が変わった。手順 3 から 5 をやり直す |

権限の不足や Graph の失敗で手順 5 が止まった場合は、ファイルができず、状態は変わらない。
これを未適用と判断しない。
エラーを直して読み戻しをやり直す。

## 確認

照合の一致は、取り込んだ時点の Entra の登録値の事実であり、端末とアプリの動作は別に確かめる。

- sync join の同期待ち：端末の最初の登録は、Entra Connect がコンピューターの `userCertificate` を同期するまで失敗する。次の同期の後まで待つ。
- 端末の参加：端末で `dsregcmd /status` を実行し、`AzureAdJoined : YES` を確かめる。
- PRT：同じ出力で `AzureAdPrt : YES` を確かめる。
- アプリの SSO：参加した端末で Microsoft 365 のアプリを開き、追加のサインインを求められないことを確かめる。

IdMagic は Microsoft Entra hybrid join の即時の登録（`windowstransport`）を提供しない。
実テナントでの受入試験の記録は WI-79 が扱う。

## ドメインの解除

一つのドメインをフェデレーションから外すときに使う。
ほかのドメインのサインインは続く。
IdMagic 側だけを先に外すと、Entra はそのドメインの利用者を IdMagic へ送り続け、IdMagic は発行を拒否するので、そのドメインの全員がサインインできなくなる。
そのため、Entra 側を先に切り戻してから IdMagic 側を外す。
解除のとき、IdMagic は Microsoft のレルム探索に問い合わせ、Entra がまだそのドメインを IdMagic へ送っていれば解除しない。
汎用の RP の削除は Entra の RP を受け付けない。

1. Entra を切り戻す。次のどちらかを行う。
   - Managed に戻す：パスワードハッシュ同期かパススルー認証を先に準備し、クラウドでサインインできる管理者を確保する（[Microsoft の移行の手順](https://learn.microsoft.com/en-us/entra/identity/hybrid/connect/migrate-from-federation-to-cloud-authentication)）。ドメインの詳細の「Managed に切り替えるコマンド」を実行する。変換には最大 60 分かかる。
   - 以前の IdP に戻す：IdMagic を適用する前に「変更前の構成を保存」で書き出したファイルがあるときだけ選べる。IdMagic を適用した後に保存したファイルは IdMagic の値を持つので使えない。「1. 接続」の後、保存したファイルの `IssuerUri`、`PassiveSignInUri`、`ActiveSignInUri`、`MetadataExchangeUri`、`SignOutUri`、`SigningCertificate`、`PreferredAuthenticationProtocol`、`FederatedIdpMfaBehavior` の値で `Update-MgDomainFederationConfiguration -DomainId <ドメイン> -InternalDomainFederationId <現在の構成の Id>` を実行する。
2. 詳細の「ドメインのフェデレーションの解除」で「ドメインを解除」を押し、確認の画面で残るドメインを確かめて確定する。
3. 切り戻し先でサインインできることと、残るドメインでサインインできることを、それぞれ確かめる。

| 解除の結果 | 次の作業 |
| --- | --- |
| 一覧へ移り、解除したことが通知される | 手順 3 へ進む |
| Entra はまだこのドメインの利用者を IdMagic へ送っている | 手順 1 が反映されていない。Managed への変換の直後なら、時間をおいて手順 2 を再実行する。続く場合は手順 1 をやり直す |
| Microsoft に問い合わせて切り戻しを確かめられなかった | 探索の通信が失敗したか、Microsoft が想定しない答えを返した。時間をおいて再実行する。続く場合は、確認だけの接続で「5. 読み戻して保存」を実行してファイルを詳細で取り込み、手順 2 を再実行する。取り込んだ観測が切り戻しを示していれば、その観測で解除する |

最後のドメインを解除すると、Entra の RP も削除する。
XML 署名鍵は残り、後で構成するドメインも同じ鍵を使う。

API のサーバーが `login.microsoftonline.com` へ通信できない構成では、`ENTRA_REALM_DISCOVERY=disabled` にし、解除のたびに手順 1 の後の読み戻しを取り込む。

## 署名鍵の更新

Entra のすべてのドメイン、Entra でない WS-Federation の RP、デフォルトの IdP プロファイルの SAML の SP は、一つの XML 署名鍵を共有する。
鍵は、準備と確認付きの切り替えの二段階で更新する（[REQ-SIGNINGKEYS-006](../modules/signing-keys/features/separation.md#req-signingkeys-006-xml-フェデレーション鍵のローテーション中も既存の信頼関係を検証できる)）。
画面は Entra の一覧か詳細の「署名鍵を更新」から開く。
画面の「同じ鍵を信頼する連携先」で、影響を受ける連携先を先に確かめる。

1. 「次の証明書を準備」を押す。署名に使う鍵は変わらず、次の証明書がメタデータに公開される。画面を離れても、再び開けば同じ準備済みの鍵から続けられる。
2. Entra のドメインごとに、画面の更新のコマンドを実行する。コマンドは `NextSigningCertificate` に次の証明書を持つ。
3. ドメインごとに「5. 読み戻して保存」を実行し、ドメインの詳細で取り込む。画面のドメインが「準備済みの証明書を確認済み」になるまで繰り返す。
4. Entra でない連携先へ次の証明書を登録し、画面で登録した連携先と全体の確認を選ぶ。確認は画面を開くたびにやり直す。
5. 「新しい鍵に切り替える」を押し、確認の画面で準備済みの kid を確かめて確定する。
6. 切り替えの後、Entra のドメインごとに更新のコマンドを再び実行して `SigningCertificate` を新しい証明書にし、読み戻しを取り込む。各ドメインと連携先で、新しくサインインできることを確かめる。Entra でない連携先の現在の証明書も新しい証明書に置き換える。

| 状況 | 次の作業 |
| --- | --- |
| 確定のときに「確認の後に…変わりました」と表示される | 別の管理者が準備を取り消したか、照合や連携先が変わった。最新の状態で手順 2 から確かめ直す |
| 切り替えの前に中止する | 「準備を取り消す」を押す。署名に使う鍵は変わらない。Entra に登録した `NextSigningCertificate` は、取り消した後の更新のコマンドで外す |
| 切り替えの後に、ある連携先でサインインが失敗する | 旧鍵で署名し直す操作はない。その連携先に新しい証明書を登録する。旧鍵の証明書は、画面が示す公開の期限（切り替えから 7 日）までメタデータに残る |

## 失敗したとき

- 適用の後にサインインが失敗する：詳細で読み戻しを取り込み、不一致の項目を確かめる。一致しているのに失敗する場合は、発行した ImmutableID と Entra の `onPremisesImmutableId` を突き合わせる（[信頼性](../modules/ws-federation/features/relying-party.md#信頼性)）。
- 適用を戻す：手順 2 で保存した JSON の値で `Update-MgDomainFederationConfiguration` を実行する。保存した構成が空だった場合（もともと管理されたドメイン）は、[ドメインの解除](#ドメインの解除)の Managed への切り戻しに従う。
- 解除の後に Entra が IdMagic へ戻された：IdMagic は解除したドメインの利用者へ発行しないので、サインインは失敗する。ドメインを構成し直すか、Entra を切り戻し直す。
- 取り込みが「保存済みの確認より前に取得した結果」で拒否される：古いファイルを取り込んでいる。手順 5 をやり直す。
