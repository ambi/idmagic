---
status: pending
authors: [tn]
risk: medium
reversibility: reversible # 名前を変える API はまだリリースしていない
created_at: 2026-11-04
priority: p2
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/modules/oauth2/features/resource-access.md, requirement: REQ-OAUTH2-086 }
  - { path: docs/modules/oauth2/features/admin-access.md, requirement: REQ-OAUTH2-087 }
  - { path: docs/modules/oauth2/features/admin-access.md, requirement: REQ-OAUTH2-088 }
  - { path: docs/modules/identity-management/features/agent.md, requirement: REQ-IDMANAGEMENT-105 }
  - { path: spec/modules/oauth2/main.tsp, symbol: ReportResourceTools }
  - { path: spec/modules/oauth2/main.tsp, symbol: GetAdminMcpResourceServerReportedTools }
  - { path: spec/modules/oauth2/models.tsp, symbol: IdMagic.Contract.ResourceToolReportRequest }
  - { path: spec/modules/oauth2/models.tsp, symbol: IdMagic.Contract.ReportedTool }
  - { path: spec/modules/oauth2/models.tsp, symbol: IdMagic.Contract.ReportedToolModification }
  - { path: spec/modules/oauth2/models.tsp, symbol: IdMagic.Contract.ReportedToolChanges }
  - { path: spec/modules/oauth2/models.tsp, symbol: IdMagic.Contract.ReportedToolsResponse }
---

# エージェントと MCP リソースサーバーの管理画面をほかの管理画面にそろえ、ツールの一覧を「カタログ」と呼ぶ

## 動機

エージェントの登録と MCP リソースサーバーの登録の画面は、デモの手順に合わせて短期間で作ったため、ほかの管理画面にない表示や、意味の伝わらない文言が残っている。
利用者から次の指摘を受けた。

- 入力欄の補足に「あなたが決める」「デフォルト値。変更できます」の印が付いている。ほかの画面にはなく、読み手の判断に役立たない。
- ツールの行の「対象ごとの権限も確認する」が何を有効にするのか分からない。
- 登録後の「登録しました」画面で、クライアント ID とクライアントシークレットをコピーできない。「Gateway の設定の例」をどう使うのか分からない。「報告されたツールを公開する」ボタンの意味が分からず、「報告された」は不具合の報告のような否定的な印象を与える。
- MCP リソースサーバーの状態が `Active`、`Disabled` と英語のまま表示され、ユーザーやエージェントの「有効」「無効」と異なる。一覧もカード型で、ほかの一覧の表と構成が異なる。

調べると、ほかにも次の不統一があった。

- 日本語の文言に「Agent」と「エージェント」、「Gateway」と「ゲートウェイ」が混在する。
- 同じものを、MCP リソースサーバーの画面は「MCP リソースサーバー」「リソースサーバー」、エージェントの画面は「接続先」と呼ぶ。
- 評価用のクライアントについて「束縛」、エージェントの認証用クライアントについて「紐付け」と、同じ操作を別の語で呼ぶ。
- 作成画面の「一覧に戻る」の置き方が、エージェント（本文の上のアイコンボタン）と MCP リソースサーバー（ヘッダーのラベル付きボタン）で異なる。
- 登録結果のカードの余白（`p-4` と `p-6`）と、結果に含める値の見せ方（`CopyableValue` と素の `code`）が異なる。

「報告」の語は画面だけの問題ではない。
API の `reported-tools` と `ReportedTool` も、英語で「通報された」と読める余地がある。
一方、Gateway の設定は同じものを `mapping.catalog` と呼んでいる。
API がまだリリースされていない今なら、画面、API、コードの語を一度に「カタログ」へそろえられる。

## 対象範囲

対象の画面は `frontend/src/features/admin-agents/` と `frontend/src/features/admin-mcp-resource-servers/`、および両者から開く `frontend/src/features/admin-authorization/` のエージェント向け画面である。

1. 値の出所の印（`OriginTag`、`InputOrigin`、辞書の `originYou`、`originDefault`）を管理画面から取り除く。補足の本文は残す。
2. ツールの行の「対象ごとの権限も確認する」を、効果が分かる表記と補足に替える（設計の 2）。
3. MCP リソースサーバーの登録結果を次のように直す（設計の 3）。
   - クライアント ID、クライアントシークレット、Gateway の設定を `CopyableValue` またはコピー付きのブロックで示す。
   - 設定の例に、何のファイルで、どこに置き、何を書き換えるかの説明を添える。
   - 次の操作のボタンと見出しから「報告された」をなくす。
4. 編集画面の「Gateway から報告されたツール」の節と、その周辺の文言を同じ語彙に直す。
5. MCP リソースサーバーの状態を「有効」「無効」で表示し、選択肢も辞書から引く。
6. MCP リソースサーバーの一覧を、エージェントの一覧と同じ表と詳細カードの構成にする（設計の 5）。
7. エージェントと MCP リソースサーバーの画面の用語を、設計の 6 の表にそろえる。英語の辞書も同じ対応で直す。
8. 作成画面の戻る操作と、登録結果のカードの余白と値の表示をそろえる。
9. 文言の変化に合わせて、REQ-OAUTH2-087、REQ-OAUTH2-088、REQ-IDMANAGEMENT-105 の表現と、`docs/development/ai-agent-demo.md` の画面の手順を直す。
10. 変わった文言と構成を参照する単体テストと E2E を直す。
11. ツールの一覧の API、型、コード、テーブルの名前を「報告」から「カタログ」へ改める（設計の 8）。仕様文書の「報告」も同じ語に直す。

## 対象外

- AI エージェントのデモ画面（`mise run demo-agent`）の「あなたが決める」「デモが渡す」「前の手順の結果」の印。手順の値を写す画面であり、印が手順の理解に役立つため残す。
- `state` の値 `Active`、`Disabled` の変更。保存値と API の列挙はそのままにし、画面の表示だけを変える。
- カタログの API の振る舞いの変更。名前だけを変え、受け付ける値、上限、応答の意味、差分の規則は変えない。
- 旧名の API を残す互換の期間。リリース前で外部の利用者がいないため、旧名は一度に取り除く。
- ユーザー、グループ、アプリケーションなど、エージェントに関わらない画面の `StatusBadge` の共通化。重複は残るが、本項目の対象の画面だけで表示を合わせる。
- MCP リソースサーバーの一覧のページングとサーバー側の検索。件数が少なく、API がページングを持たないためである。

## 設計

### 1. 値の出所の印

`OriginTag` と `InputOrigin` を削除し、辞書から `originYou` と `originDefault` を取り除く。
必須かどうかはほかの画面と同じくラベルの必須表示で、変更できるかどうかは入力欄が無効かどうかで示す。
初期値があることは入力欄の値そのものが示すため、印の代わりの文言は加えない。

### 2. 対象を限定するツール

このチェックを有効にすると、ツールの定義に `resource_id_argument` が入る。
エージェントの割当で、そのツールを使える対象（文書など）を ID で選ぶようになり、実行時は引数の対象 ID が割当の範囲にあるかを確かめる（REQ-OAUTH2-083）。
管理者にとっての効果は「このツールを、対象を選んで許可するツールにする」ことである。

ラベルと補足を次に替える。

| 項目 | 現在 | 変更後 |
| --- | --- | --- |
| チェックのラベル | 対象ごとの権限も確認する | 許可する対象をエージェントごとに限定する |
| 補足（新設） | なし | 有効にすると、エージェントの「利用する MCP リソースサーバー」で、このツールを使える対象を ID で選びます。実行時は、下の引数の値が選んだ対象に含まれるときだけ許可します。 |
| 引数のラベル | 対象 ID を読む引数 | 対象 ID を渡す引数 |

エージェント側の表示も同じ語にそろえる。
`agentResources` の `target`（対象の制限）と `targetRequired`（対象ごとの権限も必要）は、「対象の限定」と「対象を選んで許可」に替える。

採らない案：チェックをやめて「接続先全体で許可」「対象を選んで許可」の二択のラジオにする。
意味は伝わりやすいが、行ごとの縦の高さが増え、ほとんどのツールで使わない選択を常に見せることになる。
補足付きのチェックで足りると判断した。

### 3. 登録結果

MCP リソースサーバーの登録結果は、エージェントの登録結果と同じ構成にする。

| 要素 | 変更後 |
| --- | --- |
| 見出し | 登録しました（変更なし） |
| 導入 | Gateway の配置先へ次の値を渡します。クライアントシークレットはこの画面でだけ表示します。 |
| クライアント ID | `CopyableValue` |
| クライアントシークレット | `CopyableValue`。ラベルの下に「この画面を離れると再表示できません。」を添える |
| 設定 | 見出しを「Gateway の設定ファイル（YAML）」にし、コピーボタンを付ける。説明に「`idmagic-mcp-gateway` の設定ファイルとして保存します。`upstream` を保護する MCP サーバーの URL に、`client_secret_file` のファイルにクライアントシークレットを書きます。」を置く |
| 次の手順 | 「Gateway を起動すると、MCP サーバーのツールがこの画面の編集に表示されます。ツールは公開するまで実行されません。」 |
| 主ボタン | 「ツールを公開する」。編集画面のツールの節へ移る |
| 副ボタン | 一覧に戻る（変更なし） |

設定の複数行をコピーするため、`CopyableValue` に複数行を `pre` で示す形を加えるか、`pre` の横に同じコピー操作を置く部品を足す。
既存の `CopyableValue` の一行の表示は変えない。

採らない案：設定ファイルをダウンロードさせる。
シークレットを含まない設定であり、コピーで足りる。
ファイルの保存先は環境ごとに異なるため、ダウンロードしても置き直しが要る。

### 4. 「報告されたツール」の言い換え

画面では、Gateway が MCP サーバーから取得したツールの一覧を「MCP サーバーのツール」と呼ぶ。
API と仕様の語は設計の 8 で「カタログ」にするが、画面では管理者が知っている MCP サーバーを主語にする方が、何の一覧かが伝わる。

| 辞書のキー | 現在 | 変更後 |
| --- | --- | --- |
| `reportedHeading` | Gateway から報告されたツール | MCP サーバーのツール |
| `reportedNone` | まだ報告がありません。… | まだツールの一覧を受け取っていません。Gateway をカタログ参照（`mapping.catalog: true`）で起動すると表示されます。 |
| `reportedAt` | 最終報告：{time} | 一覧の受信：{time} |
| `reportedIncomplete` | 報告は一部です。… | 一覧は一部です。MCP サーバーからすべてを取得できませんでした。 |
| `reportedMissing` | 接続先に見つからない公開中のツール：… | MCP サーバーに見つからない公開中のツール：{names}。自動では削除しません。 |
| `reportedLoadFailed` | 報告を読み込めませんでした。 | ツールの一覧を読み込めませんでした。 |
| `resultEdit` | 報告されたツールを公開する | ツールを公開する |

辞書のキーは設計の 8 に合わせて `catalog*` へ改める。
REQ-OAUTH2-087 の「接続先に見つからない」の引用は、画面の文言の変更に合わせて直す。

### 5. 状態と一覧

状態は、エージェントと同じ `StatusBadge` の見た目で「有効」「無効」と示す。
エージェントの `StatusBadge` は `killed` を含み型が異なるので、MCP リソースサーバー用の小さな関数を辞書の `statusActive`、`statusDisabled` で作る。
編集画面の状態の選択肢も同じ辞書から引く。

一覧は、エージェントの一覧と同じ構成にする。

- `Card` の中の表に、表示名、リソース URI、状態、公開中のツールの数を列で示す。
- 行を選ぶと右の詳細カードに、スコープ、評価用のクライアント ID、操作（利用するエージェント、編集、削除）を示す。
- 空の一覧は、エージェントの一覧と同じアイコン付きの空状態にし、追加の操作を文章で示す。

採らない案：カードのまま状態の文言だけを直す。
指摘は見せ方の不統一も含み、[管理コンソール](../../docs/design/application/user-interface.md#管理コンソール)の「一覧の表で検索、状態、主な権限を示す」にも合わないため採らない。

### 6. 用語

| 対象 | 日本語の画面 | 英語の画面 | 採らない語 |
| --- | --- | --- | --- |
| `Agent` | エージェント | agent | Agent（日本語の文中） |
| MCP Gateway | Gateway | gateway | ゲートウェイ |
| `McpResourceServer` | MCP リソースサーバー。文脈から明らかな同じ画面の中では「リソースサーバー」 | MCP resource server | 接続先 |
| クライアントとエージェントまたはリソースサーバーの対応 | 紐付け | bind | 束縛 |
| エージェントの「利用する接続先」 | 利用する MCP リソースサーバー | MCP resource servers | 利用する接続先 |
| 割当 | 割り当て（名詞は「割当」） | assignment | — |

「接続先」をやめるのは、管理画面のナビゲーションの名前（MCP リソースサーバー）と、エージェントの画面から開く先の名前が一致せず、同じものと分からないためである。
「Gateway」は、`docs/modules/mcp-gateway/` と設定ファイルの名前に合わせ、製品の構成要素の名前として原表記を使う（[文章の言語](../../docs/development/writing-language.md)）。

### 7. 戻る操作と結果のカード

作成画面は、ユーザーとエージェントの作成画面と同じく、本文の上に戻るアイコンボタンを置く。
編集画面と詳細画面は、ヘッダーの操作にラベル付きの「一覧に戻る」を置く（現状のまま）。
登録結果のカードの余白は `p-6` にそろえる。

### 8. ツールのカタログの命名

Gateway が送り、IdMagic が保存し、管理者が読むツールの一覧を「ツールのカタログ」（tool catalog）と呼ぶ。
Gateway の設定 `mapping.catalog` が既にこの語を使っており、設定と API の語が一致する。
「最新の一覧で前回を置き換える」という保存の意味にも合う。

| 種類 | 現在 | 変更後 |
| --- | --- | --- |
| 管理 API の経路 | `GET /api/admin/v1/mcp-resource-servers/{id}/reported-tools` | `GET /api/admin/v1/mcp-resource-servers/{id}/tool-catalog` |
| 管理 API の操作 | `GetAdminMcpResourceServerReportedTools` | `GetAdminMcpResourceServerToolCatalog` |
| Gateway の送信の操作 | `ReportResourceTools` | `PutResourceToolCatalog` |
| Gateway の送信の経路 | `POST /resource-access/tools` | 変えない。「報告」を含まず、評価の入口と同じ `resource-access` の下にあるため |
| 送信の要求 | `ResourceToolReportRequest` | `ResourceToolCatalogRequest` |
| 応答 | `ReportedToolsResponse` | `ToolCatalog` |
| ツール | `ReportedTool` | `CatalogTool` |
| 差分 | `ReportedToolChanges`、`ReportedToolModification` | `ToolCatalogChanges`、`CatalogToolModification` |
| 応答の時刻の項目 | `reported_at` | `received_at`。IdMagic が受け取った時刻であり、Gateway が申告する時刻ではないため |
| テーブル | `mcp_tool_reports`（列 `reported_at`） | `mcp_tool_catalogs`（列 `received_at`）。新しいマイグレーションで改名し、適用済みのマイグレーションは書き換えない |
| Go と TypeScript の識別子、ファイル名 | `mcp_tool_report*.go`、`report_resource_tools.go`、`backend/mcpgateway/internal/report_http`、`getMcpReportedTools`、`ReportedToolsPanel` など | 同じ規則で `catalog` に改める |
| 仕様文書の語 | 報告、報告する | カタログ、カタログを送る |

`POST` のままにするのは、本文が `application/x-www-form-urlencoded` で、評価の入口と同じ認証と流量の規則を共有するためである。
操作名だけを、置き換えの意味を表す `Put` にする。

採らない案：

- `discovered-tools`。IdMagic が探しに行く印象を与え、REQ-OAUTH2-086 の「IdMagic から取得に行かない」判断と食い違う。
- `upstream-tools`。Gateway の内部の構成を管理 API の名前に出すことになる。Gateway を使わずにサーバー自身が送る構成でも意味が通る名前にしたい。

## 計画

1. 仕様を直す。TypeSpec の改名、REQ-OAUTH2-086 と REQ-OAUTH2-087 の語、REQ-OAUTH2-088 と REQ-IDMANAGEMENT-105 の画面の表現を直し、REQ-OAUTH2-088 に結果をコピーできることを加える。規則の意味は変えない。
2. 生成物を更新し、バックエンド、Gateway、テーブルの名前を改める。振る舞いを変えないので、既存のテストが改名後も通ることで確かめる。
3. 辞書を設計の表に従って直す。印を取り除く。
4. 登録結果、ツールの行、状態の表示を直し、単体テストを先に赤にしてから合わせる。
5. MCP リソースサーバーの一覧を表と詳細カードへ作り替える。
6. E2E とデモの手順文書を新しい文言に合わせる。

未解決の問いはない。
言い換えの文言は設計の表で決めており、実装中に読みにくければ表を直してから実装する。

## タスク

- [ ] T001 [Spec] REQ-OAUTH2-087、REQ-OAUTH2-088、REQ-IDMANAGEMENT-105 の画面の表現を直し、登録結果の値をコピーできることを REQ-OAUTH2-088 に加える。
- [ ] T002 [Docs] `docs/development/ai-agent-demo.md` の画面の手順の文言を新しい表記に合わせる。デモ画面の印の記述は残す。
- [ ] T003 [App] 値の出所の印を取り除く。
- [ ] T004 [App] ツールの行の対象の限定の表記と補足を直し、エージェント側の対応する表記をそろえる。
- [ ] T005 [App] MCP リソースサーバーの登録結果に、コピー、設定の説明、次の手順、ボタンの文言を入れる。複数行のコピーの部品を足す。
- [ ] T006 [App] 編集画面のツールの節の文言を直す。
- [ ] T007 [App] 状態の表示と選択肢を「有効」「無効」にする。
- [ ] T008 [App] MCP リソースサーバーの一覧を表と詳細カードの構成にする。
- [ ] T009 [App] 用語の表に従い、エージェントと MCP リソースサーバーと関連する認可の画面の辞書を直す。
- [ ] T010 [App] 作成画面の戻る操作と結果のカードの余白をそろえる。
- [ ] T011 [Spec] TypeSpec の操作と型を設計の 8 の名前へ改め、REQ-OAUTH2-086 と REQ-OAUTH2-087 の語を直す。
- [ ] T012 [App] バックエンド、Gateway、フロントエンドの識別子とファイル名を改め、テーブルと列を新しいマイグレーションで改名する。既存のテストを改名後に通す。
- [ ] T013 [Verify] 変更を検証する。

## 検証

- `mise run test-ui`：辞書、登録結果のコピー、状態の表示、一覧の表の単体テスト。文言と構成を変える画面のテストが対象である。
- `mise run test-ui-e2e-file` で、MCP リソースサーバーの登録とエージェントの登録の E2E を実行する。登録結果からツールの公開へ進む経路が文言の変更で切れていないことを確かめる。
- `mise run check-links` と `mise run check-work-items`：仕様とデモ文書の更新を確かめる。
- `mise run verify`：TypeSpec の改名が生成物、HTTP の経路、Gateway の送信、DB のマイグレーションまで通ることを確かめる。旧名の経路が残っていないことと、カタログの送信から管理 API の取得までの既存の HTTP 統合テストを含むためである。
- `rg -i "reported.?tool|ReportResourceTools|mcp_tool_report"` で、適用済みのマイグレーションと完了した作業項目のほかに旧名が残っていないことを確かめる。
- 画面を起動し、日本語と英語で、一覧、作成、登録結果、編集、エージェントの割当の各画面を目で確かめる。

## リスク

- E2E とデモの文書が文言で要素を探しているため、取りこぼすとデモの手順が壊れる。`rg` で旧文言を検索し、残りがないことを確かめる。
- Gateway とバックエンドの改名がそろわないと、Gateway のカタログの送信が 404 になり、管理画面にツールが出なくなる。Gateway の送信を含む HTTP 統合テストと、デモの E2E で確かめる。
- 開発環境の DB に旧名のテーブルが残っていても、改名のマイグレーションが既存の行を引き継ぐ。行を作り直す移行にはしない。
- 「接続先」は認可の画面（`admin-authorization/agentAccess.i18n.ts`）でも使う。用語を変える範囲を画面ごとに決めずに置き換えると、文脈に合わない表現が残る。用語の表に従い、一文ずつ読んで直す。
