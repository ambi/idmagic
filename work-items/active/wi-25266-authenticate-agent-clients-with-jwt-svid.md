---
status: pending
authors: [tn]
risk: high
reversibility: irreversible
created_at: 2026-11-11
priority: p1
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/modules/oauth2/standards.md, requirement: AIMS00-AGENT-CLIENT-AUTH }
  - { path: docs/modules/oauth2/standards.md, requirement: RFC7521-CLIENT-ID-OPTIONAL }
  - { path: docs/modules/oauth2/standards.md, requirement: RFC7523-CLIENT-ASSERTION }
  - { path: docs/modules/workloadidentity/features/attestation-exchange.md, requirement: REQ-WORKLOADIDENTITY-011 }
  - { path: spec/modules/application/models.tsp, symbol: IdMagic.Contract.TokenEndpointAuthMethod }
  - { path: spec/modules/oauth2/models.tsp, symbol: IdMagic.Contract.DiscoveryDocument }
---

# JWT-SVID をクライアントアサーションとして受け付け、Agent のクライアントを長期の鍵なしで認証する

## 動機

WorkloadIdentity の設計判断は「長期のシークレットを配らない」ことを掲げる。
しかし、JWT-SVID を使う今の経路は Token Exchange の `subject_token` だけであり、トークンエンドポイントは別にクライアント認証を求める。
Agent に束縛したクライアントは AIMS00-AGENT-CLIENT-AUTH によって `private_key_jwt` か `tls_client_auth` に限られる。
そのため、SPIRE の基盤を持つ Agent でも、クライアント認証のための長期の鍵を別に配って保管する必要が残る。

IETF OAuth WG の [draft-ietf-oauth-spiffe-client-auth-03](https://datatracker.ietf.org/doc/html/draft-ietf-oauth-spiffe-client-auth-03)（2026-11）は、JWT-SVID をクライアントアサーションとして送る方式（`client_assertion_type=urn:ietf:params:oauth:client-assertion-type:jwt-spiffe`、`token_endpoint_auth_method=spiffe_jwt`）を定める。
[draft-ietf-wimse-aims-00](https://www.ietf.org/archive/id/draft-ietf-wimse-aims-00.html) の 10.4 節も、Agent のクライアント認証の方式としてこのドラフトを挙げる。
Entra の Workload Identity Federation もクライアントアサーションの形を採る。

## 対象範囲

- クライアントの認証方式に `spiffe_jwt` を足し、`client_assertion_type` が `jwt-spiffe` の要求を、ドラフトの 3.2 節に従って検証する。
  - `aud` は認可サーバーの発行者識別子だけを値に持つ。
  - 署名は、`sub` のトラストドメインの `spiffe` の信頼設定の `jwt-svid` 鍵で検証する（REQ-WORKLOADIDENTITY-003 と同じ鍵の選択）。
  - 有効期限と、信頼設定の最大 TTL を検証する。
  - SPIFFE の JOSE ヘッダーの規則（REQ-WORKLOADIDENTITY-011）を適用する。
- SPIFFE ID とクライアントの対応付けを、ドラフトの 3.1 節の「登録による対応付け」として定める（設計を参照）。
- `client_id` を省いた要求では、ドラフトの 3.1 節に従い、SPIFFE ID と等しい `client_id` のクライアントを探す。見つからなければ `invalid_client` で拒否する。
- Authorization Server Metadata の `token_endpoint_auth_methods_supported`、`introspection_endpoint_auth_methods_supported`、`revocation_endpoint_auth_methods_supported` に `spiffe_jwt` を載せる。
- AIMS00-AGENT-CLIENT-AUTH を改め、`spiffe_jwt` を鍵の所有を示さない共有シークレットの方式と区別し、Agent のクライアントに許す。
- 管理 API と管理画面で、Agent のクライアントの認証方式に `spiffe_jwt` を選べるようにする。
- 成功と拒否の監査イベント（拒否の理由を含む）。

## 対象外

- X.509-SVID による認証（`spiffe_x509`）。[[wi-73570-accept-x509-svid-over-mtls]] が扱う。
- WIT-SVID による認証（`spiffe_wit`）。[[wi-27035-decide-wit-svid-and-wimse-proof-token-support]] が判断する。
- Client ID Metadata Document の `spiffe_id` と `spiffe_bundle_endpoint`。IdMagic は CIMD の `private_key_jwt` も扱わない（CIMD02-PRIVATE-KEY-JWT）ため、同じく除く。
- Agent に束縛していないクライアントへの `spiffe_jwt`。需要が出たときに、対応付けの方法と合わせて検討する。
- Token Exchange で、要求したクライアントと束縛先のクライアントの関係を決めること。[[wi-69290-bind-workload-token-exchange-to-the-bound-client]] が扱う。

## 設計

### SPIFFE ID とクライアントの対応付け

| 案 | 内容 | 利点 | 欠点 |
| --- | --- | --- | --- |
| A（推す） | 既存の `AgentWorkloadBinding` で SPIFFE ID から Agent を決め、要求の `client_id` が、その Agent に束縛したクライアントのどれかであることを確かめる | 対応付けの登録先が一つで済み、管理画面、試行、重なりの警告、無効化をそのまま使える。複数一致の拒否（REQ-WORKLOADIDENTITY-005）も同じ規則になる | 対応付けの先がクライアントではなく Agent なので、`client_id` の指定が要る。Agent のないクライアントは使えない |
| B | `OAuth2Client` に `spiffe_id`（完全一致か末尾の `/*` による前方一致）を持たせる | ドラフトのクライアントメタデータと同じ形で、Agent のないクライアントにも使える | SPIFFE ID の対応付けが関連付けと二か所に分かれ、同じワークロードの対応先が食い違いうる。OAuth2 が信頼設定を引く経路も別に要る |

案 A を推す。
ドラフトは登録済みのクライアントについて「登録による対応付け」を求めるだけであり、その方法を定めない。
IdMagic のクライアント識別子は SPIFFE ID ではないため、`client_id` を省いた要求は事実上拒否になる。
この挙動は管理画面と接続例で案内する。

### 再生の扱い

JWT-SVID は Bearer であり、SPIRE が発行する JWT-SVID は `jti` を持たないことがある。
ドラフトも `jti` を求めない。
`aud` を発行者だけに限ることと、信頼設定の最大 TTL で再生の窓を狭める。
`jti` があるときは、既存のクライアントアサーションの再生の記録（`ClientAssertionReplayStore`）で再利用を拒否する。
`jti` を必須にする案は、デフォルトの SPIRE で使えなくなるため採らない。

### 境界の判断の記録

- 入力：AIMS00-AGENT-CLIENT-AUTH、RFC7523-CLIENT-ASSERTION、REQ-WORKLOADIDENTITY-003、REQ-WORKLOADIDENTITY-005、REQ-WORKLOADIDENTITY-011。既存の `oauth2/ports.WorkloadTokenVerifier` は、OAuth2 が定義して WorkloadIdentity が実装し、組み立て地点で結ぶポートである。
- 適用する制約：D1、D3、D4、D6。
- 候補 1（推す）：OAuth2 の `WorkloadTokenVerifier` に、期待する audience を引数に取るクライアントアサーション用の操作を足す。SPIFFE の検証と対応付けは WorkloadIdentity に残し、OAuth2 のクライアント認証は、検証の結果（Agent、トラストドメイン、SPIFFE ID）を自分のクライアントの束縛と照合する。依存の向きと組み立て地点は変わらない。
- 候補 2：OAuth2 が `tokens_jose` の SPIFFE バンドルの部品を直接使い、信頼設定を読む。WorkloadIdentity の非公開の状態（信頼設定）を OAuth2 が読むことになり、D3 に反する。
- 候補 3：WorkloadIdentity がトークンエンドポイントのクライアント認証を担う。クライアント認証の規則がモジュールをまたいで分かれ、AIMS00 の判定と食い違いうる。
- 変更シナリオ：ドラフトが WIT-SVID や X.509-SVID を足しても、候補 1 なら WorkloadIdentity の検証の追加と、ポートの操作の追加で済む。
- 残る仮定：一つの Agent が束縛するクライアントは少数であり、`client_id` での照合で足りる。

### 主要な型と操作

- `TokenEndpointAuthMethod` に `spiffe_jwt` を足す。`OAuth2Client.ProvesKeyPossession` に相当する判定を「長期のシークレットに依らない」意味へ改め、名前を見直す。
- `WorkloadTokenVerifier.VerifyClientAssertion(ctx, tenantID, assertion, expectedAudience, now) (*WorkloadIdentityGrant, error)`。`WorkloadIdentityGrant` に検証した SPIFFE ID を足す。
- 時刻はユースケースの引数として受ける。JWKS の取得は既存の取得部品を使う。

## 計画

1. 境界の判断と再生の扱いをレビューで確定する。
2. OAuth2 の標準の表（AIMS00、新しいドラフトの節）と WorkloadIdentity の機能仕様、TypeSpec を更新する。
3. HTTP の受け入れテストを RED にし、検証、照合、メタデータ、管理 API、画面の順に実装する。
4. [[wi-76418-exchange-spire-issued-jwt-svid-end-to-end]] の SPIRE のテストに、この経路の確認を足す。

## タスク

- [ ] T001 [Design] 対応付けの案と再生の扱いを確定する。
- [ ] T002 [Spec] `standards.md` に draft-ietf-oauth-spiffe-client-auth-03 の節を足し、AIMS00-AGENT-CLIENT-AUTH を改める。WorkloadIdentity の仕様と TypeSpec を更新する。
- [ ] T003 [Acceptance] JWT-SVID のクライアントアサーションで `client_credentials` のトークンを得る経路と、`aud`、トラストドメイン、対応付け、`client_id` の不一致による拒否を RED にする。
- [ ] T004 [App] WorkloadIdentity にクライアントアサーションの検証を実装する。
- [ ] T005 [App] OAuth2 のクライアント認証とメタデータを実装する。
- [ ] T006 [UI] クライアントの認証方式に `spiffe_jwt` を足す。
- [ ] T007 [Verify] 検証を実行する。

## 検証

- `mise run test-go-package -- ./backend/workloadidentity/...` と `./backend/oauth2/...`：検証と照合の規則を確かめる。
- `mise run check-schema`：TypeSpec の列挙と、メタデータの契約を確かめる。HTTP の統合テストとあわせて localhost の待受けを使うため、サンドボックスの外で実行する。
- `mise run test-ui-e2e-file` で、クライアントの設定画面を確かめる。
- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>` とする。

## リスク

- JWT-SVID は Bearer である。漏えいしたら、有効期間の間は成りすましができる。最大 TTL を短く保つよう管理画面で案内し、送信者制約（DPoP）を推奨する。
- 公開契約に認証方式の値を足すため、後から取り除けない。ドラフトの値の名前が変わった場合に備え、標準の表にドラフトのバージョンを記す。
- AIMS00 の緩和を誤ると、共有シークレットの Agent を許してしまう。許す方式を列挙で定め、それ以外の拒否をテストで固定する。
