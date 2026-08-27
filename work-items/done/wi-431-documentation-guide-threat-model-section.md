---
depends_on: []
status: completed
authors: [tn]
risk: low
created_at: 2026-08-27
priority: p2
change_kind: docs
spec_impact: { kind: none, reason: "文書ガイドに新しい文書種の節を足すもので、製品の振る舞い、外部契約、規範シナリオ、規範 ID、TypeSpec シンボルを変えない。" }
evidence_policy: risk-based-v2
initial_context:
  specification:
    - docs/threat-model.md
    - docs/README.md
  source:
    - DOCUMENTATION_GUIDE.md
    - SPECIFICATION_FORMAT.md
  tests: []
  stop_before_reading:
    - backend
    - frontend
    - spec
---

# 文書ガイドに脅威モデルの節を足し、信頼境界の所有者を一致させる

## Motivation

wi-424 が `docs/threat-model.md` を正本文書として新設し、`SPECIFICATION_FORMAT.md` に §8 としてその形式を定めた。`DOCUMENTATION_GUIDE.md` は追随していない。同ガイドの `docs/` 直下のツリーにも、全体の仕様を扱う §4 の各節にも、脅威モデルは現れない。2 つの方法論文書が、正本文書の集合について違うことを述べている。

wi-424 の独立検証はこれを指摘したが、承認範囲外であることと、同ガイドが既にこのリポジトリに存在しない `reliability.md` と `recovery.md` を含み完全一致を意図していないことを理由に、そのときは見送った。

しかし脅威モデルはリポジトリ固有の都合ではなく、文書体系一般の概念である。`DOCUMENTATION_GUIDE.md` がリポジトリに依存しない一般論を扱う文書である以上、本来この平面が扱う対象にあたる。

ガイドの側には既に断片がある。§8 の追跡可能性の節は「シナリオ ID、規範 ID、**脅威 ID**」を並べてテスト名への記載を求めており、§11 の参考表は §4.5 の出典として OWASP Threat Modeling を挙げている。脅威 ID を前提とした記述と出典だけが先にあって、それを持つ文書の説明が無い。

あわせて所有者の食い違いも直す。ガイドの §4.5 は `deployment.md` が信頼境界を持つとしているが、wi-424 は所有を `docs/threat-model.md` へ移した。ここを揃えないと、ガイドと `SPECIFICATION_FORMAT.md` を読み比べた人が、どちらに書けばよいか判断できない。

## Scope

- §2 のディレクトリツリーへ `threat-model.md` を加える。
- §4 に脅威モデルの節を新設し、以降の節を繰り下げる。何を書き何を書かないか、状態の語彙、欠落の扱い、更新の契機を、ガイドの既存の節と同じ密度で書く。
- §4.5 の題と本文から信頼境界を外し、新しい節へ移す。`deployment.md` には実行単位、配備の構成、共有状態、可用性が残る。
- §11 の参考表を、新しい節番号に合わせて更新する。
- §12 の導入順序へ脅威モデルを置く。

## Out of Scope

- `docs/threat-model.md` の内容の変更。wi-424 が完了させている。
- ガイドのツリーに残る `reliability.md` と `recovery.md` の扱い。このリポジトリに存在しないが、ガイドは一般論を書く文書であり、一致させる義務がない。両者の関係をどう扱うかは別の判断である。
- `SPECIFICATION_FORMAT.md` §8 の変更。

## Design

節の位置は `authorization.md` の直後、`scenarios.md` の直前とする。認可の境界を読んだ人が次に問うのは「その境界を越えようとするのは何か」であり、隣接させると読み順が自然になる。ツリーの並びとも一致する。

信頼境界を `deployment.md` から移すことには反対の選択肢もある。境界は配備の形から決まるので配備の文書に置く、という考え方である。採らないのは、境界の価値が**そこで何を信用しないか**にあり、それは脅威の側から書いたときにだけ具体になるからである。配備の文書に置くと、境界は図の線として書かれて、信用しないものの列挙が抜け落ちる。ガイドの §4.5 が「信用しないものが書かれていない境界は線でしかない」と既に警告しているのは、その失敗が起きやすいことを知っているからである。

ガイドは一般論を書く文書なので、このリポジトリの ID 体系（`THREAT-NNN`）や件数を書かない。書くのは、安定した ID を持つこと、分類を ID に埋めないこと、状態の語彙を閉じた集合にすること、欠落を同じ表に残すことである。

## Plan

1. §4.5 から信頼境界の記述を抜き、新しい節の骨格へ移す。
2. 新しい節を書く。§4.8 の認可の節と同じ密度に揃える。
3. 節番号の繰り下げと、参考表と導入順序の更新を行う。

## Tasks

- [x] T001 [Spec] §2 のツリーへ `threat-model.md` を加える。
- [x] T002 [Spec] §4 へ脅威モデルの節を新設し、`scenarios.md` の節を繰り下げる。
- [x] T003 [Spec] §4.5 から信頼境界を外す。
- [x] T004 [Spec] §11 の参考表と §12 の導入順序を更新する。OWASP の出典を §4.5 から §4.9 へ移し、Shostack を足した。
- [x] T005 [Verify] ガイドと `SPECIFICATION_FORMAT.md` が、正本文書の集合と信頼境界の所有者について同じことを述べていることを確認する。

## Verification

- `DOCUMENTATION_GUIDE.md` のツリーと §4 の節が、`SPECIFICATION_FORMAT.md` の配置図と `docs/README.md` の索引に現れる文書種と一致する（このリポジトリに存在しない `reliability.md` と `recovery.md` を除く）。
- 信頼境界の所有者が、ガイドと `SPECIFICATION_FORMAT.md` と `docs/README.md` の 3 か所で一致する。
- 節番号の繰り下げによって壊れた相互参照が無い。
- `mise run verify`

## Risk Notes

節番号を繰り下げると、番号で参照している箇所が壊れる。現在 §4.9 を指す参照は無いことを確認済みだが、変更後に再確認する。

ガイドは一般論を書く文書なので、このリポジトリの事情を書き込むと役割が崩れる。件数、`THREAT-NNN` という具体的な形式、個々の脅威は書かない。

## Completion

- **Completed At**: 2026-08-27
- **Summary**:
  `DOCUMENTATION_GUIDE.md` が脅威モデルを文書種として持つようになった。§2 のツリーへ加え、§4.9 として節を新設し、`scenarios.md` を §4.10 へ繰り下げた。あわせて信頼境界の所有を §4.5 の `deployment.md` から外し、ガイドと `SPECIFICATION_FORMAT.md` と `docs/README.md` の 3 か所で所有者が一致した。ガイドは一般論を書く文書なので、このリポジトリの `THREAT-NNN` という形式も件数も書いていない。
- **Acceptance RED Evidence**:
  - **Test**: N/A: 文書ガイドの記述であり、観測できる境界を持たない。
  - **Requirement**: N/A: 製品の振る舞いを変えないため、対応する規範シナリオを持たない。
  - **Observed Failure**: 代わりに、ガイドが列挙する文書種と `SPECIFICATION_FORMAT.md` の配置図を突き合わせた。変更前は `threat-model.md` がガイドの側にだけ存在せず、`rg '信頼境界' DOCUMENTATION_GUIDE.md` は `deployment.md` を所有者とする 3 行を返した。同じ語で `SPECIFICATION_FORMAT.md` と `docs/README.md` を引くと `threat-model.md` が所有者であり、2 つの方法論文書が食い違っていた。
  - **Detection Reason**: 突き合わせは、ガイドに項目を足しただけでは通らない。所有者の記述を両側で一致させて初めて差が消えるので、片側だけを直した実装と区別できる。変更後は、ガイドの文書種の集合が配置図と一致し（このリポジトリに存在しない `reliability.md` と `recovery.md` を除く）、信頼境界の所有者は 3 か所すべてで `threat-model.md` になる。
- **Unit RED Evidence**:
  - **Test**: N/A: 検査器もコードも変更していない。
  - **Requirement**: N/A: 内側の計算を持たない変更である。
  - **Observed Failure**: 代わりに節番号の参照を確認した。繰り下げの前後で `rg -o '§[0-9]+\.[0-9]+'` を取り、`§4.9` を指す参照が 3 件（§4.5 からの誘導 1 件と §11 の出典 2 件）、`§4.10` を指す参照が 0 件であることを確かめた。繰り下げによって壊れた参照は無い。
  - **Detection Reason**: 番号だけを付け替えて本文の誘導を直さない実装では、`§4.9` を指す参照が `scenarios.md` を指したままになる。参照の数と行を実際に読むことで、その取り違えが残っていないことを確かめられる。
- **Verification Results**:
  - `mise run check-work-items` - passed
  - `mise run check-ids` - passed
  - `mise run verify` - passed
