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

管理画面の Entra 連携から「ドメインフェデレーションを追加」を開き、ドメイン、sourceAnchor の属性、Entra のテナント ID を入力して保存する。
保存するとドメインの詳細へ移る。
詳細の状態は「未確認」であり、この時点では Entra は何も変わっていない。
取り込み元の User のうち UPN がどのドメインにも属さない人数が表示されたら、取り込みの対応付けを先に確かめる。

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

## 失敗したとき

- 適用の後にサインインが失敗する：詳細で読み戻しを取り込み、不一致の項目を確かめる。一致しているのに失敗する場合は、発行した ImmutableID と Entra の `onPremisesImmutableId` を突き合わせる（[信頼性](../modules/ws-federation/features/relying-party.md#信頼性)）。
- 適用を戻す：手順 2 で保存した JSON の値で `Update-MgDomainFederationConfiguration` を実行する。保存した構成が空だった場合（もともと管理されたドメイン）は、[ドメインの解除](#ドメインの解除)の Managed への切り戻しに従う。
- 解除の後に Entra が IdMagic へ戻された：IdMagic は解除したドメインの利用者へ発行しないので、サインインは失敗する。ドメインを構成し直すか、Entra を切り戻し直す。
- 取り込みが「保存済みの確認より前に取得した結果」で拒否される：古いファイルを取り込んでいる。手順 5 をやり直す。
