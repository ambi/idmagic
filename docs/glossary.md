# 用語集

Context を跨いで意味が固定される語を置く。ここに載っている語は、どの Context でも同じものを指す。

Context の `glossary.md` は、ここに載る語をその Context での役割へ**狭める**ことがある。狭めた定義がある Context の中では、そちらが読み方になる。狭めた先で別のものを指すようになったなら、それは同じ語ではなく、ここへ吸い上げて 1 つに揃える対象でもない。

1 つの Context の中でだけ意味が定まる語は、最初からその Context の `glossary.md` が持つ。

## 主体

| 用語 | 定義 | 別名 |
|---|---|---|
| EndUser | 認証済み、または認証を試みる一般の利用者。 |  |
| ResourceOwner | OAuth 2.0 / OIDC の認可フローでリソースの所有者として認可判断を行う利用者。EndUser と同じ人物を、その文脈で呼ぶときの名前である。 |  |
| Administrator | テナント内またはテナント横断のリソースを管理する権限を持つ利用者。 |  |
| Operator | IdMagic を配備し、起動時設定を与える運用者。権限ではなく実行環境そのものが境界になる操作を持つ。 |  |
| APIConsumer | HTTP API を直接呼び出す外部クライアント。 |  |
| Agent | 人の操作を伴わずに動く実行主体。所有者、目的、ライフサイクルを持ち、資格情報は OAuth クライアントか外部のアテステーションに束ねる。 |  |

主体の種類ごとにどの境界へ到達できるかは [認可設計](design/security/authorization.md) が持つ。

## ドメインモデル

| 用語 | 定義 | 別名 |
|---|---|---|
| Aggregate | 1 つの単位として変更されるドメインオブジェクトの集まり。ちょうど 1 つのルートエンティティを持ち、その識別子が全体を名指す。常に成り立たなければならない不変条件は 1 つの Aggregate の内側に収め、境界を越える整合は結果整合として明示的に組む。外部からはルートの識別子で参照し、内部の要素を直接指さない。1 つの Aggregate はちょうど 1 つの Bounded Context に属する。境界の引き方、トランザクションとの対応、Repository の粒度は [設計規則](design/application/design-rules.md#aggregate-境界と-repository) が、テナントに属する Aggregate が `tenant_id` を持つことは [データベース設計](design/data/database.md#tenant_id-の保持区分) が定める。 |  |
| Subdomain | Bounded Context を、事業上の差別化とモデルの複雑さで `Core`、`Supporting`、`Generic` のいずれかに分ける区分。全 Context の区分は [論理アーキテクチャ](architecture/logical.md#context-の責務) の索引表が持ち、ある Context が今の区分にある理由はその Context の `decisions.md` が持つ。区分が何を左右し、何を左右しないかは [設計規則](design/application/design-rules.md#subdomain-と設計投資) が定める。 | サブドメイン |

この 2 語は Latin 表記のまま使う。「集約」は日本語で観測値や設定をまとめる操作も指し、本文書群でも [容量設計](design/performance/capacity.md) と [Observability Design](design/observability/) がその意味で使っている。同じ語に 2 つの読みを持たせると、`tenant_id` を持つかどうかのような規則がどちらの意味で書かれているのか判別できなくなる。

## 外部契約

| 用語 | 定義 | 別名 |
|---|---|---|
| InterfaceStability | インターフェースの外部契約としての安定性の区分。`stable` は互換性を保証する外部契約、`beta` は保証の対象になる前の外部契約、`internal` はブラウザーセッション専用またはドメイン内部で外部契約に含めないインターフェースを表す。規則は [API 規則](design/application/api-rules.md) が持つ。 | 安定性区分 |
| Deprecation | `stable` または `beta` のインターフェースを将来削除することの予告。`deprecated_since` 以降は `Deprecation` ヘッダーを、`sunset_at` が定まれば `Sunset` ヘッダーも付与する。 | 非推奨化 |
| BackendErrorText | バックエンドが HTTP、OAuth / OIDC リダイレクト、SAML、SCIM などの外部レスポンスで返すエラー本文。`message`、`error_description`、`detail`、プレーンテキストの本文を含む。常に英語であり、表示言語によって変化しない。 | API エラーメッセージ |
| ConfigurationReference | バックエンドプロセスが起動時に読む設定キーの網羅的な一覧。キー名、値の型、デフォルト、必須かどうか、読み取るプロセス、説明を持つ。シークレットに分類したキーの値は持たない。起動時設定の定義から生成する。 | 設定リファレンス |

## 永続化の規約

| 用語 | 定義 | 別名 |
|---|---|---|
| PersistedStateModel | `created_at` を持ち、作成後に現在状態を更新する場合は `updated_at` も持つ永続状態モデルの規約。作成後は不変で、消費または削除だけを行う記録モデルは `updated_at` を持たない。`issued_at`、`granted_at`、`occurred_at`、`expires_at`、`revoked_at` などのドメイン時刻は `created_at` を置き換えない。 |  |

型と制約の選び方は [データベース設計](design/data/database.md) が持つ。
