---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-04
priority: p3
depends_on: []
change_kind: maintenance
affected_spec:
  - { path: spec/contexts/identity-management/models.tsp, symbol: IdMagic.Contract.UserImportUnavailableError }
  - { path: spec/contexts/identity-management/models.tsp, symbol: IdMagic.Contract.GroupImportUnavailableError }
  - { path: spec/contexts/identity-management/models.tsp, symbol: IdMagic.Contract.GroupMembershipImportUnavailableError }
---

# IdManagement が宣言しているが、どの経路からも起きないエラーコードとイベントを消す

## 動機

wi-83002 で IdManagement の機能仕様を書き直したとき、`check-unspecified-vocabulary` の許容リストに残った 6 件は、どの要件にも書けなかった。
どれも、宣言はあるが、製品のどの経路からも観測できない。

| 語 | 宣言 | 起きない理由 |
| --- | --- | --- |
| `invalid_target` | TypeSpec のエラー、エクスポートのハンドラーの対応 | エクスポートの対象はルートで決まり、要求から未知の対象を渡せない |
| `user_import_unavailable` | TypeSpec のエラー、503 | `JobRepo` と `CSVArtifacts` が空のときだけ返すが、どちらのブートストラップも必ず配線する |
| `group_import_unavailable` | 同上 | 同上 |
| `group_membership_import_unavailable` | 同上 | 同上 |
| `FederationLinked` | ドメインイベント、監査の検索の候補 | 発行するコードがない |
| `FederationUnlinked` | 同上 | 同上 |

宣言だけが残ると、利用者は起きない応答やイベントに備えることになり、許容リストも減らせない。

## 対象範囲

- 6 件の宣言を消すか、実際に起きる経路を持たせるかを語ごとに決める。
  起きる経路を持たせる場合は、その条件を要件に書く。
- 消す語は、TypeSpec、ハンドラーの分岐、イベントの型、監査の検索の候補から消す。
- `tools/check/unspecified-vocabulary-debt.json` から、決めた語を消す。

## 対象外

- 連携の結び付けと解除の機能そのものの設計。
  `FederationLinked` と `FederationUnlinked` を発行する経路を持たせると決めた場合は、別の work item を起票する。

## 計画

1. 語ごとに、消すか経路を持たせるかを決める。
2. 宣言と許容リストを更新する。

## タスク

- [ ] T001 [Plan] 語ごとに扱いを決める。
- [ ] T002 [Contract] TypeSpec、ハンドラー、イベントの宣言を更新する。
- [ ] T003 [Tooling] 許容リストを更新する。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `mise run check-unspecified-vocabulary`
- `mise run check-api-compat`
- `mise run verify`

## リスク

- エラーの宣言を消すと、OpenAPI の互換性の検査が後退として報告する。
  未リリースなので、基準を更新して受け入れる。
