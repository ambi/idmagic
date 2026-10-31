# 認可モデル

## 概要

この文書は、テナントの認可モデルをバージョンとして登録する仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | 認可モデルのバージョンの登録、型と関係の検証、書き換え規則の循環の拒否 |
| 行為者 | テナント管理者 |
| 扱わないもの | 関係タプルの書き込みは[関係タプル](relation-tuple.md)が扱う |

## モデル

型の名前と関係の名前は `^[a-z][a-z0-9_]*$` で、最大 64 文字とする。
`RelationDefinition` は書き換え規則の和で成立の条件を表し、規則が空の関係は決して成り立たない。

| 規則 | 意味 |
| --- | --- |
| `direct` | 直接の関係タプル。`direct_subject_types` が受け入れる主体の形を宣言する。`user` は個別の主体、`group#member` は subject set、`user:*` はワイルドカードを表す |
| `computed_userset` | 同じオブジェクトの上の別の関係へ委ねる（`viewer` は `editor` を含む） |
| `tuple_to_userset` | `tupleset_relation` でたどった先のオブジェクトで `computed_relation` を判定する（`document#parent` をたどって `folder#viewer` を見る） |

## 操作

### 管理者による認可モデルの登録

#### REQ-AUTHORIZATION-001 管理者は認可モデルをバージョンとして登録でき、整合しないモデルは拒否される

- 管理者が認可モデルを登録したとき、AccessControl は、テナントの中で単調に増える新しいバージョンを作り、以前のバージョンを書き換えず、整合トークンを返し、`AuthorizationModelPublished` を発行する。
- 管理者が認可モデルを取得したとき、AccessControl は、最新のバージョンを返す。
- 宣言していない型か関係を参照する定義、循環する書き換えの規則、`^[a-z][a-z0-9_]*$` の形でないか 65 文字以上の型名か関係名を指定された場合、AccessControl は、422 と `authorization_model_invalid` で拒否し、バージョンを作らない。
- テナントに認可モデルを登録していない間、管理者が認可モデルを取得したとき、AccessControl は、404 と `authorization_model_not_found` で拒否する。
- **例**：EX-AUTHORIZATION-001-01、EX-AUTHORIZATION-001-02、EX-AUTHORIZATION-001-03、EX-AUTHORIZATION-001-04

#### REQ-AUTHORIZATION-011 管理コンソールは認可モデルの発行を確定する前に差分を示す

- 管理者が管理コンソールで発行の内容を確認するとき、AccessControl は、その時点の最新のバージョンと比べて追加と削除される型、関係、書き換えの規則を示す。
- 確認した差分に削除が含まれる場合、AccessControl は、既存の関係タプルが判定に数えられなくなることがあると警告する。
- 管理者が管理コンソールで発行を確定するまでの間、AccessControl は、新しいバージョンを作らない。

#### REQ-AUTHORIZATION-010 認可モデルとタプルの更新も判定の呼び出しも管理者に限られる

- `admin` のロールを持たない利用者が認可モデルの登録と取得、関係タプルの書き込みと一覧、判定、列挙を要求した場合、AccessControl は、403 と `access_denied` で拒否し、バージョンもタプルも作らない。
- API アクセストークンでこのモジュールの管理 API を要求された場合、AccessControl は、トークンのスコープによらず 403 と `insufficient_scope` で拒否する。
- **例**：EX-AUTHORIZATION-010-01、EX-AUTHORIZATION-010-02

## セキュリティ上の考慮

認可モデルと関係タプルの管理は、`AdminAuthorizationModelManage` の権限（AuthZEN の action `admin:authorization_model_manage`）を要する。
この権限はテナント管理者に属し、テナントの境界を越えない。

このモジュールの管理 API は対話のセッションに限り、API アクセストークンからは、どのスコープを持っていても到達できない。理由は[管理 API を対話のセッションに限る](../decisions.md#管理-api-を対話のセッションに限る)。
