# 通知テンプレートの内部設計

## 通知テンプレートカタログとロケール解決

通知メールの内容は、システムが同梱する日本語と英語の組み込みカタログと、必要に応じた `(tenant_id, template_key, locale)` ごとの上書きという 2 段階で解決する。版の履歴は持たず、`ResetNotificationTemplate` は常に既知の正常な組み込み文面へ戻す。`template_key` は仕様で定める固定の列挙であり、テナントは追加できない。各キーは 1 つの送信経路に対応し、送信元のないテンプレートは作成できない。

プレースホルダー（`{{name}}`）は保存時にテンプレートキーごとの許可リストと照合する。宣言されていないプレースホルダーを参照する上書きは、空の値で描画するのではなく、保存時に拒否する。アカウント復旧などの導線が実行時に欠落することを防ぐためである。許可リストは `backend/shared/notification/template` で定義して API から返す。

| Key | Placeholders |
| --- | --- |
| all keys | `product_name`, `tenant_display_name`, `user_display_name` |
| `PasswordReset`, `EmailVerification`, `EmailChangeConfirmation` | 1 つの `*_url` の導線、`expires_in_minutes` |
| `EmailChangeConfirmation` (additional) | `new_email` |
| `LifecycleWorkflowNotification` (additional) | `notification_key` |
| `AccountSecurityAlert` (additional) | `event_description`, `occurred_at`, `device_summary`, `security_review_url` |
| `ProvisioningConnectionQuarantined` (additional) | `application_id`, `quarantine_reason`, `quarantined_at` |

`ProvisioningConnectionQuarantined` の宛先は User ではなく接続の `notification_email` である。
そのため受信者の言語はなく、テナントのデフォルト言語から解決し、`user_display_name` は空文字列で描画する。

資格情報、ダイジェスト値、TOTP シークレット、API トークン、生の IP アドレスは決して差し込みにしない。メールは受信者によって転送され、引用され、無期限に保持されるため、これらの情報を差し込むと、後に受信箱が侵害された際に露出する。

描画処理は、件名、平文本文、HTML 本文を常に 1 つの単位として返す。上書きも 3 つを同時に置き換え、メールは `multipart/alternative` として送るため、平文と HTML の内容が意図せず食い違わない。

特殊文字の処理はテンプレートではなく描画処理の責務とし、HTML に差し込む値だけをエスケープする。導線の URL は呼び出すユースケースがリクエストの `issuer` から組み立て、1 つのプレースホルダー値として渡す。テンプレートは URL を配置できるが、断片から組み立てられない。

上書きできるのは、件名、HTML 本文の断片、平文本文、送信者の表示名だけである。HTML 文書の外枠と送信元アドレスはシステム側が保持し、テナントの入力をこれらへ注入させない。

言語は、受信者の `User.locale`、テナントの `default_locale`、システム設定の `DEFAULT_LOCALE`（デフォルト値は `en`）の順に解決し、カタログに翻訳がある最初の言語を使う。テナントのデフォルト言語は明示的なカラムで管理し、ある言語のテンプレートを上書きしただけで他の通知までその言語へ変わることを防ぐ。

試し送りは、操作中の管理者自身が確認済みのアドレスにだけ送信し、エンドポイントは宛先を受け取らない。任意の宛先を許可すると、テナントのブランド表示を使ったメールを第三者へ送る手段になるためである。下書きのプレビューは読み取り専用で、実際の利用者データではなく固定のサンプル値を使って描画する。
