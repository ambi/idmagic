---
status: accepted
authors: [tn]
created_at: 2026-07-18
---

# ADR-118: 環境別 seed を Seeding bounded context へ切り出す

## コンテキスト

従来の demo seed は `backend/cmd/internal/bootstrap` にあり、IdManagement、Authentication、OAuth2、Application、SAML、WS-Federation のデータを起動時に直接書き込む。これは DI の都合であって業務語彙ではなく、profile、環境許可、dry-run、semantic diff、drift、再実行、排他という独自の一貫した判断を表せない。また、first-party client の bootstrap と既知パスワードを含むデモを一緒に skip しており、production 相当の安全境界がない。

現行 manifest の棚卸しは次のとおりである。`bootstrap` は管理コンソール／account portal の first-party OAuth2 client 2 件だけを持つ。`development` と `test` は、demo confidential client、alice/root user、password history、任意の TOTP factor、engineering/support group と membership、authorization detail type、WS-Fed RP、SAML SP、4 件の Application と alice assignment を追加する。`performance` は既存 demo resource を再利用せず、profile と index から決まる synthetic User だけを作る。すべての ID は固定 UUID または logical key で決まり、秘密値は request の出力に含めない。

## 決定

`Seeding` を新しい operations bounded context とする。Seeding は `SeedProfile`、`SeedRequest`、`SeedPlan`、環境 policy、drift policy、適用順序を所有する。対象 resource の意味、validation、永続化、公開 command surface は IdManagement、Authentication、OAuth2、Application、Saml、WsFederation 等の record context に残す。profile は環境名から推測せず明示し、production は `bootstrap` のみを fail-closed で許可する。dry-run/apply の同一 planner、drift-as-conflict、bounded batch 適用、mutex/advisory lock による直列化などの policy の詳細は
[backend/seeding/ARCHITECTURE.md](../backend/seeding/ARCHITECTURE.md) に移した。

production の bootstrap は first-party client を必要とする場合、redirect URI を `SEED_FIRST_PARTY_REDIRECT_URIS` で明示する。空、`localhost`、非 HTTPS URI は拒否する。開発用 localhost URI を production manifest に持ち込まず、issuer は client metadata ではなく通常の runtime `ISSUER` 設定が所有する。

初期実装は `backend/seeding/{domain,usecases}` の policy/planner に限る。既存の起動時 seed を削除せず、CLI、contributor、checkpoint、既存 seed の移設を段階的に進める。

## 却下した代替案

- `backend/cmd/internal/bootstrap` に seed を残す: 起動用 DI と運用 policy が混在し、独立 CLI とテスト可能な plan/apply を育てられない。
- 各 record context に profile を分散する: environment policy と全体の適用順が重複し、cross-context の安全保証を一箇所で検証できない。
- DB fixture library で直接投入する: domain invariant、memory/PostgreSQL contract、dry-run、drift policy を迂回する。

## 影響

- `spec/scl.yaml` と `spec/contexts/seeding.yaml`: `Seeding` context と `SeedData` 契約を追加する。
- `backend/seeding`: 安全な request validation と planner から開始し、後続で context contributor を追加する。
- `ARCHITECTURE.md`: Seeding の context と RA module を現状の構成へ追加する。
- `work-items/wi-236-environment-aware-idempotent-seeding.md`: 実装順を context 抽出後の段階へ更新する。
