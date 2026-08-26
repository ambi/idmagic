# Glossary

Context を跨いで意味が固定される語を置く。ここに載っている語は、どの Context でも同じものを指す。

Context の `glossary.md` は、ここに載る語をその Context での役割へ**狭める**ことがある。狭めた定義がある Context の中では、そちらが読み方になる。狭めた先で別のものを指すようになったなら、それは同じ語ではなく、ここへ吸い上げて 1 つに揃える対象でもない。

1 つの Context の中でだけ意味が定まる語は、最初からその Context の `glossary.md` が持つ。

## 主体

| Term | Definition | Aliases |
|---|---|---|
| EndUser | 認証済み、または認証を試みる一般の利用者。 |  |
| ResourceOwner | OAuth 2.0 / OIDC の認可フローでリソースの所有者として認可判断を行う利用者。EndUser と同じ人物を、その文脈で呼ぶときの名前である。 |  |
| Administrator | テナント内またはテナント横断のリソースを管理する権限を持つ利用者。 |  |
| Operator | IdMagic を配備し、起動時設定を与える運用者。権限ではなく実行環境そのものが境界になる操作を持つ。 |  |
| APIConsumer | HTTP API を直接呼び出す外部クライアント。 |  |
| Agent | 人の操作を伴わずに動く実行主体。所有者、目的、ライフサイクルを持ち、資格情報は OAuth クライアントか外部のアテステーションに束ねる。 |  |

主体の種類ごとにどの境界へ到達できるかは [authorization.md](authorization.md) が持つ。

## 外部契約

| Term | Definition | Aliases |
|---|---|---|
| InterfaceStability | インターフェースの外部契約としての安定性の区分。`stable` は互換性を保証する外部契約、`beta` は保証の対象になる前の外部契約、`internal` はブラウザーセッション専用またはドメイン内部で外部契約に含めないインターフェースを表す。規則は [api-rules.md](api-rules.md) が持つ。 | 安定性区分 |
| Deprecation | `stable` または `beta` のインターフェースを将来削除することの予告。`deprecated_since` 以降は `Deprecation` ヘッダーを、`sunset_at` が定まれば `Sunset` ヘッダーも付与する。 | 非推奨化 |
| BackendErrorText | バックエンドが HTTP、OAuth / OIDC リダイレクト、SAML、SCIM などの外部レスポンスで返すエラー本文。`message`、`error_description`、`detail`、プレーンテキストの本文を含む。常に英語であり、表示言語によって変化しない。 | API エラーメッセージ |
| ConfigurationReference | バックエンドプロセスが起動時に読む設定キーの網羅的な一覧。キー名、値の型、デフォルト、必須かどうか、読み取るプロセス、説明を持つ。シークレットに分類したキーの値は持たない。起動時設定の定義から生成する。 | 設定リファレンス |

## 永続化の規約

| Term | Definition | Aliases |
|---|---|---|
| PersistedStateModel | `created_at` を持ち、作成後に現在状態を更新する場合は `updated_at` も持つ永続状態モデルの規約。作成後は不変で、消費または削除だけを行う記録モデルは `updated_at` を持たない。`issued_at`、`granted_at`、`occurred_at`、`expires_at`、`revoked_at` などのドメイン時刻は `created_at` を置き換えない。 |  |

型と制約の選び方は [database.md](database.md) が持つ。
