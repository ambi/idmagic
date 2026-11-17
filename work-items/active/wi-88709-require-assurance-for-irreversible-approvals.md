---
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-11-17
priority: p2
group: authentication
depends_on: [wi-63330-define-built-in-assurance-levels, wi-63883-show-scope-impact-on-consent-and-approval]
change_kind: feature
affected_spec:
  - { path: docs/modules/oauth2/features/approval.md, requirement: REQ-OAUTH2-043 }
  - { path: docs/modules/oauth2/features/approval.md, requirement: REQ-OAUTH2-041 }
  - { path: docs/modules/oauth2/features/resource-access.md, requirement: REQ-OAUTH2-065 }
---

# 取り消せない操作の承認に、操作の影響度に応じた認証水準を要求する

## 動機

REQ-OAUTH2-043 は承認の判断にステップアップ認証を要求するが、どの認証方式で満たしたかを操作ごとに区別しない。
送金や削除のような取り消せない操作も、取り消せる操作と同じ水準で承認できる。

[FIDO Alliance の Agentic Authentication WG](https://fidoalliance.org/fido-alliance-to-develop-standards-for-trusted-ai-agent-interactions/) は、利用者の指示をフィッシング耐性のある方法で確かめることを中心課題にしている。
[Poppy 1.0](https://personalagentprotocol.org/docs/1.0/spec) も `scope_hints.changes` で `irreversible` を区別し、取り消しにくい操作の承認を拡張で扱う。
フィッシングされた承認は Agent の誤操作をそのまま確定させるので、群の基準の p3 より上げて p2 とする。

## 対象範囲

- 承認の対象（スコープ、`authorization_details` の種類、ツール）に宣言した影響度から、判断に必要な認証水準を決める規則。
- 水準を満たさない判断の拒否と、満たす認証へのステップアップの案内。
- 判断に使った認証水準と方式（`acr`、`amr`）を承認の記録と監査に残すこと。

## 対象外

- 影響度の登録と画面の表示。[[wi-63883-show-scope-impact-on-consent-and-approval]] が扱う。
- 認証水準の語彙そのもの。[[wi-63330-define-built-in-assurance-levels]] が扱う。
- 承認結果の署名付きの証跡。[[wi-39472-decide-verifiable-approval-evidence]] が扱う。
- CIBA の `push` と承認のモバイル通知。

## 設計

承認の判断時に、`RequiredApprovalAssurance(impact, tenantPolicy) AssuranceLevel` を計算し、セッションの認証水準と比べる。
計算は作用を持たず、承認リクエストの対象と現在のテナント設定だけを入力にする。
初期値は、`irreversible` を含む承認にフィッシング耐性のある方式（パスキー）を、それ以外に現行のステップアップを要求する。
テナント管理者は水準を上げられるが、`irreversible` の水準を既定より下げられない。

影響度が未登録の対象は現行の規則のまま扱う。
未登録を `irreversible` と見なす案は、既存の承認をすべて止めるので採らない。
ただし、管理画面で未登録の対象を所見として示す。

## 計画

1. 先行項目の語彙と影響度の属性を確認し、REQ-OAUTH2-043 の拡張か新しい要件かを決める。
2. 仕様、受け入れテスト、実装の順に進める。

## タスク

- [ ] T001 [Spec] 承認の判断に必要な認証水準の規範を書く。
- [ ] T002 [Acceptance] パスキーのないセッションで取り消せない操作を承認できない RED を、正式な HTTP 入口で作る。
- [ ] T003 [Domain] 必要な水準の計算を単体テストで固める。
- [ ] T004 [App] 判断のユースケースと監査へ接続する。
- [ ] T005 [UI] 承認画面でステップアップの案内を示す。
- [ ] T006 [Verify] 検証する。

## 検証

- `mise run test-go-package -- ./backend/oauth2/...` の承認の対象パッケージ。
- `mise run test-go-package -- ./backend/shared/http/server_http`：正式な配線での拒否と成立。
- 承認画面を変えた場合は対象の `test-ui-e2e-file`。
- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>`。

## リスク

水準を上げすぎると、パスキーを持たない利用者が承認できず、Agent が止まる。
拒否時に必要な方式と登録の導線を示し、テナントの既定を段階的に適用できるようにする。
