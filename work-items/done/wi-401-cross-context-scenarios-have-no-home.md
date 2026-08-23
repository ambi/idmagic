---
depends_on: []
status: completed
authors: [tn]
initial_context:
  specification: [docs/scenarios.md, docs/contexts/authentication/scenarios.md, docs/contexts/identity-management/scenarios.md, docs/contexts/sharedsignals/scenarios.md, docs/contexts/provisioning/scenarios.md]
  source: [tools/check/src/spec-diff.ts, tools/check/src/check-security-controls.ts]
  tests: [tools/check/src]
  stop_before_reading: [backend, frontend, spec]
risk: medium
created_at: 2026-08-23
priority: p1
change_kind: docs
affected_spec:
  - { path: docs/contexts/authentication/scenarios.md, requirement: REQ-AUTHENTICATION-009 }
  - { path: docs/contexts/identity-management/scenarios.md, requirement: REQ-IDMANAGEMENT-012 }
  - { path: docs/contexts/sharedsignals/scenarios.md, requirement: REQ-SHAREDSIGNALS-002 }
---

# `docs/scenarios.md` を作り、Context を跨ぐ保証に置き場所を与える

## Motivation

[SPECIFICATION_FORMAT.md](../SPECIFICATION_FORMAT.md) §1 の正規文書一覧と `tools/check/src/specification-doc.ts:41` の `ROOT_DOCUMENTS` は `docs/scenarios.md` を認めている。**このファイルは存在しない。** `docs/README.md` の `Documents` 表にも行が無いので、読み手は置き場所があること自体を知らない。

§3 はその置き場所が何のためにあるかをこう書いている。

> A context owns only behavior it can satisfy and verify on its own. Behavior that holds only when several contexts cooperate belongs to `docs/scenarios.md` ... Splitting such a flow into per-context fragments leaves no place where the real guarantee is stated.

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

- `docs/scenarios.md` を作り、複数の Context が協調して初めて成り立つ振る舞いを置く。各シナリオは参加する Context を名指す。
- 対象を洗い出す。少なくとも次の 3 系統を候補とする。
  - 主体の停止・削除予約・完全削除が、ログイン・既存セッション・エージェントトークン・下流プロビジョニングへ伝わる連鎖。
  - `Sourcing` の取り込みが `IdManagement` を経て `Provisioning` の配信を起こす、上流から下流への伝播。
  - `Seeding` が `Tenancy` / `IdManagement` / `Application` へ発行する published command の適用。
- `docs/README.md` の `Documents` 表に行を足す。
- 既存の断片のうち、cross-context の保証へ引き上げたものの扱いを決めて実行する。

## Out of Scope

- 新しい振る舞いの追加。いま成り立っている保証を書き留めることに限る。書けない箇所が見つかったら、それは実装の欠陥なので個別の work item へ切り出す。
- 断片が持つ Context 固有の部分の移動。`REQ-AUTHENTICATION-009` のうち「無効なユーザーのログインは `AccessDeniedError`」は Authentication が単独で検証できるので、Authentication に残る。
- 対応するテストの追加。cross-context のシナリオを検証するテストがどの層に属するかは、シナリオが書けてから決まる。
- 機能の垂直分割（SPECIFICATION_FORMAT §1 が触れる `docs/contexts/<context>/<feature>/`）。今回の分断は Context 間のものであり、Context 内の分割とは別の問題である。

## Design

3 点とも着手時に確定した。**判定の基準は「その Context だけでは `WHEN` を起こせないこと」に置いた。** 全 21 Context の `scenarios.md` から 451 件の `WHEN` を機械的に抜き出し、他 Context の操作を引き金にしているものを絞り込んだ。

1. **断片ごとに扱いを分けた。** 起票時の (a)(b)(c) はいずれも全件に同じ扱いをする案だったが、断片の性質が一様でなかった。

   | 断片 | 扱い | 理由 |
   |---|---|---|
   | `REQ-AUTHENTICATION-009` | **書き直す** | `GIVEN` を「無効なユーザー」に変えれば Authentication が単独で起こせる。無効な主体を拒否することは Authentication 固有の保証であり、失うべきでない |
   | `REQ-IDMANAGEMENT-012` | **退役**（`superseded by REQ-PLATFORM-002`） | 引き金は IdManagement、観測はログインの拒否。内容が丸ごと横断の保証で、残すと二重になる |
   | `REQ-SHAREDSIGNALS-002` | **退役**（`superseded by REQ-PLATFORM-001`） | 引き金が `DisableAdminUser`。同上 |
   | `REQ-PROVISIONING-003/004/005` | **書き直す** | 引き金を「捕捉した変更を Provisioning が処理する」に変える。各変更がどの下流操作に対応するかは Provisioning 固有の詳細で、退役させると失われる |

   3 件とも**テストが 1 つも名指ししていない**ことを確認したので、退役の張り替え費用はゼロだった（`REQ-*` を名指すテストが無いこと自体は [[wi-399-burn-down-untested-refusal-debt]] の問題である）。

2. **機械化は入口までにとどめた。** `WHEN` の行を抜き出すところまでは機械でできたが、「その操作が自 Context のものか」は語彙の対応を人が与えないと判定できない。451 件から他 Context の語を含む候補を絞り、そこを人が読む形にした。**検査は置かない。** 判定できない基準を検査にすると、通ることだけが目的になる。

3. **接頭辞は `REQ-PLATFORM-` とした。** `REQ-SYSTEM-` は System Context（起動、経路の組み立て、健全性、UI）が使っており、その境界宣言と食い違う。検査は `REQ-[A-Z0-9-]+` しか要求しないので新設できる。

### 系統ごとの判定

- **主体の停止の連鎖** — 該当した。`REQ-PLATFORM-001`（無効化）と `REQ-PLATFORM-002`（削除予約と復元）にした。
- **上流からの伝播** — **Sourcing 側は該当しない。** `REQ-SOURCING-*` の `WHEN` はいずれも「SCIM クライアントが呼ぶ」であり、Sourcing 自身の受信 API である。該当したのは下流側で、`REQ-PLATFORM-003` にした。
- **Seeding の published command** — **該当しない。** `REQ-SEEDING-*` の `WHEN` はすべて `SeedOperator` が `SeedData` を呼ぶ形で、Seeding 自身の操作である。結果が他 Context のデータに現れるが、Seeding が単独で起こし観測できる。

## Plan

- 全 Context の `scenarios.md` を通し、`WHEN` が他 Context の操作であるシナリオを列挙する。ここが作業量の実測になる。
- 列挙結果を見てから 1 の案を決める。移す件数が少なければ (b)、多ければ (a) に寄る。
- `docs/scenarios.md` を作り、まず 1 系統（主体の停止の連鎖）だけを書く。書式検査と生成サイトの導線がそれで通ることを確かめてから残りへ広げる。
- 「書けない」が出たら切り出して先へ進む。止まらない。

## Tasks

- [x] T001 [Spec] 全 Context の `scenarios.md` から 451 件の `WHEN` を抜き出し、他 Context の操作を引き金にするものを絞り込んだ。
- [x] T002 [Design] 既存 id の扱い、機械化の可否、接頭辞を確定し `## Design` に記録した。
- [x] T003 [Spec] `docs/scenarios.md` を作り、主体の停止の連鎖を `REQ-PLATFORM-001` / `REQ-PLATFORM-002` として書いた。
- [x] T004 [Spec] `docs/README.md` の `Documents` 表に行を足した。
- [x] T005 [Spec] 下流への伝播を `REQ-PLATFORM-003` として書いた。上流からの取り込みと Seeding は該当しないと判定した（Design 参照）。
- [x] T006 [Spec] 断片 6 件を、書き直し 4 件・退役 2 件として処理した。
- [x] T007 [Triage] 保証を書こうとして書けなかった箇所は無かった。代わりに検査ツールの欠陥を 2 件見つけた（Completion 参照）。
- [x] T008 [Verify] `mise run verify` を通した。

## Verification

- `mise run check-spec`
  - reason: `docs/scenarios.md` は `ROOT_DOCUMENTS` にあるが、実在した状態で検査が通ったことはまだない。
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

## Completion

- **Completed At**: 2026-08-23
- **Summary**:
  `docs/scenarios.md` を作り、Context を跨がないと成り立たない保証を 3 件置いた。`REQ-PLATFORM-001` は主体の無効化がログイン・既存セッション・エージェントのトークンという 3 経路を**同時に**閉じることを 1 つの保証として述べ、外部への伝播はこの保証に含まれないこと（内部で閉じ切るのが先）も明示した。`REQ-PLATFORM-002` は削除の予約と復元が到達経路の開閉と対応することを、`REQ-PLATFORM-003` は記録の正の変更と配信行が同じトランザクションでコミットまたはロールバックすることを述べる。断片 6 件は、自 Context が引き金を持てる形へ書き直したもの 4 件と、内容が丸ごと横断で退役させたもの 2 件に分けた。判定は全 21 Context の 451 件の `WHEN` から絞り込んで行い、Sourcing と Seeding は該当しないと判定した。
- **Verification Results**:
  - `mise run verify` - passed（exit 0）
  - `mise run spec-diff` - `added: REQ-PLATFORM-001/002/003`、`changed: REQ-AUTHENTICATION-009, REQ-IDMANAGEMENT-012, REQ-PROVISIONING-003/004/005, REQ-SHAREDSIGNALS-002`
  - `mise run check-spec` - ok 138 document(s)（`docs/scenarios.md` を含む）
  - `mise run check-ids` - 407 件 OK（退役の後継 `REQ-PLATFORM-001` / `REQ-PLATFORM-002` の実在を含む）
  - `mise run check-security-controls` - ok 179 declared / 18 promised / 130 awaiting

### 途中で見つけた検査ツールの欠陥 2 件

**どちらも [[wi-405-spec-and-docs-boundary-is-not-legible]] の移動で壊れ、成功を報告し続けていた。** 本 work item の中で修正した。

| ツール | 何が起きていたか | 気付いた経緯 |
|---|---|---|
| `spec-diff` | `git ls-tree -- spec` と `walk(spec/)` しか見ておらず、散文が `docs/` へ移った後は **Markdown を 1 件も読んでいなかった。** 常に `no normative specification change` を返す | シナリオを 3 件足し 2 件退役させたのに「変更なし」と出た |
| `check-security-controls` の R4 | `.tsp` を `docs/contexts/` から読んでおり、**契約が約束する 403 を 1 件も検査していなかった** | 「promised by a 403」の件数が 18 から 0 に落ちていた |

**wi-405 の Completion は `spec-diff` の出力を検証の根拠として挙げている。** その主張自体は結果的に正しかった（本当に規範要素は動いていない）が、根拠は空だった。wi-405 の記録に訂正を追記した。

## Left Undone

- **`docs/scenarios.md` の拒否は `check-security-controls` の対象外である。** R3 と R4 は `docs/contexts/*/scenarios.md` だけを走査する。`REQ-PLATFORM-*` が宣言する拒否（無効なユーザーのログイン拒否など）にテストを要求する仕組みが無い。**横断の保証こそテストが要るのに、いまは要求されていない。** 走査対象を広げるかどうかは、`REQ-PLATFORM-*` のテストがどの層に属するかを決めてからになる。
- **`REQ-PLATFORM-*` に対応するテストを書いていない。** 起票時の Out of Scope のとおりで、シナリオが書けた今、どの層が持つべきかを決められる状態になった。
- **同じ形の欠陥がまだ他にもありうる。** 「静かにゼロを返す検査」はこのセッションで 3 件見つかっている（リンク検査、`spec-diff`、R4）。**件数を出力する検査は、件数そのものを見る習慣がないと壊れても気付けない。** 検査が 0 件を扱ったときに落ちる仕組みを持つかどうかは、[[wi-408-link-check-is-not-a-gate]] と同じ性質の課題として残る。
