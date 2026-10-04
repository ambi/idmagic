# 同意

## 概要

この文書は、利用者がクライアントへ与える同意の付与、参照、撤回の仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | 既存の同意による同意画面の出し分け、本人による撤回、管理者による参照と撤回、テナントの境界 |
| 行為者 | ResourceOwner、本人、API トークンの発行者、テナント管理者 |
| 扱わないもの | 認可コードの発行は[認可](../authorization/README.md)が扱う |

## モデル

同意は `(subject, client_id)` ごとに、付与済みのスコープの集合として永続化する。

- **判断**：この単位にする理由は、[同意を subject と client_id の組ごとのスコープの集合として持つ](../design/decisions.md#同意を-subject-と-client_id-の組ごとのスコープの集合として持つ)。

## 状態遷移

### ConsentLifecycle

同意レコードのライフサイクル。GDPR Art.7(3) により Granted → Revoked が可能。

| State | Kind | Meaning |
|---|---|---|
| Granted | initial | 付与済みスコープ集合が有効である |
| Revoked | terminal | 利用者が取り消した。以後の認可には使えない |
| Expired | terminal | 付与から 365 日が経過した |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Granted | RevokeConsent | — | Revoked |  |
| Granted | Expire | — | Expired |  |

## 操作

### 利用者による認可での同意

#### REQ-OAUTH2-008 既存同意の有無に応じて同意画面を出し分ける

### 本人による同意の撤回

#### REQ-OAUTH2-032 ユーザーは接続済みアプリの同意を自分で撤回できる

#### REQ-OAUTH2-002 API トークン発行者は account 同意スコープで自分の同意だけを操作できる

### 管理者による同意の管理

#### REQ-OAUTH2-031 管理者は所属テナントの同意を参照・撤回できるが付与は代行できない

#### REQ-OAUTH2-038 同意管理 API は別テナントの同意を公開しない

## セキュリティ上の考慮

同意の管理（`admin:consents_manage`）は、`admin` ロールを持つ、有効かつ認証済みのユーザーが、所属テナントに対して行う。
管理者は同意を参照し撤回できるが、利用者に代わって付与することはできない。
