---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-11-03
priority: p1
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/modules/identity-management/features/agent.md, requirement: REQ-IDMANAGEMENT-009 }
  - { path: docs/modules/identity-management/features/agent.md, requirement: REQ-IDMANAGEMENT-073 }
  - { path: docs/modules/identity-management/features/agent.md, requirement: REQ-IDMANAGEMENT-103 }
  - { path: docs/modules/identity-management/features/agent.md, requirement: REQ-IDMANAGEMENT-104 }
  - { path: docs/modules/authorization/features/relation-tuple.md, requirement: REQ-AUTHORIZATION-017 }
  - { path: docs/modules/oauth2/features/client.md, requirement: REQ-OAUTH2-084 }
---

# Agent の登録、認証の準備、利用する接続先を一つの作成手順にまとめる

## 動機

現在のデモの通常の管理では、Agent を使える状態にするまでに三つの画面を順に開く。
Agent を登録し、詳細から「クレデンシャルを管理」で公開鍵を貼り、さらに「利用する接続先」を編集する。
その間に Agent ID と作成した client_id を控え、デモへ転記する。
利用者の判断は「名前」「どの鍵で認証するか」「何を使わせるか」の三つだけであり、画面と転記の数はこの判断の数に見合わない。

Okta の Agent 登録は、認証方式の構成を Agent の登録作業の中で案内し、利用する接続先を同じ Agent の詳細から追加する（[Agent の手動登録](https://help.okta.com/oie/en-us/content/topics/ai-agents/ai-agent-add-manually.htm)、[リソースへの接続](https://help.okta.com/oie/en-us/content/topics/ai-agents/ai-agent-connected-resource.htm)）。
IdMagic がこれより多い画面遷移と転記を要求する理由はない。

## 対象範囲

- Agent の作成画面で、名前、責任者（既定は操作者）、区分、認証、利用する接続先と操作を一度に指定できるようにする。
- 認証は「公開鍵を貼る」「JWKS の URL を指定する」「後で設定する」から選ぶ。JWKS の URL を使えば、鍵の更新に管理画面の操作を要さない。
- 利用する接続先は、登録済みの接続先と操作を表示名で選ぶ。未選択でも作成できる。
- 保存前に、作るものを一文の要約で示す。例：「社内文書アシスタントを作成し、公開鍵で認証できるようにし、社内文書 MCP の文書の閲覧を許可します」。
- 作成後の結果画面に、配置先へ渡す値（client_id、トークンエンドポイント、リソース、要求方法）をまとめて示し、コピーと設定ファイル形式の取得を提供する。Agent ID を転記させない。
- 各入力欄に、値の出所（利用者が決める、自動で入る、前の手順の結果、既定値）と、変更したときに何が変わるかを示す。
- 既存の個別画面（資格情報の管理、利用する接続先の編集）は、作成後の変更の入口として残す。

## 対象外

- MCP 接続先と Gateway の登録の自動化。[MCP 接続準備の自動化](wi-95247-discover-mcp-tools-and-automate-connection-setup.md)で扱う。
- 対象ごとの権限の付与。[対象ごとの権限の準備](wi-24427-restrict-tools-to-targets-without-authoring-a-model.md)で扱う。
- 鍵の生成をブラウザーや IdMagic で行うこと。秘密鍵は Agent の配置先に留める。
- 期限付きのアクセス申請と承認条件の変更。

## 設計

### 一つの操作としての確定

作成は Agent、OAuth2Client、束縛、操作割当の四つの記録にまたがり、所有者のモジュールが異なる。
一つのトランザクションに見せかけるか、段階ごとの結果を返すかを着手時に決める。
候補は次の二つである。

| 案 | 利点 | 制約 |
| --- | --- | --- |
| 作成を受ける一つの管理 API が、所有モジュールのユースケースを順に呼び、段階ごとの結果を返す | 画面は一回の送信で済み、途中で止まった段階から再試行できる | 部分成功の表現と、再送で Agent やクライアントを二重作成しない冪等性キーが必要 |
| 画面が既存の API を順に呼ぶ | バックエンドの変更が少ない | 途中の失敗と再試行の扱いが画面ごとに散り、別の入口から同じ流れを再現できない |

前者を推薦する。
既存の REQ-IDMANAGEMENT-103 が定める「作成済み、束縛は未完了」の表示を、段階の一般化として引き継ぐ。
新しい組み立て地点か公開 API を追加する場合は、[境界を選ぶ判断手順](../../docs/design/application/design-guidelines.md#境界を選ぶ判断手順)の比較を記録してから実装する。

### 入力の説明

出所の表示は、入力欄の補助文に統一した語で付ける。
「あなたが決める」「自動で入ります」「前の手順の結果」「既定値。変更できます」の四つに限る。
説明が必要な語（client_id、JWKS、区分）は、初出の入力欄で一文の定義を示し、用語集へのリンクで済ませない。

### JWKS の URL

`jwks_uri` を持つクライアントは管理 API で既に扱える。
取得先の検証、期限、取得量、リダイレクトの扱いは既存の規則に従い、新しい外向き取得の経路を作らない。
既存の規則がなければ、[MCP 接続準備の自動化](wi-95247-discover-mcp-tools-and-automate-connection-setup.md)の外向き取得の決定と合わせる。

## 計画

今回は起票だけを行う。
着手時に作成 API の境界と冪等性を決め、仕様と TypeSpec を先に更新する。
作成画面の受け入れ E2E を RED にしてから、API、画面、結果表示の順に実装する。
最後にデモの手順を新しい作成画面へ合わせる。

## タスク

- [ ] T001 [Design] 一括作成の境界、段階ごとの結果、冪等性と再試行を決める。
- [ ] T002 [Spec] Agent 作成の要件と管理 API の契約を更新する。
- [ ] T003 [Acceptance] 一つの画面から作成し、転記なしでトークンを発行して割り当てた操作を実行する E2E を RED にする。
- [ ] T004 [App] 作成 API と部分失敗からの再試行を実装する。
- [ ] T005 [UI] 作成画面、入力の出所、保存前の要約、配置用の値の結果画面を実装する。
- [ ] T006 [Verify] デモの手順を更新し、変更前後の画面遷移、手入力、転記の数を記録する。

## 検証

主要ユースケースは、作成画面の一回の送信から、Agent の認証と割り当てた操作の実行までを観測する E2E とする。
途中の段階で失敗させ、再試行でクライアントと Agent が重複しないことを確認する。
公開鍵、JWKS の URL、後で設定の三経路と、割当なしの作成を含める。
実装後は `verify-changed` と対象の `test-ui-e2e-file` を選ぶ。

## リスク

一括作成の途中失敗で、束縛されないクライアントや割当のない Agent が残ると、棚卸しの所見が増える。
段階ごとの結果を示し、再試行で同じ記録へ収束させる。
作成画面に項目を集めすぎると、かえって判断が増える。
必須の入力は名前と認証の選択だけにし、残りは既定値で作成できるようにする。
