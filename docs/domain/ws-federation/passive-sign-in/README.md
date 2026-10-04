# パッシブサインイン

## 概要

この文書は、ブラウザーの WS-Federation Passive Requestor Profile によるサインインとサインアウトの仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | `wsignin1.0` の検証と、署名した Assertion の RSTR のフォームでの返却。`wsignout1.0` と `wsignoutcleanup1.0` によるローカルセッションの破棄 |
| 行為者 | EndUser（ブラウザーでサインインとサインアウトを始める利用者） |
| 扱わないもの | ログインセッションの確立は `Authentication` が、Application への割り当ては `Application` が扱う |

## モデル

サインインとサインアウトは一つのパッシブのエンドポイント `/wsfed` を共有し、`wa` のパラメーターで分かれる。

| `wa` | 結果 |
| --- | --- |
| `wsignin1.0` | `wtrealm`、`wreply`、`wfresh`、`wauth`、対象のユーザーの Application への割り当てを検証し、署名した Assertion を RSTR のフォームで返す。`wctx` は同じ値で返す |
| `wsignout1.0` | ローカルセッションを破棄し、その `wtrealm` に登録した `wreply` へリダイレクトする |
| `wsignoutcleanup1.0` | ローカルセッションを破棄し、200 を返す |

## 操作

### 利用者によるパッシブサインイン

#### REQ-WSFEDERATION-002 登録済みの RP へのパッシブサインインはトークンを発行する

- `Active` の User のログインセッションがある間、利用者が登録済みの RP の `wtrealm` で `wsignin1.0` を要求したとき、WsFederation は、署名した SAML Assertion を含む RSTR を、`wreply` へ自動で POST するフォームで返し、`wctx` を同じ値で返し、`WsFedSignInIssued` を発行する。
- `wreply` を省略した `wsignin1.0` を受けたとき、WsFederation は、RP に登録した最初の返信先へ RSTR を送る。
- `wsignin1.0` にトークンを発行するとき、WsFederation は、Assertion の audience を RP の audience（設定がない RP では `wtrealm`）に、recipient を返信先に、有効期間を発行の 1 分前から 5 分後までにし、RP の `ClaimMappingPolicy` でクレームを発行する。
- `wsignin1.0` にトークンを発行するとき、WsFederation は、RP に設定したトークンの型（SAML 1.1 または SAML 2.0。設定がない RP では SAML 1.1）の Assertion を返し、応答に `Cache-Control: no-store` を付ける。
- ログインセッションがないか、認証が完了していないか、User が `Active` でない間、`wsignin1.0` を受けたとき、WsFederation は、トークンを発行せず、元の要求を `return_to` に持つログイン画面へ 303 でリダイレクトする。
- `wfresh` の分数（30 秒未満は 30 秒とする）より前に認証したログインセッションの間、`wsignin1.0` を受けたとき、WsFederation は、トークンを発行せず、元の要求を `return_to` に持つログイン画面へ 303 でリダイレクトする。
- 統合 Windows 認証の `wauth` を指定されたか、パスワードの `wauth` を指定されたのにログインセッションがパスワードで認証していない場合、WsFederation は、400 で拒否し、`WsFedSignInRejected` を発行し、トークンを発行しない。
- クレームを発行できないか、Entra の定型設定の ImmutableID を導けない場合、WsFederation は、500 で拒否し、`WsFedSignInRejected` を発行し、トークンを発行しない。
- **例**：EX-WSFEDERATION-002-01、EX-WSFEDERATION-002-02、EX-WSFEDERATION-002-03

#### REQ-WSFEDERATION-003 信頼していない宛先へのパッシブサインインは拒否する

- `wtrealm` がないか、テナントに登録した RP の `wtrealm` でない `wsignin1.0` を受けた場合、WsFederation は、400 で拒否し、`WsFedSignInRejected` を発行し、トークンを発行しない。
- RP に登録した返信先と完全には一致しない `wreply` を受けたか、返信先を一つも登録していない RP の場合、WsFederation は、400 で拒否し、`WsFedSignInRejected` を発行し、トークンを発行しない。
- RP が属する Application に利用者が割り当てられていないか、Application のサインインポリシーを満たさない場合、WsFederation は、403 で拒否し、`WsFedSignInRejected` を発行し、トークンを発行しない。
- `wsignin1.0`、`wsignout1.0`、`wsignoutcleanup1.0` のどれでもない `wa` を受けた場合、WsFederation は、400 で拒否する。
- **例**：EX-WSFEDERATION-003-01

### 利用者によるパッシブサインアウト

#### REQ-WSFEDERATION-006 パッシブサインアウトはローカルセッションを破棄し、登録した返信先にだけリダイレクトする

- 利用者が `wsignout1.0` または `wsignoutcleanup1.0` を要求したとき、WsFederation は、ローカルのログインセッションを失効させ、セッション Cookie を消し、`WsFedSignOut` を発行する。
- `wreply` がその `wtrealm` の RP に登録した返信先と完全に一致する間、利用者が `wsignout1.0` を要求したとき、WsFederation は、`wreply` へ 303 でリダイレクトする。
- `wreply` が登録した返信先と一致しないか、`wtrealm` または `wreply` がない `wsignout1.0` を受けた場合、WsFederation は、リダイレクトせずに 200 を返す。
- 利用者が `wsignoutcleanup1.0` を要求したとき、WsFederation は、リダイレクトせずに 200 を返す。

## セキュリティ上の考慮

パッシブサインインは管理者の認可を通らず、ブラウザーのログインセッションで主体を決める。
未登録の宛先にトークンを発行することはない。

`wreply` へのリダイレクトは、サインインとサインアウトのどちらでも、その `wtrealm` に登録した宛先に限る。
登録していない宛先へ送る経路がないことが、サインアウトを開いたリダイレクターに変えないことの保証である。
