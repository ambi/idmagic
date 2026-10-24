# Directory Connector の実行環境

## 概要

この文書は、閉域の Active Directory と OpenLDAP を読み、Sourcing の [Directory Connector による取り込み](../../sourcing/features/directory.md)の Connector API へ差分を送る実行単位 `idmagic-directory-connector` の仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | 公開鍵の登録、LDAP の接続と bind、実行の種類の選択、エントリから User とコンピューターの項目への対応付け、バッチの送信と再試行、差分の位置の保存 |
| 行為者 | Connector を配置する運用者、Directory Connector |
| 扱わないもの | 取り込んだ項目の適用と実行の記録は Sourcing の [Directory Connector による取り込み](../../sourcing/features/directory.md)が扱う。グループと OpenLDAP のコンピューターの読み取り、AD への書き戻し、パスワードの取り込み |

Connector は、AD と同じ閉域に置く単体のバイナリであり、IdMagic へ外向きの HTTPS だけを行う。
状態は、Connector の手元の状態のディレクトリだけに置く。

## モデル

### 設定

Connector は、起動の引数で与えた YAML の設定ファイルと、そこから参照する bind のパスワードのファイルだけを読み、環境変数を読まない。

| 項目 | 内容 | デフォルト値 |
| --- | --- | --- |
| `idmagic.realm_url` | テナントの realm の `https://` の URL。Connector API はこの URL に `/directory-connector/v1` を続けた位置にある | 必須 |
| `idmagic.ca_file` | IdMagic のサーバーの証明書を検証する CA の PEM。省くと OS の信頼する CA を使う | なし |
| `idmagic.proxy_url` | IdMagic への HTTPS を中継する HTTP プロキシの `http://` か `https://` の URL。LDAP への接続には使わない | なし |
| `state_dir` | 秘密鍵、登録の情報、差分の位置を置くディレクトリ | 必須 |
| `ldap.url` | `ldaps://` か `ldap://` の URL | 必須 |
| `ldap.start_tls` | `ldap://` の URL で StartTLS を使うかどうか | `false` |
| `ldap.ca_file` | サーバーの証明書を検証する CA の PEM。省くと OS の信頼する CA を使う | なし |
| `ldap.server_name` | 証明書の検証に使うサーバーの名前。省くと URL のホスト名を使う | なし |
| `ldap.dialect` | `active_directory` か `openldap` | 必須 |
| `ldap.bind_dn` | サービスアカウントの識別名 | 必須 |
| `ldap.bind_password_file` | サービスアカウントのパスワードを置いたファイル。末尾の改行を除いて使う | 必須 |
| `ldap.base_dns` | 検索の起点の識別名の一覧。1 件以上で、どの起点もほかの起点の配下にない | 必須 |
| `ldap.user_filter` | User を選ぶ LDAP のフィルター | 方言のデフォルト値 |
| `ldap.page_size` | ページングした検索の 1 ページの件数。1 から 1000 | `500` |
| `ldap.attributes` | 正規の属性名から LDAP の属性名への対応のうち、デフォルト値を上書きするもの | なし |
| `computers.base_dns` | コンピューターを検索する起点の識別名の一覧。`computers` の節を書くときは 1 件以上で、どの起点もほかの起点の配下にない | 節がなければコンピューターを読まない |
| `computers.filter` | コンピューターを選ぶ LDAP のフィルター | `(objectCategory=computer)` |
| `sync.interval` | 同期を繰り返す間隔 | `5m` |
| `sync.full_interval` | 完全な再同期を行う間隔 | `24h` |

### 状態のディレクトリ

| ファイル | 内容 |
| --- | --- |
| `connector-key.pem` | Connector の P-256 の秘密鍵。PKCS#8 の PEM |
| `enrollment.json` | 登録の応答の Connector の ID、鍵の ID、JWT の `aud` |
| `cursor.json` | 差分の位置 |

### 差分の位置

差分の位置（Sourcing の[用語集](../../sourcing/glossary.md)の SourceCursor）は、最後に完了した実行が読んだ範囲と、次の差分の起点を持つ。

| 項目 | 内容 |
| --- | --- |
| 方言 | 位置を記録したときの `ldap.dialect` |
| 検索の範囲 | 位置を記録したときの `ldap.base_dns` と、方言のデフォルト値を補った `ldap.user_filter` |
| コンピューターの検索の範囲 | 位置を記録したときの `computers.base_dns` と、デフォルト値を補った `computers.filter`。`computers` の節がなければ空 |
| サーバー | AD の RootDSE の `dsServiceName`。OpenLDAP では空 |
| uSN | AD の実行を始める前に RootDSE から読んだ `highestCommittedUSN` |
| 変更の起点 | OpenLDAP の実行で検索を始めた時刻の 5 分前 |
| 最後の完全な再同期 | 最後に完了した `full` の実行を始めた時刻 |

### 方言ごとのデフォルト値

| 項目 | `active_directory` | `openldap` |
| --- | --- | --- |
| User のフィルター | `(&(objectCategory=person)(objectClass=user))` | `(objectClass=inetOrgPerson)` |
| 外部の不変 ID | `objectGUID` の 16 バイトを標準の base64（RFC 4648 の 4 節、パディング付き）にした値 | `entryUUID` の文字列 |
| 有効かどうか | `userAccountControl` の `ACCOUNTDISABLE`（`0x2`）が立っていなければ有効 | 常に有効 |
| `preferred_username` | `sAMAccountName` | `uid` |
| `name` | `displayName` | `cn` |
| `given_name` | `givenName` | `givenName` |
| `family_name` | `sn` | `sn` |
| `email` | `mail` | `mail` |
| `employee_number` | `employeeNumber` | `employeeNumber` |
| `department` | `department` | `departmentNumber` |

- **判断**：`objectGUID` を base64 にするのは、Entra Connect が `objectGUID` を sourceAnchor にするときと同じ表記にし、後の Entra の連携で同じ値を使えるようにするためである。

### コンピューターの項目

`computers` の節は `active_directory` の方言だけで使え、属性の対応を上書きできない。

| 項目 | AD の値 |
| --- | --- |
| 外部の不変 ID | `objectGUID` の 16 バイトを標準の base64 にした値 |
| 有効かどうか | `userAccountControl` の `ACCOUNTDISABLE`（`0x2`）が立っていなければ有効 |
| `sam_account_name` | `sAMAccountName` |
| `object_sid` | `objectSid` の二進表現を、`S-<revision>-<identifier authority>` に副権限を 10 進で `-` に続けて並べた文字列にした値。identifier authority は 6 バイトのビッグエンディアン、副権限はそれぞれ 4 バイトのリトルエンディアンで読む（[MS-DTYP 2.4.2](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-dtyp/f992ad60-0fe4-4b87-9fed-beb478836861)） |
| `domain` | エントリの識別名の `DC=` の値を、現れた順に `.` でつないで小文字にした値 |
| `dns_host_name` | `dNSHostName` |

## 操作

### 運用者による登録

#### REQ-DIRECTORYCONNECTOR-001 登録のコマンドは、手元で作った鍵対の公開鍵だけを送り、秘密鍵を状態のディレクトリだけに置く

- 運用者が登録のコマンドを実行したとき、DirectoryConnector は、登録トークンを標準入力から 1 行読み、新しい P-256 の鍵対を作り、公開鍵の JWK と登録トークンを登録の API へ送る。
- 登録の API が 200 を返したとき、DirectoryConnector は、秘密鍵と登録の応答を状態のディレクトリへ所有者だけが読み書きできる権限で保存し、以前の秘密鍵と登録の情報を置き換える。
- DirectoryConnector は、登録トークンを状態のディレクトリ、ログ、標準出力のどれにも書かない。
- 登録の API が 200 以外を返した場合、DirectoryConnector は、状態のディレクトリを変えず、状態コードとエラーの種類を標準エラーへ出し、0 以外で終了する。
- **判断**：再発行した登録トークンでの登録も新しい鍵対を作る。鍵の更新と失効をサーバー側の再発行（REQ-SOURCING-011）と同じ一つの手順にするためである。

### 同期

#### REQ-DIRECTORYCONNECTOR-002 Connector は、TLS で検証した LDAP の接続だけで bind し、bind のパスワードを外へ出さない

- DirectoryConnector は、`ldaps://` の URL では接続の確立時に、`ldap://` の URL では StartTLS で TLS を確立してから、`ldap.bind_dn` と bind のパスワードで simple bind する。
- `ldap://` の URL で `ldap.start_tls` が `true` でない場合、DirectoryConnector は、起動を拒否し、LDAP へ接続しない。
- サーバーの証明書を構成した CA か OS の信頼する CA で検証できない場合、DirectoryConnector は、bind せずにその回の同期を失敗させ、差分の位置を変えない。
- DirectoryConnector は、bind のパスワードを、ログ、IdMagic への要求、状態のディレクトリのどれにも含めない。
- 設定ファイルに未知のキーがあるか、必須の項目がないか、`idmagic.realm_url` が `https://` でない場合、DirectoryConnector は、起動を拒否し、LDAP にも IdMagic にも接続しない。
- `ldap.base_dns` のある起点が別の起点と同じか、その配下にある場合、DirectoryConnector は、起動を拒否し、LDAP にも IdMagic にも接続しない。
- `computers` の節があり、`ldap.dialect` が `active_directory` でないか、`computers.base_dns` のある起点が別の起点と同じかその配下にある場合、DirectoryConnector は、起動を拒否し、LDAP にも IdMagic にも接続しない。
- **判断**：重なる起点を拒否するのは、同じエントリを二度送ると、完全な再同期で送った項目の数と取り込み元の件数が食い違うからである。

#### REQ-DIRECTORYCONNECTOR-003 同期は、差分の位置と完全な再同期の間隔から実行の種類を選ぶ

- 同期の各回を始めたとき、DirectoryConnector は、LDAP を読む前に接続の報告の API を呼ぶ。
- 同期の各回で実行を始めるとき、DirectoryConnector は、次の表で実行の種類と読む User を選ぶ。

| 条件 | 種類 | 読む User |
| --- | --- | --- |
| 位置がない | `full` | 検索の範囲の全件 |
| 位置の方言、検索の範囲、コンピューターの検索の範囲のどれかが設定と異なる | `full` | 検索の範囲の全件 |
| `active_directory` で、位置のサーバーが接続先の `dsServiceName` と異なる | `full` | 検索の範囲の全件 |
| 最後の完全な再同期から `sync.full_interval` を過ぎた | `full` | 検索の範囲の全件 |
| 上のどれにも当たらない `active_directory` | `incremental` | `uSNChanged` が位置の uSN より大きい User |
| 上のどれにも当たらない `openldap` | `incremental` | `modifyTimestamp` が位置の変更の起点以後の User |

- `active_directory` の実行を始めるとき、DirectoryConnector は、検索の前に RootDSE の `highestCommittedUSN` と `dsServiceName` を読み、次の位置にする。
- `openldap` の実行を始めるとき、DirectoryConnector は、検索を始める時刻の 5 分前を次の位置の変更の起点にする。
- `incremental` の実行では、DirectoryConnector は、削除の項目を送らない。
- **判断**：削除は `uSNChanged` と `modifyTimestamp` に現れないので、完全な再同期の完了でサーバーが現れなかった User を無効にする（REQ-SOURCING-018）ことで回収する。DirSync は削除を返すが、サービスアカウントに Replicating Directory Changes の権限を求める。
- **判断**：OpenLDAP の起点を 5 分前にするのは、検索の最中の変更と時計のずれを重ねて読むためである。重ねて読んだ項目は、サーバーが `unchanged` として User を書かない（REQ-SOURCING-016）。

#### REQ-DIRECTORYCONNECTOR-004 同期は、検索の範囲のエントリを User の項目へ写し、100 件ずつ送る

- DirectoryConnector は、`ldap.base_dns` の起点を並べた順に、それぞれの配下を `ldap.user_filter` で `ldap.page_size` 件ずつページングして検索し、対応付けに使う属性だけを要求する。
- DirectoryConnector は、各エントリを[方言ごとのデフォルト値](#方言ごとのデフォルト値)と `ldap.attributes` の対応で User の項目へ写し、多値の属性は最初の値を使い、値のない属性を空にする。
- DirectoryConnector は、写した項目を読んだ順に 100 件ずつ、`<実行の ID>-<1 から始まる連番>` のバッチの ID で送り、最後に送った項目の数で実行の完了を報告する。
- 外部の不変 ID か `preferred_username` の属性がないエントリの場合、DirectoryConnector は、そのエントリを送らず、識別名と欠けた属性の名前をログへ出し、残りのエントリの同期を続ける。
- DirectoryConnector は、属性の値をログへ出さない。

#### REQ-DIRECTORYCONNECTOR-005 同期は、一時的な失敗を同じバッチで送り直し、完了が受理されてから位置を進める

- DirectoryConnector は、Connector API の要求ごとに、新しい `jti` と 2 分後の `exp` を持つ JWT を状態のディレクトリの秘密鍵で署名する。
- 要求が接続の失敗か、408、429、5xx で失敗したとき、DirectoryConnector は、新しい JWT で同じ要求を、1 秒から倍にした間隔で、最初の要求を含めて 5 回まで送る。
- 実行の完了の報告に 200 が返ったとき、DirectoryConnector は、次の位置を状態のディレクトリへ保存する。
- 再試行を使い切るか、408 と 429 以外の 4xx が返った場合、DirectoryConnector は、その回の同期を中断し、位置を変えず、状態コードとエラーの種類をログへ出す。
- 同期を中断した後の次の回では、DirectoryConnector は、保存した位置から新しい実行を始める。
- Connector を常駐させたとき、DirectoryConnector は、前の回が終わってから `sync.interval` を待って次の回を始め、停止の合図を受けたら送信中の要求を待たずに終了する。
- **判断**：IdMagic は位置を持たず（REQ-SOURCING-015）、実行の途中から再開する契約がないので、中断した実行の続きではなく、同じ位置からの新しい実行で回収する。古い実行はサーバー側で `superseded` になる。

#### REQ-DIRECTORYCONNECTOR-007 `computers` の節があれば、同期は User の後に AD のコンピューターを送る

- **上位の要件**：[REQ-DIRECTORYCONNECTOR-004](runtime.md#同期)
- `computers` の節を持つ同期で User の項目を送り終えたとき、DirectoryConnector は、`computers.base_dns` の起点を並べた順に、それぞれの配下を `computers.filter` で `ldap.page_size` 件ずつページングして検索し、[コンピューターの項目](#コンピューターの項目)に使う属性だけを要求する。
- `incremental` の実行では、DirectoryConnector は、User と同じ位置の uSN より `uSNChanged` が大きいコンピューターだけを読む。
- DirectoryConnector は、写したコンピューターの項目を読んだ順に 100 件ずつ、`<実行の ID>-computers-<1 から始まる連番>` のバッチの ID でコンピューターのバッチとして送る。
- DirectoryConnector は、User とコンピューターの送った項目の数の和で実行の完了を報告する。
- `objectGUID`、`userAccountControl`、`sAMAccountName`、`objectSid` のどれかがないか読めないか、識別名に `DC=` がないエントリの場合、DirectoryConnector は、そのエントリを送らず、識別名と欠けた属性の名前をログへ出し、残りのエントリの同期を続ける。
- `computers` の節がない同期では、DirectoryConnector は、コンピューターを検索せず、コンピューターのバッチを送らない。
- **判断**：コンピューターを User と同じ実行で送るのは、完全な再同期の完了で、現れなかったコンピューターも同じ実行の項目数の照合の後に無効にする（REQ-SOURCING-018）ためである。

### 運用者による設定の確認

#### REQ-DIRECTORYCONNECTOR-006 設定の確認は、IdMagic へ送らずに、送るはずの項目の数と送らないエントリを示す

- 運用者が設定の確認のコマンドを実行したとき、DirectoryConnector は、同期と同じ接続、検索、対応付けで検索の範囲の全件を読み、写せた項目の数と、送らないエントリの識別名と欠けた属性の名前を標準出力へ出す。
- `computers` の節を持つ設定を確認したとき、DirectoryConnector は、コンピューターの検索の範囲の全件も読み、写せたコンピューターの項目の数と送らないエントリを User と分けて出す。
- 運用者が設定の確認のコマンドを実行したとき、DirectoryConnector は、IdMagic へ接続せず、登録を求めず、差分の位置を変えない。
- 設定の確認で LDAP への接続、bind、検索のどれかが失敗した場合、DirectoryConnector は、失敗を標準エラーへ出し、0 以外で終了する。
- **判断**：IdMagic へ送る前に、検索の範囲とフィルターの誤りで多数の User が完全な再同期に現れない事態を、運用者が件数で確かめられるようにする。

## セキュリティ上の考慮

Connector の秘密鍵と bind のパスワードは、Connector の手元のファイルだけにある。
状態のディレクトリとパスワードのファイルは、Connector を動かすアカウントだけが読めるように置く。
Connector はどの待ち受けのポートも開かず、IdMagic と LDAP への外向きの接続だけを行う。

LDAP の接続は、証明書を検証した TLS に限る。
検証を省く設定は持たない。
平文の LDAP で bind すると、閉域の中で bind のパスワードを読まれ、偽のサーバーに渡すおそれがあるためである。

サービスアカウントには、検索の範囲の User と、`computers` の節を書くときはコンピューターの読み取り権限だけを与える。
`uSNChanged` と `modifyTimestamp` の差分は、通常の読み取り権限で読める。

## 実現方式

導入、登録、鍵の更新、障害時の運用の手順は [Directory Connector の導入と運用](../../../runbooks/directory-connector.md)が扱う。
