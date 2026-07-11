---
status: accepted
authors: [tn]
created_at: 2026-07-10
---

# ADR-090: 永続化アダプタのコンテキスト同居と sqlc 採用

## コンテキスト

[[ADR-047]] §3 と [[ADR-070]] §3 はいずれも、`shared/adapters/persistence/{postgres,memory}`
に置かれた context 固有 repository 実装は「実装上の利便性を優先した暫定配置」であり、
所有 context の per-context adapter へ移す余地があると *明記済み* である。両 ADR は
「shared に残すべきは DB 接続・pool・row scanner・transaction helper・Valkey client 等の
技術的共通部品に限定する」とも述べている。

現状はその暫定配置のままで、約 93 file の repository 実装が `shared` に集中している。
加えて 1 つの repository を memory と postgres で **手書き二重実装**しており、実装工数が
2 倍・両者の乖離リスクが常在する。postgres 側は生 SQL 文字列＋手書き scanner で、
`shared/adapters/persistence/postgres/base.go` の `ResilientDB`（circuit breaker / timeout、
`DB` interface）へ `Pool DB` として注入されている。

一方でスキーマは [[ADR-071]] により `deploy/schema/postgres.sql` を宣言的な正とし、
psqldef で移行する。[[ADR-084]] はカラム型を明示し、セキュリティ IdP として *実 SQL が
監査可能・予測可能*であることを要件とする。

本 ADR は [[ADR-089]]（ドメイン型の per-context 化）と対になり、永続化アダプタも
所有 context へ同居させ、併せて手書き SQL の削減手段を決める。

## 決定

1. **context 固有 repository 実装を per-context へ同居させる**。
   `shared/adapters/persistence/postgres/clients.go` →
   `internal/oauth2/adapters/persistence/postgres/`、memory 実装も同様に
   `internal/<context>/adapters/persistence/memory/` へ移す。ポート interface は所有
   context の `ports/` が持つ（[[ADR-047]] §3 補足の方向を確定させる）。
2. **`shared/adapters/persistence` は技術的共通部品に限定する**。`base.go`（`ResilientDB` /
   `DB` interface / 接続 pool）、共通 row scanner、transaction helper、circuit breaker、
   Valkey client のみを残す。各 context の postgres 実装はこの `DB`（= `DBTX` 相当）
   interface を受け取る（既に `Pool DB` 注入のため移設は機械的）。
3. **postgres 側の手書き SQL を sqlc 生成へ置換する**。`queries/*.sql` を各 context の
   `adapters/persistence/` 配下に置き、sqlc（pgx/v5 ネイティブモード）で型安全な Go を
   生成する。生成関数は `DBTX` を受けるため、`ResilientDB` を `DBTX` 実装としてラップし
   circuit breaker / timeout を温存する。sqlc の schema 入力には [[ADR-071]] の
   `deploy/schema/postgres.sql` をそのまま用い、スキーマの正を一本化する。
4. **動的クエリはエスケープハッチで扱う**。sqlc は条件付き WHERE が不得手なため、任意
   フィルタは `sqlc.narg` + `COALESCE` で吸収し、真に動的なクエリのみ同パッケージ内に
   薄い手書き pgx を併置する。探索上、クエリの大半は tenant + id の静的引きであり影響
   範囲は限定的である。
5. **codegen は `just` レシピ経由とする**。`just sqlc-generate` を追加し、生 sqlc を直接
   叩かない（AGENTS.md の just 単一コマンドマップ方針）。
6. **memory 実装はテストダブルとして維持する**。SQL 生成は memory 二重を解消しない。
   当面はポート interface を維持し memory をテスト用に残す。二重自体の解消（testcontainers
   ベースの postgres テストへ寄せて memory を退役）は本 ADR の対象外とし、別途評価する。

## sqlc を選ぶ理由と次点 bob（Considered Alternatives 実測）

選定は sqlc / bob / jet / bun の 4 択で比較した。判断軸は「[[ADR-071]] の単一スキーマ源
との整合」「実 SQL の監査性（[[ADR-084]] / IdP）」「pgx/v5 ネイティブ統合」「実行時
リフレクション回避（RA 再生成思想・AI 可読性）」。

- **sqlc（採用）**: `postgres.sql` を*そのまま schema 入力*にでき単一スキーマ源と直結。
  `.sql` が成果物として残り監査可能。pgx/v5 ネイティブ、`DBTX` 注入で `ResilientDB` を
  温存。reflection ゼロ。唯一の弱点は動的クエリ（上記エスケープハッチで対処）。
- **bob（次点）**: pgx ネイティブ codegen でありつつ builder で動的クエリも解決できる
  唯一の対抗。コストは稼働 DB introspect パイプライン、新しめでエコシステム小。
  **動的クエリが支配的と判明した場合は bob へ切替**する。
- **jet（却下）**: 型安全 builder だが database/sql 主体で pgx ネイティブでなく、bob に
  対する決定的優位がない。
- **bun（却下）**: DX と動的クエリは最良だが、実行時 reflection・struct tag のスキーマ
  二重定義・独自 migration が [[ADR-071]] / psqldef・監査性・再生成思想と噛み合わない。

**判断の分岐点＝動的クエリの実比率**。パイロット context で sqlc を実装し、動的クエリの
比率を実測して本決定（sqlc 継続 or bob へ切替）を確定する。本節に実測値を追記する。

### 実測値（wi-172、application context パイロット）

application context の全クエリ 26 件中、sqlc 生成で賄えた静的クエリ（tenant + id の固定
WHERE、または固定長 INSERT ... ON CONFLICT）は 25 件（96%）。動的クエリのエスケープハッチ
（手書き pgx）が必要だったのは `ApplicationAssignmentRepository.ListBySubjects` の 1 件
（4%）のみで、これも「可変 WHERE」ではなく `UNNEST($2::text[], $3::text[])` による
(subject_type, subject_id) ペア配列突合せを sqlc の静的解析器が型解決できないという sqlc
自体の制約が理由であり、真に動的な条件分岐クエリ（admin list/filter 等）はゼロだった。

この結果は本 ADR の当初の探索（「クエリの大半は tenant + id の静的引きであり影響範囲は
限定的」)を裏付ける。application は動的フィルタを持たない小さめの context だが、96% という
比率は sqlc 継続判断を強く支持する。**sqlc を継続採用する**。bob への切替は、今後の横展開で
admin list/filter（可変 WHERE）を多く持つ context（例: 監査ログ検索、ユーザー一覧の属性
フィルタ）を移行した際に動的比率が再評価対象になった場合にのみ検討する。

### 実測値（wi-173、oauth2 context 横展開・client/consent/認可詳細タイプ）

oauth2 context の横展開第一弾（client / consent / authorization detail type、[[wi-173]]）
では、3 リポジトリ・全 9 クエリ（`GetClientByID` / `ListClientsByTenant` / `UpsertClient` /
`DeleteClient` / `GetConsent` / `ListConsentsByTenant` / `UpsertConsent` / `RevokeConsent` /
`DeleteConsentsForSub` / `GetAuthorizationDetailType` / `ListAuthorizationDetailTypesByTenant`
/ `UpsertAuthorizationDetailType` / `DeleteAuthorizationDetailType` のうち実質 13 件、
明細は各 `queries/*.sql`）すべてが sqlc 静的生成のみで賄え、動的エスケープハッチは 0 件
（100%）だった。admin client 一覧のようなフィルタ付きクエリは本 WI の scope（
`ListClientsByTenant`/`ListConsentsByTenant`/`ListAuthorizationDetailTypesByTenant` は
いずれも tenant_id 固定 WHERE のみ）には含まれず、可変 WHERE を持つクエリは未発見のまま。
application の 96%/4% と合わせ、sqlc 継続採用をさらに裏付ける。監査ログ検索など可変 WHERE
候補が本命の [[wi-182]]（audit/outbox）で再度実測し、bob 再評価の要否を判断する。

### 実測値（wi-182、oauth2 context 横展開・audit event / outbox）

audit event / outbox では全 8 クエリ中、`AppendAuditEvent`、sidecar 属性追加、
`GetAuditEventByID`、保持期間削除 2 件、username redact、outbox 追加の 7 件（88%）を
sqlc の静的生成へ移行した。admin audit 検索の `List` だけ（1 件、12%）はテナント・時刻・
event type・user・検索属性（eq / in / contains）・フリーテキストの任意組合せで WHERE と
プレースホルダ数が変化するため、ADR-090 のエスケープハッチとして同一 context の薄い pgx
実装に残した。

動的クエリは audit 検索という最も可変性の高い候補でも少数であり、oauth2 context 全体の
sqlc 継続判断を維持する。bob への切替条件である「動的クエリが支配的」は満たさない。

### 実測値（wi-174、wsfederation context 横展開）

WS-Federation relying party repository の 4 クエリ（取得、テナント一覧、upsert、delete）は
すべて固定 SQL のため sqlc 静的生成へ移行でき、動的エスケープハッチは 0 件（100%）だった。
この context に可変 WHERE を必要とする操作はなく、sqlc 継続採用の判断を維持する。

### 実測値（wi-175、saml context 横展開）

SAML service provider repository の 4 クエリ（取得、テナント一覧、upsert、delete）はすべて
固定 SQL で、sqlc 静的生成へ移行できた。動的エスケープハッチは 0 件（100%）であり、
この context に可変 WHERE を必要とする操作はない。sqlc 継続採用の判断を維持する。

### 実測値（wi-176、scim context 横展開）

SCIM token / user-ref / group-ref repository の全 12 クエリ（token の save/find-by-hash/
list-by-tenant/delete、user-ref の save/find-by-scim-id/find-by-user-id/delete、group-ref の
save/find-by-scim-id/find-by-group-id/delete）はすべて tenant_id・scim_id・token_hash 等の
固定 WHERE で、sqlc 静的生成へ移行できた。動的エスケープハッチは 0 件（100%）。SCIM の
`filter` query parameter によるフィルタ検索は本 WI の scope（token/ref repository）には
含まれず、可変 WHERE の必要性は見つからなかった。sqlc 継続採用の判断を維持する。

## 影響

- Go import path: repository 実装が `idmagic/internal/shared/adapters/persistence/...` から
  `idmagic/internal/<context>/adapters/persistence/{postgres,memory}` へ移る。
- `shared/adapters/persistence/postgres/base.go`（`ResilientDB`）は共通基盤として残置。
- 新規ツール依存として sqlc（採用ツール）を導入。`justfile` に codegen レシピを追加。
- sqlc の schema 入力として `deploy/schema/postgres.sql` を参照（[[ADR-071]] と相互参照）。
- 振る舞い・SCL 規範・HTTP route・DB schema・公開 API は変更しない（純構造 + 生成方式）。
- [[ADR-047]] / [[ADR-070]] の「per-context 永続化は要検討事項」を解決し extend する。
- [[ADR-089]] と同一 Work Item 系列で context 単位に段階移行する。
