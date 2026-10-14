---
status: completed
archived_from: e917b95d7a0a25e5068d187677f7afecebdc71ff
---

# oauth2 コンテキストへバックエンド・コンテキストローカリティを横展開する（client / consent / 認可詳細タイプ）

oauth2 context 横展開の第一弾として client / consent / 認可詳細タイプを
[[ADR-089]]/[[ADR-090]]/[[ADR-091]] の型紙で移設した。T001 の実測で全体規模が
wi-172（95 ファイル変更）を上回ると判明し、Plan の分岐規定に従い client/consent/認可詳細
タイプ（本 WI）・token/grant（[[wi-181]]）・audit/outbox（[[wi-182]]）に 3 分割した。
`OAuth2Client`/`Consent`/`AuthorizationDetailFieldRule`/`AuthorizationDetailsSchema`/
`AuthorizationDetailType` とその zog 検証・イベント・専用 enum を `internal/oauth2/domain/`
へ移設。`ClientType`/`GrantType`/`ResponseType`/`AuthorizationDetail`（実行時インスタンス）
は shared 側の SCL permissions エンジンおよび [[wi-181]] 側の型が直接参照するため shared
残置と判断（T003、循環回避）。この判断の過程で [[ADR-089]] 決定 3 の「validation.go」解釈の
齟齬を発見し [[ADR-093]] を新設して是正した（zog スキーマは型と共に移設、汎用ラッパーのみ
shared 残置）。postgres/memory の repository 実装を `internal/oauth2/adapters/persistence/`
へ同居し、postgres 実装は新設と同時に sqlc 生成へ置換（3 リポジトリ 9 クエリすべて静的、
エスケープハッチ 0 件、[[ADR-090]] に追記）。`internal/oauth2/module.go` を新設し中央
`server/routes.go` の `Deps` と `bootstrap/deps.go` の `Dependencies` から
`ClientRepo`/`ConsentRepo`/`AuthzDetailTypeRepo` の 3 field を `OAuth2 oauth2.Module` へ
集約。副産物として `shared/adapters/persistence/memory` の `defaultTenant` を
`DefaultTenant` として export（[[wi-172]] の RowScanner/TenantKey と同じ再利用パターン）。

全文は `git show e917b95d7a0a25e5068d187677f7afecebdc71ff:<このファイルのパス>` で読む。
