# System Glossary

この Context の中でだけ意味が定まる語を置く。Context を跨いで意味が固定される語は [spec/glossary.md](../../glossary.md) が持つ。

| Term | Definition | Aliases |
|---|---|---|
| Locale | UI の表示言語を一意に決める BCP 47 言語タグ。IdMagic は `ja` と `en` だけに対応し、それ以外は未対応のロケールとして扱う。 | locale tag, 表示言語コード |
| DisplayLanguage | EndUser または Administrator が言語切り替え UI で明示的に選択したロケール。選択はブラウザーに保存し、以後のアクセスでは保存済みの設定を優先する。 | 表示言語, 言語設定 |
| FallbackLocale | 要求されたロケールが未対応である場合、または対応する辞書に翻訳キーがない場合に使うロケール。IdMagic では `en` とする。 | デフォルトロケール |
| ConfiguredDefaultLocale | 起動時設定 `VITE_DEFAULT_LOCALE` で指定するデフォルトのロケール。`ja` または `en` だけを受け付け、未設定または未対応の値であれば `FallbackLocale` を使う。 | 起動時のデフォルトロケール |
| DemoLoginAffordance | HomePage に表示する、ローカルデモ用資格情報による `authorization_code` フローへのショートカット。資格情報は Seeding の `development` プロファイルが作成する。Vite 開発サーバーではデフォルトで表示し、それ以外のビルドでは起動時設定 `VITE_DEMO_LOGIN_ENABLED=true` の場合だけ表示する。表示時に `development` プロファイルの適用状態は検査しない。 | デモログインのショートカット |
