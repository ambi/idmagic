---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-03
priority: p1
depends_on: []
change_kind: docs
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
| 同じ事実を書く場所 | ユースケースごとの要約表、規則文、エラー表、例、`design.md` | 一つの事実を変えるたびに複数箇所を同期させる |
| 書式を定める文書 | 約 2,000 行（`SPECIFICATION_FORMAT.md`、`DOCUMENTATION_GUIDE.md`、`specification-first-workflow.md`） | エージェントが毎回読む量が大きく、規則と理由の説明が混在している |
| 要判断 | 15 件 | 未決の点が現在の仕様の中に散在している |

一つの事実を手で書く場所が多いので、追加のたびにエージェントが全箇所を書き足して整合させることになり、文書が乱れ、AI のコストも増える。

さらに、追加した「詳細な内部仕様」の多くは内部のものではない。
外部から観測できるのに、誰も決めないまま実装が偶然決めていた振る舞いである。
15 件の要判断を分類すると、次のとおりになる。

| 漏れの種類 | 件数 | 例 |
| --- | --- | --- |
| 同じ種類のユースケースが経路や Aggregate ごとに異なる | 9 | ユーザー名の大文字と小文字の扱いが User と Agent、Group で異なる。メールアドレスの重複を JIT は拒否し、管理者の作成は拒否しない。削除の予約は下流へ通知し、復元は通知しない |
| 状態とユースケースの組の結果が決まっていない | 1 | 停止した Agent を削除できない |
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
| ユースケース | 行為者が起こす一つの操作の単位。API の一つの操作か、別の Context から呼ばれる一つのコマンドに当たる | 管理者による User の作成。User の無効化 |
| 横断的要件 | 複数のユースケースや Aggregate に同じ内容で適用される要件。上位の文書で一度だけ書き、各機能は例外だけを書く | 状態を変えるユースケースは、すでにその状態なら成功を返しイベントを発行しない |
| 値オブジェクト | ドメイン駆動設計の Value Object。値の正規化、比較、妥当性の判定を一か所で定義する型 | ユーザー名（前後の空白を除き、大文字と小文字を区別しない、など） |
| 状態遷移表（マトリクス形式） | 行に状態、列にユースケースを並べ、すべてのセルに結果（遷移先、何もしない、拒否とその理由）を書く表 | `PendingDeletion` の行の「無効化」の列に「拒否（409 `user_pending_deletion`）」と書く |

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
  仕様に書けない振る舞いを生まない実装の指針（状態を変えるユースケースを冪等にする、読み取りのユースケースに副作用を加えない、順序と時刻への暗黙の依存を作らない、横断的要件を一か所で実装する）も加える。

### 一つの事実を一か所に書く

- 要件を EARS 形式で書く。
  一文に一つの義務を書く。
  型は、常に成り立つもの（Ubiquitous）、イベントを契機とするもの（Event-driven）、特定の状態の間に成り立つもの（State-driven）、望まない入力や状況への応答（Unwanted behavior）、特定の構成でだけ成り立つもの（Optional feature）の五つとする。
- ユースケースごとの要約表とエラー表を廃止する。
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
  対象は、識別子の正規化と比較、状態を変えるユースケースの冪等性、拒否が作用を起こさないこと、下流への通知とその失敗の扱い、監査を残す範囲とする。
- 値オブジェクト（ユーザー名、メールアドレスなど）の正規化、比較、一意性の範囲を、モデルの節で一度だけ定義する。
- 状態遷移の節に状態遷移表（マトリクス形式）を置き、すべての組に結果（遷移、何もしない、拒否とその理由）を書く。
  時間で起きる遷移は、その起点も表に書く。
- ユースケースごとに答える問いの一覧を `spec-change` スキルに置く。
  問いは、不正な入力、上限、失敗の観測者、記録、他の項目と Aggregate への波及、同じ種類のユースケースの別経路との一致である。

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

横断的要件は、既存の「上位の規則」の仕組み（`docs/design/application/api-guidelines.md` のような上位文書の見出しへリンクし、機能の要件は例外だけを書く）を使う。
システム全体の規約は `docs/design/application/api-guidelines.md`、Context に固有の規約は Context の `README.md` のモデルの節に置く案を第一とする。
新しい ID の種類は設けない。

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

未解決の問い（着手時に決める）：

- 横断的要件の置き場所。
  上の第一案で足りるか、Context の規約に専用の文書が要るかを、`principals/user` の書き直しで確かめて決める。
- EARS 形式の日本語の型の表記。
  `principals/user` の要件で型を試し、書式文書へ載せる表記を決める。
- 担保手段の欄を既存の全文書から一括で消すか、新しい方式へ書き直すときに消すか。

## タスク

- [ ] T001 [Docs] 書式文書と仕様先行の開発ワークフローを改訂する。
- [ ] T002 [Tooling] 廃止した形式を必須とする検査を外す。
- [ ] T003 [Docs] `spec-change` と `implement-work-item` のスキルに問いの一覧と分類の手順を加える。
- [ ] T004 [Docs] 設計ガイドラインに判断基準と実装の指針を加える。
- [ ] T005 [Spec] `principals/user` を新しい方式で書き直し、前後の行数を測る。
- [ ] T006 [Spec] `principals/user` の要判断を分類し、利用者の判断を反映し、(c) を起票する。
- [ ] T007 [Docs] `SPECIFICATION_FORMAT.md` をテンプレートと理由の説明に分ける。
- [ ] T008 [Verify] 変更を検証する。

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
