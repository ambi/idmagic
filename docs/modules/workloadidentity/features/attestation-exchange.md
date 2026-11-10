# アテステーションの交換

## 概要

この文書は、外部のワークロードが提示したアテステーションを検証し、対応先の `Agent` を一意に特定する仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | 登録した信頼設定による署名、`iss`、`aud`、`exp`、TTL の上限の検証、`spiffe` の信頼設定での SPIFFE の規則の検証、関連付けによる `Agent` の一意な特定、`Agent` の状態の確認 |
| 行為者 | System（`OAuth2` の Token Exchange とクライアント認証から呼ばれる検証のユースケース） |
| 扱わないもの | アクセストークンの発行とクライアントの認証方式の確認は `OAuth2` が扱う。信頼設定と関連付けの登録は[信頼設定と関連付けの管理](trust-configuration.md)が扱う |

## モデル

検証は、いずれかの段で失敗すればフェイルクローズで拒否し、拒否の理由と用途とともに `WorkloadAttestationRejected` を発行する。
用途は、Token Exchange の `subject_token` の検証なら `token_exchange`、クライアントアサーションの検証なら `client_authentication` である。

| 拒否の理由 | 条件 |
| --- | --- |
| `unregistered_issuer` | トークンの `iss` に一致する信頼設定も、`sub` のトラストドメインに一致する `spiffe` の信頼設定もテナントにない。署名の検証で発行者が一致しない |
| `trust_bundle_disabled` | 一致した信頼設定が `disabled` である |
| `invalid_signature` | 登録した JWKS で署名を検証できない。`spiffe` の信頼設定で、JOSE ヘッダーが SPIFFE の規則を満たさない |
| `jwks_unavailable` | JWKS を取得できず、使える鍵もない |
| `audience_mismatch` | `aud` が受理する audience に含まれない |
| `expired` | `exp` を過ぎている |
| `ttl_exceeded` | `exp` と `iat` の差が信頼設定の最大 TTL を超える |
| `invalid_spiffe_id` | `spiffe` の信頼設定で、`sub` が SPIFFE ID の構文を満たさない |
| `trust_domain_mismatch` | `spiffe` の信頼設定で、`sub` のトラストドメインが信頼設定のトラストドメインと一致しない |
| `no_binding_match` | 主体がどの有効な関連付けのパターンにも一致しない |
| `ambiguous_match` | 主体が複数の有効な関連付けに一致する |
| `agent_not_active` | 対応先の `Agent` が存在しないか `Active` でない |
| `agent_unbound` | 対応先の `Agent` が `OAuth2Client` に束縛されていない |
| `client_not_bound` | クライアント認証で、要求のクライアントが対応先の `Agent` に束縛されていない |
| `replayed` | クライアント認証で、同じ `jti` の JWT-SVID を再び受けた（`OAuth2` が発行する） |

`spiffe` の信頼設定では、取得した JWKS に `use` が `x509-svid`、`jwt-svid`、`wit-svid` のいずれかの鍵があれば、その JWKS を SPIFFE バンドル（[SPIFFE Trust Domain and Bundle](https://github.com/spiffe/spiffe/blob/main/standards/SPIFFE_Trust_Domain_and_Bundle.md)）として読む。
`use` の値は大文字と小文字を区別する。

- **判断**：複数の一致を拒否する理由は、[複数の関連付けに一致した主体は優先順位で選ばず拒否する](../decisions.md#複数の関連付けに一致した主体は優先順位で選ばず拒否する)。

## 操作

### ワークロードによるトークンの交換

#### REQ-WORKLOADIDENTITY-001 登録済みの信頼設定を使ってワークロードトークンを Agent 資格情報に交換できる

- ワークロードが JWT のアテステーションを `subject_token` として Token Exchange を要求したとき、WorkloadIdentity は、`iss` の発行者か `sub` の SPIFFE ID のトラストドメインでテナントの信頼設定を探し（順序は[アーキテクチャ](#アーキテクチャ)）、登録した JWKS で署名を、受理する audience で `aud` を、`exp` と最大 TTL で有効期間を検証する。
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

- JWT として読めないトークン、`iss` に一致する信頼設定も `sub` のトラストドメインに一致する `spiffe` の信頼設定もテナントにないトークン、署名の検証で発行者が一致しないトークンを受けた場合、WorkloadIdentity は、理由 `unregistered_issuer` で交換を拒否する。
- **例**：EX-WORKLOADIDENTITY-002-01

#### REQ-WORKLOADIDENTITY-003 署名が不正なアテステーションを拒否する

- 登録した JWKS で署名を検証できない場合、WorkloadIdentity は、理由 `invalid_signature` で交換を拒否する。
- `spiffe` の信頼設定では、WorkloadIdentity は、JWKS の鍵のうち次の表の鍵だけで署名を検証する。

| JWKS の形式 | 署名の検証に使う鍵 |
| --- | --- |
| SPIFFE バンドル | `use` が `jwt-svid` の鍵 |
| SPIFFE バンドルでない JWKS | `use` が `sig` の鍵と、`use` のない鍵 |

- **判断**：[SPIFFE バンドルの鍵は取得した文書の形式で選び、Bundle Endpoint は `https_web` だけに対応する](../decisions.md#spiffe-バンドルの鍵は取得した文書の形式で選びbundle-endpoint-は-https_web-だけに対応する)。
- **例**：EX-WORKLOADIDENTITY-003-01、EX-WORKLOADIDENTITY-003-02

#### REQ-WORKLOADIDENTITY-004 期限切れのアテステーションを拒否する

- `exp` を過ぎたトークンを受けた場合、WorkloadIdentity は、理由 `expired` で交換を拒否する。
- **例**：EX-WORKLOADIDENTITY-004-01

#### REQ-WORKLOADIDENTITY-005 複数の関連付けに一致して Agent を一意に決められない主体を拒否する

- 主体が二つ以上の `enabled` の関連付けのパターンに一致する場合、WorkloadIdentity は、優先順位で選ばずに、理由 `ambiguous_match` で交換を拒否する。
- **例**：EX-WORKLOADIDENTITY-005-01

#### REQ-WORKLOADIDENTITY-011 `spiffe` の信頼設定による JWT-SVID の検証は、SPIFFE の規則を満たさないトークンを拒否する

- `spiffe` の信頼設定では、WorkloadIdentity は、`alg` が RS256、RS384、RS512、ES256、ES384、ES512、PS256、PS384、PS512 のいずれかの署名を検証する。
- `spiffe` の信頼設定では、発行者を登録していない間、WorkloadIdentity は、`iss` を検証しない。
- `oidc` の信頼設定では、WorkloadIdentity は、`sub` を SPIFFE ID として検証しない。
- `spiffe` の信頼設定では、JOSE ヘッダーに `kid` がないか、`typ` が `JWT` と `JOSE` 以外の値である場合、WorkloadIdentity は、理由 `invalid_signature` で交換を拒否する。
- `spiffe` の信頼設定では、`sub` が SPIFFE ID の構文を満たさない場合、WorkloadIdentity は、理由 `invalid_spiffe_id` で交換を拒否する。
- `spiffe` の信頼設定では、`sub` の SPIFFE ID のトラストドメインが信頼設定のトラストドメインと一致しない場合、WorkloadIdentity は、理由 `trust_domain_mismatch` で交換を拒否する。
- **判断**：[SPIFFE の規則は信頼設定の種別で選び、トークンの内容で選ばない](../decisions.md#spiffe-の規則は信頼設定の種別で選びトークンの内容で選ばない)。SPIFFE ID の構文は[データ](../data.md#spiffe-id)が定める。

#### REQ-WORKLOADIDENTITY-012 `jwks_uri` から取得した鍵の再利用は、取得から 5 分以内である

- 署名の検証または JWKS の再取得に `jwks_uri` の鍵を使うとき、WorkloadIdentity は、取得から 5 分を超えた鍵を再利用せず、`jwks_uri` から取得し直す。
- `spiffe` の信頼設定では、`jwks_uri` から SPIFFE バンドルを取得したとき、WorkloadIdentity は、`spiffe_refresh_hint` に応じた次の表の期間を超えて鍵を再利用しない。

| `spiffe_refresh_hint` | 再利用の期間 |
| --- | --- |
| 1 以上 30 未満の整数 | 30 秒 |
| 30 以上 300 以下の整数 | その秒数 |
| 300 を超える整数、1 以上の整数でない値、またはなし | 5 分 |

- WorkloadIdentity は、`jwks_uri` を Web PKI で検証した HTTPS で取得し、SPIFFE Bundle Endpoint の `https_spiffe` の方式で取得しない。
- **判断**：[SPIFFE バンドルの鍵は取得した文書の形式で選び、Bundle Endpoint は `https_web` だけに対応する](../decisions.md#spiffe-バンドルの鍵は取得した文書の形式で選びbundle-endpoint-は-https_web-だけに対応する)。

#### REQ-WORKLOADIDENTITY-007 他テナントの信頼設定は利用できない

- 交換を要求されたとき、WorkloadIdentity は、要求を受けたテナントの信頼設定と関連付けだけを探し、ほかのテナントの同じ発行者の信頼設定を参照しない。
- **例**：EX-WORKLOADIDENTITY-007-01

### クライアントアサーションの検証

#### REQ-WORKLOADIDENTITY-017 JWT-SVID のクライアントアサーションは、トラストドメインの信頼設定と発行者だけの `aud` で検証し、束縛したクライアントだけを認める

- OAuth2 が `spiffe_jwt` のクライアントの JWT-SVID の検証を求めたとき、WorkloadIdentity は、`sub` の SPIFFE ID のトラストドメインに一致する `spiffe` の信頼設定だけを探し、REQ-WORKLOADIDENTITY-003 と REQ-WORKLOADIDENTITY-011 の規則で署名と JOSE ヘッダーを、`exp` と最大 TTL で有効期間を検証する。
- 検証を通った SPIFFE ID がちょうど一つの `enabled` の関連付けに一致し、対応先の Agent が `Active` で、要求のクライアントがその Agent に束縛されているとき、WorkloadIdentity は、Agent、信頼設定、関連付け、SPIFFE ID を OAuth2 へ返し、OAuth2 が `jti` の再生を確かめた後に、それらとクライアントを載せた `WorkloadClientAuthenticated` を発行させる。
- `aud` がテナントの発行者識別子だけを値に持たない場合、WorkloadIdentity は、信頼設定の受理する audience によらず、理由 `audience_mismatch` で拒否する。
- `sub` が SPIFFE ID の構文を満たさない場合、WorkloadIdentity は、理由 `invalid_spiffe_id` で拒否する。
- `sub` のトラストドメインの `spiffe` の信頼設定がテナントにない場合、WorkloadIdentity は、`iss` に一致する信頼設定があっても、理由 `unregistered_issuer` で拒否する。
- 要求のクライアントが対応先の Agent に束縛されていない場合、WorkloadIdentity は、理由 `client_not_bound` で拒否する。
- クライアントアサーションの検証のどの段で拒否する場合、WorkloadIdentity は、[モデル](#モデル)の表の拒否の理由、信頼設定、用途 `client_authentication` を載せた `WorkloadAttestationRejected` を発行する。
- **判断**：[SPIFFECA03-JWT-SVID](../../oauth2/standards.md#oauth-spiffe-client-authentication) に従い、トークンの `iss` で鍵の源を選ばない。信頼設定の受理する audience は Token Exchange の `subject_token` のためのものであり、クライアントアサーションの `aud` はドラフトが発行者に限る。

## セキュリティ上の考慮

交換の経路は、管理者の権限を通らない。
外部のワークロードが提示するのはアテステーションのトークンだけであり、得られる権限は、登録した信頼設定と関連付けが定めた `Agent` のものに固定される。
トークンの内容が対応先の `Agent` を変えることはなく、未登録の発行者を実行時に信頼することもない（Trust On First Use を許可しない）。

信頼設定は、要求を受けたテナントの中だけで探す。
ほかのテナントに同じ発行者の信頼設定があっても、参照しない。

## 実現方式

### アーキテクチャ

検証は、署名を確かめる前にトークンの `iss` と `sub` だけを読み、それを手がかりにテナントの信頼設定を探す。
`iss` に一致する信頼設定を先に探し、なければ `sub` を SPIFFE ID として読み、そのトラストドメインの `spiffe` の信頼設定を探す。
この段で読んだ値は、信頼設定と JWKS を選ぶことにだけ使い、署名の検証の後でなければ、主体の特定に使わない。
`sub` で選んだ信頼設定でも、署名の検証の後に `sub` のトラストドメインの一致を確かめる。

対応先の `Agent` が複数の `OAuth2Client` に束縛されている場合は、最初の束縛の `client_id` を交換の結果に使う。

### 信頼性

JWKS は、検証のたびに信頼設定の `jwks_uri` またはインラインの `jwks` から解決する。
`jwks_uri` の取得結果は、REQ-WORKLOADIDENTITY-012 の期間だけ、各プロセスのメモリで URI ごとに再利用する。
取得に失敗した場合は、信頼設定ごとに最後に取得できた鍵の集合を、取得から 24 時間以内に限って使う。
24 時間を超えて取得に失敗し続けると、`jwks_unavailable` で交換を拒否する。
取得元の障害が続く場合は、取得元を直すか、管理者が信頼設定をインラインの `jwks` に切り替える。

最後に取得できた鍵の集合は、各プロセスのメモリにだけある。
プロセスを再起動した直後に取得元へ到達できなければ、24 時間の猶予なしに拒否する。
