---
status: completed
authors: [tn]
risk: low
reversibility: irreversible
created_at: 2026-09-21
priority: p2
depends_on: []
change_kind: tooling
spec_impact:
  kind: none
  reason: "作業項目の採番規則と、それを支える検査および mise タスクだけを扱い、プロダクトの振る舞いも規範要素も変えない。"
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 変更は作業項目を起票する側の規則と検査にとどまり、製品の振る舞いにも公開契約にも届かない。リリースの読み手に見える差分がない。
  references: []
initial_context:
  specification:
    - WORK_ITEM_FORMAT.md
  typespec: []
  source:
    - mise.toml
    - tools/check/src/work-item-dependencies.ts
    - tools/check/src/check-work-items.ts
    - tools/check/src/documentation-impact.ts
    - tools/check/src/runner.ts
    - tools/brief/src/main.ts
    - .agents/skills/new-work-item/SKILL.md
  tests:
    - tools/check/src/work-item-dependencies.test.ts
    - tools/check/src/check-work-items.test.ts
  stop_before_reading:
    - backend
    - frontend
    - spec
    - docs/domain
---

# 作業項目の識別番号を、未使用の 5 桁から無作為に選ぶ

## 動機

連番は、番号を配る主体が一つであることを前提にする。
この前提は、複数のメンバーがそれぞれのローカルで起票し、同じ人が並列のワークツリーで起票する今の運用では成立しない。
最大値 + 1 を数える手順は、まだ push されていない起票を見られないからである。

衝突はすでに起きている。
`work-items/done/` には次の 3 組が同じ番号で存在していた。

| 番号 | 番号を保つ記録 | 番号を移す記録 |
| --- | --- | --- |
| 112 | `wi-112-prometheus-metrics-and-authentication-golden-signals` | `wi-3-persistence-mode-rename-postgres-valkey` |
| 126 | `wi-126-admin-and-account-ui-consistency-and-navigation-policy` | `wi-42-async-job-runner` |
| 127 | `wi-127-postgres-column-type-policy` | `wi-245-mfa-enrollment-onboarding-and-enforcement` |

機械検査はこれを拾えない。
作業項目の id は拡張子を除いたファイル名全体なので、`work-item-dependencies.ts` の重複検査は題名が違えば別の記録として通す。
番号だけの衝突を見る検査は、どこにもない。

番号は `work-items/` のファイル名のほか、`depends_on`、`docs/releases/changes/wi-*.md` のファイル名、ブランチ名 `work-item/wi-48213-example`、ワークツリー名 `../idmagic-wi-48213` にも現れる。
衝突を事後の改名で直す代償はここにあり、参照がリポジトリ外へ出た後では追い切れない。
既存の 3 組は、参照がまだリポジトリの中に閉じている今のうちに解消する。

## 対象範囲

- `WORK_ITEM_FORMAT.md` のファイル名規則を、連番から未使用の 5 桁の無作為選択へ改める。
- 採番を行う `mise` タスクを追加し、人が最大値を数える手順を置き換える。
- `work-item-dependencies.ts` へ、番号の衝突を検出する検査と、空き枠が閾値を切ったことを知らせる検査を追加する。
- 既存 3 組の衝突を、各組の片方を未使用の番号へ移して解消する。参照はすべて同じ変更で張り替える。
- `new-work-item` スキル（`.agents/skills` と `.claude/skills` の両方）の採番手順を書き換える。

## 対象外

- 衝突していない既存 631 件の改名。番号空間が重ならないため必要がなく、改名は参照の張り替えを強制する。
- 識別子から起票者や起票日を読めるようにすること。起票者は識別子上は匿名とし、新旧は frontmatter の `created_at` が示す。
- 桁数を将来さらに増やすときの規則。必要になってから決める。理由は設計に書く。
- ADR の作成。採番規則は `WORK_ITEM_FORMAT.md` が正本である。

## 設計

### 採る形

新規の作業項目は `wi-<識別番号>-<ケバブケースの題名>.md` を使い、識別番号は 10000 から 99999 のうち、`work-items/` と `work-items/done/` のどちらにも現れない値を無作為に選ぶ。
ファイル名の文法は今と変わらず、変わるのは値の選び方だけである。

既存の記録は `wi-1` から `wi-634` までの 3 桁以下に収まっている。
新規を 5 桁に限れば番号空間が重ならないので、既存のファイル名、`depends_on`、リリース文書の名前、ブランチ名は一つも動かない。
`docs-work-item-links.ts` の `wi-[0-9]+` も、`documentation-impact.ts` が文書化契約の適用範囲を決める `sequence >= 452` も、そのまま意図どおりに働く。
新規は常に 10000 以上なので、後者は必ず契約の対象になる。

### 衝突の確率

既存との衝突は採番時に除外するので起きない。
残るのは、まだ push されていない起票どうしが同じ値を引く場合だけである。
空き枠を S、同時に起票される件数を n として、確率はおよそ n(n-1)/2S になる。

| 同時起票 | 枠の 1 割消費時 | 枠の 8 割消費時 |
| --- | --- | --- |
| 2 件 | 0.0012% | 0.0056% |
| 5 件 | 0.012% | 0.056% |
| 10 件 | 0.055% | 0.25% |

衝突が起きても、両方の記録はまだ push されていないため、参照は起票者のブランチの中で閉じている。
片方を採番し直せば済み、リポジトリ外へ出た識別子を追う事態にはならない。

### 桁数を 5 とする根拠

起票の速度を `created_at` から数えた。

| 月 | 起票数 |
| --- | --- |
| 2026-06 | 80 |
| 2026-07 | 228 |
| 2026-08 | 148 |
| 2026-09（21 日時点） | 171 |

立ち上がりの 6 月を除くと月 180 から 240 件、年およそ 2500 件である。
4 桁の 9000 枠はこの速度で 3.6 年、衝突確率が上がり始める 8 割消費までなら 2.9 年しかもたない。
5 桁の 90000 枠は同じ速度で 36 年もつ。
識別子は 1 文字長くなるが、`wi-48213` は郵便番号や暗証番号と同じ桁数であり、記憶の負荷は `wi-4821` と変わらない。

### 採らない案

| 案 | 採らない理由 |
| --- | --- |
| 起票元ごとに番号空間を分ける（`wi-tn-12`） | 識別子に起票者が残る。同じ人が並列のワークツリーで起票すると、なお衝突する。覚える対象が二つになる |
| 日付 + 短い無作為値（`wi-2609-482`） | 日付は同月内の衝突を防がないため、一意性にほとんど寄与しないまま長さだけを足す。同じ安全性を得るには無作為部分を 4 桁にする必要があり、全体は 8 文字になる。新旧は `created_at` がより正確に示す |
| 発音可能な短い語（`wi-tsuba`） | 辞書を数千語に収めると衝突が日常的に起き、数万語へ広げると語が長くなって利点が消える。内容と相反する語が当たると読み手を惑わせる |
| 衝突を許容して事後に改名する | 改名が `depends_on`、リリース文書名、ブランチ名の張り替えを強制する。しかも衝突が判明するのは統合時、つまり参照がすでに広がった後である |
| 題名を 2 語へ縮めたスラッグ（`wi-dpop-htu`） | 長さが可変で、起票のたびに命名を決める時間が生まれる。同じ領域の作業が増えると `wi-scim-filter` のような重複が現実的な頻度で起きる。意味は題名がすでに担っている |
| 4 桁で始め、空き枠が閾値を切ったら桁を増やす | 採番手順、文書、検査が条件分岐を抱え続ける。得るものは最初の 3 年だけ識別子が 1 文字短いことである。桁の拡張は、今回 3 桁の上に 5 桁を置くのと同じ手口でいつでも行えるので、規則を先に書いても選択肢は増えない |

### 空き枠の警告

桁を増やす規則は書かないが、空き枠が減ったことを知る手段は置く。
番号の衝突検査と同じ場所で空き枠を数え、10000 を下回ったら警告として報告する。
この時点でも 10 件同時起票の衝突確率は 0.045% であり、桁を増やす判断に必要な観測を、枯渇するより十分早く得られる。

### 既存の衝突の解消

新しい検査は、番号が重複する記録の組を報告する。
既存の 3 組は例外として除外せず、各組の片方を未使用の番号へ移して解消する。
例外を置けば、検査が報告しない衝突が恒久的に残り、しかも同じ番号を引いた将来の起票まで一緒に見逃す恐れがある。
参照は `depends_on`、`[[...]]` リンク、散文、Go と TypeScript のコメントに閉じており、リリース文書とリポジトリ外へはまだ出ていない。
張り替えられるのは今だけである。

番号を保つのはどちらかを、git に先に現れた側とする。
`created_at` が同じ組があり、そこでは順序を決められないからである。
移す側には 3、42、245 を与える。

| 移す記録 | 新しい番号 |
| --- | --- |
| `wi-112-persistence-mode-rename-postgres-valkey` | 3 |
| `wi-126-async-job-runner` | 42 |
| `wi-127-mfa-enrollment-onboarding-and-enforcement` | 245 |

新規と同じ 5 桁を与えない。
作業項目のスキーマは、識別番号を記録が書かれた時期の代理として読み、410 以上に `evidence_policy` を、452 以上に `documentation_impact` を要求する。
2026 年 7 月に完了したこの 3 件を 5 桁へ移すと、当時存在しなかった契約の対象になり、スキーマ検査が落ちる。
番号がその代理として働き続ける範囲、すなわち 410 未満の空き番号を選ぶ。

3、42、245 を選んだのは、`git log --diff-filter=A --all` がこの 3 つの番号を持つファイルを一度も記録していないためである。
410 未満の空き番号には 269 もあるが、こちらは `wi-269-audit-events-api-500-fix.md` として過去に存在した形跡があり、参照がリポジトリ外に残っている可能性を否定できない。
この選別そのものが、空き番号を数えて再利用する手順の弱さを示している。

### 置く場所とシグネチャ

番号の規則は一箇所に置き、採番と検査の両方がそこから読む。
置き場所は `tools/check/src/work-item-dependencies.ts` とする。
この単位はすでに記録の集合全体を見る検査、すなわち id の重複と依存の閉路を担っており、番号の衝突はその兄弟だからである。

| 名前 | シグネチャ | 役割 |
| --- | --- | --- |
| `IDENTIFIER_RANGE` | `{ min: 10000; max: 99999 }` | 新規の識別番号が取り得る範囲 |
| `CAPACITY_WARNING_THRESHOLD` | `number` | 空き枠がこれを下回ると警告する境目 |
| `workItemNumber` | `(id: string) => number \| undefined` | 記録の id から識別番号を読む |
| `verifyWorkItemIdentifiers` | `(records: WorkItemDependencyRecord[]) => { findings: WorkItemDependencyFinding[]; warnings: string[] }` | 番号が衝突する組を報告し、空き枠が閾値を下回ったときだけ警告を 1 行返す |
| `pickWorkItemNumber` | `(used: ReadonlySet<number>, random: () => number, range?: IdentifierRange) => number` | 未使用の値を一様に選ぶ |

衝突の報告と空き枠の警告を一つの関数から返すのは、`checkWorkItems` から見た配線を一箇所に保つためである。
空き枠の警告は、閾値を割るのに 80000 件を超える記録が要るため、実ファイルを並べる受け入れ検査では起こせない。
両者が同じ呼び出しを通るなら、衝突を観測する受け入れ検査が配線そのものを固定し、警告の条件は単体検査が固定する。

乱数は `pickWorkItemNumber` の引数として入る。
`work-items/` の走査と `Math.random` は `tools/work-item-number/src/main.ts` が担い、純粋な選択と作用を分ける。
縮めた `range` を渡せることが、使用済みの値を返さないという主張を検査可能にする。

### 警告を失敗と分ける方法

`CheckOutcome` は `ok` と `lines` だけを持ち、警告の経路がない。
`checkWorkItems` は現在 `lines.length === 0` から `ok` を導いているため、警告を `lines` へ足すと検査が落ちる。
そこで `ok` の根拠を所見の件数へ移し、警告は所見の有無にかかわらず `lines` へ並べる。
`runner.ts` は `ok` の側の行を標準出力へ出すので、警告は検査を通したまま読み手へ届く。

### 取り消せない部分

採番規則そのものは元に戻せる。
戻せないのは、新しい規則のもとで割り当てた識別番号である。
番号はブランチ名、リリース文書名、他の記録の `depends_on` へ広がるため、割り当て後の変更は参照の張り替えを伴う。
`reversibility: irreversible` はこの部分を指す。

## 計画

1. `WORK_ITEM_FORMAT.md` のファイル名規則と、そこが使う「連番」という語を改める。5 桁に限る理由と、既存が 3 桁以下であることを、規則を読む人が判断を追える形で書く。
2. 採番の `mise` タスクを追加する。`work-items/` と `work-items/done/` を読み、使用済みを除いた 10000 から 99999 の値を一つ出力する。
3. 番号の衝突検査と空き枠の警告を、失敗を観測してから実装する。
4. 既存 3 組を解消する。各組の片方を改名し、`depends_on`、`[[...]]` リンク、散文、コードのコメントの参照を同じ変更で張り替える。短い参照（`wi-126` のような番号だけの言及）は、どちらの記録を指すかを読んで判断する。
5. `new-work-item` スキルの採番手順を、最大値を数える形から採番タスクを呼ぶ形へ書き換える。`.claude/skills` は `.agents/skills` への symlink なので、編集は一度で足りる。
6. `mise run verify` まで通す。

未解決の問いはない。
桁数、空き枠の閾値、既存衝突の扱い、警告と失敗のどちらにするかは、上の設計で決めてある。

### 予定する RED 検査

`risk: low` かつ `change_kind: tooling` なので、主要ユースケースの契約は適用されず、Acceptance RED と Unit RED を一つずつ置く。

| 境界 | 検査 | 実装前に観測する失敗 |
| --- | --- | --- |
| Acceptance | `mise run test-tools-file -- check/src/check-work-items.test.ts`。一時的な workspace へ番号が衝突する 2 件を置き、登録済みの `work-items` 検査をそのまま呼ぶ | 検査が `ok: true` を返し、衝突を一件も報告しない |
| Unit | `mise run test-tools-file -- check/src/work-item-dependencies.test.ts work-item-number/src/work-item-number.test.ts` | `verifyWorkItemIdentifiers` と `pickWorkItemNumber` が存在せず、読み込みに失敗する |

Acceptance の境界を `check-work-items.test.ts` に置くのは、`mise run check-work-items` が呼ぶ入口が `checkWorkItems(snapshot)` そのものであり、作業ツリーの実データに依存せず同じ入口を通せるからである。

## タスク

- [x] T001 [Acceptance] 番号が衝突する 2 件を置いた workspace で `checkWorkItems` が何も報告しないことを、実装前に観測する。
- [x] T002 [Tooling] 番号衝突の検査と空き枠の警告を `work-item-dependencies.ts` へ実装し、`checkWorkItems` へ一箇所で接続する。
- [x] T003 [Tooling] 採番の `mise` タスクを追加し、縮めた候補範囲で使用済みの値を返さないことを確かめる。
- [x] T004 [Docs] `WORK_ITEM_FORMAT.md` の採番規則を、未使用の 5 桁の無作為選択へ改める。
- [x] T005 [Renumber] 既存 3 組の片方を 3、42、245 へ移し、参照をすべて張り替える。
- [x] T006 [Docs] `new-work-item` スキルの採番手順を、採番タスクを呼ぶ形へ書き換える。
- [x] T007 [Verify] 変更を検証する。

## 検証

- `mise run test-tools`
- `mise run check-work-items`
- `mise run check`
- `mise run verify`

## リスク

採番タスクが使用済みの値を返すと、衝突が最初から仕込まれる。
T003 で、既存の全番号を候補から除いていることを、使用済みの値だけが残る縮めた候補範囲を与えて確かめる。

改名が参照を取り残すと、記録どうしのリンクと `depends_on` が解決しなくなる。
完全な id を含む参照は機械的に置き換えられるが、`wi-126` のような番号だけの言及は、どちらの記録を指すかが文脈にしかない。
T005 では短い参照を一件ずつ読んで判断し、`mise run check-links` と `mise run check-work-items` で解決を確かめる。

規則を改めた後も、古い手順を覚えたまま連番で起票される恐れがある。
採番タスクを置くだけでは防げないため、スキルの手順から最大値を数える記述を消し、番号衝突の検査が 3 桁の新規起票も重複として報告することで受け止める。

## 完了

- **Completed At**: 2026-09-21
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返す。
  変更は起票の側の規則、それを支える検査と `mise` タスク、既存 3 組の改名にとどまり、製品にも規範文書にも届かない。
  新規の作業項目は `mise run work-item-number` が出す未使用の 5 桁を使う。
  **番号の衝突を見る検査は、どこにも無かった。** 作業項目の id は拡張子を除いたファイル名全体なので、既存の重複検査は題名が違えば別の記録として通す。
  実際、`work-items/done/` には 112、126、127 を共有する 3 組が存在したまま、`mise run check-work-items` は緑だった。
  **例外は置かず、3 組を解消した。** 起票時の依頼では既知の例外として登録する予定だったが、例外は報告されない衝突を恒久的に残し、同じ番号を引いた将来の起票まで見逃す恐れがある。
  各組のうち git に後から現れた側を、`wi-3`、`wi-42`、`wi-245` へ移した。
  **移す先を 5 桁にしなかった理由は、スキーマが番号を時期の代理として読むからである。** 作業項目スキーマは 410 以上に `evidence_policy` を、452 以上に `documentation_impact` を要求する。
  2026 年 7 月に完了した 3 件を 5 桁へ移した時点で `mise run check-work-items` は 18 件の schema 違反を報告した。
  番号がその代理として働き続ける 410 未満の空き番号のうち、`git log --diff-filter=A --all` が一度も記録していない 3、42、245 を選んだ。
  同じ条件の空き番号 269 は過去に `wi-269-audit-events-api-500-fix.md` として存在した形跡があり、外れた。
  **参照の張り替えは機械的には終わらない。** 完全な id を含む 18 ファイルは置換で済んだが、`wi-126` のような番号だけの言及は、どちらの記録を指すかが文脈にしかない。
  Go と TypeScript のコメント 5 箇所を含む 30 箇所あまりを読み分け、`ADR-084`（`wi-127-postgres-column-type-policy`）や ADR-086（`wi-126-admin-and-account-ui-consistency-and-navigation-policy`）のように番号を保つ側を指すものはそのまま残した。
  **空き枠の警告は検査を落とさない。** `CheckOutcome` には警告の経路がないため、`checkWorkItems` の `ok` の根拠を所見の件数へ移し、警告は所見の有無にかかわらず出力へ並べた。
- **Acceptance RED Evidence**:
  - **Test**: `mise run test-tools-file -- check/src/check-work-items.test.ts` の `checkWorkItems > 題名が違っても識別番号が同じ二つの記録を落とす`
  - **Requirement**: N/A: 起票の採番規則と検査であり、製品の規範要求ではない。
  - **Observed Failure**: `expect(outcome.ok).toBe(false)` が `Received: true` で失敗した。番号 40318 を共有する 2 件を置いた workspace で、登録済みの `work-items` 検査が所見を一件も返さなかった。
  - **Detection Reason**: 同じ workspace で番号だけを 40318 と 40319 に分けた対照検査は緑のまま通る。所見の有無を分けているのが番号の一致だけであることを、この 2 件が示す。スキーマ違反で `ok` が落ちる経路を避けるため、fixture はスキーマを満たす最小の記録にした。
- **Unit RED Evidence**:
  - **Test**: `mise run test-tools-file -- check/src/work-item-dependencies.test.ts work-item-number/src/work-item-number.test.ts`
  - **Requirement**: N/A: 同上。
  - **Observed Failure**: `SyntaxError: Export named 'verifyWorkItemIdentifiers' not found in module` と `error: Cannot find module './work-item-number.ts'`。
  - **Detection Reason**: 衝突の報告、空き枠の境目、範囲外の番号の除外、使用済みの値の除外を、それぞれ別の表明が固定する。縮めた候補範囲を渡せるので、採番が使用済みの値を返さないことを 5 個の枠で確かめられる。
- **Change-Resistance Results**:
  `test-go-mutation` は対象外である。変更した論理はすべて TypeScript にあり、Go の差分はコメントだけである。
  変異器が表現できない 3 つの障害を手で注入した。

  | 注入した障害 | 結果 |
  | --- | --- |
  | `checkWorkItems` から `identifiers.findings` の連結を外す | `checkWorkItems > 題名が違っても識別番号が同じ二つの記録を落とす` が失敗した |
  | `pickWorkItemNumber` から `used.has(value)` の読み飛ばしを外す | `pickWorkItemNumber` の 2 件が失敗し、使用済みの値 11 と 13 を返した |
  | 空き枠の計数から `IDENTIFIER_RANGE` の範囲判定を外す | **最初は生き残った。** 634 件の 3 桁を数えても空き枠は 89366 で、閾値 10000 を割らないためである。空き枠がちょうど閾値の状態へ範囲外の記録を足す形へ検査を書き直したところ、`5 桁の外にある番号は空き枠を減らさない` が失敗するようになった |

  生き残りが一つ出た事実そのものが、検査の名前と表明がずれていた証拠である。
  検査は範囲判定の除去を名前で主張しながら、実際には観測できる位置に立っていなかった。
- **Verification Results**:
  - `mise run verify` - 成功
  - `mise run check-work-items` - 成功
  - `mise run check-links` - 成功（883 文書）
  - `mise run test-tools` - 成功（587 件）
