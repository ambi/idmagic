# System Glossary

| Term | Definition | Aliases |
|---|---|---|
| Locale | UI の表示言語を一意に決める BCP 47 言語タグ。IdMagic は `ja` と `en` だけに対応し、それ以外は未対応のロケールとして扱う。 | locale tag, 表示言語コード |
| DisplayLanguage | EndUser または Administrator が言語切り替え UI で明示的に選択したロケール。選択はブラウザーに保存し、以後のアクセスでは保存済みの設定を優先する。 | 表示言語, 言語設定 |
| FallbackLocale | 要求されたロケールが未対応である場合、または対応する辞書に翻訳キーがない場合に使うロケール。IdMagic では `en` とする。 | デフォルトロケール |
| ConfiguredDefaultLocale | 起動時設定 `VITE_DEFAULT_LOCALE` で指定するデフォルトのロケール。`ja` または `en` だけを受け付け、未設定または未対応の値であれば `FallbackLocale` を使う。 | 起動時のデフォルトロケール |
| DemoLoginAffordance | HomePage に表示する、ローカルデモ用資格情報による `authorization_code` フローへのショートカット。資格情報は Seeding の `development` プロファイルが作成する。Vite 開発サーバーではデフォルトで表示し、それ以外のビルドでは起動時設定 `VITE_DEMO_LOGIN_ENABLED=true` の場合だけ表示する。表示時に `development` プロファイルの適用状態は検査しない。 | デモログインのショートカット |
| BackendErrorText | バックエンドが HTTP、OAuth / OIDC リダイレクト、SAML、SCIM などの外部 API レスポンスで返す利用者向けのエラー本文。`message`、`error_description`、`detail`、プレーンテキストのエラー本文を含む。常に英語であり、表示言語によって変化しない。 | API エラーメッセージ, エラーの説明 |
| PersistedStateModel | `created_at` を持ち、作成後に現在状態を更新する場合は `updated_at` も持つ永続状態モデルの規約。作成後は不変で、消費または削除だけを行う記録モデルは `updated_at` を持たない。`issued_at`、`granted_at`、`occurred_at`、`expires_at`、`revoked_at` などのドメイン時刻は `created_at` を置き換えない。各 Context のモデル定義はこの規約に従う。 |  |
| EndUser | 認証済みまたは認証を試みる一般利用者。 |  |
| Operator | IdP をデプロイ・起動時設定を行う運用者。 |  |
| ResourceOwner | OAuth2/OIDC 認可フローでリソースの所有者として認可判断を行う利用者。EndUser と同一人物を OAuth2 文脈で指す呼称。 |  |
| Administrator | テナント内または横断のリソースを管理する権限を持つ利用者。 |  |
| APIConsumer | HTTP API を直接呼び出す外部クライアント。 |  |
| InterfaceStability | インターフェースの外部契約としての安定性を表す区分。`stable` は互換性を保証する外部契約、`beta` は互換性を保証する前の外部契約、`internal` はブラウザーセッション専用またはドメイン内部で外部契約に含めないインターフェースを表す。`stable` と `beta` は同時に 2 版までパスの版として提供し、非推奨の表明から最低 12 か月は維持する。 | 安定性区分 |
| Deprecation | `stable` または `beta` のインターフェースを将来削除することの予告。`deprecated_since` 以降はレスポンスに `Deprecation` ヘッダーを付与し、`sunset_at` が定まれば `Sunset` ヘッダーも付与する。`sunset_at` は `deprecated_since` の最低 12 か月後でなければならない。 | 非推奨化 |
| ConfigurationReference | バックエンドプロセスが起動時に読む設定キーの網羅的な一覧。キー名、値の型、デフォルト、必須かどうか、読み取るプロセス、説明を持ち、シークレットに分類したキーの値は持たない。Config の定義から生成する。 | 設定リファレンス |
