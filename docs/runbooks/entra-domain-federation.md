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

## 失敗したとき

- 適用の後にサインインが失敗する：詳細で読み戻しを取り込み、不一致の項目を確かめる。一致しているのに失敗する場合は、発行した ImmutableID と Entra の `onPremisesImmutableId` を突き合わせる（[信頼性](../modules/ws-federation/features/relying-party.md#信頼性)）。
- 適用を戻す：手順 2 で保存した JSON の値で `Update-MgDomainFederationConfiguration` を実行する。保存した構成が空だった場合（もともと管理されたドメイン）の解除は、ドメイン単位の解除の手順が扱う。
- 取り込みが「保存済みの確認より前に取得した結果」で拒否される：古いファイルを取り込んでいる。手順 5 をやり直す。
