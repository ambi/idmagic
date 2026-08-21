# Tenancy Glossary

| Term | Definition | Aliases |
|---|---|---|
| Tenant | 独立した認可境界。URL 上は Realm という別名で表現される。 | テナント, Realm, realm |
| DefaultTenant | 起動時に自動作成される `realm == "default"` のテナント。ID は固定 UUID の代理キー。単一テナント運用時の互換性と、接頭辞のない HTTP リクエストの解決先を兼ねる。 | デフォルトテナント |
| TenantDisablement | Tenant.disabled_at を設定してテナント単位で停止する復活可能な操作。テナント物理削除とは独立。 | テナント無効化 |
| EntraFederation | Microsoft Entra ID の検証済みドメインを WS-Federation / WS-Trust の federated IdP として接続するプロファイル。 | Microsoft365Federation, AzureADFederation, M365Federation |
| Disabled | 復活可能な無効化状態。Tenant と (慣例的に) User の disabled_at 経路で共有される。 | disabled |
| Disable | 対象を Disabled に遷移させる。Tenant では `/api/admin/tenants/{id}/disable` から発火。 | disable |
| Enable | Disabled の対象を Active に戻す。Tenant では `/api/admin/tenants/{id}/enable` から発火。 | enable |
| System | IdP プロセス自身。起動時にデフォルトテナントを自動作成する。 |  |
| OAuth2Client | OIDC / OAuth2 プロトコルエンドポイントを呼び出す外部クライアントアプリケーション。 |  |
| EndUser | テナントに所属する人間の利用者。通知メールの受信者であり、その `locale` 属性が通知言語を解決する第 1 段になる。IdManagement が所有する User を公開用の語彙で表す。 | エンドユーザー, 利用者 |
| HardQuota | 超過するとリソース作成が同期的にエラーとなる厳格な上限。 |  |
| SoftQuota | 超過しても作成は成功するが、警告が通知される遅延評価の上限。 |  |
| NotificationTemplate | 利用者へ送る通知メール 1 通の文面定義。`template_key` と `locale` の組で一意に定まり、件名、プレーンテキスト本文、HTML 本文、差出人表示名を持つ。システムが同梱する `ja` / `en` の組込みデフォルトとテナントによる上書きの 2 段で解決する。 | 通知テンプレート, メールテンプレート |
| NotificationTemplateKey | 通知の用途を表す固定識別子。カタログに存在するキーだけが送信・上書きの対象になり、テナントはキー自体を追加できない。 | テンプレートキー |
| NotificationPlaceholder | テンプレート本文に `{{name}}` の形で書ける差し込み変数。template_key ごとに許可集合が決まっており、許可外の変数を含む上書きは保存時に拒否される。 | placeholder, 差し込み変数 |
| NotificationLocaleResolution | 通知 1 通に使うロケールを決める手順。受信者 User の `locale` 属性、テナントの `default_locale`、システムのデフォルトロケールの順に、カタログが対応する最初のロケールを採用する。 | ロケール解決順序 |
| BuiltinNotificationTemplate | システムが同梱する組込みデフォルトテンプレート。テナントによる上書きがない、または上書きが削除された場合に使われる。テナントは編集できず、「デフォルトに戻す」ことでこの文面へ復帰する。 | 組込みデフォルトテンプレート |
