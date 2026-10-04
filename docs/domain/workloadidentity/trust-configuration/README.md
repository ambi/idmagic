# 信頼設定と関連付けの管理

## 概要

この文書は、テナント管理者が `WorkloadTrustBundle` と `AgentWorkloadBinding` を管理する仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | 信頼設定の登録、更新、無効化と再有効化、削除、JWKS の再取得。関連付けの作成、無効化と再有効化、削除 |
| 行為者 | テナント管理者 |
| 扱わないもの | 登録した信頼設定によるトークンの検証は[アテステーションの交換](../attestation-exchange/README.md)が扱う |

## モデル

信頼設定は、JWKS の取得元（`jwks_uri`）かインラインの `jwks` のどちらかを持つ。
JWKS の再取得は、保存する鍵素材を差し替えるので、変更の操作として扱う。

## 状態遷移

### WorkloadTrustBundleLifecycle

登録時に `enabled` として作成する。無効化すると `disabled` に遷移し、それ以降は配下の関連付けを交換に使えない。再有効化すれば `enabled` に戻せる。削除は状態遷移ではなくレコードそのものを取り除く終端操作であり、配下の関連付けもカスケード削除する。

| State | Kind | Meaning |
|---|---|---|
| enabled | initial | 配下の関連付けを交換に使える |
| disabled | — | 配下の関連付けを交換に使えない。再有効化できる |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| enabled | WorkloadTrustBundleDisabled | — | disabled |  |
| disabled | WorkloadTrustBundleEnabled | — | enabled |  |

### AgentWorkloadBindingLifecycle

作成時は `enabled` とする。無効化すると `disabled` に遷移し、それ以降の交換には使えない。再有効化すれば `enabled` に戻せる。削除は状態遷移ではなくレコードそのものを取り除く終端操作である。

| State | Kind | Meaning |
|---|---|---|
| enabled | initial | 交換に使える |
| disabled | — | 交換に使えない。再有効化できる |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| enabled | AgentWorkloadBindingDisabled | — | disabled |  |
| disabled | AgentWorkloadBindingEnabled | — | enabled |  |

## 操作

### 管理者による信頼設定の登録

#### REQ-WORKLOADIDENTITY-008 管理者は信頼設定を登録・無効化・再有効化できる

#### REQ-WORKLOADIDENTITY-010 信頼設定と関連付けの管理は管理者に限られる

### 管理者による関連付けの作成

#### REQ-WORKLOADIDENTITY-009 管理者は他テナントの Agent への関連付けを作成できない

## セキュリティ上の考慮

信頼設定と関連付けの登録、更新、無効化、削除は、`admin` ロールを持つ、有効かつ認証済みのユーザーだけが、所属テナントに対して行える。
関連付けの対象にできる `Agent` も、同じテナントのものに限る。
