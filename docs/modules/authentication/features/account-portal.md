# アカウントポータル

## 概要

この文書は、本人が自分のアカウントを操作するセルフサービス API の境界と、機微な操作のステップアップ認証の仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | 本人の範囲に閉じたセルフサービス API、本人が変えられる項目の限定、機微な自己操作のステップアップ認証、API トークンの発行者による本人の認証情報の操作 |
| 行為者 | 本人（認証済みのセッションまたは API トークンの発行者） |
| 扱わないもの | 管理者による操作は管理 API が、個々の認証の要素の登録と解除は各機能が扱う |

## モデル

本人が変えられるのは、自分の表示名、`editable_by_user=true` の属性、パスワードに限る。
ロール、状態、組織の属性、`editable_by_user=false` の属性は管理者専用のままであり、`required_actions` は本人からは閲覧だけで、付与も取り消しもできない。
ただし、本人の操作の副作用として解除されるもの（パスワードの変更の成功による `update_password` の解除など）は除く。

| 機微な自己操作 | 求めるもの |
| --- | --- |
| `ChangePassword`、`RemoveTotpFactor`、`RequestEmailChange`、`RevokeMyOtherSessions` | CSRF と同一オリジンの検査に加えて、直近の再認証（ステップアップ認証） |

ステップアップ認証の根拠は、要求が届いた資格情報で決まる。

| 資格情報 | 根拠 | 満たさないときの応答 |
| --- | --- | --- |
| ログインセッションの Cookie | `max(session.auth_time, session.step_up_at)` が `StepUpRecencySeconds`（5 分）以内 | `403 step_up_required` |
| アクセストークン（ポータルの Bearer を含む） | トークンの `auth_time` が 5 分以内 | RFC 9470 の `401 insufficient_user_authentication` |
| 管理発行の API アクセストークン | トークンの保持 | なし |

Cookie の要求は認証済みだが、この操作に必要な直近の認証を示していないので、`401` ではなく `403` を返す。
アクセストークンの要求は、トークンが示す認証が古いので、RFC 9470 に従ってクライアントに再認可を求める。
アクセストークンの発行時刻はリフレッシュでも新しくなるので、認証の時刻として使わない。

- **判断**：機微な自己操作にステップアップ認証を求める理由は、[機微な自己操作に CSRF の防御に加えてステップアップ認証を求める](../decisions.md#機微な自己操作に-csrf-の防御に加えてステップアップ認証を求める)。

## 操作

### API トークンの発行者による本人の認証情報の操作

#### REQ-AUTHENTICATION-004 API トークンの発行者は機密操作のスコープで自身の認証情報だけを操作できる

- 本人の User に固定した API アクセストークンでアカウントの操作を要求されたとき、Authentication は、`account:read` で本人のアカウントの情報、セキュリティの設定、サインイン履歴、セッションの参照を、`account:mfa:write` で本人の認証の要素と復旧コードの変更を、`account:sessions:write` で本人のセッションの失効を、`account:password:write` と現在のパスワードの提示で本人のパスワードの変更を許可する。
- 本人がセルフサービス API を呼び出したとき、Authentication は、URL、本文、クエリ文字列の `sub` や `tenant_id` ではなく、認証した主体の `sub` に対してだけ作用する。
- 操作に対応しないスコープのトークンで変更を要求された場合、Authentication は、403 と `insufficient_scope` で拒否する。
- 別のテナントで発行したトークンを受けた場合、Authentication は、401 と `invalid_token` で拒否する。
- 他の User の資源を名指しした要求を受けた場合、Authentication は、本人の資源として見つからないものとして拒否し、他の User の資源を変えない。
- API アクセストークンでステップアップ認証のエンドポイントを要求された場合、Authentication は、403 と `insufficient_scope` で拒否し、必要な資格として対話のセッションを `WWW-Authenticate` で示す。
- 本人の User に固定した API アクセストークンで、そのスコープが許すステップアップ認証を求める操作を受けたとき、Authentication は、トークンの保持をステップアップ認証として扱い、401 の `insufficient_user_authentication` を返さない。
- **例**：EX-AUTHENTICATION-004-01、EX-AUTHENTICATION-004-02、EX-AUTHENTICATION-004-03、EX-AUTHENTICATION-004-04

### 本人によるステップアップ認証

#### REQ-AUTHENTICATION-039 本人は利用できる手段で再認証し、セッションをステップアップ認証済みにできる

- 本人が認証済みのセッションでステップアップ認証を始めたとき、Authentication は、200 と、パスワードと、登録済みの TOTP、WebAuthn、未消費の復旧コードのうち使える手段を返し、`StepUpRequested` を発行する。
- 本人が使える手段の正しい資格情報でステップアップ認証を完了したとき、Authentication は、セッションの `step_up_at` を現在の時刻にし、`StepUpCompleted` を発行する。
- `max(auth_time, step_up_at)` が 5 分の内にあるセッションで機微な自己操作を受けたとき、Authentication は、ステップアップ認証を済ませたものとして操作を受け付ける。
- 誤った資格情報でステップアップ認証の完了を受けた場合、Authentication は、403 と `step_up_failed` で拒否し、`step_up_at` を変えない。
- 本人が使えない手段でステップアップ認証の完了を受けた場合、Authentication は、400 と `invalid_request` で拒否する。
- `max(auth_time, step_up_at)` が 5 分を過ぎたセッションで機微な自己操作を受けた場合、Authentication は、403 と `step_up_required` で拒否する。
- `auth_time` が 5 分の内にあるアクセストークンでステップアップ認証を求める操作を受けたとき、Authentication は、ステップアップ認証を済ませたものとして操作を受け付ける。
- `auth_time` が 5 分を過ぎたか `auth_time` を持たないアクセストークンでステップアップ認証を求める操作を受けた場合、Authentication は、401 と `insufficient_user_authentication` で拒否し、`WWW-Authenticate` に `error="insufficient_user_authentication"` と `max_age="300"` を示し、操作の対象を変えない。
- Authentication は、アクセストークンの要求のステップアップ認証の判定に、トークンの発行時刻とセッションの `step_up_at` を使わない。
- **判断**：ポータルは自身の IdP の OAuth クライアントとして Bearer で API を呼ぶので、リソース側の判断をトークンで完結させ、RFC 9470 の再認可でトークンの認証を新しくする。セッションの記録をトークンの判定へ持ち込む案と、アクセストークンにセッションを結び付ける案は採らない。前者は二つの資格情報の整合を自前で保つことになり、後者はセッションの Cookie の値をリソースサーバーへ渡すことになる。

## セキュリティ上の考慮

セルフサービス API（`/api/account/*`）は、認証済みのセッション自身の `actor.sub` に対してだけ作用する。
URL、本文、クエリ文字列で与えられた `sub` や `tenant_id` を信頼することはないので、ユーザーをまたぐアクセスとテナントをまたぐアクセスは構造的に生じない。

利用者自身が操作するアカウントのポータルと、管理用のアカウント API（`/api/auth/account`、ロールを含む）は別の契約である。
ポータル自身の要約のエンドポイント（`/api/account/summary`）は意図的にロールを省くので、管理用の情報を漏らさない。
ポータルの UI は独立した外枠であり、たまたま管理者のロールを持つユーザーにも、管理用の案内を出さない。

## 実現方式

### アーキテクチャ

ステップアップ認証を求める操作は、TypeSpec の 401 の応答に `InsufficientUserAuthenticationError` を、403 の応答に `StepUpRequiredError` を宣言する。
拒否の応答は共有の HTTP 支援の一つの関数が書き、要求が Authorization の資格情報を持つかで RFC 9470 の 401 と Cookie の 403 を選ぶ。
ログインの直後のセッションとトークンもステップアップ認証済みであり、Google や Okta で利用者が慣れている再認証の形と一致する。

ポータルは `insufficient_user_authentication` の 401 を、トークンの失効を表す `invalid_token` の 401 と区別する。
確認を表示し、利用者が押したボタンで、応答が示した `max_age` と `acr_values` を付けた `/authorize` をポップアップに開く。
ポップアップの `/callback` は `state` と `code` を同じオリジンの `postMessage` で元の画面へ返し、元の画面がコードを交換して元の要求を一回だけ送り直す。
ポップアップを開けない場合は画面全体で再認可し、入力は引き継がない。
Cookie の要求のための `POST /api/account/v1/step-up/complete` と `step_up_at` の記録は残す。

認証の新しさは `LoginSession` のレコード自体に `step_up_at` として保存するので、Cookie とともに別の端末へ引き継がれることはない。
