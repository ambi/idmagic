# RP と Entra フェデレーションの管理

## 概要

この文書は、WS-Fed の RP の登録、参照、削除と、Entra のドメインフェデレーションの定型設定の仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | RP の登録と更新、一覧、削除。Entra のドメインフェデレーションの定型設定による RP の作成と更新 |
| 行為者 | テナント管理者、管理 API のクライアント |
| 扱わないもの | アプリケーションに属する RP の設定は `Application` のプロトコル設定からも行える。クレームのポリシーの意味は `ClaimMapping` が定める |

## モデル

`EntraFederationProfile` は、WS-Fed の RP のための定型設定である。
ドメイン、IssuerUri、sourceAnchor の属性、受動、能動、MEX の各エンドポイントを受け取り、`wtrealm` と audience に同じ IssuerUri を持つ `WsFedRelyingParty` を作成または更新する。

| 必須のクレーム | 発行のしかた |
| --- | --- |
| UPN | `preferred_username` から `http://schemas.xmlsoap.org/claims/UPN` として発行する |
| ImmutableID | 正規化した sourceAnchor（`entra_immutable_id`）から導き、永続的な NameID と `http://schemas.xmlsoap.org/claims/nameidentifier` の両方に含める |

必須のクレームは定型設定で固定し、フェイルクローズで扱う。
sourceAnchor は、設定の時点では既存のユーザーの欠落、重複、変換できない値を拒否し、発行の時点では対象のユーザーの ImmutableID を導けなければクレームの発行を拒否する。

- **判断**：定型設定にする理由は、[Entra のドメインフェデレーションを専用の定型設定として扱う](../design/decisions.md#entra-のドメインフェデレーションを専用の定型設定として扱う)。

## 操作

### 管理 API のクライアントによる RP と Entra フェデレーションの操作

#### REQ-WSFEDERATION-001 管理 API クライアントは WS-Fed スコープの信頼設定だけを操作できる

- 同じ `wtrealm` の RP がない間、管理者が RP を登録したとき、WsFederation は、RP を作り、201 と保存した RP を返す。
- 同じ `wtrealm` の RP がある間、管理者が RP を登録したとき、WsFederation は、作成時刻を保って RP を置き換え、200 と保存した RP を返す。
- 管理者が RP を一覧したとき、WsFederation は、テナントのすべての RP を 200 で返す。
- 管理者が `wtrealm` を指定して RP を削除したとき、WsFederation は、その RP を消して 204 を返す。
- 管理者が Entra のドメインフェデレーションを構成したとき、WsFederation は、`wtrealm` と audience を IssuerUri（省略時は `urn:idmagic:entra:<ドメイン>`）とする SAML 1.1 の RP を作成または置き換え、`EntraFederationConfigured` を発行し、PowerShell に渡す設定値とともに返す。
- `wsfed:read` のスコープの API アクセストークンで呼び出されたとき、WsFederation は、RP の一覧だけを許可する。
- `wsfed:write` のスコープの API アクセストークンで呼び出されたとき、WsFederation は、RP の登録と削除と Entra のドメインフェデレーションの構成だけを許可する。
- `wsfed:write` を持たないトークンで変更の操作を要求された場合、WsFederation は、403 と `insufficient_scope` で拒否し、RP を変えない。
- 別のテナントで発行したトークンを提示された場合、WsFederation は、401 と `invalid_token` で拒否し、RP を変えない。
- `wtrealm`、返信先、`claim_policy.name_id` の `format` か `source_attribute` のない RP の登録を要求された場合、WsFederation は、400 と `invalid_request` で拒否し、RP を変えない。
- 2,048 文字または 2,048 バイトを超える `wtrealm` の RP の登録を要求された場合、WsFederation は、422 と `field_length_exceeded` で拒否し、RP を変えない。
- `wtrealm` を指定しない削除を要求された場合、WsFederation は、400 と `invalid_request` で拒否する。
- Application に属する RP の削除を要求された場合、WsFederation は、409 と `application_owned_protocol` で拒否し、RP を残す。
- ドメインがないか URL の形であるか、sourceAnchor の属性がないか、返信先が絶対 URL でない Entra の構成を要求された場合、WsFederation は、400 と `invalid_request` で拒否し、RP を変えない。
- テナントの既存の User のどれかで sourceAnchor の属性が欠けているか、変換できないか、ほかの User と重複する場合、WsFederation は、Entra の構成を 400 と `invalid_request` で拒否し、RP を変えない。
- `admin` のロールを持たない利用者が RP か Entra のドメインフェデレーションの管理 API を要求した場合、WsFederation は、403 と `access_denied` で拒否し、RP を変えない。
- **例**：EX-WSFEDERATION-001-01、EX-WSFEDERATION-001-02、EX-WSFEDERATION-001-03、EX-WSFEDERATION-001-04

## セキュリティ上の考慮

RP の登録、参照、削除と Entra のドメインフェデレーションの設定は、`AdminFederationTrustsManage` の権限（AuthZEN の action `admin:federation_trusts_manage`）を要する。
対話のセッションでは、`admin` ロールを持つ、有効かつ認証済みのユーザーが、所属テナントに対して行える。
