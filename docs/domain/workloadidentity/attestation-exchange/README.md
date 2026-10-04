# アテステーションの交換

## 概要

この文書は、外部のワークロードが提示したアテステーションを検証し、対応先の `Agent` を一意に特定する仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | 登録した信頼設定による署名、`iss`、`aud`、`exp`、TTL の上限の検証、関連付けによる `Agent` の一意な特定、`Agent` の状態の確認 |
| 行為者 | System（`OAuth2` の Token Exchange から呼ばれる検証のユースケース） |
| 扱わないもの | アクセストークンの発行は `OAuth2` が扱う。信頼設定と関連付けの登録は[信頼設定と関連付けの管理](../trust-configuration/README.md)が扱う |

## モデル

検証は、いずれかの段で失敗すればフェイルクローズで拒否し、拒否の理由とともに `WorkloadAttestationRejected` を発行する。

| 拒否の理由 | 条件 |
| --- | --- |
| `unregistered_issuer` | トークンの `iss` に一致する信頼設定がテナントにない。署名の検証で発行者が一致しない |
| `trust_bundle_disabled` | 一致した信頼設定が `disabled` である |
| `invalid_signature` | 登録した JWKS で署名を検証できない |
| `jwks_unavailable` | JWKS を取得できず、使える鍵もない |
| `audience_mismatch` | `aud` が受理する audience に含まれない |
| `expired` | `exp` を過ぎている |
| `ttl_exceeded` | `exp` と `iat` の差が信頼設定の最大 TTL を超える |
| `no_binding_match` | 主体がどの有効な関連付けのパターンにも一致しない |
| `ambiguous_match` | 主体が複数の有効な関連付けに一致する |
| `agent_not_active` | 対応先の `Agent` が存在しないか `Active` でない |
| `agent_unbound` | 対応先の `Agent` が `OAuth2Client` に束縛されていない |

- **判断**：複数の一致を拒否する理由は、[複数の関連付けに一致した主体は優先順位で選ばず拒否する](../design/decisions.md#複数の関連付けに一致した主体は優先順位で選ばず拒否する)。

## 操作

### ワークロードによるトークンの交換

#### REQ-WORKLOADIDENTITY-001 登録済みの信頼設定を使ってワークロードトークンを Agent 資格情報に交換できる

#### REQ-WORKLOADIDENTITY-006 対応先 Agent が Killed になった後は交換を拒否する

#### REQ-WORKLOADIDENTITY-002 未登録の発行者を拒否する

#### REQ-WORKLOADIDENTITY-003 署名が不正なアテステーションを拒否する

#### REQ-WORKLOADIDENTITY-004 期限切れのアテステーションを拒否する

#### REQ-WORKLOADIDENTITY-005 複数の関連付けに一致して Agent を一意に決められない主体を拒否する

#### REQ-WORKLOADIDENTITY-007 他テナントの信頼設定は利用できない

## セキュリティ上の考慮

交換の経路は、管理者の権限を通らない。
外部のワークロードが提示するのはアテステーションのトークンだけであり、得られる権限は、登録した信頼設定と関連付けが定めた `Agent` のものに固定される。
トークンの内容が対応先の `Agent` を変えることはなく、未登録の発行者を実行時に信頼することもない（Trust On First Use を許可しない）。

信頼設定は、要求を受けたテナントの中だけで探す。
ほかのテナントに同じ発行者の信頼設定があっても、参照しない。
