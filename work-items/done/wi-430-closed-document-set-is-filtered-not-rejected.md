---
depends_on: []
status: completed
authors: [tn]
risk: medium
created_at: 2026-08-27
priority: p1
change_kind: tooling
evidence_policy: risk-based-v2
initial_context:
  specification: [SPECIFICATION_FORMAT.md]
  source:
    [
      tools/check/src/specification-doc.ts,
      tools/check/src/check-specifications.ts,
      tools/workspace/src/workspace.ts,
      tools/workspace/src/check-workspace.ts,
      tools/check/src/slo-references.ts,
    ]
  tests: [tools/workspace/src/workspace.test.ts]
  stop_before_reading: [backend, frontend, spec, docs/runbooks]
spec_impact: { kind: none, reason: "検査の追加と SPECIFICATION_FORMAT.md の記述の是正であり、製品の振る舞いと外部契約を変えない。" }
---

# 正本文書の閉じたファイル集合を、絞り込みではなく拒否として強制する

## Motivation

`SPECIFICATION_FORMAT.md` はこう述べる。

> The file set and the file names are *(checked)* for `docs/` and `docs/contexts/<context>/`; anything else at those two levels is not a canonical document and is rejected.

**拒否されない。** `tools/workspace/src/workspace.ts` の `scanCanonicalDocuments` は、許可リストに一致する名前だけを拾う絞り込みである。一致しないファイルは検証の対象から外れるだけで、何のエラーも出ない。関数のコメント自体が「an unrelated Markdown file next to them is not mistaken for specification source」と、絞り込みが意図であることを述べている。

結果として、`docs/` 直下または `docs/contexts/<context>/` に置いた未登録の Markdown は、次のすべてを満たしたまま存在できる。`mise run check-spec` は exit 0 を返す。ファイル名は出力に一度も現れない。生成される仕様サイトにも載らない。正本文書として不正な本文（H1 が 2 つある、遷移表の `From` が宣言されていない状態を指す、など）を持っていても、何も落ちない。

これは wi-424 の実装中に、Acceptance RED を仕掛けようとして発見した。二重の H1 を持つファイルを `docs/` に置いて `check-spec` が拒否することを期待したところ、exit 0 で通った。

**害は「間違ったファイルが通る」ことではなく、「正しいつもりのファイルが黙って無視される」ことである。** 名前を打ち間違えた正本文書（`decision.md`、`scenario.md`、`glossary.MD`）は、書いた人からは存在して見え、検査からは存在しない。書いた内容は誰にも検証されず、生成された仕様サイトにも現れない。気づく契機が無い。

これは `docs/standards.md` の「各行は、規範 ID をテスト名に含めた対応するテストを持つ」が現在偽であること（wi-418 が扱う）と同じ型の欠陥である。宣言された規則と、実際に強制されている規則が違う。

## Scope

- `docs/` 直下と `docs/contexts/<context>/` の直下にある、許可リストに無い Markdown を検出して落とす。
- 意図的な例外の扱いを決める。`docs/contexts/` 自体のような、ディレクトリと同居する非文書をどう区別するかを含む。
- 落とすときの失敗メッセージに、名前の打ち間違いを疑わせるだけの情報を持たせる。近い名前が許可リストにあるならそれを示す。
- `SPECIFICATION_FORMAT.md` の記述を、実装される挙動と一致させる。
- 既存の作業ツリーに未登録の Markdown が無いことを確認し、あれば登録するか移動するかを判断する。

## Out of Scope

- `docs/development/` と `docs/runbooks/` の中身の検査。それらは固定した種類の集合を持たないという既存の判断を変えない。
- Markdown 以外のファイル。図表や添付の扱いは本件では決めない。
- 正本文書の集合そのものの変更。
- 閉じた 2 段より外側の Markdown。`docs/contexts/` 直下に置いた Markdown や、`docs/` の下に新しく掘ったディレクトリの中身は、依然として黙って無視される。`SPECIFICATION_FORMAT.md` が閉じていると言っているのはこの 2 段であり、その言明と実装は一致した。3 段目をどうするかは別の判断であり、独立検証がこの穴を指摘している。

## Design

検査の位置には 2 案がある。文書を集める関数を絞り込みから拒否へ変える案と、`check-spec` 側に「許可リストに無い Markdown が無いこと」を確かめる別の規則を足す案である。

採るのは後者である。集める関数の役割は「仕様の対象を集める」ことで、集める関数が同時に拒否も担うと、呼び出し側すべてが拒否の副作用を持つことになる。`spec-diff` や `render-spec-docs` が同じ関数を使っているため、差分を取るだけの操作が未登録ファイルを理由に落ちるのは筋が悪い。検査は検査として独立させる。

例外を許すかどうかは、実際に必要になるまで決めない。現時点で `docs/` 直下と各 Context 直下に未登録の Markdown が無ければ、例外の仕組みは持たない。持たせてから使われない仕組みは、次に必要になったときに誤って使われる。

失敗メッセージは、許可リストとの編集距離が近い名前を示す形にする。この欠陥の主な害が打ち間違いの見逃しである以上、「許可されていない」とだけ言うメッセージでは、書いた人が何を間違えたかに到達できない。

実装での配置は次のとおりである。判定は `tools/check/src/canonical-document-set.ts` の `verifyCanonicalDocumentSet` が引き受ける。入力はディレクトリ名とその直下のファイル名の一覧（`DirectoryListing[]`）、出力は所見の一覧（`Finding[]`）で、ファイルシステムも終了コードも触らない純関数である。走査は `tools/workspace/src/workspace.ts` の `listCanonicalDirectories` が、終了コードは `check-workspace.ts --documents` が持つ。集める側の `discoverSpecificationDocuments` は同じ `listCanonicalDirectories` の結果を許可名で絞り込むだけになり、絞り込みという性格は変わっていない。

候補提示は、両辺を小文字にそろえて測った編集距離が 2 以下の許可名を示す。近い名前が無ければ候補を出さず、代わりにその段の許可名を並べる。距離の上限を持つのは、遠い名前に候補を出すと、そもそも正本文書を書いたつもりでない人へ誤った誘導になるからである。小文字にそろえてから測るので、`glossary.MD` のように大文字小文字だけが違う名前は距離 0 になり必ず候補に届く。拡張子の大文字は打ち間違いの中でも特に見つけにくいので、この扱いが要る。

## Plan

1. 現在の作業ツリーに未登録の Markdown があるかを確認する。
2. 未登録の Markdown を置いた状態で `check-spec` が通ることを観測する。
3. 検査を実装し、近い名前の提示を含める。
4. `SPECIFICATION_FORMAT.md` の記述を実装と一致させる。

観測する RED は次の 2 つである。**Acceptance RED** は `tools/workspace/src/check-workspace.test.ts` で、仮の作業ツリーに `docs/decision.md` を置いて `check-workspace.ts --documents` を起動し、終了コードが 0 でないことと、標準エラーにその名前が現れることを確かめる。これが観測できる最も狭い境界である。`mise run check-spec` はこの起動を含む 3 段のうちの 1 つで、TypeSpec の再コンパイルを伴うため、同じ性質をより高い費用で確かめることになる。**Unit RED** は `tools/check/src/canonical-document-set.test.ts` で、ディレクトリの一覧を入力に取る純関数が、許可名に無い Markdown 1 件に対して所見を 1 件返し、その所見が近い許可名を含むことを確かめる。

## Tasks

- [x] T001 [Baseline] 現在の作業ツリーに未登録の Markdown があるかを確認する。`docs/` 直下 12 件と `docs/contexts/<context>/` 21 ディレクトリの直下 124 件はすべて許可リスト内で、未登録の Markdown は無い。例外の仕組みは持たない。
- [x] T002 [Acceptance] 未登録かつ内容の不正な Markdown を置いても `check-spec` が通ることを観測する。二重 H1 を持つ `docs/decision.md` を置いて `mise run check-spec` が exit 0 を返し、出力にその名前が 1 度も現れないことを確認した。
- [x] T003 [Acceptance RED] `check-workspace.ts --documents` が未登録の Markdown を拒否することを確かめるテストを書き、失敗を観測する。`tools/workspace/src/check-workspace.test.ts` の 4 件のうち拒否を見る 2 件が落ち、受け入れを見る 2 件は通った。
- [x] T004 [Unit RED] 閉じた集合を判定する純関数のテストを書き、失敗を観測する。`tools/check/src/canonical-document-set.test.ts` は所見を返さない骨格に対して 8 件中 6 件が落ちた。
- [x] T005 [Tooling] 許可リストに無い Markdown を拒否する検査を実装し、失敗メッセージに近い名前の候補を含める。`tools/check/src/canonical-document-set.ts` に純関数を置き、`check-workspace.ts --documents` から呼ぶ。
- [x] T006 [Spec] `SPECIFICATION_FORMAT.md` の記述を実装と一致させる。
- [x] T007 [Verify] 打ち間違えた名前で落ち、正しい名前で通ることを確認する。実際の作業ツリーで `docs/scenario.md` と `docs/contexts/system/decision.md` がそれぞれ `scenarios.md`、`decisions.md` を候補として示して落ち、取り除くと `mise run check-spec` は exit 0 に戻った。`glossary.MD` は macOS のファイルシステムが大文字小文字を区別しないため実ツリーでは再現できず、純関数の単体テストで確かめている。

## Verification

- `docs/` 直下と `docs/contexts/<context>/` の直下に許可リストに無い Markdown を置くと `mise run check-spec` が落ちる。
- 打ち間違えた名前が、その段の許可名に近いなら、候補として失敗メッセージに現れる。近い名前が無いときは、代わりにその段の許可名が並ぶ。
- 現在の作業ツリーで `mise run check-spec` が通り続ける。
- `docs/development/` と `docs/runbooks/` の自由な命名が引き続き許される。
- `mise run verify`

## Risk Notes

拒否を入れると、これまで黙って無視されていたファイルが表に出る。移行の途中や下書きとして `docs/` に置かれた Markdown があれば、それが検査を落とす。T001 で先に確認し、登録するか `docs/development/` へ移すかを決める。

集める関数を変えずに検査を足す方針のため、集める側と拒否する側で許可リストの解釈が分かれる余地が残る。同じ定数（`ROOT_DOCUMENTS`、`CONTEXT_DOCUMENTS`）を両方が参照することで、分岐を防ぐ。実装では、許可リストだけでなくディレクトリの一覧そのものを `listCanonicalDirectories` 1 つに集約した。どの段を閉じた集合とみなすかで両者が食い違う余地も、これで消える。

大文字小文字を区別しないファイルシステム（macOS の既定）では、`glossary.MD` のような拡張子だけが違う名前は別のファイルとして存在できない。この検査が拾えるのは、区別するファイルシステム上（CI の Linux を含む）で作られた名前である。拡張子の照合を大文字小文字に鈍感にしてあるのは、そこで作られた名前を手元でも読めるようにするためである。

## Completion

- **Completed At**: 2026-08-28
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返す。規範仕様は動いていない。変わったのは検査の強制力である。`docs/` 直下と `docs/contexts/<context>/` 直下に置かれた Markdown のうち、配置が定めない名前を持つものが `mise run check-spec` を落とすようになった。それまでは、そのファイルは検証の対象から外れるだけで、終了コードにも出力にも現れなかった。落とすときは、両辺を小文字にそろえて測った編集距離が 2 以下の許可名を候補として示し、近い名前が無ければその段の許可名を並べる。`SPECIFICATION_FORMAT.md` の宣言も 2 点で実装に合わせた。「file set が checked」は必須ファイルの存在検査が元から無いため偽であり、`file names` に改めた。候補提示についても、無条件に最も近い文書を示すという書き方をやめ、近い名前があるときとないときを書き分けた。判定は `verifyCanonicalDocumentSet` という純関数が持ち、走査は `listCanonicalDirectories` が、終了コードは `check-workspace.ts --documents` が持つ。集める側の `discoverSpecificationDocuments` は同じ走査結果を絞り込むだけで、絞り込みという性格は変わっていない。
- **Acceptance RED Evidence**:
  - **Test**: `tools/workspace/src/check-workspace.test.ts` の `check-workspace --documents > rejects a Markdown file the closed set does not name` と `> names the canonical document a misspelled file was meant to be`
  - **Requirement**: N/A: リポジトリの検査ツールであり、対応する規範的な製品要件を持たない。代わりに失敗したのは、`SPECIFICATION_FORMAT.md` が *(checked)* と宣言している「`docs/` と `docs/contexts/<context>/` の名前は検査される」という言明である。
  - **Observed Failure**: 両方とも `expect(result.code).not.toBe(0)` が `Expected: not 0` で失敗。`2 pass 2 fail`。同じ性質を実ツリーでも観測した。二重 H1 を持つ `docs/decision.md` を置いた状態で `mise run check-spec` は exit 0 を返し、`decision.md` は出力に 1 度も現れなかった。
  - **Detection Reason**: この 2 件は、`mise run check-spec` の利用者が見るのと同じ境界、すなわち `check-workspace.ts --documents` の起動の終了コードと標準エラーを見ている。置くファイルの本文は H1 が 2 つある不正な本文なので、通ってしまうということは、そのファイルが内容検査にすら届いていないということである。同じ 4 件のうち受け入れを見る 2 件は RED の時点でも通っており、失敗が拒否の欠如だけに由来することを分けている。
- **Unit RED Evidence**:
  - **Test**: `tools/check/src/canonical-document-set.test.ts` の `verifyCanonicalDocumentSet` 一式を、所見を返さない骨格実装に対して実行した。
  - **Requirement**: N/A: 上と同じ理由で、対応する規範的な製品要件を持たない。
  - **Observed Failure**: `2 pass 6 fail`。拒否と候補提示を見る 6 件が落ち、許可名だけを置いた受け入れの 2 件が通った。
  - **Detection Reason**: 判定はファイルシステムにも終了コードにも触れない純関数なので、この 6 件が見ているのは「どの名前を許すか」と「何を候補として示すか」だけである。受け入れの 2 件が同時に通ることで、単に何でも拒否する実装は区別される。候補提示の主張は `did you mean X?` という文言まで含めて書いてある。候補が無いときの文言はその段の許可名を全部並べるので、許可名が含まれることだけを見る主張では、候補提示を丸ごと削った実装も通ってしまう。
- **Independent Verification**:
  実装していない新しい文脈のエージェントが、仕様レビューと規約レビューを分けて実施した。Scope の 5 項目がすべて実装され、Out of Scope の 3 項目が混入していないこと、実ツリー全体が通ることを確認したうえで、欠陥を 7 件報告した。うち 6 件を修正した。最も重いのは、単体テストが候補提示ロジックを一切識別していなかったという指摘である。候補が無いときの文言が許可名を全列挙するため、許可名の部分一致を見る主張が候補提示なしでも成立していた。検証者は候補提示を全廃する変異が単体 8 件を全部生存することを実測して示した。次に、`nearestName` の大文字小文字分岐が到達不能な死にコードであり、その根拠コメントが事実に反するという指摘。距離を両辺小文字で測るため大文字小文字だけが違う名前は距離 0 になり、必ずループ側で選ばれる。実測で確認して分岐を削除した。ほかに、`SPECIFICATION_FORMAT.md` が候補提示について実装より強い無条件の約束をしていたこと、閉じた集合の検査が `config.documents.length > 0` の内側にあり、名前を全部打ち間違えた作業ツリーでは検査ごと飛ぶこと、T001 の記録が 129 件と誤っていたこと (正しくは 124 件)、`workspace.ts` へ新規追加した doc コメントが言語表に反して英語だったことを修正した。修正しなかったのは、閉じた 2 段より外側の Markdown が依然として黙殺されるという指摘で、Out of Scope に理由を記録した。同じ検証者が修正後に再検証し、6 件すべてが閉じたこと、修正が退行を持ち込んでいないことを確認した。再検証はさらに 1 件を見つけた。候補提示の距離を測る前の小文字化のうち、入力側だけがテストで固定されておらず、テストのコメントが「両辺」と書いていて実際の識別力を上回っていた。全部が大文字の名前を 1 件足して固定した。`/^  - ALT/` を `/^ {2}- ALT/` に変えた正規表現についても、6 種の入力でマッチ位置とキャプチャが完全に一致することを確認した。
- **Change-Resistance Results**:
  代表的な誤実装を 8 つ入れ、いずれも検出されることを実測した。両段の許可名を合併する / 拡張子の照合を大文字小文字に敏感にする / 候補提示を全廃する (`nearestName` が常に `undefined`) / 候補提示の距離上限を 2 から 3 へ 1 だけ動かす / 同じく 8 へ広げる / 距離上限を 99 にする / 距離の測定で候補側の小文字化を落とす / 同じく入力側の小文字化を落とす。最初の版では、候補提示に関わる 3 つ (全廃・上限 3・上限 8) が単体テストを生存していた。独立検証がこれを指摘したのを受けて、主張を `did you mean X?` の全文に変え、距離 2 と距離 3 の境界と `readme.md` → `README.md` を足した。再検証がさらに入力側の小文字化の未固定を見つけたので、`GLOSSARY.MD` → `glossary.md` を足した。結果として 8 つすべてが単体テストだけで落ちる。
- **Verification Results**:
  - `mise run verify` - passed
  - `mise run spec-diff` - `no normative specification change against main`
