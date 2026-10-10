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

`max(session.auth_time, session.step_up_at)` が `StepUpRecencySeconds`（5 分）以内であれば、セッションをステップアップ認証済みとみなす。
満たさない場合は、`401` ではなく `403 step_up_required` を返す。
セッションは認証済みだが、この操作に必要な直近の認証を示していないからである。

- **判断**：機微な自己操作にステップアップ認証を求める理由は、[機微な自己操作に CSRF の防御に加えてステップアップ認証を求める](../design/decisions.md#機微な自己操作に-csrf-の防御に加えてステップアップ認証を求める)。

## 操作

### API トークンの発行者による本人の認証情報の操作

#### REQ-AUTHENTICATION-004 API トークンの発行者は機密操作のスコープで自身の認証情報だけを操作できる

- 本人の User に固定した API アクセストークンでアカウントの操作を要求されたとき、Authentication は、`account:read` で本人のアカウントの情報、セキュリティの設定、サインイン履歴、セッションの参照を、`account:mfa:write` で本人の認証の要素と復旧コードの変更を、`account:sessions:write` で本人のセッションの失効を、`account:password:write` と現在のパスワードの提示で本人のパスワードの変更を許可する。
- 本人がセルフサービス API を呼び出したとき、Authentication は、URL、本文、クエリ文字列の `sub` や `tenant_id` ではなく、認証した主体の `sub` に対してだけ作用する。
- 操作に対応しないスコープのトークンで変更を要求された場合、Authentication は、403 と `insufficient_scope` で拒否する。
- 別のテナントで発行したトークンを受けた場合、Authentication は、401 と `invalid_token` で拒否する。
- 他の User の資源を名指しした要求を受けた場合、Authentication は、本人の資源として見つからないものとして拒否し、他の User の資源を変えない。
- API アクセストークンでステップアップ認証のエンドポイントを要求された場合、Authentication は、403 と `insufficient_scope` で拒否し、必要な資格として対話のセッションを `WWW-Authenticate` で示す。
- **例**：EX-AUTHENTICATION-004-01、EX-AUTHENTICATION-004-02、EX-AUTHENTICATION-004-03、EX-AUTHENTICATION-004-04

### 本人によるステップアップ認証

#### REQ-AUTHENTICATION-039 本人は利用できる手段で再認証し、セッションをステップアップ認証済みにできる

- 本人が認証済みのセッションでステップアップ認証を始めたとき、Authentication は、200 と、パスワードと、登録済みの TOTP、WebAuthn、未消費の復旧コードのうち使える手段を返し、`StepUpRequested` を発行する。
- 本人が使える手段の正しい資格情報でステップアップ認証を完了したとき、Authentication は、セッションの `step_up_at` を現在の時刻にし、`StepUpCompleted` を発行する。
- `max(auth_time, step_up_at)` が 5 分の内にあるセッションで機微な自己操作を受けたとき、Authentication は、ステップアップ認証を済ませたものとして操作を受け付ける。
- 誤った資格情報でステップアップ認証の完了を受けた場合、Authentication は、403 と `step_up_failed` で拒否し、`step_up_at` を変えない。
- 本人が使えない手段でステップアップ認証の完了を受けた場合、Authentication は、400 と `invalid_request` で拒否する。
- `max(auth_time, step_up_at)` が 5 分を過ぎたセッションで機微な自己操作を受けた場合、Authentication は、403 と `step_up_required` で拒否する。

## セキュリティ上の考慮

セルフサービス API（`/api/account/*`）は、認証済みのセッション自身の `actor.sub` に対してだけ作用する。
URL、本文、クエリ文字列で与えられた `sub` や `tenant_id` を信頼することはないので、ユーザーをまたぐアクセスとテナントをまたぐアクセスは構造的に生じない。

利用者自身が操作するアカウントのポータルと、管理用のアカウント API（`/api/auth/account`、ロールを含む）は別の契約である。
ポータル自身の要約のエンドポイント（`/api/account/summary`）は意図的にロールを省くので、管理用の情報を漏らさない。
ポータルの UI は独立した外枠であり、たまたま管理者のロールを持つユーザーにも、管理用の案内を出さない。
