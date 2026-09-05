---
depends_on: []
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-08-27
priority: p1
change_kind: tooling
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 検査と負債台帳を足す変更であり、規範行もシナリオも増減しない。製品の利用者から観測できる差は無い。
  references: []
spec_impact:
  {
    kind: none,
    reason: "`docs/standards.md` の冒頭の文だけを実態に合わせて直す。規範行、シナリオ、TypeSpec symbol のいずれも増減も変更もしない。",
  }
initial_context:
  specification:
    [
      docs/standards.md,
      SPECIFICATION_FORMAT.md,
      docs/development/specification-first-workflow.md,
    ]
  source:
    [
      tools/check/src/security-controls.ts,
      tools/check/src/check-security-controls.ts,
      tools/check/src/specification-doc.ts,
      tools/check/src/check-specifications.ts,
      tools/workspace/src/check-workspace.ts,
      tools/workspace/src/workspace.ts,
      tools/check/security-refusal-debt.json,
    ]
  tests:
    [
      tools/check/src/specification-doc.test.ts,
      tools/check/src/security-controls.test.ts,
    ]
  stop_before_reading: [backend, frontend, spec]
---

# 規範とシナリオの被覆をゲートにする

## Motivation

`docs/standards.md` は冒頭でこう宣言している。「`Statement` は製品が何をするかを書き、標準の側の義務を要約しない。各行は、規範 ID をテスト名に含めた対応するテストを持つ。」

確認したところ、この文は現在ただの偽である。`WCAG22-KEYBOARD`、`GDPR-ERASURE`、`GDPR-CONSENT-WITHDRAWAL` のいずれも、`backend/` と `frontend/` に 1 件も出現しない。`tools/check/src/specification-doc.ts` は表の見出しと列の値集合は検証するが、テストからの参照は見ていない。GDPR と WCAG は外部の監査で問われる種類の規範であり、根拠を求められたときに示せるものが無い。

シナリオ側にも同じ非対称がある。`tools/check/src/security-controls.ts` の `checkRefusalCoverage` は、拒否を宣言したシナリオを名指しするテストが存在することを検査し、未対応分を `tools/check/security-refusal-debt.json` に負債として明示管理している。仕組みとしては完成しているが、対象は拒否を宣言したシナリオだけである。それ以外の `REQ-<CONTEXT>-NNN` は、どのテストからも名指しされないまま存在できる。

`docs/development/specification-first-workflow.md` は「テストやコードに要求 ID を書くことが、後から `spec-where` と生成されたトレーサビリティのページの両方でその対応を見つけられるようにする」と述べており、トレーサビリティのページは `render-spec-docs` が生成する。しかしそれは view であってゲートではない。名指しが無いことは、生成された表に空欄として現れるだけで、何も止めない。Behavior-Driven Development の中心はシナリオが実行されることにあるので、拒否だけが検査されている現状はその中心を半分しか満たしていない。

第三の穴として、`docs/glossary.md` と各 Context の `glossary.md` が Published Language を定めているのに、そこに無い語をシナリオが使っても検査は通る。用語集とシナリオが同じ語彙を使っていることは、誰も見ていない。

## Scope

- **規範の被覆**：`docs/standards.md` と各 Context の `standards.md` の全行について、規範 ID を名指しするテストの存在を検査する。
- **シナリオの被覆**：拒否を宣言したシナリオに限らず、全 `REQ-<CONTEXT>-NNN` について、名指しするテストの存在を検査する。
- **負債の明示管理**：既存の未対応分を負債ファイルに列挙し、新規の未対応だけを落とす。`security-refusal-debt.json` の形式に揃え、拒否の負債ファイルとは併存させる。
- **退役の扱い**：後継へ差し替えられたシナリオは被覆の対象から外す。
- **宣言の修正**：`docs/standards.md` の冒頭の文が偽であり続けないよう、記述と実態を一致させる。

## Out of Scope

- テストの追加そのもの。負債ファイルの各項目を解消する作業は、Context ごとに別の work item が扱う。
- 網羅性の逆方向、すなわち「必要な `REQ` が足りないこと」の検出。仕様の完全性は本 work item の対象外であり、脅威モデリング（wi-424）が別の角度から扱う。
- カバレッジ率の閾値。行や分岐の被覆は wi-131 が扱う関心事であり、本件は規範 ID の名指しだけを見る。
- **語彙の対応の検査**。着手前の測定（下記 Design）で誤検出が支配的になることが分かったため、Scope から外した。

## Design

### 測定した現状

| 対象 | 宣言 | テストが名指し | 未対応 |
|---|---|---|---|
| 規範行（`standards.md` 10 ファイル） | 154 | 0 | 154 |
| シナリオ（`scenarios.md` 22 ファイル、退役 2 件を除く） | 306 | 115 | 191 |

191 件の未対応シナリオのうち 102 件は `security-refusal-debt.json` に載る拒否の負債と完全に一致する。

### 検査の置き場所と形

検査は既存の `check-security-controls` を拡張するのではなく、`check-spec` の側へ置く。理由は対象が仕様の全域であり、セキュリティ制御に限らないからである。ただし実装は `security-controls.ts` の `checkRefusalCoverage` と同じ形（宣言された ID の集合、テストが名指しした ID の集合、許可された負債の集合の三者比較）を使い、二つ目の実装様式を持ち込まない。

判定は純粋関数 `checkNormativeCoverage(declared, cited, debt, debtPath, accounted): CoverageFinding[]` に閉じる。主要な型は `DeclaredId = { id: string; path: string }`、`DebtEntry = { id: string; reason: string }`、`CoverageFinding = { path: string; message: string }` である。ファイルの走査、負債ファイルの読み取り、テスト本文の収集という効果は、すべて `check-specifications.ts` の側に置く。`check-spec` は `check-workspace.ts --documents` を経由してこの入口を全正本文書に対して 1 度だけ呼ぶので、規範行とシナリオを 1 か所で集約できる。

規範行の ID は `standards.md` の文法を既に持つ `specification-doc.ts` が返す。`SpecificationValidation` に `standardIds` を足し、シナリオ ID と同じ経路で入口へ渡す。退役したシナリオは見出しが後継を名乗るので、既存の `supersededBy` を見て `declared` から外す。これで T004 の退役の扱いは追加の規則を持たない。

### 負債ファイル

拒否の負債とは併存させる。`security-refusal-debt.json` は「セキュリティ制御の拒否がテストされていない」という、他とは危険度の違う負債を表しており、一般の被覆漏れと同じ一覧に混ぜると、その一覧を読む理由が失われる。

ただし併存を「重なり合う 2 つの一覧」にはしない。未対応シナリオ 191 件のうち 102 件は拒否の負債と一致するので、そのまま列挙すると過半が二重記載になり、テストが 1 件増えるたびに 2 つのファイルを直すことになる。シナリオの被覆は拒否の負債に載る ID を既知の負債として受理し、シナリオの負債ファイルがそれを重ねて持つことは検査で拒否する。併存の意味は「意味の違う、互いに素な 2 つの一覧」である。

負債の項目は `{ "id": ..., "reason": ... }` とし、`reason` は非空を必須とする。理由の欄を空にできると、その後に足された項目と初期投入分が区別できなくなり、解消の進捗が読めなくなる。初期投入分の理由は `present when the check was introduced` で揃える。項目は ID 順に並べる。193 行の一覧が読まれなくなる最大の原因は並びが崩れて差分が読めなくなることなので、これも検査する。

### 名指しをどこで探すか

規範 ID の名指しを探す場所は、拒否の被覆と同じくテストファイルに限る（`*_test.go`、`*.test.ts`、`*.test.tsx`、`*.spec.ts`、`*.spec.tsx`）。実装コードの中の言及を数えると、コメントに ID を書くだけで検査が通ってしまう。

テストファイルの中では、名前に限らず任意の位置の言及を認める。Risk Notes が挙げた命名の窮屈さは実在し、`TestAuthorizeCode_WCAG22_KEYBOARD_GDPR_ERASURE` のような名前を強制すると意図が読めなくなる。拒否の被覆が既に「ファイル内の任意の位置」で運用されており、二つ目の規則を持ち込む理由も無い。この決定の帰結として、`docs/standards.md` の「規範 ID をテスト名に含めた」という表現自体が実態と違うことになるので、T006 でそこも直す。

### 語彙の対応を Scope から外した根拠

着手前に検出可能性を測った。`docs/glossary.md` と 21 個の Context の `glossary.md` が定義する語と別名は 590 件、TypeSpec の symbol は 2997 件ある。全 `scenarios.md` に現れる PascalCase の語 314 件をこの 3587 件と突き合わせると、解決できない語は 60 件残る。その 60 件を読むと、過半は外部プロトコルの要素名（`AuthnRequest`、`NameIDPolicy`、`RelayState`、`IssueInstant`、`ForceAuthn`、`ProtocolBinding`、`UsernameToken`）、Web プラットフォームの語（`SameSite`、`HttpOnly`、`WebAuthn`、`PublicKeyCredentialRequestOptions`）、製品名（`PostgreSQL`、`WebP`）、Context 名（`IdManagement`、`SharedSignals`、`DataKeys`、`WorkloadIdentity`）であり、用語集に無いことは欠陥ではない。本当に定義の無い語は `AdminDashboard`、`HomePage`、`SeedData`、`KeyStore` のような少数である。

誤検出を消すには、外部プロトコルの要素名と Context 名を列挙した許可一覧を保守し続けることになる。それは「リポジトリのどこかに現れる PascalCase の語か」を問う検査に退化し、用語集との対応を見ていない。Design の予告どおり、絞れなかったので入れない。

## Plan

1. 全 `standards.md` の規範 ID と全 `scenarios.md` の `REQ` を集め、テストからの名指しと突き合わせて現状の被覆率を出す。（完了、上表）
2. 被覆されていない ID がある状態で `mise run check-spec` が通ることを観測する。
3. `normative-coverage.ts` に純粋な判定を実装し、`specification-doc.ts` に規範 ID の収集を足す。
4. `check-specifications.ts` から負債ファイル、正本文書、テスト本文を渡して配線する。
5. 負債ファイル 2 種を初期値で生成する。
6. `docs/standards.md` の冒頭と `SPECIFICATION_FORMAT.md` の記述を検査の実態と一致させる。
7. 負債に無い規範 ID とシナリオを足して落ちることを確認する。

## Tasks

- [x] T001 [Baseline] 規範 ID と `REQ` の現在の被覆率を測り、負債ファイルの初期値を作った。規範 154 件中テストが名指しするのは 20 件、シナリオ 306 件中 115 件。`tools/check/standards-coverage-debt.json` に 134 件、`tools/check/scenario-coverage-debt.json` に 89 件を初期投入した。
- [x] T002 [Acceptance] 被覆されていない規範 ID が 134 件ある作業ツリーで `mise run check-spec` が終了コード 0 で通ることを観測した。
- [x] T003 [Tooling] `normative-coverage.ts` の `checkNormativeCoverage` と、`specification-doc.ts` が返す `standardIds` で規範 ID の被覆を検査する。`normative-coverage.test.ts` の `rejects a declaration no test names and no debt entry covers` が RED から GREEN。
- [x] T004 [Tooling] シナリオの被覆を同じ検査で扱い、`supersededBy` を持つ見出しを `check-specifications.ts` が `declared` から外す。`check-workspace.test.ts` の `leaves a retired scenario out of the coverage gate` と `rejects a scenario no test names when no debt list admits it` が対応する。
- [x] T005 [Tooling] 負債の項目を `{ id, reason }` とし、非空の理由、ID 順、重複の排除、宣言の消滅、テストが付いた項目、拒否の負債との二重記載を検査する。`normative-coverage.test.ts` の 7 件が対応する。
- [x] T006 [Spec] `docs/standards.md` の冒頭を「テスト名に含めた」から「テストが名指しし、無い行は理由付きで負債台帳に残る」へ直した。`SPECIFICATION_FORMAT.md` §5 と §6 に同じ規則を `*(checked)*` として置いた。
- [x] T007 [Scope] 検出可能性を測り、誤検出が支配的であることを確認して Out of Scope へ移した（Design に測定を記載）。
- [x] T008 [Verify] 新しい規範行、新しいシナリオ、理由の削除、テストが付いた負債項目、二重記載、並び順の崩れの 6 通りで `mise run check-spec` が落ちること、テストを足すか退役させると通ることを実測した。

## Verification

- `mise run check-spec` が現状の作業ツリーで通る。
- 負債ファイルに載っていない新しい規範行を `standards.md` へ足すと落ち、対応するテストを足すと通る。
- 負債ファイルに載っていない新しいシナリオを足すと落ち、`REQ` を名指しするテストを足すと通る。
- 負債ファイルの項目から理由を削ると落ちる。
- 後継へ差し替えたシナリオが被覆の対象から外れる。
- `mise run verify`

## Risk Notes

規範 ID の名指しをテスト名で行う規則は、テストの命名を検査に従わせることになる。命名が窮屈になりすぎると、ID を名前に押し込むためにテストの意図が読みにくくなる。Design のとおり、テストファイル内の任意の位置での言及を認めることでこれを避ける。

シナリオの被覆を全 `REQ` へ広げると、負債が大きく、かつ解消が長期にわたる。負債ファイルが「読まれない一覧」になることが最大の失敗であり、理由の必須化、並び順の固定、解消済み項目の検出がそれを防ぐ仕掛けである。

語彙の検査は誤検出が支配的になれば信頼を失い、他の検査ごと無視されるようになる。測定の結果、絞れなかったので入れない。

## Completion

- **Completed At**: 2026-09-05
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返す。規範行もシナリオも増減せず、`docs/standards.md` の冒頭の文だけが実態に合う表現へ変わった。意味の差は仕様の内容ではなく、仕様と実装の対応が検査されるかどうかにある。`mise run check-spec` は今後、規範行 154 件とシナリオ 306 件のそれぞれについて、製品のテストがその ID を名指ししているか、負債台帳が理由付きで保持しているかのどちらかを要求する。着手時点で名指しがあるのは 134 件で、残りは `tools/check/standards-coverage-debt.json`（134 件）、`tools/check/scenario-coverage-debt.json`（89 件）、既存の `tools/check/security-refusal-debt.json`（102 件）が保持する。3 つの台帳は互いに素であり、縮む方向にしか動かない。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`
  - **Requirement**: N/A: 製品の規範要求ではなく、仕様文書に対する検査を足す変更である。
  - **Observed Failure**: 実装前、`WCAG22-KEYBOARD` と `GDPR-ERASURE` を含む 134 件の規範行と 191 件のシナリオをどのテストも名指ししていない作業ツリーで、`mise run check-spec` が終了コード 0 で通った。これが観測した「落ちるべきときに落ちない」失敗である。実装後、同じ作業ツリーに `PROBE-COVERAGE-GATE` の行と `REQ-APITOKENS-900` のシナリオを足すと、それぞれ `is declared, but no test names it` で落ち、ID を書いたテストファイルを足すと通った。
  - **Detection Reason**: 検査はテストファイルの中に宣言された ID そのものが現れることだけを見る。実装コード中の言及やコメントだけの一致では通らず、名指しの無い行を通す実装は上の 2 つの追加で必ず落ちる。
- **Unit RED Evidence**:
  - **Test**: `tools/check/src/normative-coverage.test.ts`（`bun test` = `mise run test-tools`）
  - **Requirement**: N/A: 同上。判定は仕様の内容ではなく被覆の三者比較である。
  - **Observed Failure**: `Cannot find module './normative-coverage.ts'` で 1 件 fail、続いて `validateDocument(...).standardIds` が `undefined` を返して `specification-doc.test.ts` の 2 件が fail。
  - **Detection Reason**: 各規則を 1 件ずつの期待に分けてあるので、どれか 1 つを緩めた実装は対応する 1 件だけが落ちる。理由の必須、ID 順、二重記載、宣言の消滅、テストが付いた項目、他の台帳との排他がそれぞれ独立して観測できる。
- **Change-Resistance Results**:
  実装した検査に対し、通ってはならない 6 つの形を作業ツリーへ実際に注入して確認した。(1) 規範行 `PROBE-COVERAGE-GATE` の追加 → 落ちる。テストを足すと通り、`154 → 155 standard(s)` と `134 → 135 id(s) named by a test` に動いた。(2) シナリオ `REQ-APITOKENS-900` の追加 → 落ちる。テストを足すと通り、退役見出しへ変えると被覆の対象から外れて通った。(3) 負債項目の理由を空にする → `is listed without a reason` で落ちる。(4) 負債項目に対応するテストを足す → `now has a test that names it` で落ちる。(5) 拒否の負債にある ID をシナリオの台帳へ足す → `Keep the two lists disjoint` で落ちる。(6) 台帳の並びを崩す → `Keep the list in id order` で落ちる。
  実装中の誤りを 1 つ、検査自身が検出した。最初の版は ID を「大文字の区切りをハイフンで繋いだもの」という形で認識しており、`SAML2Core-BearerAssertion` や `WSFed-PassiveSignIn` のような SAML / WS-Federation の 13 行はテストが名指ししても永久に一致しなかった。宣言された ID そのものを検索する方式へ変えたところ、負債の初期値が 135 件から 134 件へ減った。この 1 件が、形を推測する実装と宣言を読む実装を区別する観測である。
  台帳を失った場合の向きも確認した。台帳ファイルの無い作業ツリーでは、被覆されていない ID がすべて報告される（`check-workspace.test.ts` の `rejects a scenario no test names when no debt list admits it`）。台帳の削除はゲートの解除ではなく、223 件の報告として現れる。
- **Verification Results**:
  - `mise run verify` - passed
