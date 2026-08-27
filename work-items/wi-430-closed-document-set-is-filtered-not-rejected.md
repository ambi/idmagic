---
depends_on: []
status: pending
authors: [tn]
risk: medium
created_at: 2026-08-27
priority: p1
change_kind: tooling
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

## Design

検査の位置には 2 案がある。`scanCanonicalDocuments` を絞り込みから拒否へ変える案と、`check-spec` 側に「許可リストに無い Markdown が無いこと」を確かめる別の規則を足す案である。

採るのは後者である。`scanCanonicalDocuments` の役割は「仕様の対象を集める」ことで、集める関数が同時に拒否も担うと、呼び出し側すべてが拒否の副作用を持つことになる。`spec-diff` や `render-spec-docs` が同じ関数を使っているため、差分を取るだけの操作が未登録ファイルを理由に落ちるのは筋が悪い。検査は検査として独立させる。

例外を許すかどうかは、実際に必要になるまで決めない。現時点で `docs/` 直下と各 Context 直下に未登録の Markdown が無ければ、例外の仕組みは持たない。持たせてから使われない仕組みは、次に必要になったときに誤って使われる。

失敗メッセージは、許可リストとの編集距離が近い名前を示す形にする。この欠陥の主な害が打ち間違いの見逃しである以上、「許可されていない」とだけ言うメッセージでは、書いた人が何を間違えたかに到達できない。

## Plan

1. 現在の作業ツリーに未登録の Markdown があるかを確認する。
2. 未登録の Markdown を置いた状態で `check-spec` が通ることを観測する。
3. 検査を実装し、近い名前の提示を含める。
4. `SPECIFICATION_FORMAT.md` の記述を実装と一致させる。

## Tasks

- [ ] T001 [Baseline] 現在の作業ツリーに未登録の Markdown があるかを確認する。
- [ ] T002 [Acceptance] 未登録かつ内容の不正な Markdown を置いても `check-spec` が通ることを観測する。
- [ ] T003 [Tooling] 許可リストに無い Markdown を拒否する検査を実装する。
- [ ] T004 [Tooling] 失敗メッセージに近い名前の候補を含める。
- [ ] T005 [Spec] `SPECIFICATION_FORMAT.md` の記述を実装と一致させる。
- [ ] T006 [Verify] 打ち間違えた名前（`decision.md`、`scenario.md`）で落ち、正しい名前で通ることを確認する。

## Verification

- `docs/` 直下と `docs/contexts/<context>/` の直下に許可リストに無い Markdown を置くと `mise run check-spec` が落ちる。
- 打ち間違えた名前に対して、近い許可名が失敗メッセージに現れる。
- 現在の作業ツリーで `mise run check-spec` が通り続ける。
- `docs/development/` と `docs/runbooks/` の自由な命名が引き続き許される。
- `mise run verify`

## Risk Notes

拒否を入れると、これまで黙って無視されていたファイルが表に出る。移行の途中や下書きとして `docs/` に置かれた Markdown があれば、それが検査を落とす。T001 で先に確認し、登録するか `docs/development/` へ移すかを決める。

`scanCanonicalDocuments` を変えずに検査を足す方針のため、集める側と拒否する側で許可リストの解釈が分かれる余地が残る。同じ定数（`ROOT_DOCUMENTS`、`CONTEXT_DOCUMENTS`）を両方が参照することで、分岐を防ぐ。
