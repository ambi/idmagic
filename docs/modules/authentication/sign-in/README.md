# ログイン

## 概要

この文書は、ブラウザーでのパスワードによるログイン、ログインの流量の制限、無効なユーザーの拒否、ブラウザーの初期化の情報の仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | パスワードによる認証と認可の継続、アカウント単位と IP 単位の流量の制限、無効な主体の拒否、認証の状態と CSRF の境界を保つ初期化の情報 |
| 行為者 | ResourceOwner、EndUser、本人（認証済みのセッション） |
| 扱わないもの | 第二要素は[多要素認証](../mfa/README.md)が、セッションの一覧と失効は[ログインセッション](../session/README.md)が、外部 IdP によるログインは[外部 IdP との連携](../federation/README.md)が扱う |

## モデル

ログインの試行は、アカウント単位と IP 単位で独立に抑制する。

| 単位 | 閾値 | 抑止するもの |
| --- | --- | --- |
| アカウント | 900 秒間に 10 回の失敗で 900 秒間拒否する | 特定のアカウントへの辞書攻撃 |
| IP | 900 秒間に 30 回の失敗で 900 秒間拒否する | 一つの発信元から多数のアカウントを狙う credential stuffing |

どちらかの閾値を超えれば 429 を返す。

- **判断**：二つの単位を独立に数え、恒久的な締め出しを使わない理由は、[ログインの流量の制限をアカウントと IP で独立に数え恒久的な締め出しを使わない](../design/decisions.md#ログインの流量の制限をアカウントと-ip-で独立に数え恒久的な締め出しを使わない)。

## 操作

### リソースオーナーによるパスワードのログイン

#### REQ-AUTHENTICATION-007 ResourceOwner はブラウザーでパスワード認証し、認可を継続する

- `Active` の User が認可のトランザクションの中で正しいユーザー名とパスワードをログインの API に送ったとき、Authentication は、`pwd` の認証方式のログインセッションを作ってセッションの Cookie を発行し、`UserAuthenticated` を発行し、認可を続けて `redirect_uri` へ認可コードを返させる。
- 正しいパスワードでログインしたとき、Authentication は、アカウントのログインの失敗の回数を消し、IP のログインの失敗の回数を残す。
- ログインに成功した User が必須操作 `update_password` を持つとき、Authentication は、認可を続けずにパスワードの変更の画面へ進ませる。
- ユーザー名かパスワードを送らないか、JSON として読めない本文を受けた場合、Authentication は、400 と `invalid_request` で拒否する。
- 認可のトランザクションを解決できない場合、Authentication は、401 と `transaction_unavailable` で拒否する。
- SameSite の Cookie と要求の CSRF トークンが一致しない場合、Authentication は、403 と `csrf_failed` で拒否し、セッションを作らない。
- 存在しないユーザー名か誤ったパスワードを受けた場合、Authentication は、401 と `invalid_credentials` で拒否し、ログインの失敗を数え、`AuthenticationFailed` を発行する。
- 900 秒の間にアカウントのログインの失敗が 10 回に達した間、そのアカウントのログインを受けたとき、Authentication は、パスワードを検証せずに 429 と `rate_limited` を `Retry-After` とともに返し、`LoginThrottled` を発行する。
- 900 秒の間に同じ IP からのログインの失敗が 30 回に達した間、その IP からのログインを受けたとき、Authentication は、429 と `rate_limited` を `Retry-After` とともに返し、`LoginThrottled` を発行する。
- 抑制の閾値を超えた後のログインの失敗を記録するとき、Authentication は、失敗を 5 分の窓ごとに集約し、各窓の最初の失敗で件数を載せた `AuthenticationEventAggregated` を発行する。
- 失敗の回数の共有のストアへ到達できない場合、Authentication は、ログインを抑制なしに通さずに失敗させる。
- **例**：EX-AUTHENTICATION-007-01、EX-AUTHENTICATION-007-02、EX-AUTHENTICATION-007-03、EX-AUTHENTICATION-007-04

#### REQ-AUTHENTICATION-009 無効なユーザーは新規ログインも既存セッションも拒否される

無効化そのものは IdManagement の操作であり、無効化から到達経路が閉じるまでの連鎖は REQ-PLATFORM-001 で定める。ここは、無効な主体を Authentication が単独で拒否することだけを述べる。

- `Active` でない User が正しいパスワードでログインしようとした場合、Authentication は、401 と `invalid_credentials` で拒否し、理由 `account_disabled` の `AuthenticationFailed` を発行し、セッションを作らない。
- User が `Active` でない間、その User の既存のセッションで認証を要する API を呼び出されたとき、Authentication は、セッションを失効させ、未認証として 401 と `authentication_required` で拒否する。
- **例**：EX-AUTHENTICATION-009-01

### 本人によるブラウザーの初期化

#### REQ-AUTHENTICATION-005 ブラウザーの初期化情報は認証状態と CSRF 境界を保持する

- 認証済みのセッション、`idmagic.admin` か `idmagic.account` のトークン、`account:read` の API アクセストークンでアカウントの文脈を要求されたとき、Authentication は、subject、realm、実効ロール、CSRF トークンを返す。
- 未認証の利用者がパスワードのリセットの文脈を要求したとき、Authentication は、CSRF トークンを含む文脈を返す。
- セッションが未認証か認証の途中の場合、Authentication は、アカウントの文脈の要求を 401 と `authentication_required` で拒否する。
- 許可したポータルのスコープも `account:read` も持たないトークンでアカウントの文脈を要求された場合、Authentication は、403 と `insufficient_scope` で拒否する。
- **例**：EX-AUTHENTICATION-005-01、EX-AUTHENTICATION-005-02、EX-AUTHENTICATION-005-03

## セキュリティ上の考慮

アカウント単位のカウンターは、アカウントが存在するかを確かめる前に加算する。
存在しない場合は、パスワードの検証の段で固定の番人のハッシュを使う。
そうしないと、処理の時間（Argon2 の検証が起きるかどうか）や 429 のレスポンスの形から、どの利用者名が存在するかが漏れるからである。

ログインの成功はアカウントのカウンターを消すが、IP 単位のカウンターは意図的に残す。
共有のオフィスや NAT の IP から一回ログインが成功しても、その IP の残りの通信が信頼できることにはならないからである。

共有のスロットルのストアへ到達できない場合、ログインは抑制なしに通すのではなく、失敗させる。
クライアントの IP は、デフォルトでは直接の接続の相手から読む。
`X-Forwarded-For` を尊重するのは、`TRUSTED_FORWARDED_HOPS` を明示的に有効にした場合だけである。
無条件に信頼すると、攻撃者が IP 単位の軸を偽装で回避できるからである。
