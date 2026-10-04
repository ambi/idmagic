# サービスプロバイダーの管理

## 概要

この文書は、SAML の SP を登録、参照、削除する仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | SP の登録と更新、一覧、削除 |
| 行為者 | テナント管理者、管理 API のクライアント |
| 扱わないもの | アプリケーションに属する SP の設定は `Application` のプロトコル設定からも行える。クレームのポリシーの意味は `ClaimMapping` が定める |

## 操作

### 管理 API のクライアントによる SP の操作

#### REQ-SAML-005 管理 API クライアントは SAML スコープに従ってサービスプロバイダーを操作できる

- 同じ entityID の SP がない間、管理者が SP を登録したとき、Saml は、SP を作り、201 と保存した SP を返す。
- 同じ entityID の SP がある間、管理者が SP を登録したとき、Saml は、作成時刻を保って SP を置き換え、200 と保存した SP を返す。
- 管理者が SP を登録したとき、Saml は、Assertion への署名をデフォルトで有効にし、IdP プロファイルを省略した SP を `default`（既存の SP では現在のプロファイル）に関連付ける。
- 管理者が SP を一覧したとき、Saml は、テナントのすべての SP を 200 で返す。
- 管理者が entityID を指定して SP を削除したとき、Saml は、その SP を消して 204 を返す。
- `saml:read` のスコープの API アクセストークンで呼び出されたとき、Saml は、SP の参照だけを許可する。
- `saml:write` のスコープの API アクセストークンで呼び出されたとき、Saml は、SP の登録と削除だけを許可する。
- `saml:write` を持たないトークンで変更の操作を要求された場合、Saml は、403 と `insufficient_scope` で拒否し、SP を変えない。
- 別のテナントで発行したトークンを提示された場合、Saml は、401 と `invalid_token` で拒否し、SP を変えない。
- 不正な SP の登録を要求された場合、Saml は、400 と `invalid_request` で拒否し、SP を変えない。
- 長さの上限を超える項目の SP の登録を要求された場合、Saml は、422 と `field_length_exceeded` で拒否し、SP を変えない。
- entityID を指定しない削除を要求された場合、Saml は、400 と `invalid_request` で拒否する。
- Application に属する SP の削除を要求された場合、Saml は、409 と `application_owned_protocol` で拒否し、SP を残す。
- `admin` のロールを持たない利用者が SP の管理 API を要求した場合、Saml は、403 と `access_denied` で拒否し、SP を変えない。
- **例**：EX-SAML-005-01、EX-SAML-005-02、EX-SAML-005-03、EX-SAML-005-04

## セキュリティ上の考慮

SP の登録、参照、削除は、`AdminFederationTrustsManage` の権限（AuthZEN の action `admin:federation_trusts_manage`）を要し、`admin` ロールを持つ、有効かつ認証済みのユーザーが、所属テナントに対して行える。
