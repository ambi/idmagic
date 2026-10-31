---
status: pending
authors: [tn]
risk: high
created_at: 2026-10-31
priority: p3
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/modules/token-vault/features/execution.md, requirement: REQ-TOKENVAULT-008 }
---

# 非 OAuth の下流資格情報を Vault で仲介するか判断して対応する

## 動機

Vault は OAuth のアクセストークンを Bearer で仲介するが、API key や mTLS だけを受ける下流には使えない。
[PingGateway](https://developer.pingidentity.com/blog/securing-mcp-servers-with-pings-mcp-gateway/) は API key や mTLS への変換を説明している。
一方、長期の鍵を保管すると漏えいの影響と失効の責任が増え、[Okta の Gateway](https://www.okta.com/blog/product-innovation/okta-agent-gateway-runtime-security/) は共有のプラットフォーム鍵を拒否する方向を示す。
必要性と危険の釣り合いが確定していないので、先に判断する。

## 対象範囲

- 判断を最初に行う。対象の SaaS、利用者単位の鍵か共有の鍵か、ローテーションと失効の責任を比べる。
- 採用する場合、限定した形式の資格情報の保管と注入。

## 対象外

- 利用者を区別できない共有の鍵の無条件な受け入れ。監査の主体が曖昧になる。
- Agent へ平文の鍵を返すこと。

## 設計

見送りを既定とし、実在の導入先と、利用者単位で失効できる形が確認できたときだけ採用する。
採用する場合は、REQ-TOKENVAULT-009 から 011 の外向きの制限と暗号化の束縛を再利用し、ヘッダーの形式を管理者の宣言に限る。
見送る場合は、再検討の条件を `## 要約` に書いて中止する。

## 計画

1. 導入先の要件を確かめて採否を決める。
2. 採用時だけ脅威モデルを更新し、実装する。

## タスク

- [ ] T001 [Decision] 対象と責任の範囲を確かめ、採否を確定する。
- [ ] T002 [Spec] 資格情報の形式と脅威モデルを更新する。
- [ ] T003 [Acceptance] 注入、宛先の制限、応答への非露出の RED を作る。
- [ ] T004 [App] 実装する。

## 検証

- 採用した場合に `mise run test-go-package -- ./backend/tokenvault/...` と `mise run verify-changed -- <着手時の基準リビジョン>` を実行する。

## リスク

共有の鍵は本人の撤回と両立しない。
採用条件にこの差を明記する。
