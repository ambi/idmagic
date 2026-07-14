---
depends_on: [wi-49-agent-identity-first-class-principal, wi-50-token-exchange-delegation-actor-chain, wi-51-rich-authorization-requests-agent-scopes]
status: pending
authors: ["tn"]
risk: high
created_at: 2026-06-22
---

# エージェントのデータアクセス向け関係ベース細粒度認可 (ReBAC / FGA)

## Motivation
エージェント、特に RAG (検索拡張生成) パイプラインは大量の文書・レコードへ
横断アクセスするため、「代行しているユーザーが本来見られるものだけを取得する」
細粒度の認可が不可欠になる。粗い RBAC では「ユーザー U が文書 D を読めるか」を
リソース単位で判定できない。Google Zanzibar に始まる関係ベースアクセス制御
(ReBAC) と、その実装である OpenFGA (Auth0 / Okta の Fine-Grained Authorization)
が、エージェントの RAG データアクセスを per-resource で絞る標準的手法になっている。

idmagic は AuthZEN スタイルの PDP を持つ (ADR-010) が、判定はクライアント認可
ルール中心で、リソース×主体の関係タプルに基づく判定を持たない。本 WI は AuthZEN の
`{subject, action, resource, context}` インターフェースを拡張し、関係タプル
(user/agent ⇄ resource) に基づく ReBAC 判定を追加する。これにより
[[wi-50-token-exchange-delegation-actor-chain]] で代行する actor チェーンを考慮した
「ユーザーとして、かつエージェント経由で」のアクセス判定が成立する。

## Scope
- **decision**:
  - 新規 ADR [[ADR-052]]: authorization model (Zanzibar 風の type / relation / tuple)、ローカル 評価エンジンと外部 PDP (OpenFGA 等) の差し替え方針 (ADR-010 の adapter 路線を踏襲)、 actor チェーンを判定 context にどう載せるか、整合性 (consistency) の扱いを確定する。
- **scl**:
  - 新規 model: RelationTuple / RelationType / ResourceType / FgaCheckRequest / FgaCheckResult。AuthZEN 判定 context に actor チェーンを追加する。
  - 新規 event: RelationTupleWritten / RelationTupleDeleted / FgaCheckEvaluated。
  - 新規 interface: WriteRelationTuples / CheckAccess / ListAccessibleResources。 permission AdminAuthorizationModelManage。
- **go**:
  - ローカル ReBAC 評価エンジン (tuple ストア + グラフ探索) と Postgres adapter、 および外部 PDP への差し替え可能な adapter 境界 (ADR-010 の local-authzen-adapter 路線)。
  - 代行 (actor) を考慮した CheckAccess を fail-closed で実装する。
- **http**:
  - 関係タプル管理 API と CheckAccess / ListAccessibleResources エンドポイント。

## Out of Scope
- 外部 OpenFGA / Zanzibar サービスとの本番接続実装 (adapter 境界の提供まで)。
- アプリ側 RAG パイプラインそのものの実装。
- 大規模 tuple のシャーディング・キャッシュ最適化 (まず正しさ優先)。

## Plan
- [[ADR-052-relationship-based-fine-grained-authorization]] を [[ADR-010-authzen-policy-as-spec]] の既存 `PolicyDecision` contract に合わせて accepted にし、ReBAC を新しい HTTP middleware ではなく AuthZEN evaluator の relationship facts provider として組み込む。
- tuple は `(tenant, resource_type:id, relation, subject_type:id)`、model は relation rewrite/permission を versioned に保持する。tenant を tuple key と evaluator input の双方で必須にし、wildcard/subject-set の初期対応範囲を ADR で固定する。
- 既存 admin RBAC は管理面の coarse gate として残す。ReBAC は agent がデータ resource へアクセスする箇所で RBAC/RAR/actor chain と AND 合成し、判定不能・循環・store failure は fail-closed にする。
- 初期 backend は PostgreSQL closure/evaluation とし、OpenFGA等の外部 service は port adapter の将来候補に留める。model update と tuple write の整合 token を返し、read-your-writes が必要な管理操作で使う。
- decision audit は許可/拒否、model version、relation path の非PII要約を記録し、tuple 全量や機密 resource name を event に複製しない。

## Tasks
- [ ] T001 [ADR/SCL] ADR-052 の relation language、AuthZEN 合成順、consistency/fail mode を確定し、models/interfaces/events/constraints/contracts を再生成する。
- [ ] T002 [Domain] tuple、model version、subject set と bounded graph evaluator を実装し、循環・深さ・unknown relation をテストする。
- [ ] T003 [Postgres] tenant-scoped tuple/model schema、query/index、transactional write と consistency token を追加し、memory contract test と揃える。
- [ ] T004 [Policy] ReBAC facts provider を既存 AuthZEN evaluator に接続し、RBAC、RAR、actor chain、agent status との fail-closed 合成を実装する。
- [ ] T005 [Management] model/tuple の検証付き write/read API と専用 permission、監査 event を追加する。汎用 check endpoint は内部/診断用途に限定する。
- [ ] T006 [Verify] nested group/owner、delete propagation、concurrent update、cross-tenant tuple injection、store outage と decision trace を検証する。

## Verification
- `just test-go`
  - reason: tuple に基づく許可 / 拒否、グラフ探索 (継承・グループ)、actor 考慮、tenant 越境拒否の境界。
- `just lint-go`
- `just build-go`
- 手動: ユーザーに文書 A のみ許可するタプルを書く → エージェント代行で A は許可・B は拒否されることを確認する。

## Risk Notes
ReBAC は判定ロジックの中枢で、グラフ探索の誤りや既定許可は情報漏洩に直結する。
既定拒否 (fail-closed) を徹底し、actor チェーンを判定 context に明示的に載せる。
ADR-010 の adapter 境界を踏襲し、ローカル実装と外部 PDP を同一契約で検証する。
