---
depends_on: []
status: completed
authors: [tn]
risk: medium
created_at: 2026-08-27
priority: p1
change_kind: docs
evidence_policy: risk-based-v2
initial_context:
  specification: []
  typespec: []
  source:
    - DEVELOPMENT.md
    - SPECIFICATION_FORMAT.md
    - WORK_ITEM_FORMAT.md
    - docs/README.md
    - docs/structure.md
    - docs/design-rules.md
    - tools/check/src/specification-doc.ts
    - work-items/wi-414-boundary-fitness-functions.md
    - work-items/wi-416-ddd-strategic-and-tactical-gaps.md
    - backend/saml/domain
    - backend/saml/handlers_http
    - backend/application/usecases
  tests:
    - tools/check/src/specification-doc.test.ts
  stop_before_reading:
    - frontend
    - infra
    - spec/contexts
spec_impact: { kind: none, reason: "設計原則の正本文書を新設し SPECIFICATION_FORMAT.md の配置図を更新するが、規範シナリオ、規範 ID、TypeSpec シンボルを追加も変更もしない。" }
---

# 設計原則を現在状態の文書として持つ

## Motivation

`docs/` と方法論文書を検索しても、情報隠蔽、深いモジュール、インターフェースの深さに相当する語が 1 件も出てこない。層の方向、Context の境界、ファイルの種類分けはきわめて厳密に定めているのに、「良いインターフェースとは何か」を語る言葉が体系に存在しない。

同じ空白がいくつもの形で現れている。`ports/` にあるのは Repository と外部サービス、つまり駆動される側だけであり、ユースケースをアプリケーションの API として型で定義する駆動する側のポートが無い。HTTP ハンドラーが usecase の具象構造体を直接呼ぶため、同じユースケースを CLI やジョブから呼ぶときに境界が無い。アダプターがドメイン型を外へ漏らさないという規範も無い。

型についても同じである。`docs/database.md` は列型の選択規則を「自由形式の文字列は `TEXT`、上限があるなら `CHECK` を併記、有限の値集合は `CHECK (col IN (...))`」と細部まで定めているのに、Go 側の型規律には同等の記述がない。不正な状態を型で表現できなくするという方針も、別名型を使うかどうかの方針も無い。TypeSpec が持つのは境界の型だけで、ドメイン型は手書きであるため、型の正本が事実上二重化していることも書かれていない。

`DEVELOPMENT.md` の「Type and effect design」はデータ・計算・作用の分離を良く定めているが、それは変更時に守る手順であって、システムが現在持っている性質としては記述されていない。結果として `docs/` だけを読んだ人は、このシステムがどういう設計原則で建っているかを知ることができない。これは `DEVELOPMENT.md` 自身の「現在の設計は正本文書と work item だけから理解できなければならない」に反する。

さらに `SPECIFICATION_FORMAT.md` が「不変条件を列挙するな」と明示的に禁じた結果、不変条件の居場所が体系から消えた。判断としては妥当だが、代わりの居場所が示されていない。

## Scope

- **モジュールの深さと情報隠蔽**：インターフェースの複雑さに対して提供する機能が大きいことを良いモジュールの基準として書く。`internals.md` を書く必要が生じたこと自体が、モジュールが浅い兆候でありうるという診断を明示する。
- **ポートの両翼**：駆動されるポート（Repository、外部サービス、通知）と駆動するポート（ユースケースをアプリケーション API として型で定義したもの）を区別し、どちらを `ports/` に置くかを定める。
- **アダプターの責務**：アダプターが外部表現と内部表現を変換する責務を持つこと、同じ Go 型を再利用しても外部契約の正本は TypeSpec であること、外部形式の規則がドメインのインターフェースに現れる場合はアダプター固有の型へ分けることを書く。既存の直接利用が残る現状も併記する。
- **型の規律**：不正な状態を構築できない型を優先すること、識別子に別名型を使うかどうか、TypeSpec の型とドメイン型の関係（境界の型とドメインの型は別物であり、どちらが何の正本か）を書く。
- **作用の境界**：ユースケースが時刻、乱数、識別子生成、設定、永続化、通知を編成し、決定可能な計算へ値またはポートとして渡す設計規則を書く。`domain/` 内に作用を作る既存例外が残り、配置だけでは無作用を保証しない現状も併記する。
- **不変条件の居場所**：散文で列挙する代わりに、構築時に検証する型、事後条件、スキーマ制約のいずれが持つかを定める。`SPECIFICATION_FORMAT.md` の「不変条件を列挙するな」に、代わりの居場所を示す一文を足す。
- **エラーの扱い**：失敗を値として返す方針、部分関数を避ける方針、拒否をどこで表現するかを書く。
- **配置先の決定**：上記を `docs/` のどこに置くかを決め、必要なら `SPECIFICATION_FORMAT.md` の配置図と検査器を更新する。

## Out of Scope

- 既存コードを新しい原則へ合わせる改修。本 work item は原則を現在状態の文書として持つところまでを担う。
- 駆動するポートの実装。導入の可否は原則を書いた後に、対象を絞った別の work item で判断する。
- 不変データ構造や永続データ構造の採用。Go の標準的な書き方から離れる判断は本件では行わない。
- 検査による強制。作用の禁止は wi-414 が持ち、それ以外の原則は当面レビューの判断とする。
- `domain/` 内で時刻や乱数を生成する既存箇所の改修。例外の棚卸しと新規違反の防止は wi-414 が持つ。
- HTTP 応答型や外部プロトコル表現がドメイン型を直接使う既存箇所の改修。新しい設計規則で変更箇所を評価し、改修対象と効果を絞れた時点で別の work item にする。

## Design

配置先には 2 つの案がある。

第一案は `docs/structure.md` に節を足すことである。既存ファイルの範囲は「ディレクトリ、依存の向き、層の構成、アーキテクチャスタイル」であり、設計原則はその延長として読める。`docs/` 直下のファイル集合は `SPECIFICATION_FORMAT.md` で閉じた集合として検査されているため、この案は形式文書にも検査器にも手を入れずに済む。

第二案は `docs/design-rules.md` を新設することである。設計原則は構造の記述とは寿命も読まれ方も違い、`structure.md` に混ぜると「ディレクトリを確認しに来た人」が原則を読み飛ばす。ただし閉じた集合を広げるため、`SPECIFICATION_FORMAT.md` の配置図と `tools/check/src/specification-doc.ts` の許可リストを同時に変える必要がある。

採るのは第二案である。`SPECIFICATION_FORMAT.md` 自身が「セクションが仕様を分けるのではなく、ファイルが分ける。ファイル名がその中身の種類を語る」と定めており、構造の記述と設計原則は明らかに別の種類だからである。第一案はその原則に反する形で既存ファイルを太らせることになる。ファイル集合が閉じているのは無秩序な追加を防ぐためであって、種類の異なる内容を既存ファイルへ押し込めるためではない。

原則の書き方は、判断とその理由という `decisions.md` の形式に倣う。原則を抽象的な標語として並べると読み手が適用できないため、それぞれについて「この原則に反している状態はどう見えるか」を 1 文添える。

着手時の仕分けでは、`ports/` が Repository と外部サービスを表す駆動されるポートを持ち、HTTP アダプターが具象ユースケースを直接呼ぶ構成を確認した。駆動するポートは一律に導入せず、二つ以上のアダプターまたは呼び出し方式が実在する Seam に限って導入する。これは「一つのアダプターしかない Seam は仮想上の差し替えにすぎない」というモジュールの深さの判断と整合する。

同じ仕分けで、`domain/` 内の時刻取得と乱数生成、HTTP 応答型によるドメイン型の直接利用も確認した。そのため、「ドメインは常に無作用である」「アダプターはドメイン型を外部表現へ一切露出しない」を現在の普遍的な性質とは書かない。作用の既存負債と機械検査は wi-414、Aggregate と Repository の粒度は wi-416 が扱い、既存アダプターの表現分離は対象と効果を絞るまで Out of Scope とする。

型の正本は、外部境界を TypeSpec、内部の状態と操作を Go のドメイン型、永続データの整合性をデータベース制約が持つ形に決める。別名型は識別子へ一律に適用せず、値の取り違えを防ぐか一つの検証済み表現を作れる場合に限る。構築後の `Validate` を必要とする既存型があるため、「不正な状態を型だけでは構築できない」というリポジトリ全体の保証は置かない。

## Plan

1. 現在のコードから、原則として書けるものと書けないものを仕分ける。既に守られている性質は現在状態として書き、守られていない性質は原則ではなく wi-414 の負債または後続 work item の対象として切り出す。
2. 配置先を `docs/design-rules.md` に決め、`SPECIFICATION_FORMAT.md` の配置図と検査器の許可リストを更新する。
3. 原則を書く。それぞれに理由と、反している状態の見え方を添える。
4. `SPECIFICATION_FORMAT.md` の不変条件に関する記述へ、代わりの居場所を示す一文を足す。
5. `docs/README.md` の文書索引へ新しいファイルを加える。

## Tasks

- [x] T001 [Baseline] 既に守られている性質と守られていない性質を仕分け、後者を wi-414、wi-416、または本項目の Out of Scope へ切り出した。
- [x] T002 [Acceptance] `docs/design-rules.md` を置いた状態の `mise run check-spec` が `not a canonical document` で拒否し、`mise run test-tools -- check/src/specification-doc.test.ts` が `Expected: "prose" / Received: undefined` で失敗することを観測した（`REQ-*` は N/A：正準文書集合を広げる文書および検査器変更であり、製品の規範的振る舞いを変えないため）。
- [x] T003 [Tooling] `SPECIFICATION_FORMAT.md` の配置図と `specification-doc.ts` の許可リストへ新しいファイル名を加え、`documentKind` の単体検査を GREEN にした。
- [x] T004 [Spec] 設計原則を `docs/design-rules.md` に書き、`docs/README.md` の索引へ加えた。
- [x] T005 [Spec] `SPECIFICATION_FORMAT.md` の不変条件の記述へ、型、操作、スキーマ、`scenarios.md` という代わりの居場所を示す一文を足した。
- [x] T006 [Verify] `mise run check-spec` と `mise run verify` を通し、生成された `specification/design-rules.html` が全体索引とナビゲーションの両方から到達できることを確認した。

## Verification

- `mise run check-spec` が新しい正本文書を受け入れ、許可リストに無いファイル名は引き続き拒否する。
- 生成された仕様サイトの `docs/README.md` に対応するページから、新しいファイルへ到達できる。
- `mise run verify`

## Risk Notes

正本文書のファイル集合を広げると、以後「ここに書くべきか迷う内容」の受け皿として使われ、原則ではない雑多な記述が集まるおそれがある。ファイルの境界宣言を最初に書き、何を持たないか（判断は `decisions.md`、機構は `internals.md`、構造は `structure.md`）を明示する。

既に守られていない性質を原則として書いてしまうと、文書が現在状態ではなく願望になる。これは Context Map が実装と乖離したのと同じ失敗であり、T001 の仕分けがそれを防ぐ唯一の工程である。守られていない性質は、原則としてではなく負債として記録する。

## Completion

- **Completed At**: 2026-08-28
- **Summary**:
  `mise run spec-diff` は `main` に対する規範仕様の変更がないことを報告した。正準文書の閉集合へ `docs/design-rules.md` を加え、モジュールの深さ、Seam、アダプター、型と不変条件、作用、エラーの現在の設計規則と既存例外を記録し、仕様サイトから到達可能にした。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`
  - **Requirement**: N/A: 正準文書集合を広げる文書および検査器変更であり、製品の規範的振る舞いを変えないため。
  - **Observed Failure**: `docs/design-rules.md: not a canonical document` と拒否され、許可されたルート文書の一覧に新しいファイル名が現れなかった。
  - **Detection Reason**: 正準文書を作成しただけで検査器と生成器の閉集合を更新し忘れる誤実装を、仕様全体の観測境界で検出するため。
- **Unit RED Evidence**:
  - **Test**: `mise run test-tools -- check/src/specification-doc.test.ts` の `documentKind > names the grammar of each canonical document`
  - **Requirement**: N/A: 文書名から検査文法を選ぶ内部ツールの単体変更であり、製品の規範的振る舞いを変えないため。
  - **Observed Failure**: `documentKind('docs/design-rules.md')` に対して `Expected: "prose" / Received: undefined` となった。
  - **Detection Reason**: 新しいファイル名を正準な prose 文書として分類しない誤実装を、許可リストの単位で検出するため。
- **Independent Verification**:
  新しい文脈の `standards_review` と `spec_review` が、規約軸と仕様軸を独立に確認した。初回の Scope 不整合 3 件、古い未解決記述 1 件、日本語用語の規約違反 1 件を修正し、再確認では両軸とも指摘なしとなった。
- **Change-Resistance Results**:
  代表的な誤実装として `ROOT_DOCUMENTS` から `design-rules.md` を一時的に削除したところ、追加した単体検査が `Expected: "prose" / Received: undefined` で失敗した。復元後は 29 件すべてが成功した。
- **Verification Results**:
  - `mise run check-spec` - passed
  - `mise run spec-render` - passed; 140 canonical documents and 980 generated pages
  - `mise run check-api-compat` - passed; no breaking changes
  - `mise run check-boundaries` - passed
  - `mise run check-links` - passed
  - `mise run verify` - passed
