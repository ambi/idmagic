---
status: completed
authors: ["tn"]
risk: high
created_at: 2026-06-20
---

# SCIM 2.0 provisioning を user / group lifecycle の外部契約として実装する

## Motivation
Okta / Google IAM / Entra ID と連携する production IdP では、SCIM 2.0 による
user / group provisioning、deprovisioning、group sync が重要になる。
手作業の admin CRUD だけでは、入退社・異動・グループ更新を安全に同期できない。

## Scope
- **decision**:
  - 新規 ADR: idmagic が SCIM server として振る舞う範囲を定義する。 User / Group schema mapping、active=false の扱い、hard delete を受けた場合の soft-delete/anonymization との関係、Bearer token 認証方式を決める。
- **scl**:
  - ScimUser / ScimGroup / ScimServiceProviderConfig / ScimResourceType / ScimSchema を追加する。
  - CreateScimUser / PatchScimUser / DeleteScimUser / CreateScimGroup / PatchScimGroup / DeleteScimGroup を追加する。
  - provisioning event と permission を追加する。
- **go**:
  - `/scim/v2/Users`, `/scim/v2/Groups`, `/scim/v2/ServiceProviderConfig`, `/scim/v2/ResourceTypes`, `/scim/v2/Schemas` を realm 配下に公開する。
  - SCIM bearer token / provisioning client を tenant-scoped client として管理する。
  - PATCH Operations を仕様準拠で処理し、user/group aggregate に変換する。
  - `active=false` は disable/deprovision として扱い、hard delete は policy に従う。
  - group membership は `wi-9` の Group aggregate と同期する。
- **ui**:
  - admin settings に SCIM endpoint、token 発行/rotation、last sync、error history を表示する。
  - group/user detail に SCIM source を表示し、source-of-truth 属性は直接編集不可にする。
- **documentation**:
  - README に Okta / Google / Entra ID からの SCIM 設定例を書く。

## Out of Scope
- idmagic が SCIM client として外部アプリへ push provisioning すること。
- password sync。
- HRIS connector。
- custom enterprise schema の完全汎用 mapping。初期は core + enterprise user 最小 subset。

## Verification
- `go test ./...` (in: idmagic)
- `golangci-lint run ./...` (in: idmagic)
- `go build ./...` (in: idmagic)
- `bun --cwd idmagic/ui typecheck`
- `bun --cwd idmagic/ui lint`
- `bun --cwd idmagic/ui build`
- 手動: SCIM Create User → Patch active=false → Reactivate → Delete の流れが internal user lifecycle と一致することを確認する。
- 手動: SCIM Group member 更新で effective roles が更新されることを確認する。

## Risk Notes
SCIM は外部 system of record との契約なので、内部 admin 操作よりデータ破壊の
影響が大きい。`active=false` と delete の意味を ADR で先に固定し、hard delete
は既存 ADR-036 と矛盾させない。

## Completion
- **Completed At**: 2026-07-04
- **Summary**:
  SCIM 2.0 inbound provisioning の ADR と SCL を追加し、User / Group lifecycle と同期する
  SCIM API、永続化、管理 UI の設定導線、同期対象 principal の編集抑止を実装した。
- **Verification Results**:
  - `scim_test.go` - passed
  - Go backend lint / format / tests - passed
  - UI lint / format / typecheck / build - passed

- **ADR & SCL**:
  - `ADR-080-scim2-inbound-provisioning.md` を作成し、SCIM Inbound Provisioning の仕様とデータモデル、セキュリティ方針を定義。
  - `spec/contexts/scim.yaml` に SCIM 仕様を SCL-first で定義し、`scl.yaml` に統合。
- **Backend & Database**:
  - `deploy/schema/postgres.sql` に SCIM 関連テーブルスキーマを追加。
  - `internal/scim` に SCIM Inbound Provisioning 用の Echo ハンドラ、ポート、ユースケースおよびリポジトリ層を実装。
  - メモリリポジトリを使用した SCIM User/Group ライフサイクル・メンバーシップ同期の結合テスト `scim_test.go` がすべて PASS することを確認。
  - Go バックエンドのリンター (`golangci-lint` / `gofumpt`) とテストが完全にパスする状態を達成。
- **Frontend UI**:
  - `AdminSettingsPage.tsx` に SCIM 同期設定タブを実装し、Bearer トークンの発行・失効ライフサイクルを構築。
  - `AdminUsersPage.tsx`, `AdminGroupsPage.tsx` に SCIM 同期対象 principal に対する直接編集の抑止 (readonly / disabled 制御) および警告表示を統合。
  - フロントエンドの Biome Lint/Format および TypeScript Typecheck, Vite Build が正常に完了することを確認。
