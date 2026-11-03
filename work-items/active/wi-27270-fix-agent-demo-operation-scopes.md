---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-11-03
priority: p1
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/oauth2/features/client.md, requirement: REQ-OAUTH2-084 }
  - { path: docs/modules/oauth2/features/resource-access.md, requirement: REQ-OAUTH2-083 }
  - { path: docs/modules/mcp-gateway/features/runtime.md, requirement: REQ-MCPGATEWAY-010 }
primary_use_cases:
  - id: demo-operation-scopes
    requirement: REQ-OAUTH2-084
    observable_result: 通常のデモで閲覧、更新、削除を割り当て、更新と削除が同じスコープでも未割当の削除を拒否する。
    boundary: e2e
    reason: 管理画面の操作登録と割当からデモのトークン発行、Gateway、実在の MCP ツールの効果までを確認する。
    test: { path: frontend/tests/e2e/agent-demo-operations.spec.ts, name: administrator enables writing and deletion independently in the agent demo, task: test-ui-e2e }
    fault_model: デモの要求スコープが閲覧に固定されるか、変更用スコープだけで未割当の削除を許可する。
    setup:
      state: 初期デモテナントと管理者だけがある。
      entry_points: 接続先、ツール、Agent、鍵と割当を管理画面で準備し、デモの接続と実行を使う。
      fixtures: 認証済み管理者とデモプロセスの起動だけを共通基盤とする。
---

# デモの閲覧固定を外し、作成・更新と削除を個別に許可する

## 動機

デモは `tools:read` だけを要求するため、削除の許可にも閲覧用スコープを指定する必要があった。
閲覧と削除だけを実装していたため、基本的な文書の作成・更新も試せなかった。

## 対象範囲

- `write_document` と文書本文の作成・更新、結果を読み直す操作を追加する。
- カタログを使う通常の管理ではスコープを省略して発行を要求し、現在の割当と認証上限から決める。
- 閲覧には `tools:read`、作成・更新と削除には `tools:write` を接続定義で対応させ、各操作は個別に割り当てる。
- 設定手順、発行結果、追加・取消の説明と実ブラウザーの検証を更新する。
- 操作の追加は既存のツール登録と割当の選択の中で行い、デモの手順数、手入力、転記を増やさない。

## 対象外

- ツールとスコープの1対1対応の強制と、新しい認可方式。
- スコープの自動生成、MCP ツールの自動発見、管理コンソール全体の再設計。
- 一回の操作の承認を Gateway へ渡す機能と、競合製品の実機測定。

## 設計

### 契約と方式

MCP の [Tools](https://modelcontextprotocol.io/specification/2026-07-28/server/tools) は名前付きの呼び出し可能な機能を定義する。
[認可仕様](https://modelcontextprotocol.io/specification/2026-07-28/basic/authorization) と [RFC 6749 §3.3](https://www.rfc-editor.org/rfc/rfc6749#section-3.3) は、ツール名とスコープ名の1対1対応を要求しない。
今回は粗い認証上限と個別の操作割当を保ち、`write_document` と `delete_document` を `tools:write` へ対応させる。
ツールごとにスコープを作る案も妥当だが、今回は利用者へ同じ許可の二重選択を追加しない。
製品の REQ-OAUTH2-084 と083は既にこの組み合わせを規定しているため、要件と公開 TypeSpec は変更しない。
利用例とデモの手引きを先に更新する。

### 入力と作用

`write_document` は `document_id`、`title`、`body` を受け、プロセス内だけの本文を作成・上書きする。
本文の所有者は `documents` とし、共有定数の初期本文を変更しない。
既存の mutex で閲覧、変更、削除、初期化を同期する。
空の ID とタイトルはツールのエラーとし、本文は空文字列も保存できる。
同じ内容の再送は同じ本文へ確定し、削除後の書き込みは作り直しとして扱う。
上流に届かなかった要求は本文を変更せず、既存の実行カウンターで区別する。
入力のサイズ、JSON の解釈、通信の期限は既存の HTTP と MCP SDK の境界に従い、新しい永続化、キャッシュ、再試行を導入しない。
公開するツール名は MCP サーバーの実名と一致させ、利用者向けの表示名は自由に設定する。
通常の接続では省略したスコープを現在の割当へ射影し、従来の文書別の構成では登録済みの閲覧スコープを明示して互換性を保つ。
通常発行と承認機能が使うクライアント資格情報の発行は同じ要求構築を通す。
既発行トークンのスコープは変えず、新たなスコープが必要な追加後は既存の接続ボタンで再発行する。
同じスコープ内の操作の追加と取消は次の評価へ反映する。
デモだけの API と上流ツールの変更であり、製品の公開 HTTP 契約、モジュール境界、DB スキーマ、監査イベントは変更しない。

## 計画

今回は起票だけを行い、実装と製品仕様の変更には着手しない。
手引きと例を更新し、ツールの効果と発行要求のテストを RED にする。
デモに本文の書き込みとカタログ発行を実装し、画面の手順と結果を合わせる。
実ブラウザーで閲覧のみ、更新の追加、同じスコープでの削除の追加と取消を観測する。

## タスク

- [ ] T001 [Spec] ツールとスコープの対応、設定と再発行の説明を更新する。
- [ ] T002 [App] 文書の作成・更新と発行要求の固定値の撤去を実装する。
- [ ] T003 [UI] 閲覧・更新・削除の実演と、個別の許可を確かめるブラウザーテストを追加する。
- [ ] T004 [Verify] 差分を検証し、完了記録とコミットを作る。

## 検証

`test-go-package -- ./examples/agent-demo` で実際の MCP ツール呼び出し、本文の変化、スコープ要求を確認する。
`test-ui-e2e-file` は新しい `agent-demo-operations.spec.ts` と既存の `ai-agent-demo.spec.ts` を対象にする。
最終差分には `verify-changed -- main`、完了記録には `check-work-items` と `check-links` を選ぶ。
独自パーサーと判定器は追加しないため、新しい fuzz と変異テストは追加しない。

## リスク

変更用のスコープを個別の操作の許可と混同しないよう、更新を許可しても削除が拒否されることを独立に確かめる。
通常の例と文書別の例でトークンの要求方法が異なるため、両方を実行する。

ツール名や接続設定の手入力の削減は [MCP 接続準備の自動化](wi-95247-discover-mcp-tools-and-automate-connection-setup.md)で扱う。
デモの設定済みの開始点と入力の出所の表示は、この項目の後に [wi-68754](wi-68754-start-the-agent-demo-preconfigured-and-explain-every-input.md)で扱う。
