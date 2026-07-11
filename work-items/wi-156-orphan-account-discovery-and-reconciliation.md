---
depends_on: []
status: pending
authors: ["tn"]
risk: high
created_at: 2026-07-10
---

# 外部アプリの orphan account 検出と照合を導入する

## Motivation
Outbound SCIM provisioning を導入しても、外部アプリ側には IdMagic 管理外で作成された
local account、退職者の残存アカウント、属性不一致アカウントが残り得る。Microsoft Entra
ID Governance の orphan account discovery のように、外部アプリのアカウント台帳を取り込み、
IdMagic の User / Group / Application assignment と照合できないと、未管理アカウントを
発見・是正できない。

本 WI は、外部アプリの account inventory を IdMagic に取り込み、内部 User との match /
unmatched / ambiguous / disabled mismatch を管理者が確認・解決できる reconciliation
機能を導入する。

## Scope
- **scl**:
  - `Application` に ExternalAccountInventory / ExternalAccount / ReconciliationMatch / ReconciliationStatus を追加する。
  - `IdentityManagement` の UserRef と external account の照合 rule を定義する。
  - `Scim` / outbound provisioning と連携する inventory import events / scenarios を追加する。
- **go**:
  - external account import port、matching engine、reconciliation repository、memory / postgres adapter を実装する。
  - CSV または SCIM-like JSON inventory からの初期取り込みを提供する。
- **http**:
  - inventory import、match preview、reconciliation decision、orphan account 一覧 API を追加する。
- **ui**:
  - Application detail に external accounts、orphan accounts、ambiguous matches、解決 action UI を追加する。
- **documentation**:
  - README に inventory 形式、match rule、解決 workflow を追記する。

## Out of Scope
- Outbound SCIM provisioning 本体。これは `wi-45-outbound-scim-provisioning` が扱う。
- 全 SaaS connector の実装。
- 外部アプリの危険な自動削除。
- HR source of truth との同期。

## Plan
- 初期 inventory 取り込みは CSV / JSON upload または admin API に限定し、connector は port 境界だけ用意する。
- matching は email / username / immutable external id の優先順位を構造化し、ambiguous match は自動解決しない。
- orphan account は report と remediation suggestion までに留め、外部削除は outbound provisioning 側の明示 action に委ねる。
- reconciliation decision は監査イベントに残し、後から match rule を変えて再評価できるよう raw inventory snapshot を保持する。

## Tasks
- [ ] T001 [SCL] ExternalAccountInventory、match 状態、events / scenarios を追加する。
- [ ] T002 [Decision] inventory 形式、matching 優先順位、危険 action の境界を ADR に記録する。
- [ ] T003 [App] import / matching / reconciliation store を実装する。
- [ ] T004 [HTTP] inventory import と reconciliation API を追加する。
- [ ] T005 [UI] orphan account discovery / reconciliation UI を追加する。
- [ ] T006 [Verify] SCL、Go、UI、手動シナリオを検証する。

## Verification
- `just yaml-check`
- `just check-ids`
- `just test-go`
- `just verify-ui`
- 手動: 外部 inventory に IdMagic に存在しない account を含めると orphan として表示されることを確認する。
- 手動: 同一 email の候補が複数ある場合は ambiguous として自動解決されないことを確認する。

## Risk Notes
外部アカウント照合は誤削除・誤紐付けのリスクが高い。初期実装では自動削除や自動 disable を行わず、match preview と管理者 decision に限定する。ambiguous match は必ず人手確認とし、inventory snapshot と decision を監査可能に残す。
