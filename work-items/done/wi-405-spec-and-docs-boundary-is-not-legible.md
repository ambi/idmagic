---
depends_on: []
status: completed
authors: [tn]
initial_context:
  specification: [docs/README.md, docs/structure.md]
  source: [DOCUMENTATION_GUIDE.md, SPECIFICATION_FORMAT.md, DEVELOPMENT.md, tools/check/src/specification-doc.ts, mise.toml]
  tests: [tools/check/src/specification-doc.test.ts]
  stop_before_reading: [backend, frontend, work-items/done]
risk: medium
created_at: 2026-08-23
priority: p2
change_kind: docs
spec_impact: { kind: none, reason: "文書の配置を変える変更である。正規文書 134 件を spec/ から docs/ へ移すが、規範的な要素そのもの（REQ-* のシナリオ、規範 ID、TypeSpec symbol）は 1 件も追加も削除も変更もしない。affected_spec が索引するのは要素であって置き場所ではないため、指すべき差分が無い。移動の事実は spec-diff が no normative specification change と報告することで裏づけられる。" }
---

# `spec/` と `docs/` の境界を、名前から読める形にする

## Motivation

[DOCUMENTATION_GUIDE.md](../../DOCUMENTATION_GUIDE.md) §4 は文書の置き場所を 3 つに分ける。`spec/`（境界の宣言、用語、規範、状態遷移、判断、機構、シナリオ、TypeSpec）、`docs/`（product-overview、build、ci、testing）、`operations/`（SLO、リリースと後退、バックアップ、Runbook）である。

**区別の基準そのものは筋が通っている。** §5.9 がそれを書いている。

> 判定は「**それが変わったとき、外部から観測できる振る舞いか、守るべき境界が変わるか**」である。

`spec/` は Yes、`docs/` は No。原理としては明確である。

**問題は名前がその基準を裏切っていることである。** `docs` は「文書すべて」と読める語だが、`spec/` の中身の大半——`decisions.md`、`internals.md`、`scenarios.md`、`glossary.md`——は、読み手の素朴な感覚ではどう見ても「文書」である。ディレクトリ名を見た人が `decisions.md` を `docs/` に置こうとするのは誤読ではなく、名前がそう誘導している。§5.9 の基準へ辿り着いて初めて、そうではないと分かる。**基準を読まないと使えない構成は、構成として弱い。**

さらに悪いことに、**ガイド自身の参照実装であるこのリポジトリに `docs/` も `operations/` も無い。**

| ガイドが置く場所 | このリポジトリの実態 |
|---|---|
| `docs/build.md`、`docs/ci.md`、`docs/testing.md` | `DEVELOPMENT.md`（ルート）、`README.md` の「主なコマンド」、`mise.toml` |
| `docs/product-overview.md` | `README.md`（[[wi-404-repository-entrance-documents-are-missing]] で「対象外」節を追加） |
| `operations/runbooks/` | `infra/runbooks/` |
| `operations/backup-and-recovery.md` | `infra/runbooks/backup-restore-dr.md` |
| `operations/reliability.md` | 無い。サービス目標は `spec/capacity.md` にある（[[wi-400-service-objectives-need-stable-ids]]） |

実態は偶然の寄せ集めではなく、**一貫した別の規則**になっている。**手順は、それが動かすものの隣に置く。** Kubernetes と監視の手順は `infra/README.md`、UI の指針は `frontend/README.md`、スキーマ運用は `infra/schema/README.md`、障害手順は `infra/runbooks/`。この規則は `docs/` に集める案より優れている点がある。手順を読む人はたいてい対象のディレクトリを開いており、そこに手順があれば探さずに済む。

[[wi-404-repository-entrance-documents-are-missing]] の Design 1 は、対象外 3 行を置くためにこの未決着へ突き当たり、「このリポジトリの実態に合わせる」で回避した。**回避であって解決ではない。** 次に同じ種類の文書を足す人が、同じところで同じ時間を使う。

## Scope

- §4 のディレクトリ構成と、§10（開発文書）・§11（運用文書）を、**仕様と手順の区別が名前から読める形**に改める。
- §5.9 の判定基準を、仕様に入れないものの節だけでなく、構成を示す節からも参照できるようにする。基準を読まないと構成が使えない状態を解消する。
- 「手順は、それが動かすものの隣に置く」を、案として書くか規則として書くかを決めて反映する。
- `operations/` の扱いを同時に決める。Runbook をインフラ資材の隣に置く構成をガイドが許すかどうかは、同じ問いである。
- §13 の参考と §15 の導入順序に、構成の変更が波及していないか確認する。
- **決めた配置へリポジトリを実際に移す**（着手時に Scope へ追加。Design の「Scope を広げた」を参照）。正規文書 134 件と runbook 3 件の移動、検査ツールと生成ツールの経路の付け替え、294 箇所の参照の張り替えを含む。

## Out of Scope

- `SPECIFICATION_FORMAT.md` の `ROOT_DOCUMENTS` / `CONTEXT_DOCUMENTS` **が何を含むか**。集合を適用する場所を `spec/` から `docs/` へ移すだけで、文書の種類を増減する判断は別の問題である。

  例外が 1 件ある。**`product-overview.md` を `ROOT_DOCUMENTS` に足した。** ガイド §4 と §9.2 が `docs/product-overview.md` を挙げるのに集合へ入っていないと、その名前で作った瞬間に検査が「正規文書ではない」と拒否する。ガイドと検査が食い違う状態は本 work item が診断した病そのものなので、ここだけは直した。
- 方法論文書（`DEVELOPMENT.md`、`SPECIFICATION_FORMAT.md`、`WORK_ITEM_FORMAT.md`、`DOCUMENTATION_GUIDE.md`）自身の置き場所。別リポジトリへ抽出する想定があるため、ルートに残す。`docs/development/` へ入れるかどうかはその抽出と一緒に決まる。
- `docs/development/` と `docs/operations/` の中身を書くこと。ガイドが場所を定めるだけで、`build.md` や `reliability.md` を新しく書き起こしはしない。該当する内容が無いファイルは作らない（§3）。
- `docs/scenarios.md` の新設（[[wi-401-cross-context-scenarios-have-no-home]]）とサービス目標の正本化（[[wi-400-service-objectives-need-stable-ids]]）。どちらも文書の中身の話であり、配置とは独立である。
- 完了済み work item の散文に残る歴史的なパス（`spec/scl.yaml` など）。当時そこにあった記録であり、書き換えれば履歴の改竄になる。解決を要求される `affected_spec` だけを張り替えた。

## Design

3 点とも着手時に確定した。**採ったのは起票時に挙げた 3 案のどれでもない、(a) と (b) を組み合わせた 4 案目である。**

1. **`docs/` を人が書く文書の単一ルートにし、`spec/` は機械が食う契約だけを持つ。**

   起票時の (a)「すべて `docs/` に寄せる」への私の反論は「分割の基準ごと失う」だったが、これは **(a) をサブディレクトリ無しの平坦な案として読んだときにだけ成り立つ。** `docs/` の中で手順を `development/` と `operations/` に分ければ、基準は残り、しかも名前が基準を言う。反論は案ではなく私の読みに当たっていた。

   同時に (b) の「`spec/` を TypeSpec だけにする」も採った。**`.tsp` は文書ではなくコンパイラの入力なので、`docs/` の下に置くと `decisions.md` を `docs` と呼ぶより深く名前を裏切る。**

   | 決めたこと | 置き場所 |
   |---|---|
   | 現在の仕様と設計（正規文書 134 件） | `docs/` 直下と `docs/contexts/<context>/` |
   | 手順 | `docs/development/`、`docs/operations/` |
   | TypeSpec、`tspconfig.yaml`、OpenAPI ベースライン | `spec/` |
   | 生成物 | 追跡しない `spec/generated/` |

   (b) 単独の弱点として挙げた「言語非依存の正本という位置づけを失う」は、**成り立たないと判断した。** その性質は正本が単一で言語に依存しないことであって、`.tsp` と同じディレクトリにあることではない。2 つの木が `contexts/<context>/` で対応していれば、1 つの Context の仕様は 1 つの名前を 2 か所で引くだけである。DEVELOPMENT.md §5 の該当文はそう言い直した。

   得たものは 1 つ大きい。**生成物が `spec/generated/` に集まるので、「`docs/` の下にあるものはすべて人が書いたものである」と言い切れる。** これは移動前には言えなかった。

2. **runbook は `docs/operations/runbooks/` へ移し、読み手で分ける規則を対象で分ける規則より優先する。** 競合の解き方はガイド §11 に理由ごと書いた。**当番担当者は「どのディレクトリの資材の話か」を知らない状態で呼び出される。** 資材の隣は変更する人にとって近く、障害の最中に探す人にとっては遠い。ただしこれは手順一般の規則ではないので、そう限定した。あるコンポーネントを動かすための README はそのコンポーネントの隣でよい（`infra/README.md`、`frontend/README.md` は動かしていない）。

3. **リポジトリ非依存は保てた。** 起票時に懸念した「実態を一般的な規則として言い直す」必要は、実際には逆向きに解決した。ガイドを直した結果がこのリポジトリの実態になったので、言い直す対象が消えた。ガイドには「深さは重要度の裏返しにする」「読み始める時点で原因が分かっていないから runbook だけは読み手で分ける」という、どのプロジェクトでも判定できる形の理由だけを書いた。

### Scope を広げた

起票時の Out of Scope は「このリポジトリのファイルの移動」を除外していた。**着手時の指示でこれを取り消し、移動まで本 work item に含めた。** 分けるとガイドが実態に無い配置を勧める期間が残り、それは本 work item が診断した病そのものだからである。

## Tasks

- [x] T001 [Docs] §4、§10、§11 で `docs/` と `operations/` を前提にしている記述を列挙する。
- [x] T002 [Design] 採る案、`operations/` の扱い、リポジトリ非依存との両立を確定し `## Design` に記録する。
- [x] T003 [Docs] §4 の構成を改め、§5.9 の判定基準を構成の節から参照できるようにする。「深さは重要度の裏返しにする」「境界の判定」「生成物」の 3 つの小節を足した。
- [x] T004 [Docs] §10 と §11 を 2 の判断に合わせる。§11 には runbook だけ読み手で分ける理由を書いた。
- [x] T005 [Docs] §13 の参考と §15 の導入順序に波及がないか確認し、あれば直す。波及なし（どちらも配置に依存しない）。
- [x] T006 [Migrate] 正規文書 134 件を `docs/` へ、runbook 3 件を `docs/operations/runbooks/` へ移す。
- [x] T007 [Tooling] `specification-doc.ts`、`workspace.ts`、`render-spec-docs`、`security-controls.ts`、`report-refusal-debt.ts`、`work-item.schema.json`、`mise.toml` の経路を付け替える。
- [x] T008 [Docs] `SPECIFICATION_FORMAT.md` §1、`DEVELOPMENT.md` §2/§5、`AGENTS.md` の言語表、`CONTRIBUTING.md` を追随させる。
- [x] T009 [Verify] `mise run verify` を通し、ガイド内の相互参照とリンクが解決することを確認する。

## Verification

- `mise run verify`
- 手動: `DOCUMENTATION_GUIDE.md` 内の節番号への参照（§5.9、§9.2、§11.1 など）が、構成を変えた後もすべて実在する節を指していることを確認する。**節を動かすと、この種の参照が静かに壊れる。**
- 手動: 直した §4 だけを読んで、`decisions.md` と `build.md` をそれぞれどこへ置くかが判断できるかを確かめる。§5.9 を読まないと判断できないなら、この変更は目的を達していない。
- 手動: このリポジトリの実際の配置（`DEVELOPMENT.md`、`docs/operations/runbooks/`、`frontend/README.md`）が、直した後のガイドに照らして違反でないことを確認する。違反になるなら、ガイドと参照実装のどちらを直すかを Design へ追記する。

## Risk Notes

リスクは medium。**134 件の文書を移し、検査ツールと生成ツールの経路を付け替え、294 箇所の参照を張り替えるためである。** 製品のコードには触れないが、仕様を守っている仕組みそのものに触れる。

**失敗の形は、診断を取り違えて基準のほうを緩めることである。** 「`spec/` と `docs/` が分かりにくい」という訴えに対して最も安易な応答は 1 案 (a)——1 か所にまとめる——だが、それは分かりにくさの原因だった**基準ごと捨てる**ことになる。正規文書の集合が閉じていて検査が守っている性質は、このリポジトリで実際に機能している数少ない仕組みの 1 つである。名前の問題を構造の変更で解こうとしない。

もう 1 つは、**ガイドだけを直してリポジトリを放置し、両者が食い違ったままにすること**である。Verification の最後の手動確認は、そのための歯止めである。食い違うなら、どちらを直すかをこの work item の中で決めて記録する。決めずに終えると、次に読む人は「ガイドと実態が違う」という同じ発見を最初からやり直す。

## Completion

- **Completed At**: 2026-08-23
- **Summary**:
  人が書く文書を 1 つの根へ集めた。`docs/` が正規文書 134 件（直下 10 件と `contexts/<context>/` 124 件）と `operations/runbooks/` を持ち、`spec/` は TypeSpec 43 件・`tspconfig.yaml`・OpenAPI ベースラインだけになった。2 つの木は `contexts/<context>/` で対応する。DOCUMENTATION_GUIDE §4 は構成図を差し替えたうえで、判定の理由——深さは重要度の裏返しにする、境界の判定は §5.9、`docs/` 直下に生成物が混ざらない——を 3 つの小節として明示した。§10 と §11 には配置を書き、§11 には runbook だけが読み手で分かれる理由（当番担当者は原因が分からない状態で呼び出される）を、手順一般の規則ではないと限定したうえで書いた。`ROOT_DOCUMENTS` には `product-overview.md` を足し、ガイドが挙げる名前を検査が拒否する食い違いを解消した。検査・生成・スキーマ・タスクの経路と、294 箇所の参照を張り替えた。規範的な要素は 1 件も動いていない。
- **Verification Results**:
  - `mise run verify` - passed（exit 0）。ただしこの移動は `check-security-controls` の R4 も壊しており、`.tsp` を `docs/contexts/` から読んで**契約が約束する 403 を 1 件も検査していなかった**。`verify` は成功を報告し続けた。[[wi-401-cross-context-scenarios-have-no-home]] で修正した。
  - `mise run spec-render` - 137 document(s), 330 operation(s), 19 API tag(s), 836 TypeSpec symbol(s)
  - `mise run spec-diff` - no normative specification change against main - **無効（後日訂正）**。この移動で `spec-diff` 自身が壊れており、`git ls-tree -- spec` と `walk(spec/)` しか見ていないため散文を 1 件も読んでいなかった。常に「変更なし」を返す状態だった。主張（規範要素は動いていない）は結果的に正しいが、根拠は空である。[[wi-401-cross-context-scenarios-have-no-home]] で両方の木を読むよう修正した。
  - `mise run test-tools` - 167 pass / 0 fail
  - 手動: 全 Markdown の相対リンクを全件解決確認 - **無効（後日訂正）**。使った `fd` の呼び出しが 0 件を返しており、1 つも検査していなかった。[[wi-406-operations-holds-only-runbooks]] で検査を作り直したところ、本 work item が壊した実リンクが 2 件あった（`infra/backup/README.md` と、`done/` へ移した本ファイル自身の `../DOCUMENTATION_GUIDE.md`）。いずれも wi-406 で修正した。
  - 手動: ガイド内の `§N.M` 参照が実在する節を指すことを全件確認 - passed
  - 手動: `docs/product-overview.md` を一時的に置いて検査が受理することを確認 - passed
  - 手動: このリポジトリの実配置がガイドに照らして違反でないことを確認 - passed

## Left Undone

- **`docs/development/` は作っていない。** ガイドが場所を定めただけで、`build.md` / `ci.md` / `testing.md` に相当する内容は現在 `DEVELOPMENT.md`（ルート）と `README.md` の「主なコマンド」にある。§3 の「必要が生じていない文書を作らない」に従い、空のディレクトリを置かなかった。方法論文書を別リポジトリへ抽出するときに、ルートに残すか `docs/development/` へ入れるかを決める。
- **`docs/operations/` 直下は runbooks だけである。** `reliability.md` は [[wi-400-service-objectives-need-stable-ids]] が扱う。その work item の Design 1 が「`ROOT_DOCUMENTS` に `reliability.md` を足すか、`capacity.md` に ID を足すか」を問うているが、**本 work item で `ROOT_DOCUMENTS` の適用先が `docs/` へ移り、`product-overview.md` を足す前例もできたので、判断の材料は変わっている。** wi-400 に着手するときはこの completion を読むこと。
- **sd による一括置換で 4 箇所のテンプレートリテラルを壊した**（`workspace.ts`、`security-controls.ts`、`render-spec-docs/src/main.ts` の 2 箇所）。うち 1 箇所は検査を通したまま**文書を 137 件ではなく 3 件しか描画しない**状態で成功していた。件数を読まなければ気付かなかった。**生成物の件数は、通ったかどうかとは別に必ず確認する。**
