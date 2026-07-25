---
status: completed
authors: [tn]
risk: medium
created_at: 2026-07-19
depends_on: []
---

# inbound 統合 (identity import) の taxonomy と context 構造を確定する

## Motivation

ADR-128 で outbound provisioning を独立 `Provisioning` context として切り出したが、inbound
(外部 → idmagic の identity import) 側には統一された境界規範が無く、実装が場当たりに散っている:

- **受動 server 型** (外部が我々の API を叩く): SCIM server = `Scim` context
  ([[wi-31-scim2-provisioning]] とその拡張 wi-246〜251)。
- **upload / batch 型** (管理者がファイルを渡す): CSV user import =
  `backend/idmanagement/usecases/user_import.go` (IdManagement context 内。ADR-128 §影響 が
  「適所でない」と指摘)。
- **能動 pull / connector 型** (我々が外部 API を叩いて取り込む): LDAP/AD federation
  ([[wi-95-ldap-ad-user-federation]])、inbound federation / identity broker
  ([[wi-30-inbound-federation-and-identity-broker]])、orphan reconciliation
  ([[wi-156-orphan-account-discovery-and-reconciliation]]) が該当。

この 3 種は起動契機 (外部 HTTP / ファイル投入 / スケジュール・イベント)、権威方向、runtime 形状
(request handler / batch job / connector + 取り込み) が異なり、"inbound" 一語では束ねられない。
outbound を先に整理した今、inbound の**目標 context 構造を確定**し、後続の物理再配置
([[wi-259-rename-scim-inbound-server-context]]、[[wi-260-relocate-csv-user-import-to-inbound]]) と
将来 feature ([[wi-95-ldap-ad-user-federation]]、[[wi-30-inbound-federation-and-identity-broker]])
の受け皿を定める。

## Scope
- **decision (ADR)**: inbound taxonomy を定める新規 ADR。3 runtime shape の境界、target context
  (数・名前・責務)、CSV import・SCIM server・将来 connector の帰属、active-pull machinery の
  受け皿を記録する。ADR-128 の原則「protocol/source 非依存コア + protocol/source 別 feature」
  「構造は作り替えが高コストゆえ前もって決め、共有コードは早期結合が有害・後追い抽出が安価ゆえ
  on-demand で切り出す」を踏襲する。
- **scl (context_map)**: 確定した inbound context を `spec/scl.yaml` の context_map に scaffold
  する (エントリと depends_on の骨子)。model / interface の物理移設は後続 WI。
- **architecture**: `ARCHITECTURE.md` に target 構造 (context・module・依存方向) を反映する。

## Out of Scope
- 物理コード移動 ([[wi-259-rename-scim-inbound-server-context]] の Scim rename、
  [[wi-260-relocate-csv-user-import-to-inbound]] の CSV import 移設)。
- active-pull connector の実装 ([[wi-95-ldap-ad-user-federation]]、
  [[wi-30-inbound-federation-and-identity-broker]])。
- outbound provisioning ([[wi-45-outbound-scim-provisioning]]、ADR-128)。
- inbound SCIM server の振る舞い拡張 (wi-246〜251)。

## Plan
- 候補構造を評価して 1 つ選ぶ:
  - **(A)** 単一 `Inbound` / `Sourcing` context + source 別 feature slice
    (`inbound/scim-server`, `inbound/upload`, `inbound/connector-*`)。outbound (=`Provisioning`) と対称で
    拡張が自明。ADR-128 決定 2 の feature-slice variant を inbound にも適用する。
  - **(B)** 受動 server SCIM は protocol context のまま (rename のみ)、能動 pull connector は別 context、
    upload は小さな import feature。runtime 形状ごとに context を分ける。
  - **(C)** ADR-128 と同型: `Sourcing` context に source 非依存コア + source 別 feature slice。
- 命名: outbound を `Provisioning` にした対称として inbound を何と呼ぶか
  (`Sourcing` / `Import` / `Inbound`)。ADR-128 は symmetric 命名を outbound 側では見送ったので、
  inbound taxonomy 確定の本 WI で symmetric 化の是非を決める。
- ADR-128 の申し送り「client として外部 API を能動駆動する machinery (connection 登録・credential・
  スケジューリング・retry・remote 相関) は outbound push と active-pull inbound で共通化し得る」を
  検討し、共有可否を判断する (今は on-demand 抽出の方針を継承)。

## Tasks
- [x] T001 [Decision/ADR] taxonomy と target context 構造を確定し ADR に記録する (`new-adr`) →
      [[ADR-141-inbound-identity-sourcing-taxonomy]]。ADR-128 の申し送り部分を partial supersede
      (双方向リンク + ADR-128 本文に一文注記)。
- [x] T002 [SCL] 確定 context を context_map に scaffold し `just check-scl` を通す →
      `spec/scl.yaml` に `Sourcing` エントリ、`spec/contexts/sourcing.yaml` に taxonomy の glossary。
- [x] T003 [Architecture] `ARCHITECTURE.md` を target 構造へ同期する (`new-architecture`) →
      contexts 台帳・Context Map・Structural Decisions。module 台帳は実体が入る wi-259 で追加。
- [x] T004 [Verify] `just check` (scl / work-items / ids / architecture / traceability) を緑にする。

## Verification
- `just check` (= `check-scl` / `check-work-items` / `check-ids` / `check-architecture` /
  `check-traceability`) — context_map・ADR ref・Architecture 整合。WI 起草時の `just yaml-check` は
  現行 justfile に存在しないレシピ名だったため読み替えた。
- ADR レビュー: 3 shape の帰属先と、将来 feature (wi-95 / wi-30) の受け皿が明記されていること。

## Risk Notes
純設計 WI で振る舞いは変えないが、ここで決める境界が後続の物理再配置 (259 / 260) と将来 feature
(wi-95 / wi-30) を規定するため、決定の質が下流コストを左右する。過大なら runtime shape 単位に
分割する。

## Completion

- **Completed At**: 2026-07-25
- **Summary**: inbound の分類軸を「方向 / runtime 形状」から「上流の外部権威 + durable な source
  binding の有無」へ引き直し、条件を満たすものだけを単一 bounded context `Sourcing` (source 別 feature
  slice、thin root) に束ねる目標構造を [[ADR-141-inbound-identity-sourcing-taxonomy]] に確定した。
  Plan の候補では (A)+(C) の統一 context 案を採用し、命名は `Provisioning` と対称の capability 名
  `Sourcing` とした。調査の結果、当初 inbound 3 shape とされていたもののうち 2 つは別族と判定し帰属を
  明記した: 管理者 CSV import は IdManagement に残す (権威は idmagic 自身、source binding 無し、
  ADR-140 の CSV export と対称・wi-284 が対を拡張)、login-time federation は Authentication
  (wi-30 Plan / ADR-064)、downstream target の台帳照合は Application / Provisioning 側 (wi-156)。
  ADR-128 §コンテキスト (2)(b) / §影響 の「CSV import は適所でない」という申し送りはこれに伴い覆し、
  partial supersede として双方向リンクと一文注記を張った。SCL は context_map に `Sourcing` を
  scaffold し、`spec/contexts/sourcing.yaml` に taxonomy の語彙 (IdentitySource / SourceCorrelation /
  Ingestion / IngestionRun / SourceCursor / SourceDrift) と混同封じを置いた。models / interfaces の
  実体は wi-259 / wi-95 が入れる。
- **Verification Results**:
  - `just check` - passed (scl / work-items / ids / architecture / traceability)
  - `just scl-render` - passed (`spec/idmagic.html` のみ更新。models / interfaces 未追加のため
    JSON Schema / OpenAPI は不変)
- **Human Decisions**: 境界構造 (統一 context + source slice)、命名 (`Sourcing`)、管理者 CSV import の
  帰属 (IdManagement に残す) の 3 点はユーザーが選択した。
- **Affected Guarantees State**: 振る舞い・契約・データは不変 (設計のみ)。Go 実装は未配置で、
  `Scim` context と `backend/scim/` は本 WI では変更していない。
- **Residual Risk**: ADR-141 は `status: suggested`。決定 5 により
  [[wi-260-relocate-csv-user-import-to-inbound]] は前提が消滅して不要になるが、ADR 受理待ちのため
  本 WI では wi-260 のレコードを変更していない。受理時に wi-260 の廃止 (または「将来の scheduled
  file feed に限定」への書き換え) と、wi-259 の対象を `backend/sourcing/scim` slice 移設へ確定する
  更新が必要。
