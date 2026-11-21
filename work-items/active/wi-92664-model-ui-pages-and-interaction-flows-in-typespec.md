---
status: in_progress
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-11-17
priority: p2
group: frontend-ui
depends_on: []
change_kind: docs
affected_spec:
  - { path: spec/modules/frontend/models.tsp, symbol: IdMagicUi.PageInfo }
  - { path: spec/modules/frontend/models.tsp, symbol: IdMagicUi.Area }
  - { path: spec/modules/frontend/models.tsp, symbol: IdMagicUi.Category }
  - { path: spec/modules/frontend/models.tsp, symbol: IdMagicUi.FormInfo }
  - { path: spec/modules/frontend/models.tsp, symbol: IdMagicUi.FieldInfo }
  - { path: spec/modules/frontend/models.tsp, symbol: IdMagicUi.TransitionInfo }
  - { path: spec/modules/frontend/models.tsp, symbol: IdMagicUi.RouteInfo }
  - { path: spec/modules/frontend/models.tsp, symbol: IdMagicUi.DestinationInfo }
  - { path: spec/modules/frontend/models.tsp, symbol: IdMagicUi.FlowInfo }
  - { path: spec/modules/frontend/models.tsp, symbol: IdMagicUi.FlowViewInfo }
  - { path: spec/modules/frontend/models.tsp, symbol: IdMagicUi.FlowLinkInfo }
  - { path: spec/modules/frontend/shared.tsp, symbol: IdMagicUi.ReadStates }
  - { path: spec/modules/frontend/shared.tsp, symbol: IdMagicUi.FormStates }
  - { path: spec/modules/frontend/shared.tsp, symbol: IdMagicUi.ListStates }
  - { path: spec/modules/frontend/shared.tsp, symbol: IdMagicUi.ImportStates }
  - { path: spec/modules/frontend/shared.tsp, symbol: IdMagicUi.Destinations }
  - { path: spec/modules/frontend/shared.tsp, symbol: IdMagicUi.PageData }
  - { path: spec/modules/frontend/pages.tsp, symbol: IdMagicUi.FrontendPages }
  - { path: spec/modules/frontend/pages.tsp, symbol: IdMagicUi.FrontendRouteNotes }
  - { path: spec/modules/authentication/pages.tsp, symbol: IdMagicUi.AuthenticationPages }
  - { path: spec/modules/authentication/pages.tsp, symbol: IdMagicUi.AccountAuthenticationPages }
  - { path: spec/modules/authentication/pages.tsp, symbol: IdMagicUi.RecoverPasswordFlow }
  - { path: spec/modules/identity-management/pages.tsp, symbol: IdMagicUi.AdminUserPages }
  - { path: spec/modules/identity-management/pages.tsp, symbol: IdMagicUi.AccountProfilePages }
  - { path: spec/modules/identity-management/pages.tsp, symbol: IdMagicUi.ManageUsersFlow }
  - { path: spec/modules/identity-management/pages.tsp, symbol: IdMagicUi.ImportUsersFlow }
  - { path: spec/modules/oauth2/pages.tsp, symbol: IdMagicUi.OAuthPages }
  - { path: spec/modules/application/pages.tsp, symbol: IdMagicUi.PortalApplicationPages }
  - { path: spec/modules/token-vault/pages.tsp, symbol: IdMagicUi.TokenVaultAccountPages }
---

# TypeSpec を基に UI のページ一覧と操作の詳細設計を整備する

## 動機

着手前の[フロントエンド設計のページの一覧](../../docs/design/application/frontend.md#ページの一覧)は、認証、アカウントポータル、管理コンソール、システムコンソールを四行にまとめていた。
個々のページ、同じ URL 内の操作、入力項目と遷移先を列挙しておらず、設計資料だけでは UI 全体を確認できない。
ページごとの入力バリデーション規則を漏れなく定義するためにも、先にページと入力項目を参照できる一覧が必要である。

## 対象範囲

- 既存の UI 全体を、ページ、ページ内の操作単位、入力項目、表示状態と遷移に分けて棚卸しする。
- UI Flows と IFML を参考に、TypeSpec を一次情報とするページモデルと、設計文書への派生表示を整備する。
- ページ単位の詳細設計と、既存経路との対応を確かめる検査を整備する。
- 今回扱う UI 設計では「ページ（page）」を使い、後続の入力バリデーション定義から参照する識別子を定める。

## 対象外

- 入力項目ごとのバリデーション規則の定義と実装。別の作業項目で扱う。
- ページの追加、URL の変更、UI の再配置と製品の振る舞いの変更。
- React のページ、ルーター、フォーム検証コード、クライアント SDK の生成。
- IFML 全体の実装、適合の主張、XMI の入出力、専用のモデリングツールの導入。
- 全文書と実装の「画面」「screen」の一括改名。アクセシビリティのスクリーンリーダーなど、別の意味で使う用語の変更。

## 設計

### 着手時に確定したモデルと仕様差分

製品の要件と公開 HTTP 契約は変更しない。
UI 設計の TypeSpec 宣言も仕様差分の対象になるため、追加した宣言を `affected_spec` に列挙する。
ページは領域ごとのモデルのプロパティで宣言し、`auto dec` によってページ、フォーム、入力項目、遷移、API 操作とプロパティの対応を付ける。
遷移先は別のプロパティへの参照であり、文字列の名前解決を別に実装しない。
共通フォームと状態はモデルとして共有し、入力項目は各利用ページの識別子で展開する。

抽出は `loadUiCatalog(root): Promise<UiCatalog>`、文書生成は `renderUiDocuments(catalog, { preview? }): Map<string, string>`、照合は `validateUiCatalog(catalog, routes, { allowUnassignedRoutes? }): string[]` とする。
ファイルと compiler を使う取得処理から、文書生成と照合の計算を分ける。
既存の文書生成と検査に実装を置き、製品モジュールの境界と実行時の依存は変更しない。
OMG の正式な XMI にある ViewContainer、ViewComponent、Event、Action、NavigationFlow、DataFlow と ParameterBindingGroup を確認した。
ページ内の表示要素、契機、API の処理、移動とデータ受け渡しに対応させ、業務状態機械と IFML の全メタモデルは導入しない。

ツールの検査では参照切れと経路の欠落、生成物の古さを故意に作る。
製品の Domain、Use Cases と HTTP の振る舞いは変更しないため、製品の受け入れ RED と単体 RED は対象外とし、実物の TypeSpec compiler と生成文書の境界で代替する。

### 参考資料と採用する考え方

| 資料 | 採用する考え方 | 今回への適用 |
| --- | --- | --- |
| [Ryan Singer による UI Flows の原文](https://signalvnoise.com/posts/1926-a-shorthand-for-designing-ui-flows) | 利用者に見える内容と、利用者が行う操作を分け、操作から次の表示へつなぐ | ページの名前だけを結ぶ図にせず、状態、操作、結果を識別する。メールや外部 IdP もフロー上の境界として扱う |
| [OMG の IFML 仕様](https://www.omg.org/spec/IFML)と[IFML の概要](https://www.ifml.org/) | 表示内容、利用者との対話、フロントエンドの制御をモデル化する | ViewContainer、ViewComponent、Event、Action、NavigationFlow、DataFlow とパラメーターの考え方を参照し、今回必要な要素へ絞る |
| [TypeSpec の emitter 拡張](https://typespec.io/docs/extending-typespec/emitters-basics/) | 対象となる宣言を選択し、同じ型情報から複数の成果物を生成する | 既存 compiler の API を使い、UI 設計の宣言から一覧、詳細とフローを導く。公開 API の生成とは対象を分ける |

IFML の正式なメタモデルとの対応は着手時に確認し、採用する概念と簡略化を記す。
今回作るのは IdMagic の UI 設計を記述するためのモデルであり、IFML の適合実装は対象外とする。

### ページとページ内の操作単位

利用者がある目的で開く表示単位を**ページ（page）**と呼ぶ。
URL を持つページを一覧の基本単位とし、タブ、ダイアログ、フォーム、一覧内の操作は、そのページに属する表示要素と操作として別に列挙する。
同じ URL 内に複数のフォームや手順があっても、一つの説明に畳み込まない。

認証、アカウントポータル、管理コンソール、システムコンソールの全ページを対象とする。
入口のホーム、コールバック、ステータス表示、共通のエラー表示、独立した URL を持たない再認証や確認のダイアログも、ページまたは共有要素として所在を記す。
外部 IdP、呼び出し元アプリケーション、メールなどは外部の到達先として識別し、IdMagic のページ数へ含めない。

ページ、フォーム、入力項目、操作と状態には、表示文言や実装のファイル名に依存しない安定した識別子を付ける。
入力項目は `pageId / formId / fieldId` で一意に参照できるようにする。
既存のページマーカーとの対応は調査するが、今回のために実装のマーカーを改名しない。

### 一覧とページの詳細

生成する一覧は一ページ一行とし、識別子、ページ名、所属する領域、利用者、正規の URL、入口、目的、詳細へのリンクを示す。
一覧と詳細の列挙を別々に手で保守しない。
詳細は次の情報を持つ。

| 項目 | 内容 |
| --- | --- |
| 入口と利用条件 | 通常の入口、直接アクセス、パスと検索パラメーター、認証状態、利用者、テナントの文脈。認可の規則は既存仕様への参照で示す |
| 表示内容 | 表示領域、一覧の列、詳細の情報、タブ、共有要素。レイアウトの座標や翻訳済みの文言は複製しない |
| フォームと入力項目 | 検索、絞り込み、ファイル選択、確認用入力を含む全項目の識別子、意味、入力の種類、値の型、初期値の取得元、表示と編集の条件、繰り返しの有無 |
| データとの対応 | 表示と入力が参照する API 操作、リクエストとレスポンスのプロパティ。API へ送信しない UI 固有の値、複数プロパティへ分解する値も区別する |
| 操作と遷移 | 起点のページと状態、契機、条件、実行する操作、渡すデータ、結果、遷移先。保存、キャンセル、戻る、削除、再試行と外部への遷移を扱う |
| 表示状態 | 読み込み、空、送信中、成功、入力不備、操作失敗、権限不足、認証切れ、部分失敗。適用しない状態には理由を付ける |
| 仕様との対応 | 機能要件と共通 UI 規則への参照。個別の記述が共通規則を適用するのか、既存仕様に基づく例外なのかを区別する |

カスタム属性や繰り返し入力は、具体的な属性名や件数を固定せず、入力項目のテンプレートと定義元を記す。
共通要素は一度定義し、各ページから参照して、入力項目の一覧では利用箇所まで展開できるようにする。
本項目では必須性、長さ、形式、数値範囲、項目間の依存などのバリデーション規則を新たに定義しない。
既存 TypeSpec の型とプロパティへの参照を保持し、後続の作業項目が制約の出典と UI 固有の規則を結び付けられるようにする。

遷移図は同じ宣言から生成し、利用目的ごとに分ける。
ページが変わらない操作結果も状態の変化として表し、外部への遷移では渡す情報と復帰先を示す。
業務オブジェクトのライフサイクルは既存の状態遷移定義を参照し、UI の状態として再定義しない。

### TypeSpec と設計文書の分担

UI の共通モデルとコンパイルの入口は `spec/modules/frontend/` に置き、ページの宣言は主担当モジュールの `pages.tsp` に置く。
UI 専用のエントリーポイントから各モジュールの宣言を読み込む。
UI の名前空間は HTTP サービスと分け、API のモデル、操作とプロパティを TypeSpec のシンボルとして参照する。
UI のための架空の HTTP 操作や、API 型をコピーした入力 DTO は作らない。
既存の API エントリーポイントと標準 OpenAPI emitter は UI の宣言を出力対象にせず、公開 API の生成結果を保つ。

TypeSpec の model、enum、union と参照で表現し、UI 固有のメタデータに必要な最小限の decorator だけを補う。
既存の文書サイトには compiler の型情報からモデルカタログを作る処理があるため、その取得方法を使って UI の宣言を抽出する。
本項目では汎用の外部公開ライブラリや独立した emitter パッケージを追加せず、既存の文書生成と検査の責務の中へ組み込む。
実行時のフロントエンドから設計の宣言や生成ツールへ依存を追加しない。

人が書く共通規則とモデルの読み方は、[フロントエンド設計](../../docs/design/application/frontend.md)と[ユーザーインターフェース設計](../../docs/design/application/user-interface.md)が扱う。
ページの宣言は主担当の機能モジュールに置き、詳細とモジュール別の一覧を `docs/modules/<module>/pages/` へ生成する。
利用者の目的から主担当を一つ決め、横断ページと共有の表示は Frontend が担当する。
モジュール名は宣言元の配置から導き、別の対応表やページごとの owner 注釈を追加しない。
全体とカテゴリ別の索引は `docs/design/application/pages/` に生成し、モジュールの詳細への参照を持たせる。
いずれも Git で追跡する。
利用者の領域、作業カテゴリ、個別ページの三段に分け、それぞれの索引から詳細を探せるようにする。
一つのページが複数モジュールの API を使う場合も、UI 上の作業の主担当を維持し、カテゴリと領域からの見つけ方を別に提供する。
利用手順の図はモジュールのページ一覧に置き、利用者が一つの目的を達成する表示と操作を選ぶ。
ページや内部状態を機械的に全件並べず、図に含める表示と操作を `flow`、`view` と `flowLink` で明示する。
`flowLink` はページの既存の操作を TypeSpec のプロパティとして参照し、起点、到達先と状態が一致することを検査する。
個別ページとカテゴリの索引には図を生成しない。
UI Flows の見える内容と操作、IFML の ViewElement と NavigationFlow、NN/g の Wireflows にある同じページ内の表示の変化を参照する。
塗り分けを増やさず、図の枠は表示、矢印は操作として、意味のある手順を白黒で表す。
派生文書の先頭で入力元と再生成コマンドを示し、手編集を禁止してドリフトを検査する。
文書サイトにも同じ内容を表示し、一覧から詳細、関連する要件と API へたどれるようにする。

この分担に合わせて、次の既存規約と索引を同じ作業で更新する。

- [文書ガイド](../../docs/formats/documentation-guide.md#一次情報文書に置かない内容)の「画面ごとの URL、フィールド、状態」の担当を、UI の TypeSpec 宣言と派生表示に改める。配置図と追跡する生成文書の例外も更新する。
- [TypeSpec の範囲](../../docs/formats/specification-format.md#typespec-の範囲)に UI 設計の宣言を加え、HTTP 契約や業務上の制約の担当との違いを記す。
- アプリケーション設計の索引と Frontend の文書索引からページ一覧を参照する。現在の四行の一覧と「完全な一覧の一次情報は経路ファイル」という説明は置き換える。

### 網羅性と検査の限界

実装の経路集合と、宣言したページの URL の集合を双方向に照合する。
レイアウト専用経路、リダイレクト専用経路、別名、テナント接頭辞の有無は区別し、ページを重複して数えない。
ページを描画しない経路も、除外理由または遷移先を持つ対応として残し、黙って検査対象から外さない。
設計にだけある未実装のページは、実装済みの一覧と区別した状態と根拠を要求する。
将来の作業項目が予定するページを一括で先取りして作らない。

機械検査は識別子の重複、未解決のページと API の参照、遷移の起点と終点の欠落、経路対応の欠落、生成文書のドリフトを検出する。
フォームと入力項目の完全性は、経路の一致だけでは確かめられない。
各ページの描画、条件分岐、共有フォームと設定に応じて増える入力を棚卸しし、入力項目と操作の対応をレビューする。
実装にしかない振る舞いは、既存の[分類手順](../../docs/development/specification-first-workflow.md#仕様にない振る舞いの分類)に従い、偶然の動作を新しい規範として固定しない。

### 比較した案

| 案 | 利点 | 採否と理由 |
| --- | --- | --- |
| Markdown の一覧と詳細を手で増やす | 導入が容易 | 採らない。入力項目、API 参照と遷移の整合を同時に保守しにくい |
| 経路や JSX から全情報を自動抽出する | 実装済み URL の列挙には向く | 経路の照合に限って使う。目的、条件付きの入力、外部への遷移と設計上の意図は抽出だけでは補えない |
| IFML のメタモデルと図式を全面導入する | 標準の概念と交換形式を使える | 採らない。必要な一覧と詳細に対して、導入と保守の範囲が広い |
| TypeSpec の UI 宣言から設計文書とフローを生成する | 既存 API の型を参照し、一覧と詳細を同じ入力から導ける | 採る。UI 固有の宣言と検査を必要な範囲に絞る |

## 計画

1. 全経路とページ内の操作を調べ、IFML との対応と、代表例を表現できる最小の TypeSpec モデルを固める。認証と MFA、一覧から詳細と編集、条件付き入力、確認と再認証、CSV の取り込みを代表例にする。
2. 担当と配置の規約を更新し、UI 専用コンパイル、型情報の抽出、文書生成と検査を接続する。公開 API の生成結果への影響を確認する。
3. 四領域の全ページへ宣言を広げ、共有要素、入力項目、状態、操作と外部への遷移を埋める。既存仕様と実装の不一致は個別に分類する。
4. ページ一覧と詳細を生成し、経路との照合、入力項目の棚卸し、文書サイトの表示を確認する。
5. ページ、フォームと入力項目の識別子を引き継ぐ、入力バリデーション規則の別作業項目を起票する。その項目は本項目へ依存し、必須性、形式、範囲、項目間条件、検証の担当と表示を定義する。

作る成果物、対象領域とバリデーションとの分担は、この項目で固定する。
TypeSpec の具体的な構文と decorator のシグネチャは、代表例の確認で決めてから全ページへの展開に進む。

### 途中コミットからの引き継ぎ（2026-11-17）

利用者の指示により、全面展開の途中で、再開できる状態をコミットする。
本項目は `in_progress` のままとし、完了への移動と最終コミットの手順はまだ実施しない。
その後、利用者から保守負荷について懸念が示されたため、未照合の全ページを同じ方式で書き足す作業はいったん止めた。
再開時は、以下の「保守負荷の見直し」を先に扱う。

#### 作業場所と利用者が確定した方向

- ブランチは `work-item/wi-92664`、worktree はリポジトリ直下の `.worktrees/wi-92664/`。着手時の基点は `425896ee2`。
- 元のチェックアウトには未追跡の本項目の起票版と別項目があったため、変更しない。以後はこの worktree 内の WI と実装を使う。
- 個別のページ詳細は主担当モジュールの `docs/modules/<module>/pages/` に置く。担当は利用者の目的で決め、使用する API の個数では決めない。
- 横断ページと共有表示は Frontend が担当する。領域・カテゴリの全体索引は、モジュールの詳細へリンクする。
- ページごとの操作フロー図は不要。モジュールのページ一覧に、主要な利用目的ごとの図を置く。
- 図は白黒を基調にする。箱は利用者に見える表示、矢印は操作を示す。共通の読み込み・失敗処理を機械的に並べない。
- 入力の必須性、形式、長さ、範囲、項目間条件の規則を今回追加しない。別 WI の起票は未実施。

#### 実装済みの範囲と進捗

全体の進捗は約35%と報告したが、これはタスク全体の概算である。
経路の照合率や、入力項目の網羅率を表す値ではない。
現在の生成結果は29ページ、描画しない経路の注記6件、実装の経路139件に対して未照合104件である。
未照合には独立したページ以外も含まれるため、残り104ページという意味ではない。

| 宣言元 | 定義済みページ |
| --- | --- |
| `spec/modules/authentication/pages.tsp` | ログイン、二要素認証、再設定依頼、パスワード再設定、本人のパスワード変更、本人のセキュリティ設定、本人の活動履歴、認可中の MFA 登録の8件 |
| `spec/modules/identity-management/pages.tsp` | 管理ユーザーの一覧・詳細・追加・編集・CSV 取り込み、本人のプロフィール・編集・メール・メール確認・データ取得の10件 |
| `spec/modules/frontend/pages.tsp` | サービス入口、本人の概要、管理ダッシュボード、コールバック、結果表示の5件。ほかに root、account、admin、system、system index、profile の6経路の注記 |
| `spec/modules/oauth2/pages.tsp` | 同意、デバイス認可、本人の同意と委任、本人宛て承認の4件 |
| `spec/modules/application/pages.tsp` | 本人のアプリケーションランチャーの1件 |
| `spec/modules/token-vault/pages.tsp` | 本人の外部サービス接続の1件 |

元の9ページ分は HTML を見ながら配置と図を調整した。
その後に追加した本人向けと入口の20ページは、宣言、コンパイルと生成を確認する段階まで進めたもので、入力と操作の完全性のレビューは未完了である。
特に API 操作の `@uses`、共通状態の適用、同じページ内の確認・キャンセル・失敗の表現は再確認する。
定義済みというだけで、棚卸し済み・全項目検証済みとは扱わない。

| タスク | 現在の状態 |
| --- | --- |
| T001 | UI Flows、IFML と Wireflows を調査し、代表例とモデルを実装。保守負荷を踏まえた記述量の確定が残る |
| T002 | 文書の担当、生成例外、モジュール配下の許可とナビゲーション、モジュールの構成説明と現在の担当モジュールの索引を更新。未定義のモジュールの索引は今後の追加時に扱う |
| T003 | UI 専用コンパイル、抽出、文書・図の生成と mise タスクを実装。2026-11-22 に `check-ui-pages` として通常検査へ組み込んだ |
| T004 | ID、遷移、フローと操作の対応、経路の双方向照合、ページマーカーと入力名の照合、ドリフト検査を実装し通常ゲートへ接続した。`--preview` の除去は全経路の宣言後 |
| T005 | 29ページの宣言を作成。104経路の分類と、全フォーム・条件付き入力のレビューは未完了 |
| T006 | 局所テストとプレビューの検証まで。最終差分の集約検査、公開 API の出力の比較、全件の入力・操作レビューは未実施 |
| T007 | 未着手。本項目に依存する入力バリデーションの WI を、識別子と API プロパティの参照を引き継いで起票する |

#### 保守負荷の見直しを先に行う

時間を使ったのは、初回のリバースエンジニアリングと、利用者による配置・図のレビューを反映する試作である。
ただし、現在の方式なら保守が軽いと確認できたわけではない。
実装と宣言の重複があり、経路の一致だけではフォームの追加漏れを検出できない。
このまま全件へ展開すると保守負荷が残る。

1. 手書きする値を棚卸しし、URL、API の型と項目など参照から導ける情報と、目的・入力の意味・表示条件など人が判断する情報を分ける。`route` と `path`、`@bind` とフィールドの型などの二重記述を減らせるか検討する。
2. 一つのページの項目追加・変更を例に、TypeSpec と実装の更新箇所を数え、保守の手順を確かめる。モデルを細かくすること自体を目標にしない。
3. JSX の通常の入力に加え、共有フォームと定義駆動の入力の対応をどう検査するか決める。自動抽出だけで目的や条件を補ったことにはしない。動的属性は定義の参照とテンプレートで扱う。
4. この分担を設計と生成ツールへ反映してから、全件へ広げる。利用者の保守負荷への懸念を、全件追加を急ぐことで先送りしない。

先に報告した残り4〜6時間は、従来の方式で全面棚卸しする場合の概算であり、この見直し後の確約ではない。

#### 保守負荷の見直しの結果（2026-11-22）

最新の main（`086a2ec2d`）へ rebase した後、上の1〜2を実施した。
利用者と合意した方針で、参照から導ける値を宣言から除いた。

| 除いた手書きの値 | 導出元と検査 |
| --- | --- |
| ページの `route` と `path` | ページ ID を実装の `PageMarker` の kind と一致させ、その kind を静的に描画する経路から導く（`resolveRoutes`）。kind を式で決める経路と、同じ kind を持つ経路が複数ある場合だけ `route` を書く。明示した経路のマーカーが別の ID を指せば拒否する |
| 束縛した入力の型と、束縛先と同じ `mapping` | `@bind` の先の API プロパティから型を導き、宣言の型は `unknown` と書く。束縛しない入力では `mapping` を必須にする |
| 入力、フォームと遷移の既定値 | 入力の `initial` は「空」、入力とフォームの `condition` は「常時」。遷移は ready から ready への無条件の移動で、渡すデータはない。ページ内の更新（`mode: "data"`）には `result` を必須にする |
| 一覧の検索条件の二重宣言 | URL と同期する検索項目はフォームにだけ書き、初期値で URL から取ることを示す |
| `@flowLink` の操作の型参照 | `PageData` で宣言したページでは `::type.actions` がテンプレートを指して解決できなかった。起点の表示のページ内の操作名（`action`）で参照し、存在と到達先の一致を生成時に検査する |

29ページの宣言は 64,024 バイト・608 行から 53,624 バイト・451 行に減った。
ページの `route` が残るのは、kind を認可トランザクションから決める login、totp、consent、mfa-enrollment の4件である。
照合で実装との不一致を2件見つけ、実装のマーカーに合わせて宣言の ID を `admin-home` から `admin-dashboard`、`account-password` から `change-password` に改めた。
一つのページへ入力を足すときの更新は、実装の JSX と、ラベルと入力の種類と束縛先を書いた TypeSpec の1行、再生成である。

3の入力の照合は、利用者が「JSX の静的な `name` を宣言と照合する」案を選んだ。
実装の `frontend/src/` のテスト以外にあるフォーム部品381個のうち、静的な `name` を持つのは28個だった。
静的な `id` を持つ部品を含めても210個前後で、`id` は `user-editor-username` のように DOM の識別子として項目名と別に命名されているため、照合に使えない。
利用者の判断により、照合できない部品は生成文書へ列挙し、全部品へ `name` を付ける作業は [wi-80172](wi-80172-name-every-frontend-form-control.md) へ分けた。

`tools/build-site/src/ui-inputs.ts` が次の照合を行う。

- 宣言したページの経路ファイルから、`frontend/src/routes/` と `frontend/src/features/` の静的な import をたどる。共通部品と辞書はたどらない。
- 到達したファイルのフォーム部品と静的な `name` を ast-grep で取る。
- 一つの部品を複数のページが描画しうるため（認証フローの出し分け、一覧が詳細のダイアログを import するなど）、部品のファイルへ到達するページのどれかがその名前を宣言していれば受け入れ、どのページも宣言していない名前を拒否する。
- 照合する名前は `@field` の `name`、なければ束縛先のプロパティ名、なければ宣言のプロパティ名とする。回復コードの入力が `code` へ束縛され実装では `recovery` と名付けられていたため、この上書きを加えた。
- 静的な `name` を持たない部品は、ページ詳細の「実装と照合していない入力」にファイルと件数を出す。行番号は無関係な編集で生成文書を古くするため出さない。

ファイル単位の帰属のため、同じファイルを描画する別ページの宣言で受け入れてしまう見逃しは残る。
import の探索は Bun の `scanImports` を試したが、一部の経路ファイルで JSX の警告を例外として投げたため、ast-grep に揃えた。

経路、マーカー、import と入力の抽出は `ast-grep` CLI を呼ぶ。
`@ast-grep/napi` を tools の依存へ固定する変更は、実行環境の権限判定で拒否されたため採らなかった。
代わりに、ほかのゲートのツールと同じく `mise.toml` の `[tools]` へ `aqua:ast-grep/ast-grep` 0.51.3 を固定し、CI でも mise から導入されるようにした。

#### 通常ゲートへの接続（2026-11-22）

`mise run check-ui-pages` が `write-ui-pages.ts --preview --check` を実行し、`check` と CI の `check-ci-spec` に含めた。
`verify-changed` は変更があれば必ず `check` を実行するため、同じ検査が走る。
全経路を宣言するまでは `--preview` で未宣言の経路だけを許し、宣言済みのページの参照、経路とマーカー、宣言にない入力名、生成文書の古さは拒否する。
全経路を宣言した時点で `--preview` を外し、生成文書のプレビュー注記を除く。
生成文書へ一行を足すと `check-ui-pages` が古い生成文書として拒否し、戻すと通ることを確かめた。

残る作業は、未照合104経路のページの宣言、`--preview` の除去、最終レビュー、要約と完了への移動である。
104経路への展開では、各ページ詳細の「実装と照合していない入力」を、入力項目のレビューの対象にする。

#### 実装の入口と既知の残作業

- `spec/modules/frontend/main.tsp` は UI 専用の入口。`tspconfig.yaml` の `auto-decorators` が必要。HTTP 用の `spec/main.tsp` から UI の宣言を import しない。
- `models.tsp` が decorator と情報の型、`shared.tsp` が共通状態、`PageData` と値の共有を持つ。主担当は宣言元の `spec/modules/<module>/pages.tsp` から抽出する。
- `tools/build-site/src/ui-pages.ts` が compiler からの抽出、経路の取得と整合検査を持つ。API の参照と遷移先は TypeSpec が解決する。`routeDeclarations` と `ui-inputs.ts` は `ast-grep` CLI を呼び、そのバージョンは `mise.toml` の `[tools]` が固定する。
- `ui-page-documents.ts` が Markdown の生成、`ui-flow.ts` が主要フローの Mermaid を担当する。4表示までは横、5表示以上は縦に配置する試作。全件の図を自動生成するのではなく、`flow`、`view`、`flowLink` で主要手順を選ぶ。
- `write-ui-pages.ts` が書き出しと `--check`。`--preview` は未照合経路だけを許し、宣言内の不正な参照・状態・操作とフローの不一致は拒否する。現時点で通常モードは未照合の経路のため失敗する。これを成功扱いにしない。
- `tools/build-site/src/render.ts` がモジュール配下のナビゲーション、表と白黒の図を描画する。`tools/workspace/src/document-layout.ts` が配置を許可する。
- `tools/check/src/specification-doc.ts` も新しい生成文書を散文として受け付ける。配置の許可だけを変えると `check-spec` の文書検査で拒否されたため、分類と回帰テストを加えた。
- 生成 Markdown は追跡するが、HTML の `site/` は追跡しない。全生成文書に途中プレビューの注記がある。`build-site` 自体は UI の再生成・ドリフト検査をまだ行わない。
- 通常の `check`、CI の `check-ci-spec` と `verify-changed` から `check-ui-pages` が走る。全経路の宣言までは `--preview` 付きであり、このまま完了にしない。
- 今回の通常 API の生成入口は変えていないが、OpenAPI の比較を最終差分で実行する。製品の `frontend/src/` と `backend/` の振る舞いは変更していない。

未照合の内訳は、本人用・管理用 API トークン、グループ、Agent、アプリケーション管理、認可、MCP、外部 IdP、Entra、ディレクトリ連携、ライフサイクル、設定、テナント、ワークロード ID、監査、ジョブ、鍵などである。
`frontend/src/routes/` の経路と `frontend/src/features/` の表示・共有フォームを合わせて調べる。
既存の `pages.tsp` を一律にコピーして内容を埋める作業を始める前に、上の保守方式を見直す。

#### 再開コマンド

worktree 内で `mise tasks` と `mise run brief -- wi-92664 --target spec/modules/frontend --target tools/build-site --target docs/modules` から再開する。
依存がなければ `mise run setup` を実行する。

```sh
mise run compile-ui-spec
mise run generate-ui-pages -- --preview
mise run generate-ui-pages -- --preview --check
mise run build-site
mise run serve-site -- 8086
```

HTML の入口は `http://localhost:8086/docs/design/application/pages/index.html`。
代表的な図は `/docs/modules/identity-management/pages/index.html` と `/docs/modules/authentication/pages/index.html`、詳細は `/docs/modules/identity-management/pages/admin-user-create.html`。
サーバーの起動状態と一時ファイルには依存せず、必要なら起動し直す。
ブラウザーの目視確認は別途必要である。

```sh
mise run test-tools-file -- build-site/src/ui-pages.test.ts build-site/src/render.test.ts check/src/canonical-document-set.test.ts
mise run typecheck-tools
mise run lint-tools
mise run check-spec
mise run check-links
mise run check-work-items
mise run spec-diff
```

`--preview` を外した `mise run generate-ui-pages` は、未照合の経路の一覧を返して失敗する。
全経路を対応付けてから通常モードで再生成し、プレビュー注記を除く。
その後、通常ゲートの配線、必要な `verify-changed`、バリデーション WI の起票、最終レビュー、要約・完了への移動を行う。
途中コミットと最終コミットの整理は、利用者の追加指示と開発サイクルに従う。

localhost の待受けとブラウザー起動を要する検査は初回からサンドボックス外で実行する。
この環境では mise の導入・キャッシュの書き込みでも権限が必要だった。
権限エラーを検査対象の不具合や成功へ読み替えない。

#### 途中コミット時点の検証結果

| 検査 | 結果と限界 |
| --- | --- |
| `compile-ui-spec` | 成功。29ページと API プロパティ・操作への参照をコンパイルできる |
| `generate-ui-pages -- --preview` と `--preview --check` | 成功。29ページ・139経路・未照合104経路。定義済みの参照・状態と主要フローの整合、生成文書の一致を確認 |
| `generate-ui-pages`（通常モード） | 未照合104経路を列挙して終了コード1。途中成果を完成扱いにしないことを確認。網羅性のゲートは未達 |
| 局所テスト4ファイル | `ui-pages.test.ts`、`render.test.ts`、`canonical-document-set.test.ts`、`specification-doc.test.ts` の141件成功 |
| `typecheck-tools`、`lint-tools` | 成功。既存の Biome 設定スキーマのバージョン差の情報表示があるが、lint の失敗ではない |
| `check-spec` | 成功。UI の文書配置・散文の分類、既存の仕様とサイトを検査。ただし、このタスクだけでは UI 専用コンパイルと棚卸しの完全性を保証しない |
| `build-site` | 成功。455文書などから1509 HTML ページを生成。Mermaid の構文とサイト内参照も検査 |
| `check-links`、`check-work-items` | 成功。途中記録は `in_progress` のまま |
| `spec-diff` | 追加した UI 宣言を確認し `affected_spec` へ追記 |

`途中プレビューでは未棚卸しの実装経路だけを許し、壊れた宣言は拒否する` と `UI の索引とモジュールのページ詳細を散文として検査する` のテストは、変更前に失敗を確認してから成功させた。
前者は入力不整合を無視するプレビューの危険、後者は配置だけを許して文書の種別判定を更新し忘れる故障を対象にする。
全ページの項目の一致、通常 UI 検査の配線、最終 `verify-changed` と製品の E2E は未実施である。
製品の振る舞いを変更していないため、この途中コミットでは製品の E2E を追加の証拠に選んでいない。

## タスク

- [ ] T001 [Design] IFML の採用概念と TypeSpec のモデルを代表例で確定する。
- [ ] T002 [Docs] 一次情報の担当、生成物の配置と用語を更新し、ページ一覧への索引を整備する。
- [ ] T003 [Tooling] UI 専用のコンパイルと抽出、ページ一覧と詳細とフローの生成を既存ツールへ組み込み、再生成と検査の mise タスクを用意する。
- [ ] T004 [Tooling] 経路の双方向照合、参照と識別子の検査、生成文書のドリフト検査を加える。
- [ ] T005 [Docs] 全領域のページ、共有要素、フォームと入力項目、状態と遷移を定義し、設計文書を生成する。
- [ ] T006 [Verify] 機械検査の故障検出と公開 API の生成結果を確認し、ページ内の入力と操作の網羅性をレビューする。
- [x] T007 [Plan] 入力バリデーション規則を定義する別の作業項目を、識別子と出典を引き継いで起票する。[wi-24528](wi-24528-declare-ui-input-validation-rules.md) として起票した。束縛先の API プロパティの制約は導出し、UI 固有の規則だけを宣言して、実装の `formSchema` と照合する。

## 検証

起票時は `mise run check-work-items` と `mise run check-links` で記録の形式と参照を検査する。
実装時は次を対象とする。

- `mise run compile-spec` と UI 専用のコンパイルタスク：UI の宣言と API シンボルの参照が解決できること、既存 OpenAPI の生成結果が変わらないことを確認する。
- `mise run test-tools-file -- <対象テスト>`：重複 ID、存在しない参照、未記載の実装経路、対応先のない設計上の経路、壊れた遷移、古い生成文書を拒否する。別名、共有要素と条件付き入力の代表例も扱う。
- ページ用に追加する検査タスク：全ページの経路対応と生成文書の一致を確認し、通常の `check` からも実行する。
- `mise run check-spec` と `mise run check-links`：宣言の分離、更新した文書の配置と索引、生成した詳細へのリンクを検査する。
- `mise run check-site`：生成したページ一覧、詳細、遷移図と API 参照が文書サイトでも読めることを確認する。
- `mise run verify-changed -- <base>`：生成と検査のコードに変更があるため、実際の差分に応じたツールの型検査、lint とテストを含む最終検査を実行する。
- 四領域の全ページについて、条件分岐と共有要素を含む入力項目、操作と表示状態を設計と突き合わせる。経路集合の一致だけを網羅性の証拠にしない。

## リスク

- モデルが細かすぎると、ページを変更するたびに宣言の保守が増える。UI の振る舞いと入力の把握に必要な情報へ絞り、DOM 構造、コンポーネント構造と文言を複製しない。
- 設計の宣言と実装が別々に古くなる。経路と生成物のドリフトを検査し、フォームと条件付き入力はページ変更時の確認対象にする。
- 棚卸しで見つかった欠落を補う過程で、バリデーションや認可の新しい規則まで混ぜやすい。規則の変更は別項目へ切り出し、本項目では既存仕様への参照と項目の所在を定める。
- UI の宣言が公開 API の出力へ混入する可能性がある。コンパイルの入口と名前空間を分け、既存 OpenAPI の差分で検出する。
