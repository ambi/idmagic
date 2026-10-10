---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-09
priority: p1
depends_on: [wi-17193-move-context-specific-code-out-of-support-http, wi-39119-publish-the-resolved-tenant-as-tenancy-public-language, wi-65906-inject-time-randomness-and-network-into-domain, wi-92970-adopt-information-hiding-modular-design]
change_kind: tooling
spec_impact: { kind: none, reason: "境界の負債の台帳の理由を書き直し、残りの解消を work item として起票するだけである。製品のコード、HTTP の応答、ドメインイベント、永続状態は変えない。" }
---

# 境界の負債を変更の局所性と開発コストで順位付けし、削減する作業へ分解する

## 動機

`tools/check/boundary-debt.json` の `private-import`、`module-cycle`、`shared-dependency` の項目の理由は、旧台帳から引き継いだ定型文か、移行のときに機械的に作った文である。
定型文の理由からは、どの依存をどう直すのかが読み取れず、解消の work item も起票されていない。

wi-17193、wi-39119、wi-65906 で、原因が一か所に集まった負債（`support_http` を経由した依存、Tenancy への依存、`domain` の作用）が消える。
残るのは、モジュールの組ごとに原因の異なる依存である。
現時点で多いのは、`private-import` の参照先の `backend/claimmapping/usecases`（11）と `backend/jobs/usecases`（9）、および 57 件の `module-cycle` である。

`table-write` の 7 件は、原子性の理由と参照を持つ具体的な理由を移行の時点で書いてある。
ただし、公開操作を別のトランザクションで呼ぶ形へ変えるだけでは原子性を保てないので解消にならない。

2026-10-10 の設計調査では、既存負債は合計 211 件だった。
これは調査時点の値であり、前提項目の完了後に再計測する。
負債の増加を拒否しても、既存の結合による探索範囲、変更の波及、業務規則を迂回できる公開範囲は残る。
件数だけでなく、繰り返し払う開発コストと守るべき保証から削減順を決める。

現在は未リリースであり、当面も未リリースを継続する。
旧版との互換性維持より、現在の設計を小さな文脈で変更できることを優先して p1 とする。
公開済み利用者がいるとは仮定せず、パッケージの変更では既存の製品仕様と原子性を維持する。

## 対象範囲

- 前提の 3 項目が完了した後の `mise run check-boundaries` の結果から、残った違反をモジュールの組ごとにまとめる。
- 組ごとに、[境界を選ぶ判断手順](../../docs/design/application/design-guidelines.md#境界を選ぶ判断手順)の D1〜D8 で直す方法を決める。
  非公開の処理への依存は D3、循環は D4、共有ライブラリからの依存は D5、変換の配置は D6、所有者外の書き込みは D2 と D7 を適用する。
- 直す方法と、それが必要な理由を、台帳の各項目の理由として書き直す。
- 直すための work item を、まとめて直せる単位で起票する。
- 変更頻度、共変更、参照する公開範囲、所有者外の書き込み、探索するモジュールの範囲から削減順と根拠を記録する。
- Repository や内部型を公開する案と、必要な業務操作だけを公開する案を比較する。
- 解消項目を既存の internal 移行項目へ対応付け、依存の解消と機械的な配置変更を重複起票しない。
- `table-write` の 7 件は、呼び出し側のトランザクションに参加する公開操作を所有者が公開する案（D2、D7）を比べ、採るなら解消の work item を起票する。

## 対象外

- 個々の依存と書き込みの解消そのもの。起票した work item で行う。
- 件数だけからの一律な分割と統合、マイクロサービス化。
- internal への配置変更そのものは[呼び出し側のないモジュールの移行](wi-97546-move-modules-without-private-callers-behind-go-internal.md)と[残るモジュールの移行](wi-75383-move-remaining-modules-behind-go-internal-and-retire-legacy.md)が扱う。

## 設計

前提項目が除く共通原因と、残るモジュールごとの原因を分ける。
依存の実態はコードと境界検査、共変更の履歴は mise run report-change-coupling から導き、手書きの依存グラフは作らない。
共変更の回数は調査候補の選択に使い、件数だけで境界を変更しない。
比較には履歴の対象期間、全体更新の除外条件、母数を添える。

各解消案は、隠す業務規則、必要な公開操作、変更が閉じる範囲、トランザクションの不変条件を説明する。
Repository を単に公開へ変えて違反を消す案や、所有者への呼び出しを別トランザクションにするだけの案は、保証を保つ根拠がなければ採らない。
本項目は順位付けと解消項目の起票を完了条件とし、配置変更や依存の削減が実施済みだとは報告しない。

## 計画

1. 前提項目の完了後に残る負債と参照元を再計測する。
2. 変更頻度、共変更、探索範囲と守るべき保証から優先する組を選ぶ。
3. D1〜D8 で候補を比較し、台帳の理由と解消項目へ反映する。
4. 既存の internal 移行との役割分担と依存を確認し、削減後に再評価する指標を記録する。

## タスク

- [ ] T001 [Inventory] 残った違反をモジュールの組ごとにまとめ、変更頻度、共変更、探索範囲、原子性から削減順を決める。
- [ ] T002 [Design] 公開する業務操作、隠す規則、変更の局所性を D1〜D8 で比較する。
- [ ] T003 [Tooling] 台帳の理由を書き直す。
- [ ] T004 [Plan] 解消の work item を起票する。
- [ ] T005 [Verify] 変更を検証する。

## 検証

- `mise run check-boundaries`
- `mise run check-boundary-debt-ratchet`
- `mise run report-change-coupling` の条件と母数を記録し、優先順位の根拠を説明する。
- すべての残る項目に、保証を維持する具体的な解消項目があり、既存の internal 移行と重複しない。
- `mise run check-work-items`

## リスク

- 理由を書き直すだけでは負債は減らない。
  この項目の完了の条件を、すべての残りの項目に解消の work item が対応していることにする。
