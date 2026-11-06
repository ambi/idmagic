---
status: pending
authors: [tn]
risk: high
reversibility: irreversible
created_at: 2026-11-07
priority: p1
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/modules/workloadidentity/features/attestation-exchange.md, requirement: REQ-WORKLOADIDENTITY-001 }
  - { path: docs/modules/workloadidentity/features/attestation-exchange.md, requirement: REQ-WORKLOADIDENTITY-002 }
  - { path: docs/modules/workloadidentity/features/attestation-exchange.md, requirement: REQ-WORKLOADIDENTITY-003 }
  - { path: docs/modules/workloadidentity/features/trust-configuration.md, requirement: REQ-WORKLOADIDENTITY-008 }
  - { path: spec/modules/workloadidentity/models.tsp, symbol: IdMagic.Contract.WorkloadTrustBundle }
---

# SPIFFE の信頼設定では、JWT-SVID を SPIFFE の規則どおりに検証する

## 動機

WorkloadIdentity は JWT-SVID を主なアテステーションとして挙げるが、検証は OIDC 互換の汎用 JWT と同じで、SPIFFE 固有の規則を確かめていない。

| 規則（SPIFFE JWT-SVID 仕様） | 現在の実装 |
| --- | --- |
| `sub` は SPIFFE ID（`spiffe://<trust-domain>/<path>`）である | 空でないことだけを確かめる |
| `sub` のトラストドメインは、検証に使ったバンドルのトラストドメインと一致する | `WorkloadTrustBundle.TrustDomain` を保存するが、検証に使わない |
| JOSE ヘッダーの `kid` は必須 | 鍵が一つなら `kid` なしを受け付ける（`pickJWK` の共通の挙動） |
| `typ` は省略か `JWT` または `JOSE` | 確かめない |
| `alg` は RS/ES/PS の 256/384/512 | PS256、ES256、RS256 だけを受け付ける |
| `iss` は任意 | 必須で、信頼設定の検索のキーにしている |

SPIRE は `jwt_issuer` を設定しない限り JWT-SVID に `iss` を入れない。
既定の SPIRE が発行した JWT-SVID は、現在 `unregistered_issuer` で拒否される。
また、別のトラストドメインの SPIFFE ID を `sub` に持つトークンを、`iss` と鍵が一致する限り受け付けてしまう。

## 対象範囲

- 信頼設定に検証の種別（たとえば `profile: oidc | spiffe`）を足す。既存の信頼設定は `oidc` とみなす。
- `spiffe` の信頼設定では、`sub` の SPIFFE ID の構文、`sub` のトラストドメインと `TrustDomain` の一致、`kid` の存在、`typ` の値を検証する。
- 受け付ける `alg` に RS384、RS512、ES384、ES512、PS384、PS512 を足すかを決める。
- `iss` のない JWT-SVID の扱い（信頼設定の探し方）を決めて実装する。
- 新しい拒否の理由（たとえば `invalid_spiffe_id`、`trust_domain_mismatch`）を、拒否の理由の表、TypeSpec、管理画面の拒否の一覧へ足す。
- 管理画面の信頼設定のフォームで、種別を選べるようにする。

## 対象外

- SPIFFE Bundle Endpoint の形式の扱い。[wi-88550](wi-88550-read-spiffe-bundle-endpoint-format.md) が扱う。
- X.509-SVID。[wi-73570](wi-73570-accept-x509-svid-over-mtls.md) が扱う。
- `oidc` の信頼設定（Kubernetes の ServiceAccount トークンなど）の検証の変更。

## 設計

### 検証の種別を信頼設定に持たせる

Kubernetes の投影 ServiceAccount トークンの `sub` は `system:serviceaccount:<ns>:<sa>` であり、SPIFFE ID の検証を全体に掛けられない。
信頼設定ごとに種別を持たせ、`spiffe` のときだけ SPIFFE の規則を足す。

採らない案：`sub` が `spiffe://` で始まるときだけ SPIFFE の規則を掛ける。
トークンの内容で検証の厳しさが変わり、攻撃者が `sub` の形を選べば規則を外せるため採らない。

### `iss` のない JWT-SVID

候補は次のとおり。着手時にどちらにするか決める。

| 案 | 内容 | 欠点 |
| --- | --- | --- |
| A | `spiffe` の信頼設定では `issuer` を任意にし、署名の前に `sub` のトラストドメインを読んで信頼設定を探す。署名の検証後にトラストドメインの一致を確かめる | 信頼設定の一意性の制約が `issuer` から `trust_domain` へ移る |
| B | `iss` を必須のままにし、SPIRE 側で `jwt_issuer` の設定を求める | 既定の SPIRE では使えず、運用の手順が増える |

案 A を推す。
SPIFFE の信頼の単位はトラストドメインであり、`iss` はその代わりにならない。
署名前に読んだ値を信頼設定の選択にだけ使う現在の規則（[実現方式](../../docs/modules/workloadidentity/features/attestation-exchange.md#実現方式)）は保てる。

### 型と作用

検証は `tokens_jose.VerifyWorkloadSVID` に、種別とトラストドメインを入力として渡す純粋な判定のままにする。
SPIFFE ID の構文の判定は、`workloadidentity/domain` の値型（たとえば `SpiffeID`）として置き、パースと正規化（大文字の禁止、`.`、`..` の拒否、末尾の `/` の拒否）をそこに閉じる。

## 計画

1. 案 A と案 B を決め、仕様と TypeSpec を更新する。
2. SPIFFE ID の値型と、種別ごとの検証を実装する。
3. 管理 API と管理画面に種別を足す。既存の行は移行で `oidc` にする。

## タスク

- [ ] T001 [Spec] 種別、`iss` の扱い、新しい拒否の理由を仕様と TypeSpec に書く。
- [ ] T002 [Acceptance] 別のトラストドメインの `sub` を持つ JWT-SVID が受け付けられることを RED にする。
- [ ] T003 [Domain] SPIFFE ID の値型を、構文の表に沿ったテストとともに作る。
- [ ] T004 [App] 種別ごとの検証と、`iss` のない JWT-SVID の信頼設定の探索を実装する。
- [ ] T005 [Adapter] スキーマの移行と、管理 API の種別のフィールドを足す。
- [ ] T006 [UI] 信頼設定のフォームと拒否の一覧に種別と新しい理由を足す。
- [ ] T007 [Verify] 検証を実行する。

## 検証

- `mise run test-go-package -- ./backend/workloadidentity/...` と `./backend/shared/security/tokens_jose/...`：検証の規則の変更。
- `mise run test-go-package -- ./backend/oauth2/...`：Token Exchange の経路。
- `mise run check-schema`：TypeSpec と移行の変更。
- 管理画面のフォームのテスト。

## リスク

- 検証を厳しくするため、既存の信頼設定を `spiffe` に変えると、それまで通っていたトークンが拒否されうる。既存の行は `oidc` のまま移行し、種別の変更は管理者の明示の操作に限る。
- 信頼設定の一意性の制約を変えると、同じトラストドメインを持つ既存の行が衝突しうる。移行前に重複を数える。
