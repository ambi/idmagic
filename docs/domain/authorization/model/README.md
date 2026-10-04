# 認可モデル

## 概要

この文書は、テナントの認可モデルを版として登録する仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | 認可モデルの版の登録、型と関係の検証、書き換え規則の循環の拒否 |
| 行為者 | テナント管理者 |
| 扱わないもの | 関係タプルの書き込みは[関係タプル](../relation-tuple/README.md)が扱う |

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

#### REQ-AUTHORIZATION-001 管理者は認可モデルを版として登録でき、整合しないモデルは拒否される

#### REQ-AUTHORIZATION-010 認可モデルとタプルの更新も判定の呼び出しも管理者に限られる

## セキュリティ上の考慮

認可モデルと関係タプルの管理は、`AdminAuthorizationModelManage` の権限（AuthZEN の action `admin:authorization_model_manage`）を要する。
この権限はテナント管理者に属し、テナントの境界を越えない。

この Context の管理 API は対話のセッションに限り、API アクセストークンからは、どのスコープを持っていても到達できない。理由は[管理 API を対話のセッションに限る](../design/decisions.md#管理-api-を対話のセッションに限る)。
