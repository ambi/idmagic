---
depends_on: []
status: pending
authors: [tn]
risk: medium
created_at: 2026-08-29
priority: p2
change_kind: bugfix
affected_spec:
  - { path: docs/contexts/provisioning/scenarios.md, requirement: REQ-PROVISIONING-013 }
  - { path: spec/contexts/provisioning/models.tsp, symbol: IdMagic.Contract.ProvisioningFeatureFlags }
---

# `push_groups` が配信を生まない状態を解消する

## Motivation

Provisioning の正本文書は Group の送出を能力として宣言している。`glossary.md` は Push Groups を「`ProvisioningFeatureFlags.push_groups` が有効なとき、Group とメンバーシップを下流へ配信する機能」と定義し、`states.md` の配信ライフサイクルは `GroupPushed` と `GroupMembershipPushed` を終端への遷移として持ち、TypeSpec は `push_groups` と `GroupPushConfig` を接続の設定として公開している。

**実装には、Group の配信を生む経路が無い。**

- 配信を捕捉するのは User の変更と割り当ての変更の 2 つだけで、Group の変更を捕捉する通知先はどの Context にも配線されていない。
- Full Resync が配信を作るのは適用範囲の User だけで、割り当ての一覧からも `user` 種別しか拾わない。
- 配信実行時に属性を解決する経路は User の Aggregate しか扱わず、Group を渡すと「対象が無い」として何も送らずに成功で終わる。
- 送出クライアントには Group の作成・更新とメンバーシップの PATCH が実装されているが、呼び出す側がいない。

つまり `push_groups` を有効にしても、下流へは何も起きない。**設定は保存され、画面は有効と表示し、配信は 1 件も生まれない。** 失敗として現れないぶん、動いていないことに気付く手掛かりが無い。

[[wi-403-provisioning-declares-no-scim-conformance]] は外向き SCIM の準拠範囲を宣言したが、Group リソースの行を置けなかった。`excluded` と書けば上記の正本文書と矛盾し、`partial` と書けば動かないものを動くと言うことになるからである。**この非一貫が解けるまで、外向きの Group 送出は規範として宣言できない。**

## Scope

- Group の変更とメンバーシップの変更を配信として捕捉する経路を作る。捕捉は既存の同一トランザクション捕捉のポートに揃える。
- Group の属性を解決する経路を作り、`GroupPushConfig` の表示名の取得元を反映する。
- Full Resync と On-Demand Provision が Group を対象に含む条件を決める。
- `push_groups` が無効なときに Group の配信が生まれないことを、否定テストとして固定する。
- 解決したうえで `docs/contexts/provisioning/standards.md` に Group リソースの行を足す。

## Out of Scope

- Group の入れ子構造の送出。直接所属だけを扱う内向きの方針に揃える。
- 下流の Group を IdMagic へ取り込む向き。`Sourcing` の関心である。
- 能力そのものを取り下げる判断。取り下げるなら `glossary.md`、`states.md`、TypeSpec、画面の 4 つを同時に外すことになり、この work item より大きい。着手時に費用を比較して選ぶ。

## Design

未定。着手時に、能力を実装するのか取り下げるのかを先に確定して本節に記録する。実装を選ぶ場合、Group の変更を誰が通知するか——`IdManagement` の Group 側か `Application` の割り当て側か——を Context Map の関係と照らして決める。

## Plan

1. 実装と取り下げのどちらを取るかを決める。
2. 実装を選ぶなら、捕捉、属性解決、配信の順に 1 挙動ずつ広げる。
3. `standards.md` に Group の行を足し、証拠テストを付ける。

## Tasks

- [ ] T001 [Design] 実装と取り下げのどちらを取るかを確定する。
- [ ] T002 [Acceptance] Group の変更が配信を生むことの受け入れ検査を RED で置く。
- [ ] T003 [App] 捕捉、属性解決、配信を実装する。
- [ ] T004 [Spec] `standards.md` に Group の行を足す。
- [ ] T005 [Verify] `mise run check-spec`、`mise run verify`。

## Verification

- `mise run check-spec`
- `mise run test-go`
- `mise run verify`

## Risk Notes

リスクは medium。動いていなかった経路が動き出すため、`push_groups` を有効にしたまま放置されている接続があれば、この変更の後に初めて下流へ書き込みが始まる。誤削除ガードと適用範囲の判定が Group にも効くことを、実装より先に確かめる。
