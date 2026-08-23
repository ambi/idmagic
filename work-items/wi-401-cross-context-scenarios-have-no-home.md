---
depends_on: []
status: pending
authors: [tn]
risk: medium
created_at: 2026-08-23
priority: p1
change_kind: docs
affected_spec:
  - { path: spec/contexts/authentication/scenarios.md, requirement: REQ-AUTHENTICATION-009 }
  - { path: spec/contexts/identity-management/scenarios.md, requirement: REQ-IDMANAGEMENT-012 }
  - { path: spec/contexts/sharedsignals/scenarios.md, requirement: REQ-SHAREDSIGNALS-002 }
---

# `spec/scenarios.md` を作り、Context を跨ぐ保証に置き場所を与える

## Motivation

[SPECIFICATION_FORMAT.md](../SPECIFICATION_FORMAT.md) §1 の正規文書一覧と `tools/check/src/specification-doc.ts:41` の `ROOT_DOCUMENTS` は `spec/scenarios.md` を認めている。**このファイルは存在しない。** `spec/README.md` の `Documents` 表にも行が無いので、読み手は置き場所があること自体を知らない。

§3 はその置き場所が何のためにあるかをこう書いている。

> A context owns only behavior it can satisfy and verify on its own. Behavior that holds only when several contexts cooperate belongs to `spec/scenarios.md` ... Splitting such a flow into per-context fragments leaves no place where the real guarantee is stated.

**すでにその状態になっている。** 「主体を止めたら到達経路が閉じる」という 1 つの保証が、3 つの Context に断片として散っている。

| ID | 置き場所 | WHEN を起こす主体 |
|---|---|---|
| `REQ-AUTHENTICATION-009` | Authentication | 管理者がユーザーを無効化する（IdManagement の操作） |
| `REQ-IDMANAGEMENT-012` | IdManagement | ユーザーがログインを試みる（Authentication の操作） |
| `REQ-SHAREDSIGNALS-002` | SharedSignals | 管理者が `DisableAdminUser` する（IdManagement の操作） |

3 件とも、**自分の Context だけでは `WHEN` を起こせない。** それぞれの Context は自分の担当分だけを述べており、どれも嘘ではない。しかし 3 件を足しても次のことはどこにも書かれていない。

- 無効化してから、既存セッション・新規ログイン・エージェントのトークンという 3 つの経路が閉じるまでに、どこまでの遅れを許すのか。
- 3 つの経路のうち 1 つが閉じなかったとき、それは仕様のどの記述に違反するのか。
- SharedSignals の外部配送が失敗している間も、ローカルの失効は先に確定するのか（`REQ-SHAREDSIGNALS-007` が「受信側の障害はローカル失効を遅らせない」と書くが、これは受信側の話であり、無効化から始まる連鎖全体の保証ではない）。

無効化がセキュリティ機能である以上、**この 3 経路が同時に閉じることこそが製品の保証**であり、断片のどれもそれを述べていない。実装が 1 経路だけ塞ぎ忘れても、矛盾する記述がどこにも無いので仕様上は正しいままになる。

## Scope

- `spec/scenarios.md` を作り、複数の Context が協調して初めて成り立つ振る舞いを置く。各シナリオは参加する Context を名指す。
- 対象を洗い出す。少なくとも次の 3 系統を候補とする。
  - 主体の停止・削除予約・完全削除が、ログイン・既存セッション・エージェントトークン・下流プロビジョニングへ伝わる連鎖。
  - `Sourcing` の取り込みが `IdManagement` を経て `Provisioning` の配信を起こす、上流から下流への伝播。
  - `Seeding` が `Tenancy` / `IdManagement` / `Application` へ発行する published command の適用。
- `spec/README.md` の `Documents` 表に行を足す。
- 既存の断片のうち、cross-context の保証へ引き上げたものの扱いを決めて実行する。

## Out of Scope

- 新しい振る舞いの追加。いま成り立っている保証を書き留めることに限る。書けない箇所が見つかったら、それは実装の欠陥なので個別の work item へ切り出す。
- 断片が持つ Context 固有の部分の移動。`REQ-AUTHENTICATION-009` のうち「無効なユーザーのログインは `AccessDeniedError`」は Authentication が単独で検証できるので、Authentication に残る。
- 対応するテストの追加。cross-context のシナリオを検証するテストがどの層に属するかは、シナリオが書けてから決まる。
- 機能の垂直分割（SPECIFICATION_FORMAT §1 が触れる `spec/contexts/<context>/<feature>/`）。今回の分断は Context 間のものであり、Context 内の分割とは別の問題である。

## Design

未定。着手時に次の 3 点を確定して本節に記録する。

1. **既存の断片を移すのか、残して上に足すのか。** `REQ-*` は「一度参照されたら変更しない」ので、`REQ-AUTHENTICATION-009` を `spec/scenarios.md` へ物理的に移してもその id のままである。Context 名を持つ id がルートの文書に載る状態になる。取りうる案は 3 つある。
   - **(a) id を保ったまま移す。** 参照は壊れない。ただし id の接頭辞が置き場所を表さなくなる。
   - **(b) 新しい id を作り、既存を退役させる。** §6 の退役の形（`superseded by`）が使え、置き場所と id が一致する。既存 3 件を参照しているテストの張り替えが要る。
   - **(c) 断片を残し、その上に cross-context の保証を新しい id で足す。** 移動が無いので安全だが、同じことを 2 か所が述べる状態になり、§3 が避けようとしたものに戻る。

   **(c) は選ばない。** 残る 2 案のどちらを取るかは、既存 3 件を名指しているテストの数を数えてから決める。

2. **対象の洗い出しをどこまで機械化するか。** 「`WHEN` の操作を自 Context が持たない」シナリオは候補として抽出できる可能性がある。ただし `WHEN` は自然言語なので、操作名と Context の対応を人が与えないと判定できない。**まず手で全 21 Context の `scenarios.md` を通し、機械化できる形が見えるかを確かめる。** 見えなければ手作業のままにして、検査は置かない。

3. **`REQ-` の接頭辞。** ルートの文書に置くシナリオが新しい id を持つ場合、`<CONTEXT>` の部分に何を書くか。`REQ-SYSTEM-*` は `system` Context（起動、経路の組み立て、健全性、フロントエンド）が既に使っている。ルート用の接頭辞を新設するか、`system` に相乗りするかを決める。相乗りは、`system` Context の境界宣言と食い違う。

## Plan

- 全 Context の `scenarios.md` を通し、`WHEN` が他 Context の操作であるシナリオを列挙する。ここが作業量の実測になる。
- 列挙結果を見てから 1 の案を決める。移す件数が少なければ (b)、多ければ (a) に寄る。
- `spec/scenarios.md` を作り、まず 1 系統（主体の停止の連鎖）だけを書く。書式検査と生成サイトの導線がそれで通ることを確かめてから残りへ広げる。
- 「書けない」が出たら切り出して先へ進む。止まらない。

## Tasks

- [ ] T001 [Spec] 全 Context の `scenarios.md` から、`WHEN` を自 Context が起こせないシナリオを列挙する。
- [ ] T002 [Design] 既存 id の扱い、洗い出しの機械化の可否、接頭辞を確定し `## Design` に記録する。
- [ ] T003 [Spec] `spec/scenarios.md` を作り、主体の停止の連鎖を書く。参加する Context を名指す。
- [ ] T004 [Spec] `spec/README.md` の `Documents` 表に行を足す。
- [ ] T005 [Spec] 残る系統（上流からの伝播、published command の適用）を書く。
- [ ] T006 [Spec] 引き上げた断片を、2 で決めた方式で処理する。
- [ ] T007 [Triage] 保証を書こうとして書けなかった箇所を、実装の欠陥として個別の work item へ切り出す。
- [ ] T008 [Verify] `mise run check-spec`、`mise run check-ids`、`mise run spec-render`、`mise run verify` を通す。

## Verification

- `mise run check-spec`
  - reason: `spec/scenarios.md` は `ROOT_DOCUMENTS` にあるが、実在した状態で検査が通ったことはまだない。
- `mise run check-ids`
  - reason: id の一意性と、退役させた場合の後継の実在を確かめる。
- `mise run spec-render`
  - reason: 生成サイトが新しいルート文書へ導線を張れることを確かめる。ROOT_DOCUMENTS にあっても、生成側が実ファイルの存在を前提にしている箇所があれば、ここで分かる。
- `mise run spec-diff`
  - reason: 引き上げと退役で仕様がどう動いたかを、記憶ではなく差分から読む。
- `mise run verify`

## Risk Notes

リスクは medium。**既存の `REQ-` id に触れる可能性があるためである。** id はテスト名、`affected_spec`、完了済み work item から参照されている。案 (b) を取る場合、参照元をすべて張り替えないと `check-ids` と `check-work-items` が落ちる。落ちれば気付けるので、静かに壊れることは無い。

内容の側の危険は、**cross-context の保証を書くときに、実装が実際に何を保証しているかを確かめずに書くこと**である。「無効化から 60 秒以内に全経路が閉じる」と書くのは簡単だが、それを満たしていることを誰も測っていなければ、その行は仕様ではなく願望である。測れない保証は、測れる形に言い換えるか、書かない。

3 系統すべてを 1 度に書こうとすると止まる。1 系統ずつ入れれば、中断しても入った分は残る。
