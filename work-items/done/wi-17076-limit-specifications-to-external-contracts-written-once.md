---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-03
priority: p1
depends_on: []
change_kind: docs
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  reason: 仕様と設計の書き方と、IdManagement の User の機能仕様の記述だけを変え、製品の振る舞い、公開 API、設定、運用手順は変えないので、リリースの読者へ知らせる差分がない。
  references: []
initial_context:
  specification:
    - SPECIFICATION_FORMAT.md
    - docs/domain/identity-management/principals/user/README.md#REQ-IDMANAGEMENT-042
    - docs/domain/identity-management/principals/user/lifecycle.md#REQ-IDMANAGEMENT-010
  typespec: []
  source:
    - tools/check/src/specification-rules.ts
    - tools/check/src/feature-nodes.ts
    - tools/check/src/specification-doc.ts
    - tools/render-docs/src/render.ts
  tests:
    - tools/check/src/feature-layout.acceptance.test.ts
  stop_before_reading: [frontend, backend]
affected_spec:
  - { path: docs/domain/identity-management/principals/user/README.md, requirement: REQ-IDMANAGEMENT-001 }
  - { path: docs/domain/identity-management/principals/user/README.md, requirement: REQ-IDMANAGEMENT-005 }
  - { path: docs/domain/identity-management/principals/user/README.md, requirement: REQ-IDMANAGEMENT-042 }
  - { path: docs/domain/identity-management/principals/user/README.md, requirement: REQ-IDMANAGEMENT-043 }
  - { path: docs/domain/identity-management/principals/user/README.md, requirement: REQ-IDMANAGEMENT-044 }
  - { path: docs/domain/identity-management/principals/user/README.md, requirement: REQ-IDMANAGEMENT-045 }
  - { path: docs/domain/identity-management/principals/user/README.md, requirement: REQ-IDMANAGEMENT-047 }
  - { path: docs/domain/identity-management/principals/user/lifecycle.md, requirement: REQ-IDMANAGEMENT-010 }
  - { path: docs/domain/identity-management/principals/user/lifecycle.md, requirement: REQ-IDMANAGEMENT-011 }
  - { path: docs/domain/identity-management/principals/user/lifecycle.md, requirement: REQ-IDMANAGEMENT-013 }
  - { path: docs/domain/identity-management/principals/user/lifecycle.md, requirement: REQ-IDMANAGEMENT-046 }
  - { path: docs/domain/identity-management/principals/user/lifecycle.md, requirement: REQ-IDMANAGEMENT-048 }
  - { path: docs/domain/identity-management/principals/user/lifecycle.md, requirement: REQ-IDMANAGEMENT-049 }
  - { path: docs/domain/identity-management/principals/user/lifecycle.md, requirement: REQ-IDMANAGEMENT-050 }
---

# 仕様には外部から観測できる振る舞いだけを EARS 形式の要件として一か所に書き、仕様にない振る舞いを見つけたら分類して仕様、設計、実装へ反映する方式へ改める

## 動機

IdManagement の仕様と設計は、内部仕様の不足を補う追加を重ねた結果、整理されないまま量だけが増えた。
2026-10-03 時点の IdManagement の文書を測ると、次のとおりである。

| 測定項目 | 値 | 問題 |
| --- | --- | --- |
| 文書の総行数 | 約 4,900 行（`REQ-*` は 87 件） | 1 件あたり約 56 行 |
| 手書きの Gherkin の例 | 約 2,050 行（総行数の約 42%） | 多くは規則文の言い換えであり、新しい情報を加えていない |
| 話題の索引の「該当なし：」の行 | 103 行 | 内容のない記入欄になっている |
| 同じ事実を書く場所 | 操作ごとの要約表、規則文、エラー表、例、`design.md` | 一つの事実を変えるたびに複数箇所を同期させる |
| 書式を定める文書 | 約 2,000 行（`SPECIFICATION_FORMAT.md`、`DOCUMENTATION_GUIDE.md`、`specification-first-workflow.md`） | エージェントが毎回読む量が大きく、規則と理由の説明が混在している |
| 要判断 | 15 件 | 未決の点が現在の仕様の中に散在している |

一つの事実を手で書く場所が多いので、追加のたびにエージェントが全箇所を書き足して整合させることになり、文書が乱れ、AI のコストも増える。

さらに、追加した「詳細な内部仕様」の多くは内部のものではない。
外部から観測できるのに、誰も決めないまま実装が偶然決めていた振る舞いである。
15 件の要判断を分類すると、次のとおりになる。

| 漏れの種類 | 件数 | 例 |
| --- | --- | --- |
| 同じ種類の操作が経路や Aggregate ごとに異なる | 9 | ユーザー名の大文字と小文字の扱いが User と Agent、Group で異なる。メールアドレスの重複を JIT は拒否し、管理者の作成は拒否しない。削除の予約は下流へ通知し、復元は通知しない |
| 状態と操作の組の結果が決まっていない | 1 | 停止した Agent を削除できない |
| 時間で起きる遷移の起点が決まっていない | 1 | 完全削除が一覧の取得のときにしか起きない |
| 項目の間の不変条件が決まっていない | 1 | メールアドレスを変えても `email_verified` が残る |
| 失敗や記録が観測できない | 2 | 確認メールの送信失敗を本人が知る手段がない。動的グループによるロールの付与が監査に残らない |
| 上限に達したときの挙動が決まっていない | 1 | エクスポートの一覧が 200 件で切れる |

「内部仕様が要る」という状況は、仕様の漏れか実装の欠陥のどちらかを示す。
文書を足して説明するのではなく、仕様を決めるか実装を直して解消する方式に改める。
あわせて、漏れを仕様の作成、実装、テストの各工程で見つけて反映する手順を定める。

## 対象範囲

### 用語

この項目と、改訂後の書式文書では、次の語を使う。
書式文書では、初出の箇所に同じ定義を置く。

| 用語 | 定義 | 例 |
| --- | --- | --- |
| 要件 | `REQ-*` の ID が付いた一件の約束。今の書式文書が「規則」と呼ぶもの | REQ-IDMANAGEMENT-042 |
| EARS 形式 | 要件の文を、EARS（Easy Approach to Requirements Syntax）の五つの型のどれかで書く書き方 | 「User が `Disabled` のとき、サインインを拒否する」 |
| 外部から観測できる振る舞い | API の応答、保存されて後から読める状態、発行するイベントと監査記録、外部への通知として、製品の外から確かめられる結果 | 409 を返す。`UserDeleted` を発行する |
| 操作 | 行為者が製品に一回要求し、製品が結果を返すまでの単位。HTTP の operation と一対一とは限らず、別の Context からの呼び出しや、時間を契機とする処理も含む | 管理者による User の作成。User の無効化。猶予期間の経過による完全削除 |
| 横断的要件 | 複数の操作や Aggregate に同じ内容で適用される要件。上位の文書で一度だけ書き、各機能は例外だけを書く | 状態を変える操作は、すでにその状態なら成功を返しイベントを発行しない |
| 値オブジェクト | ドメイン駆動設計の Value Object。値の正規化、比較、妥当性の判定を一か所で定義する型 | ユーザー名（前後の空白を除き、大文字と小文字を区別しない、など） |
| 状態遷移表（マトリクス形式） | 行に状態、列に操作を並べ、すべてのセルに結果（遷移先、何もしない、拒否とその理由）を書く表 | `PendingDeletion` の行の「無効化」の列に「拒否（409 `user_pending_deletion`）」と書く |

- 「規則」という語を、書式文書、skills、検査の出力で「要件」に置き換える。

### 仕様に書く範囲

- 外部から観測できる振る舞いはすべて仕様に書き、観測できないもの（内部の手順、トランザクションの範囲、関数の分け方など）は仕様に書かない。
- 書式文書の分かりにくい語を、一般的な技術用語か具体的な言い方に置き換える（例：「段」は「階層」にする）。
- 未記載の振る舞いを見つけたら、次のどれかに分類する。
  全部を書き写すことはしない。

| 分類 | 条件 | 対応 |
| --- | --- | --- |
| (a) 要件にする | 維持すべき約束である | 要件として書く |
| (b) 書かない | その細部だけが異なる二つの実装をどちらも正しいと判断できる | 仕様に書かず、テストでも固定しない |
| (c) 実装を直す | 偶然の振る舞いであり、なくすべきである | 実装方針を切り替える work item を起票する |

- 「仕様を短く書けない設計は直す」を設計の判断基準として設計ガイドラインに加える。
  仕様に書けない振る舞いを生まない実装の指針（状態を変える操作を冪等にする、読み取りの操作に副作用を加えない、順序と時刻への暗黙の依存を作らない、横断的要件を一か所で実装する）も加える。

### 一つの事実を一か所に書く

- 要件を EARS 形式で書く。
  一文に一つの義務を書く。
  型は、常に成り立つもの（Ubiquitous）、イベントを契機とするもの（Event-driven）、特定の状態の間に成り立つもの（State-driven）、望まない入力や状況への応答（Unwanted behavior）、特定の構成でだけ成り立つもの（Optional feature）の五つとする。
- 操作ごとの要約表とエラー表を廃止する。
  要約が要る場合は、要件と TypeSpec から生成サイトで作る。
- 担保手段の欄を廃止する。
  要件からコードへの追跡は、テストの `//spec:covers` と `spec-route` で行う。
- 例を必須から任意に改める。
  例を置くのは、境界値や意外な挙動を示すときだけとし、そのときは `testdata/*.examples.json` からの生成を優先する。
- 機能の `design.md` を任意にし、機能の階層の話題の索引を廃止する。
  `design.md` に書くのは、コードから読み取れない仕組みと、壊れたときの直し方だけとする。
- 要判断の欄を廃止する。
  仕様には現在の挙動を要件として書き、未決の点は work item として起票する。

### 漏れを作成時に見つける仕組み

- 横断的要件を上位の階層で一度だけ定義し、機能の仕様には例外だけを書く。
  対象は、識別子の正規化と比較、状態を変える操作の冪等性、拒否が作用を起こさないこと、下流への通知とその失敗の扱い、監査を残す範囲とする。
- 値オブジェクト（ユーザー名、メールアドレスなど）の正規化、比較、一意性の範囲を、モデルの節で一度だけ定義する。
- 状態遷移の節に状態遷移表（マトリクス形式）を置き、すべての組に結果（遷移、何もしない、拒否とその理由）を書く。
  時間で起きる遷移は、その起点も表に書く。
- 操作ごとに答える問いの一覧を `spec-change` スキルに置く。
  問いは、不正な入力、上限、失敗の観測者、記録、他の項目と Aggregate への波及、同じ種類の操作の別経路との一致である。

### 漏れを実装時とテスト時に見つけて反映する手順

- `implement-work-item` スキルと仕様先行の開発ワークフローに、仕様にない分岐、エラー、イベント、副作用を書くことになったら止めて (a)(b)(c) に分類し、work item に記録する手順を加える。
- テスト時に `spec-review-candidates`、ミューテーションテストの生存した変異、`//spec:covers` のないテストを、同じ分類で読む手順を加える。

### 更新の方法と書式文書

- 仕様の変更は、work item に要件の差分（追加、変更、廃止）を書いてから適用する。
- 既存コードから仕様を一括で書き出す作業をやめ、仕様に触れる変更のときだけ書き出す。
- `SPECIFICATION_FORMAT.md` を、エージェントが読む短いテンプレートと、人が読む理由の説明に分ける。
- 廃止した形式を必須とする検査（例の必須、担保手段の必須、機能の話題の索引、規則という語）を外す。
  既存の文書がそのまま検査を通るように、外す方向だけの変更にする。

### 試行

- `principals/user` の文書を新しい方式で書き直す。
  要件の ID は変えず、本文を EARS 形式に改める。
- 書き直しの前後で、行数と、変更一件あたりにエージェントが読む行数を測り、完了の節に残す。
- `principals/user` の要判断 6 件を (a)(b)(c) に分類した案を作り、利用者の判断を受けて、(a) と (b) を反映し、(c) を work item として起票する。

## 対象外

- IdManagement の `principals` の階層の廃止。
  wi-87813 が扱う。
- 状態遷移表（マトリクス形式）の網羅、エラーコードとイベントの要件への記載、モデルベースのテストの検査と仕組み。
  試行で方式の効果を確かめてから、wi-89346 が扱う。
- `principals/user` 以外の機能の書き直し。
  試行の結果を見てから、Context ごとに起票する。
- (c) に分類した振る舞いの実装の変更。
  分類ごとに起票する work item が扱う。
- 書式文書の `practices/` への移動。
  wi-99632 が扱う。

## 設計

### 参考にした手法と取り入れる点

| 手法 | 取り入れる点 | 取り入れない点 |
| --- | --- | --- |
| OpenSpec | 現在の仕様と変更の差分を分け、変更は要件の追加、変更、廃止として書く | ディレクトリ構成と専用ツール。既存の work item と検査がすでに同じ役割を担う |
| Kiro の EARS 記法 | 要件を五つの型の一文で書く | 機能ごとの `requirements.md`、`design.md`、`tasks.md` の構成。変更単位の文書であり、現在状態の仕様に合わない |
| Living Documentation | コードとテストから導けるものを手で書かず、生成する | 仕様をコードの注釈からだけ生成すること。外部から観測できる振る舞いを実装より先に決める仕様先行の原則に反する |
| arc42 | 必要な章だけを書く | 全話題を並べる索引。内容のない記入欄を増やす |

### 横断的要件の置き場所

要件は機能ノードでだけ宣言できるので（検査済みの規則）、Context の `README.md` には要件を置かない。
次のように置き、新しい ID の種類は設けない。

| 範囲 | 置き場所 |
| --- | --- |
| 一つの機能の複数の操作 | 不変条件が最も直接に関わる操作の H3 に一つの要件として置き、ほかの操作は「上位の要件」の欄で参照する |
| 複数の機能 | その振る舞いを実装する共有の仕組みの機能ノード |
| システム全体 | `docs/design/application/api-guidelines.md` などのシステム文書 |
| 複数の機能が使う値 | Context の `README.md` のモデルの節に、値オブジェクトとして定義する |

`principals/user` の試行では、値オブジェクトを機能のモデルの節に置いた。
User、Group、Agent の名前をそろえる wi-83020 で、Context の `README.md` へ移す。

### 用語の決定

| 対象 | 決定 | 退けた候補と理由 |
| --- | --- | --- |
| `REQ-*` の一件 | 要件 | 規則：検査の規則、Gherkin の `Rule`、ガイドラインの規則と区別できない |
| 機能仕様の H3 の単位 | 操作 | ユースケース：要求工学では複数手順のシナリオを指す。システム操作：「システム」が誤解を招く。処理、振る舞い、機能、コマンド：既存の語とぶつかるか、読み取りを含まない |

### 未記載の振る舞いの分類（principals/user）

| # | 今の挙動 | 分類 | 扱い |
| --- | --- | --- | --- |
| 1 | ユーザー名は大文字と小文字を区別する（Agent と Group は区別しない） | (c) | wi-83020。Okta、Entra ID、Google Workspace、Keycloak、SCIM に合わせ、User、Group、Agent の名前を、表記を保ったまま大文字と小文字を区別せずに比較する |
| 2 | 管理者による作成だけがメールアドレスの重複を拒否しない | (c) | wi-83020 |
| 3 | フェデレーションの JIT だけが動的グループを評価しない | (c) | wi-83020 |
| 4 | 期限切れの完全削除が一覧の取得のときにしか起きない | (c) | wi-92312 |
| 5 | 管理者がメールアドレスを変えても `email_verified` が残る | (c) | wi-93929 |
| 6 | 削除の予約は下流へ通知するが、復元は通知しない | (c) | wi-93929 |
| 7 | 途中で失敗した完全削除は、再実行しても回収できない | (c) | wi-92312 |
| 8 | `Deleted` の User への無効化、再有効化、削除の予約、復元は 404 `user_not_found` で拒否される | (a) | 状態遷移表（マトリクス形式）に書いた。表を作る過程で見つかった |

(c) の各項目は、仕様には現在の挙動を要件として残し、変更はそれぞれの work item で行う。

### 書き起こしの work item の削除

コンテキスト全体の暗黙の仕様を一括で書き起こす 19 件の work item（`wi-*-transcribe-implicit-specifications-of-*`）は、この方式と相いれないので削除した。
書き起こしは、機能に触れる変更の作業で、触れる範囲だけを行う。

### 採らない案

| 案 | 採らない理由 |
| --- | --- |
| 今の形式を保ち、検査を増やして同期の漏れを防ぐ | 手で書く場所の数が変わらないので、作成と更新の費用が下がらない |
| 例（Gherkin）だけを仕様にする | 要件が増えるほど全体を把握できなくなる |
| 内部設計を機能ごとに詳しく書く | 内部仕様が要る状況は仕様の漏れか実装の欠陥なので、文書で説明しても原因が残る |
| 全 Context を一度に書き直す | 方式の効果を測る前に費用をかけることになる。試行で測ってから広げる |

## 計画

1. 書式文書の改訂案を作る。
   語の置き換え、仕様の範囲、EARS 形式、廃止する欄と表、横断的要件、値オブジェクト、状態遷移表（マトリクス形式）、三つの分類を含める。
2. 廃止した形式を必須とする検査を外す。
3. `spec-change` と `implement-work-item` のスキルに、問いの一覧と分類の手順を加える。
4. `principals/user` を書き直し、前後の行数を測る。
5. `principals/user` の要判断 6 件の分類案を利用者に示し、判断を受けて反映する。
6. `SPECIFICATION_FORMAT.md` をテンプレートと理由の説明に分ける。

着手時に決めたこと：

- 横断的要件の置き場所は、上の「横断的要件の置き場所」の表のとおりとした。
- EARS 形式の日本語の型は、常時「〈対象〉は、〈結果〉。」、契機「〈契機〉とき、〈結果〉。」、状態「〈状態〉の間は、〈結果〉。」、望まない入力「〈条件〉場合は、〈結果〉。」、構成「〈構成〉では、〈結果〉。」とした。
- 担保手段の欄と要判断の欄は、その機能を書き直すときに消す。残っている担保手段の値は、引き続き宣言の存在を検査する。

## タスク

- [x] T001 [Docs] 書式文書と仕様先行の開発ワークフローを改訂する。代替検査：`mise run check-links`、`mise run check-agent-guidance`。
- [x] T002 [Tooling] 廃止した形式を必須とする検査を外す。RED：`feature-layout.acceptance.test.ts` の、例のない要件、担保手段のない要件、機能の話題の索引、エラーの節の各テストが、検査を変える前の期待と食い違って失敗した。
- [x] T003 [Docs] `spec-change` と `implement-work-item` のスキルに問いの一覧と分類の手順を加える。
- [x] T004 [Docs] 設計ガイドラインに判断基準と実装の指針を加える。
- [x] T005 [Spec] `principals/user` を新しい方式で書き直し、前後の行数を測る。
- [x] T006 [Spec] `principals/user` の要判断を分類し、利用者の判断を反映し、(c) を起票する。
- [x] T007 [Docs] `SPECIFICATION_FORMAT.md` をテンプレートと理由の説明に分ける。
- [x] T008 [Verify] 変更を検証する。

## 検証

- `mise run check-work-items`
- `mise run spec-diff` で、`principals/user` の要件の差分が本文の書き直しだけであり、ID の追加と削除がないこと。
- 書き直しの前後の行数と、変更一件あたりにエージェントが読む行数を比べる。
- `mise run verify`

## リスク

- 書き直しで、要件の義務が落ちる。
  要件ごとに書き直し前後の義務を突き合わせ、`spec-diff` で確かめる。
  義務を変える判断は (a)(b)(c) の分類として利用者に示し、書き直しに紛れ込ませない。
- 検査を外すと、旧形式の文書の質が下がっても気付けなくなる。
  外すのは廃止した形式を必須とする検査だけに限る。
- wi-99632 も書式文書を移動するので、同時に進めると衝突する。
  本項目を先に完了させ、wi-99632 は改訂後の文書を移す。
- wi-87813 が先に完了すると、`affected_spec` のパスが変わる。
  着手時にパスを更新する。

## 完了
- **Completed At**: 2026-10-04
- **Summary**:
  `mise run spec-diff -- main` の結果、規範の差分は REQ-IDMANAGEMENT-001、005、010、011、013、042、043、044、045、046、047、048、049、050 の本文の書き直しと、UserLifecycle の状態遷移の変更（状態遷移表（マトリクス形式）の追加）だけであり、要件の追加と削除はない。
  仕様フォーマットを規範だけの文書に改め、理由を `docs/development/specification-format-rationale.md` へ分けた（847 行から 593 行と 135 行）。
  `principals/user` の 4 文書は 719 行から 393 行になり、例は 38 件から 15 件になった。
  状態遷移表を作る過程で、`Deleted` の User に対する無効化、再有効化、削除の予約、復元の結果（404 `user_not_found`）という未記載の振る舞いが見つかり、表に書いた。
  (c) に分類した 7 件を wi-83020、wi-93929、wi-92312 として起票し、一括の書き起こしの work item 19 件を削除した。
- **Acceptance RED Evidence**:
  - **Test**: `tools/check/src/feature-layout.acceptance.test.ts`
  - **Requirement**: N/A: 文書の書式と検査の変更であり、製品の要件を変えない。
  - **Observed Failure**: 検査を変える前の期待のままでは、例のない要件、担保手段のない要件、機能の話題の索引、エラーの節を扱う 4 件のテストが失敗した。修正後の `mise run check-spec` は、エラーの節が残る `user/README.md` を `section エラー is not a section of a feature specification` で拒否した。
  - **Detection Reason**: 廃止した形式を必須とし続ける検査と、廃止した節を受け付け続ける検査の両方を、実際のワークスペースを組み立てて区別する。
- **Unit RED Evidence**:
  - **Test**: `tools/check/src/specification-rules.test.ts`、`tools/check/src/specification-doc.test.ts`
  - **Requirement**: N/A: 文書の書式と検査の変更であり、製品の要件を変えない。
  - **Observed Failure**: 上位の要件の欄の名前の変更と、機能の `design.md` の文書種別の変更で、それぞれのテストが旧い期待のまま失敗した。
  - **Detection Reason**: 欄の名前と文書種別の判定を、文字列だけから確かめる。
- **Change-Resistance Results**:
  エラーの節を語彙へ戻す誤実装は、`rejects an errors section` のテストが検出する。
  旧い欄の名前「上位の規則」を欄として読まない誤実装は、黙って要件文として扱われるので、`rejects the retired parent field name` のテストを加えて検出するようにした。このテストは修正と同時に加えたので、修正前の失敗は観測していない。
  例のない要件を再び拒否する誤実装は、`accepts a requirement that has no example in the appendix` が検出する。
- **Verification Results**:
  - `mise run verify` - 成功
