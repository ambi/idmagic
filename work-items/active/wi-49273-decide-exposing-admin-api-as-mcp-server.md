---
status: pending
authors: [tn]
risk: high
created_at: 2026-11-17
priority: p3
group: admin-governance
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/modules/oauth2/features/admin-access.md, requirement: REQ-OAUTH2-003 }
  - { path: docs/modules/identity-management/features/admin-access.md, requirement: REQ-IDMANAGEMENT-025 }
---

# 採否を判断したうえで、管理 API を MCP サーバーとして公開する

## 動機

Ping は MCP サーバー、Agent Skills、CLI で管理操作を AI Agent に行わせる形を打ち出し、Entra も AI コーディング Agent による Agent ID の初期設定を案内している。
管理者が Agent に IdMagic の設定や棚卸しを任せる需要は増えるとみられる。

IdMagic の管理 API は OAuth のスコープで保護されており（REQ-OAUTH2-003）、外部の汎用 MCP サーバーから呼ぶことはできる。
ただし、管理権限を Agent に渡す場合の承認、範囲の制限、監査の扱いは定めていない。

## 対象範囲

- 判断を最初に行う。IdMagic 自身の MCP サーバーを提供する案、自前の MCP Gateway の背後に管理 API を置く案、手順書だけを示す案を比べる。
- 採用した場合の、公開する操作、読み取りと変更の区別、変更操作の承認の要否。

## 対象外

- 管理コンソールに組み込む会話型の Agent。
- システムコンソール（テナント横断）の操作の公開。

## 設計

判断の材料は、管理権限を持つ Agent を IdMagic 自身の Agent と委譲の仕組みで表せるか、変更操作に承認（REQ-OAUTH2-065）を必須にできるか、プロンプトインジェクションで管理操作が実行される脅威の評価である。
第一候補は、自前の MCP Gateway の背後に管理 API を置き、変更操作を承認付きにする案である。
自社の部品を自分で使う構成になり、新しい実行単位を増やさない。
見送る場合は、再検討の条件を `## 要約` に書いて中止する。

## 計画

1. 脅威モデルへの影響を含めて三つの案を比べ、採否を決める。
2. 採用した案だけを、仕様、受け入れテスト、実装の順に進める。

## タスク

- [ ] T001 [Decision] 調査して採否と方式を確定する。
- [ ] T002 [Spec] 採用した方式の規範と脅威モデルの更新を書く。
- [ ] T003 [Acceptance] 承認のない変更、範囲外のテナント、停止した Agent の RED を作る。
- [ ] T004 [App] 実装する。

## 検証

- 採用した場合に、対象のパッケージと `mise run verify-changed -- <着手時の基準リビジョン>` を実行する。

## リスク

管理権限を持つ Agent が乗っ取られると、テナント全体の設定を変えられる。
既定では読み取りだけを公開し、変更は承認と期限付きの権限を必須にする。
