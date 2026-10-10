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

- ワークロードが JWT のアテステーションを `subject_token` として Token Exchange を要求したとき、WorkloadIdentity は、`iss` に一致するテナントの `enabled` の信頼設定を探し、登録した JWKS で署名を、受理する audience で `aud` を、`exp` と最大 TTL で有効期間を検証する。
- 検証を通った主体がちょうど一つの `enabled` の関連付けのパターンに一致し、対応先の Agent が `Active` で `OAuth2Client` に束縛されている間、交換を要求されたとき、WorkloadIdentity は、その Agent の束縛先の `client_id` の資格情報として短い有効期間のアクセストークンを発行させ、信頼設定、関連付け、Agent、audience を載せた `WorkloadTokenExchanged` を発行する。
- 関連付けのパターンを照合するとき、WorkloadIdentity は、パスのワイルドカード（`*` と `?`）を一つの階層の中だけで一致させる。
- 交換のどの段で拒否する場合、WorkloadIdentity は、OAuth2 の 400 と `invalid_grant` で拒否し、拒否の理由と信頼設定を載せた `WorkloadAttestationRejected` を発行し、トークンを発行しない。
- JWKS を取得できず、使える鍵もない場合、WorkloadIdentity は、理由 `jwks_unavailable` で拒否する。
- `aud` が受理する audience に含まれない場合、WorkloadIdentity は、理由 `audience_mismatch` で拒否する。
- `exp` と `iat` の差が信頼設定の最大 TTL を超える場合、WorkloadIdentity は、理由 `ttl_exceeded` で拒否する。
- 一致した信頼設定が `disabled` の場合、WorkloadIdentity は、理由 `trust_bundle_disabled` で拒否する。
- 主体がどの `enabled` の関連付けのパターンにも一致しない場合、WorkloadIdentity は、理由 `no_binding_match` で拒否する。
- 対応先の Agent が `OAuth2Client` に束縛されていない場合、WorkloadIdentity は、理由 `agent_unbound` で拒否する。
- **例**：EX-WORKLOADIDENTITY-001-01

#### REQ-WORKLOADIDENTITY-006 対応先 Agent が Killed になった後は交換を拒否する

- 対応先の Agent が存在しないか `Active` でない場合、WorkloadIdentity は、理由 `agent_not_active` で交換を拒否する。
- **例**：EX-WORKLOADIDENTITY-006-01

#### REQ-WORKLOADIDENTITY-002 未登録の発行者を拒否する

- JWT として読めないか `iss` のないトークン、テナントのどの信頼設定にも一致しない `iss` のトークン、署名の検証で発行者が一致しないトークンを受けた場合、WorkloadIdentity は、理由 `unregistered_issuer` で交換を拒否する。
- **例**：EX-WORKLOADIDENTITY-002-01

#### REQ-WORKLOADIDENTITY-003 署名が不正なアテステーションを拒否する

- 登録した JWKS で署名を検証できない場合、WorkloadIdentity は、理由 `invalid_signature` で交換を拒否する。
- **例**：EX-WORKLOADIDENTITY-003-01

#### REQ-WORKLOADIDENTITY-004 期限切れのアテステーションを拒否する

- `exp` を過ぎたトークンを受けた場合、WorkloadIdentity は、理由 `expired` で交換を拒否する。
- **例**：EX-WORKLOADIDENTITY-004-01

#### REQ-WORKLOADIDENTITY-005 複数の関連付けに一致して Agent を一意に決められない主体を拒否する

- 主体が二つ以上の `enabled` の関連付けのパターンに一致する場合、WorkloadIdentity は、優先順位で選ばずに、理由 `ambiguous_match` で交換を拒否する。
- **例**：EX-WORKLOADIDENTITY-005-01

#### REQ-WORKLOADIDENTITY-007 他テナントの信頼設定は利用できない

- 交換を要求されたとき、WorkloadIdentity は、要求を受けたテナントの信頼設定と関連付けだけを探し、ほかのテナントの同じ発行者の信頼設定を参照しない。
- **例**：EX-WORKLOADIDENTITY-007-01

## セキュリティ上の考慮

交換の経路は、管理者の権限を通らない。
外部のワークロードが提示するのはアテステーションのトークンだけであり、得られる権限は、登録した信頼設定と関連付けが定めた `Agent` のものに固定される。
トークンの内容が対応先の `Agent` を変えることはなく、未登録の発行者を実行時に信頼することもない（Trust On First Use を許可しない）。

信頼設定は、要求を受けたテナントの中だけで探す。
ほかのテナントに同じ発行者の信頼設定があっても、参照しない。
