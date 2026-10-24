# Directory Connector による取り込み

## 概要

この文書は、閉域の Active Directory と LDAP を取り込み元とする Directory Connector を、テナントに登録し、その呼び出しを認証する仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | Directory Connector の作成、登録、無効化と、Connector の呼び出しの認証 |
| 行為者 | テナント管理者、Directory Connector |
| 扱わないもの | ユーザーとグループの取り込み。Connector の実行環境と LDAP の読み取り |

Directory Connector は、AD と同じ閉域のネットワークに置く実行単位であり、IdMagic へ外向きの接続だけを行う。
IdMagic は AD への到達経路も、AD の資格情報も持たない。

## モデル

| 項目 | 内容 |
| --- | --- |
| ID | IdMagic が採番する Connector の ID。Connector の JWT の `iss` と `sub` になる |
| 表示名 | 管理者が付ける名前。前後の空白を除いて保存し、200 文字まで |
| 状態 | `AwaitingEnrollment`、`Active`、`Disabled` |
| 登録トークン | 一回だけ使える、有効期限付きのシークレット。IdMagic は SHA-256 のハッシュと有効期限だけを保存する |
| 公開鍵 | 登録で受け取った JWK。P-256 の EC 鍵か 2048 ビット以上の RSA 鍵であり、秘密鍵の成分を持たない |
| 鍵の ID | 公開鍵の RFC 7638 のサムプリント |
| 時刻 | 作成、登録、最終接続、無効化の時刻 |

一つの Connector が一つの取り込み元を表し、取り込み元（[用語集](../glossary.md)の IdentitySource）の関連付けと資格情報を兼ねる。

Connector の呼び出しは、Connector の秘密鍵で署名した JWT を `Authorization: Bearer` で提示して認証する。
JWT の `aud` には、テナントの発行者識別子に `/directory-connector/v1` を続けた値を使う。

- **判断**：mTLS を採らないのは、エッジに証明書の転送の設定を求めず、Connector の秘密鍵を閉域から出さずに、鍵の更新と失効を IdMagic 側の登録の差し替えだけで行うためである。

## 状態遷移

### DirectoryConnectorLifecycle

作成直後の Connector は登録待ちであり、登録トークンで公開鍵を登録すると呼び出せるようになる。
再発行は登録済みの公開鍵を破棄して登録待ちへ戻す。
無効化は終端であり、置き換えるときは新しい Connector を作る。

| State | Kind | Meaning |
|---|---|---|
| AwaitingEnrollment | initial | 登録トークンを発行し、公開鍵の登録を待っている |
| Active | — | 公開鍵を登録し、その鍵で署名した JWT で呼び出せる |
| Disabled | terminal | 管理者が無効化した。登録も呼び出しも受け付けない |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| AwaitingEnrollment | Connector による登録 | 登録トークンが一致し、有効期限内であり、公開鍵が条件を満たす | Active | DirectoryConnectorEnrolled |
| AwaitingEnrollment | 管理者による登録トークンの再発行 | — | AwaitingEnrollment | DirectoryConnectorEnrollmentReissued |
| Active | 管理者による登録トークンの再発行 | — | AwaitingEnrollment | DirectoryConnectorEnrollmentReissued |
| AwaitingEnrollment | 管理者による Connector の無効化 | — | Disabled | DirectoryConnectorDisabled |
| Active | 管理者による Connector の無効化 | — | Disabled | DirectoryConnectorDisabled |

| State | Connector による登録 | Connector による接続の報告 | 管理者による登録トークンの再発行 | 管理者による Connector の無効化 |
|---|---|---|---|---|
| AwaitingEnrollment | → Active（登録トークンが一致し、有効期限内であり、公開鍵が条件を満たす）<br>拒否：401 invalid_token（登録トークンが一致しないか、有効期限を過ぎている）<br>拒否：400 invalid_request（公開鍵が条件を満たさない） | 拒否：401 invalid_token | → AwaitingEnrollment | → Disabled |
| Active | 拒否：401 invalid_token | 何もしない | → AwaitingEnrollment | → Disabled |
| Disabled | 拒否：401 invalid_token | 拒否：401 invalid_token | 拒否：409 directory_connector_disabled | 何もしない |

## 操作

### 管理者による Connector の作成

#### REQ-SOURCING-008 Directory Connector の作成は、登録待ちの Connector と一回だけ使える登録トークンを返す

- 管理者が表示名を指定して Directory Connector を作成したとき、Sourcing は、`AwaitingEnrollment` の Connector を作り、201 で Connector と登録トークンを返し、`DirectoryConnectorCreated` を発行する。
- 登録トークンを発行したとき、Sourcing は、`idmagic_dce_` に 32 バイトの乱数の base64url を続けた値を作り、発行時刻の 1 時間後を有効期限として記録する。
- 登録トークンを発行したとき、Sourcing は、トークンを応答で一度だけ返し、SHA-256 のハッシュだけを保存する。
- 表示名が空か空白だけの場合、Sourcing は、400 と `invalid_request` で拒否し、Connector を作らない。
- 表示名が 200 文字を超える場合、Sourcing は、422 と `field_length_exceeded` で拒否し、Connector を作らない。
- `admin` または `system_admin` のロールを持たない利用者が作成を要求した場合、Sourcing は、403 と `access_denied` で拒否し、Connector を作らない。
- **判断**：再発行がいつでもできるので、有効期間を長くする利点より、漏えいしたトークンで偽の Connector を登録される期間を短くすることを選ぶ。

### 管理者による Connector の一覧

#### REQ-SOURCING-013 Directory Connector の一覧は、テナントの Connector を作成順に、シークレットを含めずに返す

- 管理者が Directory Connector を一覧したとき、Sourcing は、テナントのすべての Connector を作成時刻の昇順に返す。
- 管理者が Directory Connector を一覧または取得したとき、Sourcing は、各 Connector の ID、表示名、状態、作成時刻と、記録した登録時刻、最終接続時刻、無効化時刻を返す。
- `Active` の Connector を一覧または取得したとき、Sourcing は、鍵の ID を返す。
- `AwaitingEnrollment` の Connector を一覧または取得したとき、Sourcing は、登録トークンの有効期限を返す。
- 管理者が Directory Connector を一覧または取得したとき、Sourcing は、登録トークンとそのハッシュを返さない。
- `admin` または `system_admin` のロールを持たない利用者が一覧を要求した場合、Sourcing は、403 と `access_denied` で拒否する。

### 管理者による Connector の取得

#### REQ-SOURCING-014 Directory Connector の取得は、要求したテナントの Connector だけを返す

- **上位の要件**：[REQ-SOURCING-013](directory.md#管理者による-connector-の一覧)
- 管理者が Directory Connector を取得したとき、Sourcing は、200 でその Connector を返す。
- 存在しないか別のテナントの Connector の ID を指定された場合、Sourcing は、404 と `directory_connector_not_found` で拒否する。
- `admin` または `system_admin` のロールを持たない利用者が取得を要求した場合、Sourcing は、403 と `access_denied` で拒否する。

### Connector による登録

#### REQ-SOURCING-009 Directory Connector の登録は、有効な登録トークンと公開鍵で Connector を `Active` にする

- `AwaitingEnrollment` の Connector の有効期限内の登録トークンと公開鍵の JWK を受けたとき、Sourcing は、Connector を `Active` にし、公開鍵と鍵の ID と登録時刻を保存し、登録トークンのハッシュを消し、`DirectoryConnectorEnrolled` を発行する。
- Connector を登録したとき、Sourcing は、200 で Connector の ID、鍵の ID、JWT の `aud` に使う値を返す。
- 登録トークンが要求したテナントの `AwaitingEnrollment` の Connector のものでないか、有効期限を過ぎている場合、Sourcing は、401 と `invalid_token` で拒否し、どの Connector も変えない。
- 公開鍵が P-256 の EC 鍵でも 2048 ビット以上の RSA 鍵でもないか、秘密鍵の成分を含む場合、Sourcing は、400 と `invalid_request` で拒否し、登録トークンを使用済みにしない。
- **判断**：使用済み、期限切れ、無効化済みのトークンを同じ応答で拒否し、トークンを持たない呼び出し元に Connector の有無と状態を明かさない。

### Connector による接続の報告

#### REQ-SOURCING-010 Directory Connector の呼び出しは、登録した鍵で署名した短命の JWT だけを認証する

- `Active` の Connector が登録した鍵で署名した JWT を提示して接続を報告したとき、Sourcing は、Connector の最終接続時刻を記録し、204 を返す。
- Sourcing は、`iss` と `sub` が Connector の ID であり、`aud` が登録の応答の値であり、`alg` が `ES256` か `PS256` であり、`exp` が検証の時刻の 5 分後までにあり、`jti` が未使用である JWT だけを、時計のずれを前後 60 秒まで許して認証する。
- JWT を認証したとき、Sourcing は、`jti` を使用済みとして 6 分間記録する。
- JWT がないか、形式、署名、`aud`、`exp`、`alg` のいずれかが条件を満たさない場合、Sourcing は、401 と `invalid_token` で拒否し、最終接続時刻を変えない。
- 使用済みの `jti` を持つ JWT を提示された場合、Sourcing は、401 と `invalid_token` で拒否し、最終接続時刻を変えない。
- JWT の `iss` が要求したテナントの `Active` の Connector でない場合、Sourcing は、401 と `invalid_token` で拒否し、最終接続時刻を変えない。

### 管理者による登録トークンの再発行

#### REQ-SOURCING-011 登録トークンの再発行は、登録済みの鍵を破棄して Connector を `AwaitingEnrollment` に戻す

- **上位の要件**：[REQ-SOURCING-008](directory.md#管理者による-connector-の作成)
- `AwaitingEnrollment` か `Active` の Connector の登録トークンを管理者が再発行したとき、Sourcing は、Connector を `AwaitingEnrollment` にし、以前の登録トークンと公開鍵を破棄し、201 で Connector と新しい登録トークンを返し、`DirectoryConnectorEnrollmentReissued` を発行する。
- `Disabled` の Connector の再発行を要求された場合、Sourcing は、409 と `directory_connector_disabled` で拒否し、Connector を変えない。
- 存在しないか別のテナントの Connector の ID を指定された場合、Sourcing は、404 と `directory_connector_not_found` で拒否する。
- `admin` または `system_admin` のロールを持たない利用者が再発行を要求した場合、Sourcing は、403 と `access_denied` で拒否し、Connector を変えない。
- **判断**：鍵の更新と、漏えいが疑われる鍵の失効を、同じ一つの操作で行えるようにする。

### 管理者による Connector の無効化

#### REQ-SOURCING-012 Directory Connector の無効化は、以後の登録と呼び出しを拒否する

- `AwaitingEnrollment` か `Active` の Connector を管理者が無効化したとき、Sourcing は、Connector を `Disabled` にし、登録トークンのハッシュを消し、無効化時刻を記録し、200 で Connector を返し、`DirectoryConnectorDisabled` を発行する。
- `Disabled` の Connector を管理者が無効化したとき、Sourcing は、200 で Connector を返し、Connector を変えず、イベントを発行しない。
- 存在しないか別のテナントの Connector の ID を指定された場合、Sourcing は、404 と `directory_connector_not_found` で拒否する。
- `admin` または `system_admin` のロールを持たない利用者が無効化を要求した場合、Sourcing は、403 と `access_denied` で拒否し、Connector を変えない。

## セキュリティ上の考慮

Connector の管理 API は、`admin` または `system_admin` のロールを持つ、有効かつ認証済みの利用者の対話のセッションに限り、API アクセストークンでは到達できない。
作成と再発行は Connector の資格情報の元になるシークレットを返すので、API アクセストークンの発行と同じ扱いにする。

Connector の呼び出しのテナントは、要求の経路が解決したテナントで決まる。
JWT の `iss` は、そのテナントの Connector の ID として引き、別のテナントの Connector は存在しないものとして扱う。
`aud` を発行者識別子そのものにしないので、Connector の JWT は、同じ発行者の別の経路の受け手に通用しない。

IdMagic が保存するのは、登録トークンのハッシュと公開鍵だけである。
Connector の秘密鍵、AD の資格情報、利用者のパスワードは、IdMagic のデータベース、API の応答、イベントのどれにも含まれない。
