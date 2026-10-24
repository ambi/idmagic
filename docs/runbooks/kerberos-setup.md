# Kerberos の信頼設定と導入確認

## 着手条件

既存 User に Windows のサインインを追加する場合に使う。
AD の管理者、API の運用管理者と、ブラウザーの管理者が協力して行う。
認証と同期の役割は [Kerberos の機能仕様](../modules/authentication/features/negotiate.md)、方式の比較は [設計判断](../modules/authentication/decisions.md#kerberos-の検証と利用者の同期を分ける)を参照する。

## 最初に確認すること

- 利用者が開く HTTPS のホスト名と証明書を決める。SPN のホスト名をブラウザーが開く名前と合わせる。
- 対象 AD に参加した試験用 PC と、KDC へ到達できるネットワークを用意する。API から AD への常時接続は必要ない。
- 既存 User と KDC の principal が同一人物であることを管理者が確認する。メールアドレスの一致では代用しない。
- 未参加端末でも使える idmagic のパスワードか別 IdP を用意する。AD のパスワードでのフォーム認証と、AD の無効化の自動同期は、この機能には含まれない。
- AD アカウントの削除、名前の再利用、手動の対応付けの失効を担当する運用管理者を決める。

## 実施

### 案内に従って設定する

```sh
mise run setup-kerberos
```

このウィザードは設定と PowerShell を作成し、AD 側の作業、鍵の受け渡し、ブラウザー設定と試験ログインを順に案内する。
AD の管理者パスワードは受け取らない。
生成先は一時ディレクトリーであり、表示された `negotiate.json` をデプロイ時の秘密情報として管理する。

生成した `configure-ad.ps1` を AD 管理用 Windows へ渡し、内容を確認して実行する。
スクリプトは重複する SPN と既存アカウントを拒否し、新規の非管理者サービスアカウントだけを作る。
`ktpass /pass *` に入力するのは、その専用サービスアカウントのパスワードである。
途中で失敗すると専用アカウントだけが残る場合があるため、AD の状態を確認してからやり直す。
準備の根拠は [Microsoft の ktpass](https://learn.microsoft.com/en-us/windows-server/administration/windows-commands/ktpass)で確認できる。

keytab を承認済みの秘密情報の受け渡し経路で API 運用管理者へ渡す。
API の全レプリカへ同じ設定と鍵をデプロイし、ファイルは API の実行ユーザーだけが読める権限にする。
TLS をゲートウェイで終端する場合は、その公開側のリーフ証明書の PEM を配置する。

### 自動試行を無効にしたまま試験する

生成設定は `automatic: false`、出口 IP の条件なしで始まる。
設定の `bindings` は principal の名前と内部 User ID の対応であり、realm は別の項目に指定する。
例として `alice@EXAMPLE.COM` の principal は、realm `EXAMPLE.COM` と名前 `alice` に分ける。
利用者が自身で編集できるプロフィールの属性には保存しない。

```sh
mise run check-kerberos-config -- /path/to/negotiate.json
```

このコマンドは構文、対応付けの重複、SPN、keytab と TLS 証明書の整合を検査する。
AD との通信、実際のチケットの取得、ブラウザーの成功を検査するものではない。
`NEGOTIATE_CONFIG_FILE` にデプロイ先を設定し、API を再起動する。
起動ログの `configuration_sha256` で各レプリカが同じ設定を使っていることを確認する。

管理対象ブラウザーの `AuthServerAllowlist` などへ、対象の公開ホスト名だけを登録する。
ブラウザーのバージョンと OS により設定と channel binding の対応が異なる。
[Microsoft Edge のポリシー](https://learn.microsoft.com/en-us/deployedge/microsoft-edge-browser-policies/authserverallowlist)と [Okta のブラウザー設定例](https://help.okta.com/oie/en-us/Content/Topics/Directory/ad-dsso-configure-browsers.htm)を参照し、対象端末で確認する。
ワイルドカードでインターネット全体へ資格情報を送る設定にはしない。

試験用 PC から Windows のサインインを選び、監査記録と実際の利用先へのサインインを確認する。
試験に成功してから `automatic: true` と `intranet_cidrs` の出口の公開 IP 条件を設定する。
出口 IP は API から到達する先ではなく、API に届いた要求の発信元である。
信頼するゲートウェイだけが転送ヘッダーを設定する構成にし、`TRUSTED_FORWARDED_HOPS` を合わせる。
未設定や解決できない IP では自動試行せず、手動の選択を残す。

### 鍵を更新する

サービスアカウントの鍵を更新すると、AD 側と keytab の KVNO が変わる。
AD 管理者と更新時刻を合わせ、旧新のチケットの有効期間、全レプリカへのデプロイと後退の条件を決める。
旧新 KVNO を併記した keytab を使い、各ファイルは原子的に置き換える。
ファイルの置換は部分的な読取を防ぐが、AD の更新との間を含む無停止のローテーションを保証しない。
鍵の更新中も通常ログインが使えることを確認する。
初期作成用 PowerShell は既存アカウントを変更しないため、ローテーションにそのまま再実行しない。

## 確認

- ドメイン参加 PC でパスワードの入力なくログインし、`NegotiateAuthSucceeded` と完了時の `UserAuthenticated` を確認する。
- OIDC の `amr` は `wia` であり、Kerberos だけでは MFA を満たさないことを確認する。
- 未参加端末、KDC へ到達できない端末、未設定のブラウザーで通常ログインへ戻れることを確認する。
- OS の資格情報ダイアログが表示された場合はキャンセルし、通常ログインを使う。ダイアログをサーバー側で必ず抑制できるとは扱わない。
- 改竄、期限切れ、再送、未知の principal、無効な User が拒否され、新しいセッションを作らないことを確認する。
- realm と SPN を取り違えた設定、異なる TLS 終端証明書、Cookie を許可しないブラウザーを切り分ける。

開発時には次の検査を使う。

```sh
mise run test-kerberos-ad
```

Docker 上に隔離した Samba AD DC を作り、実際の KDC のチケットと PAC を検証してからコンテナを削除する。
Windows のブラウザーの設定と実際の顧客 AD での成功を、この検査で代用しない。

## 失敗したとき

自動試行を無効にし、通常ログインを維持する。
Kerberos 全体を停止する場合は `NEGOTIATE_CONFIG_FILE` を外して API を再起動する。
AD 側で鍵を変更した場合は、古いファイルへ戻すだけでは復旧しないことがあるため、AD 管理者と合わせて対処する。
時計のずれ、SPN の重複、KVNO、TLS 終端証明書、ブラウザーの許可先、共有ストアの順に切り分ける。
keytab、チケット、パスワードと生成設定の内容はログや障害チケットに貼らず、設定の SHA-256 と拒否の分類を共有する。
