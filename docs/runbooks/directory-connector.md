# Directory Connector の導入と運用

## 概要

この手順書は、利用者の閉域に `idmagic-directory-connector` を配置し、IdMagic に登録して、AD または OpenLDAP の User を取り込み続けるための運用作業を扱う。
Connector の振る舞いは [Directory Connector の実行環境](../modules/directory-connector/features/runtime.md)が、IdMagic 側の取り込みは [Directory Connector による取り込み](../modules/sourcing/features/directory.md)が定める。

## 配置

| 項目 | 要件 |
| --- | --- |
| ホスト | AD の DC または OpenLDAP のサーバーへ到達できる、閉域の Linux か Windows のホスト。DC そのものに置かない |
| 外向きの通信 | IdMagic の realm の URL への HTTPS（443）と、LDAP のサーバーへの LDAPS（636）か StartTLS の LDAP（389） |
| 内向きの通信 | なし。Connector は待ち受けのポートを開かない |
| 実行のアカウント | Connector だけに使う OS のアカウント。状態のディレクトリとパスワードのファイルを、このアカウントだけが読めるようにする |
| 配置の数 | 一つの Connector の登録につき一つ。同じ登録を二台で同時に動かさない。新しい実行が他方の実行を打ち切り合うためである |

## 最小権限のサービスアカウント

| ディレクトリ | 与える権限 |
| --- | --- |
| Active Directory | 通常のドメインユーザー。検索の範囲の OU に対するデフォルトの読み取り権限で足り、Replicating Directory Changes などの追加の権限は与えない |
| OpenLDAP | `ldap.base_dns` の配下の User の、対応付けに使う属性、`entryUUID`、`modifyTimestamp` の読み取り（`read`）だけを許す ACL |

サービスアカウントのパスワードは、パスワードのファイルに 1 行で置く。

## 導入

1. バイナリを `/opt/idmagic-directory-connector/`（Windows では `C:\Program Files\IdMagic\DirectoryConnector\`）へ置く。
2. 状態のディレクトリ（例 `/var/lib/idmagic-directory-connector`）を作り、実行のアカウントだけが読み書きできる権限（`0700`）にする。
3. パスワードのファイルを作り、実行のアカウントだけが読める権限（`0600`）にする。
4. 設定ファイルを作る。

```yaml
idmagic:
  realm_url: https://idp.example.com/realms/acme
state_dir: /var/lib/idmagic-directory-connector
ldap:
  url: ldaps://dc1.corp.example.com:636
  dialect: active_directory
  ca_file: /etc/idmagic-directory-connector/corp-ca.pem
  bind_dn: CN=svc-idmagic,OU=Service Accounts,DC=corp,DC=example,DC=com
  bind_password_file: /etc/idmagic-directory-connector/bind-password
  base_dns:
    - OU=Staff,DC=corp,DC=example,DC=com
    - OU=Contractors,DC=corp,DC=example,DC=com
sync:
  interval: 5m
  full_interval: 24h
```

StartTLS を使う場合は、`url` を `ldap://dc1.corp.example.com:389` にし、`start_tls: true` を足す。
平文の LDAP で bind する設定は起動時に拒否される。
`base_dns` には同期する OU を並べる。互いに含み合う OU は拒否されるので、親の OU だけを書く。
IdMagic へ HTTP プロキシを通して出る場合は、`idmagic:` の下に `proxy_url: http://proxy.corp.example.com:8080` を足す。LDAP への接続はプロキシを通らない。
Microsoft Entra hybrid join のために AD のコンピューターアカウントも取り込む場合は、最上位に次の節を足す。`active_directory` の方言でだけ使える。

```yaml
computers:
  base_dns:
    - CN=Computers,DC=corp,DC=example,DC=com
```

`filter` を省くと `(objectCategory=computer)` を使う。サービスアカウントには、この起点の配下のコンピューターの読み取り権限も与える。

5. IdMagic に送る前に、設定の確認のコマンドで LDAP の読み取りを確かめる。IdMagic へは何も送らず、登録も要らない。

```sh
idmagic-directory-connector check -config /etc/idmagic-directory-connector/config.yaml
```

出力の送るはずの User の数が AD 上の対象の人数と大きく違う場合は、`base_dns` と `user_filter` を直してから登録へ進む。
完全な再同期は、現れなかった User を IdMagic で無効にするためである。

## 登録

1. テナントの管理者が、管理 API の `POST /api/admin/v1/directory-connectors` で Connector を作り、応答の登録トークンを受け取る。トークンは 1 時間で期限が切れる。
2. Connector のホストで、実行のアカウントとして登録のコマンドを実行し、標準入力へ登録トークンを渡す。

```sh
idmagic-directory-connector enroll -config /etc/idmagic-directory-connector/config.yaml
```

3. 成功すると、状態のディレクトリに `connector-key.pem` と `enrollment.json` ができる。登録トークンはどこにも保存されない。

## 起動

常駐させるときは `run` を、一回だけ同期するときは `run -once` を使う。

```sh
idmagic-directory-connector run -config /etc/idmagic-directory-connector/config.yaml
```

Linux では systemd のユニットとして、Windows ではタスク スケジューラの「起動時」のタスクとして、実行のアカウントで動かす。
ログは JSON Lines で標準エラーへ出るので、サービスマネージャーのログへ集める。

## 鍵の更新と失効

1. 管理者が管理 API の `POST /api/admin/v1/directory-connectors/{connector_id}/enrollment-tokens` で登録トークンを再発行する。この時点で以前の鍵は使えなくなる。
2. Connector を止め、新しい登録トークンで登録のコマンドを実行する。新しい鍵対が作られ、以前の鍵を置き換える。
3. Connector を起動する。差分の位置は残るので、次の同期は前回の続きから始まる。

ホストを失った、または鍵の漏えいが疑われるときも同じ手順で失効させる。
Connector を廃止するときは、管理 API で Connector を無効化し、状態のディレクトリを消す。

## 障害時の対応

| 症状 | 原因の候補 | 対応 |
| --- | --- | --- |
| ログに TLS の検証の失敗が出る | `ca_file` が DC の証明書の発行者と合わない、`server_name` が証明書の名前と合わない | 正しい CA を置くか `server_name` を直す。検証を省く設定はない |
| ログに `invalid_token` の 401 が出る | 再発行で鍵が失効した、Connector が無効化された、ホストの時計が 60 秒より大きくずれている | 時計を合わせる。直らなければ鍵の更新の手順で登録し直す |
| ログに bind の失敗が出る | サービスアカウントのパスワードの期限切れ、ロック | AD 側で直し、パスワードのファイルを更新する。Connector の再起動は要らない |
| 同期が中断を繰り返す | IdMagic への到達の失敗、IdMagic の 5xx | 再試行は自動で行う。中断しても差分の位置は進まないので、復旧後の次の同期で回収される |
| 多数の User が無効になった | `base_dns` か `user_filter` の誤りで、完全な再同期に User が現れなかった | `check` で件数を確かめて設定を直す。次の完全な再同期で現れた User は再び有効になる。検索の範囲を変えると次の同期は自動で完全な再同期になる |
| AD で削除した User が IdMagic に残る | 差分は削除を読まない | 次の完全な再同期で無効になる。急ぐ場合は Connector を止め、状態のディレクトリの `cursor.json` を消して起動し、完全な再同期を行わせる |
| DC を入れ替えた | uSN は DC ごとの値である | 対応は要らない。接続先の DC が変わると、次の同期は自動で完全な再同期になる |

取り込みの結果は、管理 API の取り込みの実行の一覧と詳細（REQ-SOURCING-019）で、件数と失敗した項目を確かめる。
