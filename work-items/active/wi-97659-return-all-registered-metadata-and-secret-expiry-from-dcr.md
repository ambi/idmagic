---
status: pending
authors: [tn]
risk: medium
reversibility: irreversible
created_at: 2026-11-29
priority: p1
group: spec-conformance
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/oauth2/standards.md, requirement: RFC7591-REGISTER }
  - { path: docs/modules/oauth2/features/client.md, requirement: REQ-OAUTH2-016 }
  - { path: spec/modules/oauth2/main.tsp, symbol: IdMagic.Contract.RegisterClient }
  - { path: spec/modules/oauth2/models.tsp, symbol: IdMagic.Contract.ClientRegistrationRequest }
  - { path: spec/modules/oauth2/models.tsp, symbol: IdMagic.Contract.ClientRegistrationResponse }
---

# 動的クライアント登録で、登録したメタデータと client_secret の有効期限を返す

## 動機

2026-11-29 に [[wi-21740-decide-recordings-for-requests-idmagic-receives]] で、RFC 7591 §3.1 の二つの登録の要求の例を `/register` へ流した（`backend/oauth2/internal/handlers_http/testdata/rfc7591-examples.json`）。
どちらも 201 で登録されたが、応答と保存の状態に三つの食い違いがあった。

- 要求の `client_name`（`My Example Client`）は保存されるが、応答に含まれない。RFC 7591 §3.2.1 は「the authorization server MUST return all registered metadata about the client」と求め、IdMagic の `ClientRegistrationResponse` の説明も「登録したメタデータをサーバーが保存したとおりに繰り返し」と書く。
- 応答に `client_secret_expires_at` がない。RFC 7591 §3.2.1 は `client_secret` を発行したときに REQUIRED とし、期限がなければ 0 を返す。
- 動的登録は有効期限のない `ClientSecretCredential` を作る（`backend/oauth2/internal/client/usecases/register_client.go`）。`ClientSecretCredential.expires_at` の説明は「新たに加える資格情報では必須である」と定める。

あわせて、`ClientRegistrationRequest.grant_types` は TypeSpec で必須だが、実装は RFC 7591 §2 のとおり省略を `authorization_code` とみなして受け付ける。契約の記述が実装と規範に追いついていない。

規範に反し、クライアントは secret の期限を知る手段を持たないので、p1 とする。

## 対象範囲

- 動的登録の secret の有効期限の方針を決める（管理者が作る secret と同じ既定の期限を付けるか、期限なしを契約の例外として明記するか）。
- 応答に `client_secret_expires_at`（期限なしなら 0）と、保存した `client_name` を返す。`client_id_issued_at` を返すかも決める。
- `ClientRegistrationRequest.grant_types` を任意にし、省略の既定を説明に書く。
- `rfc7591_examples_test.go` に、上の三つの期待値の検査を足す。

## 対象外

- 言語の付いたメタデータ（`client_name#ja-Jpan-JP`）の保存。今は無視しており、RFC 7591 §2.2 はそれを許す。
- 理解しないメタデータ（`example_extension_parameter`、`logo_uri`、`policy_uri`）の保存。RFC 7591 §2 のとおり無視している。

## 設計

secret の期限は、管理者の secret の発行（ClientSecretCredential の発行の規則）と同じ所有者に置き、動的登録はそれを使う。

## 計画

1. `spec-change` で TypeSpec と要件を更新する。
2. RFC 7591 の例の再生に期待値を足して RED を観測し、実装を直す。

## タスク

- [ ] T001 [Spec] 契約と要件を更新する。
- [ ] T002 [Acceptance] 例の再生で RED を確認する。
- [ ] T003 [App] 応答と secret の期限を直す。
- [ ] T004 [Verify] 差分から選んだ検査を実行する。

## 検証

- `mise run test-go-package -- ./backend/oauth2/internal/handlers_http`、`mise run test-go-package -- ./backend/oauth2/internal/client/usecases`、`mise run check-api-compat`。

## リスク

既存の動的登録のクライアントの secret に後から期限を付けると、そのクライアントが突然認証できなくなる。
期限は新しく発行する secret にだけ付け、既存の資格情報は変えない。
