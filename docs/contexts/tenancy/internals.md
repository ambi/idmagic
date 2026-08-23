# Tenancy Internals

## Tenant resolution

**1 テナント = 1 正規ロケーション = 1 発行者。** テナントは `endpoint_style` が指す正規ロケーションからだけ到達でき、もう一方の経路では不在として扱う。同一テナントへ 2 つのオリジンから到達できると発行者が一意に定まらず、Discovery Metadata の `issuer` が取得元 URL と一致しなくなる (OpenID Connect Discovery 1.0 §4.3 / RFC 8414 §3.3 違反)。この 1 対 1 が、以下の解決規則が守っているものである。

解決は Host ヘッダーとパスだけを見て、次の順に進む。

1. `tenant_base_domain` が設定され、Host が `{label}.{tenant_base_domain}` に一致するなら、ラベルを realm として対応付ける。見つかったテナントの `endpoint_style` が `Subdomain` でなければ不在として扱う。
2. パスが `/realms/{realm}/...` に一致するなら realm を対応付ける。見つかったテナントの `endpoint_style` が `Path` でなければ不在として扱う。
3. どちらにも一致しないリクエストは、テナントが存在しないものとして扱う。任意の Host や接頭辞のないパスをデフォルトテナントへフォールバックさせない。

ミドルウェアは `^/realms/([a-z0-9][a-z0-9-]{0,62})(/|$)` で realm の区間を取り出し、解決した `Tenant` と発行者の文字列をリクエストコンテキストに付ける。発行者、URL の接頭辞、Cookie のスコープ、WebAuthn の RP ID は、いずれもこの正規ロケーションから組み立てる。`Path` では発行者が `{base}/realms/{realm}`、`Subdomain` では `{scheme}://{realm}.{tenant_base_domain}` になる。

存在しないテナントには `404 tenant_not_found`、無効なテナントには OAuth / OIDC のプロトコルルートで `400 invalid_request` を返す。レスポンスの形が場合ごとに変わらないため、解決器の応答だけからテナントを列挙することはできない。

プロトコルと管理のルートはすべて `/realms/{realm}/...` の下に置き、テナントをまたぐ制御面のテナント管理だけを `/realms/default/admin/tenants/...` に置く。こうすると、デフォルトテナントのセッション Cookie のパスだけで対象を覆えるので、Cookie のスコープをルートパスまで広げずに済む。

`Subdomain` を選べるのは配備時に基底ドメインを設定した場合だけであり、設定しない配備は `Path` のままでワイルドカード DNS も証明書も要らない。`realm` は変更できるが、発行者にも `Subdomain` ではホスト名にも現れるため、その変更は `endpoint_style` の変更と同じく既存クライアントとの互換性を壊す。RP の再設定、既存パスキーの再登録、進行中のセッションの終了を伴うので、アイデンティティの移行として計画する。

## Tenant identity: UUID key and realm slug

`tenants` は、不変の代理キー `id UUID` と、変更可能で一意な識別子 `realm TEXT` を持つ。これにより、組織名やブランド名の変更、綴りの訂正で realm を改名しても、他のテーブルの `tenant_id` 外部キーは変更せずに済む。URL の接頭辞、OIDC の発行者、Discovery Metadata など外部に公開する識別子には `realm` を使い、`tenant_id` 外部キー列、`spec.DefaultTenantID`、Context 内の `TenantID` など内部参照には UUID を使う。解決ミドルウェアが `FindByRealm(realm)` で両者を対応付け、管理 API は URL の `realm` をユースケースの呼び出し前に UUID へ解決する。

デフォルトテナントを表す 2 つの定数も同じ分離に従う。`spec.DefaultTenantID` は固定の UUID であり、IdMagic が生成する ID の列が全体を通じて UUID 型であることと整合する。`spec.DefaultRealm` は文字列 `"default"` であり、テナントを URL に表す箇所だけで使う。`tenants(id)` を参照する外部キー列は UUID 型とし、`tenant_id` に SQL のデフォルト値は持たせない。すべての挿入で `tenant_id` を明示しなければならず、値が欠けた場合はデフォルトテナントへ黙って混入させず、明確に失敗させる。これはリポジトリ全体の [`tenant_id` retention classes](../../persistence.md#tenant_id-retention-classes) 方針をさらに厳しくした例である。`tenants` への外部キーを持たない追記専用テーブル、または不透明なキーを持つテーブル（`audit_events.tenant_id`、`authentication_event_buckets.tenant_id`）では、`tenant_id` を `UUID` ではなく `TEXT` のままにする。テナントに属さない監査イベントには、UUID 列で自然に表せない番兵値が必要なためである。

## Tenant security policy overrides

`Tenant` が持つセキュリティポリシーの上書きは、デプロイ全体の製品既定を緩めず、厳しい方向にだけ働く。パスワードポリシーでは最小長と履歴件数を下げず最大長を上げない。Token Exchange の `max_delegation_depth` ではシステム既定の 3 を超えて上げず、未設定なら 3 を継承する。管理 API の `0` は委譲を全面禁止する値ではなく、上書きを解除して SQL の `NULL` へ戻す操作として扱う。設定取得では現在の任意上書きとシステム既定を別々に返し、管理 UI が継承状態と実効値を区別できるようにする。

`trusted_device_max_age_seconds` だけは既定が「機能なし」の側にあるので、方向が逆になる。未設定と `0` はどちらも Authentication の信頼済みデバイスを丸ごと無効にする値であり、上書きを解除する操作ではない。正の値はテナントが第二要素の省略を明示的に有効にしたことを意味するので、システム上限 (7,776,000 秒 = 90 日) を超える値だけを拒否する。緩める方向の値を保存できるのは、この設定の既定が最も厳しい状態そのものだからである。

OAuth2 Context は `TenantRepository` を直接参照せず、委譲深さを返す小さなポートに依存する。`oauth2/policy_tenancy` アダプターがこのポートを `Tenant` の実効値へ接続する。上書きを読めないときにシステム既定へ退避すると、テナントが意図して下げた認可境界を黙って広げるため、解決失敗では Token Exchange を拒否する。

## Tenant branding

`TenantBranding` は `Tenant` に埋め込まず、`tenant_id` をキーとする独立したエンティティとする。独立して更新される外観設定によって、認可と realm 解決が依存する中核の `Tenant` Aggregate を肥大化させないためである。設定項目は、製品名、ロゴ、ファビコン、2 つのブランドカラー、サポート導線、法務導線、フッター文言に限る。任意の CSS、HTML、スクリプト、背景画像は受け付けない。

信頼できないテナントの入力をマークアップや自由形式のスタイルとしてホステッドログインシェルへ渡さない。ブランドカラーは `#rrggbb` として検証し、固定した 2 個の CSS カスタムプロパティ（`--tenant-brand-primary` / `--tenant-brand-accent`）にだけ注入する。テキストフィールドはデフォルトのエスケープ処理で描画し、`dangerouslySetInnerHTML` は決して使わない。`support_url` / `legal_url` は `https://` スキームだけを許可リストに含め、`javascript:`、`data:`、平文の `http://` を書き込み時に拒否する。コントラストは保存時の制約に含めず、管理 UI で確認できるようにして、可読性の結果はテナントが負う。

ロゴとファビコンには、Application アイコンと同じ検証処理を使う。先頭バイト、サイズ、形式を検証し、`nosniff` を付けて配信する。検証処理は `backend/shared/mediavalidation` で共有するが、保存先は専用の `tenant_branding_assets` テーブルとし、管理は Tenancy に残す。`GetTenantBranding` は、設定やアセットが欠けていてもシステムデフォルトへフォールバックし、ログイン画面を失敗させない。更新時は `updated_at` を進め、公開レスポンスではキャッシュ版または ETag として使う。`tenant_id` は URL の一部なので、テナント間でキャッシュを混同せずに古い外観を無効化できる。

## Tenant resource quotas

リソースの作成にはテナントごとの上限を設ける。共有基盤上で、負荷の高い、または暴走した 1 つのテナントが及ぼす影響範囲を抑えるためである。上限は強制方法によって 2 つに分かれる。**Hard** の上限（`users`、`groups`、`agents`、`applications`、`oauth2_clients`、`active_sessions`、`consents`、`active_jobs`、`ssf_streams`）は作成トランザクション内で同期的に確認し、超過していれば操作を拒否する。**Soft** の上限（`audit_events_retained`、`export_artifacts_bytes`）は操作を成功させ、代わりに非同期の警告と監査イベントを発行する。Hard の上限はデータベースの枯渇を防ぎ、Soft の上限は記録の欠落を避けながら長期的な蓄積を検知する。

新しいテナントには固定のデフォルトの上限を与える (たとえばユーザー 10,000、グループ 1,000、エージェント 100、アプリケーション 50、OAuth2 のクライアント 100、有効なセッション 50,000、同意 10,000、実行中のジョブ 10、SSF ストリーム 20)。SSF ストリームの上限は送信側と受信側で分けず、`SsfStream` の行数として 1 つの上限で数える。向きは同じ集合の属性でしかなく、上限を分けても抑えたい資源 (ストリーム 1 本ごとに増える配送先・鍵取得先・保持する配送記録) は変わらないためである。System Admin は特定のテナントの上限を個別に上書きできる。Tenant Admin は自身の上限に対する使用量を閲覧できるが変更はできない。上限を決める権限を、テナント自身ではなく共有のプラットフォームの運用者に残すためである。

既存テナントへ初めて上限を適用する際は、現在の使用量を下回って直ちに操作を拒否しないよう、十分な余裕を持つ値を割り当てる。たとえば現在の使用量の 2 倍またはデフォルトの 10 倍を使う。その後、バックグラウンドの照合ジョブで使用量カウンターと実際の行数を一致させてから、System Admin が意図した値へ上限を引き下げる。

## Notification template catalog and locale resolution

通知メールの内容は、システムが同梱する日本語と英語の組み込みカタログと、必要に応じた `(tenant_id, template_key, locale)` ごとの上書きという 2 段階で解決する。版の履歴は持たず、`ResetNotificationTemplate` は常に既知の正常な組み込み文面へ戻す。`template_key` は仕様で定める固定の列挙であり、テナントは追加できない。各キーは 1 つの送信経路に対応し、送信元のないテンプレートは作成できない。

プレースホルダー（`{{name}}`）は保存時にテンプレートキーごとの許可リストと照合する。宣言されていないプレースホルダーを参照する上書きは、空の値で描画するのではなく、保存時に拒否する。アカウント復旧などの導線が実行時に欠落することを防ぐためである。許可リストは `backend/shared/notification/template` で定義して API から返す。

| Key | Placeholders |
| --- | --- |
| all keys | `product_name`, `tenant_display_name`, `user_display_name` |
| `PasswordReset`, `EmailVerification`, `EmailChangeConfirmation` | 1 つの `*_url` の導線、`expires_in_minutes` |
| `EmailChangeConfirmation` (additional) | `new_email` |
| `LifecycleWorkflowNotification` (additional) | `notification_key` |
| `AccountSecurityAlert` (additional) | `event_description`, `occurred_at`, `device_summary`, `security_review_url` |

資格情報、ダイジェスト値、TOTP シークレット、API トークン、生の IP アドレスは決して差し込みにしない。メールは受信者によって転送され、引用され、無期限に保持されるため、これらの情報を差し込むと、後に受信箱が侵害された際に露出する。

描画処理は、件名、平文本文、HTML 本文を常に 1 つの単位として返す。上書きも 3 つを同時に置き換え、メールは `multipart/alternative` として送るため、平文と HTML の内容が意図せず食い違わない。

特殊文字の処理はテンプレートではなく描画処理の責務とし、HTML に差し込む値だけをエスケープする。導線の URL は呼び出すユースケースがリクエストの `issuer` から組み立て、1 つのプレースホルダー値として渡す。テンプレートは URL を配置できるが、断片から組み立てられない。

上書きできるのは、件名、HTML 本文の断片、平文本文、送信者の表示名だけである。HTML 文書の外枠と送信元アドレスはシステム側が保持し、テナントの入力をこれらへ注入させない。

言語は、受信者の `User.locale`、テナントの `default_locale`、システム設定の `DEFAULT_LOCALE`（既定値は `en`）の順に解決し、カタログに翻訳がある最初の言語を使う。テナントの既定言語は明示的な列で管理し、ある言語のテンプレートを上書きしただけで他の通知までその言語へ変わることを防ぐ。

試し送りは、操作中の管理者自身が確認済みのアドレスにだけ送信し、エンドポイントは宛先を受け取らない。任意の宛先を許可すると、テナントのブランド表示を使ったメールを第三者へ送る手段になるためである。下書きのプレビューは読み取り専用で、実際の利用者データではなく固定のサンプル値を使って描画する。
