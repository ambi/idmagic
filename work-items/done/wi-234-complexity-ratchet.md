---
status: completed
archived_from: d3f23b968ea18e874c4e9a4a2b8db53ad0cc0036
---

# UI container と Go source の複雑性を分割し再増加を ratchet で防ぐ

Applications(2,677行/state81)、Users(2,457行/state44)、Groups(984行/state30)、Agents(918行/state26)、Settings(749行/state22)の5管理画面モノリスを route/resource operation 単位(list/detail/edit/create/import + 共有 presentation/dialog)へ分割し、Go 側は唯一 budget 超過だった `authorize_handler.go`(1,028行)を既存の concern 別ファイル規約に沿って8ファイルへ分割した。wi-232 が導入済みの complexity budget/debt(`ui-page-lines` 400行、`ui-page-local-state` 10個、`go-source-lines` 800行)を全対象ファイルで新パス・実測値に追従させ、budget 内に収まったファイルの debt entry は削除、収まらなかった edit page 系のみ ceiling を実測値へ更新して残した。抽出後に単体 test の無かった presentation/helper(`AdminAttributeField`、`AssignmentManager`、`CategoryManager`、Go throttle helper 群ほか)へ test を追加した。挙動変更は無く、全て behavior-preserving なファイル間移動と re-export の再配線。

全文は `git show d3f23b968ea18e874c4e9a4a2b8db53ad0cc0036:<このファイルのパス>` で読む。
