---
status: accepted
authors: [tn]
created_at: 2026-08-10
supersedes: [ADR-101]
---

# ADR-161: ユーザー CSV を可逆な部分 upsert と成功済み preview 結合で再適用する

## コンテキスト

固定 4 列の create-only import は初期投入には十分だったが、より多い列を持つ export と非対称で、
無編集ファイルさえ既存 username の競合になる。さらに preview と apply がそれぞれ CSV を受け取るため、
管理者が確認した内容と実際に適用する payload の同一性を保証できない。日常的な一括管理には、既存 User の
部分更新、読み取り専用列を含む export の再入力、tenant custom schema、上流管理権威を壊さない拒否境界が必要になった。
旧 import の 1,000 行・1 MiB 上限をそのまま往復保証へ使うと、10,000 User 規模の tenant では成功した
export を分割せず再適用できず、export/import 対称性という目的そのものを満たさない。

## 決定

ユーザー CSV を machine-key header と可逆な formula-safe codec を持つ partial upsert とする。User export と
import は一つの configurable transfer policy を共有し、成功した export は必ず同じファイルのまま preview
できる契約にする。既定 policy は 100,000 data rows・64 MiB/file・64 KiB/field とし、tenant 容量ではなく
単一非同期 artifact の resource safety boundary として扱う。

CSV payload は job JSON に埋め込まず、tenant-scoped immutable artifact store に streaming で保存する。job は
artifact reference、server-computed digest、size、summary だけを持つ。preview upload は副作用のない計画を作り、
apply は CSV を再送せず、同一 tenant の成功済み preview payload と digest に結合する。apply 時は stale plan を
使わず現在状態から再計画し、bounded chunk で進めながら各行の mutation を原子的にする。

現在の parser / planner / port / job composition は
[`backend/idmanagement/ARCHITECTURE.md`](../backend/idmanagement/ARCHITECTURE.md) を正本とする。

## 却下した代替案

- preview と apply の双方で CSV を upload する: 確認済み payload と適用 payload の差し替えを防げず、
  digest をクライアント申告にしても信頼境界を改善しない。
- preview で作った mutation plan をそのまま apply する: 対象 User が並行更新された場合に stale な差分を
  上書きするため、preview payload を現在状態へ再計画する方式より安全性が低い。
- 旧 1,000 行・1 MiB 上限を維持して複数ファイルへの手動分割を求める: 成功した export の再入力を保証できず、
  10,000 User 規模で管理者に不必要な分割・結果集約を強いる。
- job params/result 内の CSV string/base64 はそのまま上限だけを引き上げる: database row と worker heap を
  payload size に比例させ、queue、取得、監視の各経路へ不要な PII と memory pressure を広げる。
- 既存の単件 create/update use case を列ごとに逐次呼び出す: roles、required actions、custom attributes の
  途中で失敗すると 1 行が部分保存され、再試行結果を予測できない。
- Group と GroupMembership も同じ変更へ含める: 識別子、immutable membership type、dynamic/source-managed
  制約が User と異なり、User CSV から得る実装・レビュー知見を反映できないまま blast radius を広げる。

## 影響

- `spec/contexts/identity-management.yaml` の `glossary.UserImport`、`models.UserImportJob` /
  `UserImportRowError` / `UserImportMode` / `DataExportColumn`、`interfaces.ImportAdminUsers` /
  `ApplyAdminUserImport` / `StartUserCsvExport`、User CSV scenarios、`flows.AdminUsers` を変更する。
- immutable artifact store port と durable/local adapters を追加し、Jobs には payload reference、SHA-256、size、
  summary を保存する。apply job には preview reference と digest だけを保存する。
- User export/import は shared transfer policy と streaming reader/writer を使い、10,000-row round-trip を
  integration evidence として固定する。
- IdManagement は tenant schema と source ownership を use-case port 経由で参照し、source-managed User の
  update を fail closed で拒否する。
- password、password hash、MFA secret、token は引き続き import/export 対象外とし、row error、job view、
  audit event に CSV 値を含めない。
