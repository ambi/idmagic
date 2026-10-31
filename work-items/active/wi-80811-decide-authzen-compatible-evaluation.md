---
status: pending
authors: [tn]
risk: medium
created_at: 2026-10-31
priority: p3
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/modules/oauth2/features/resource-access.md, requirement: REQ-OAUTH2-062 }
---

# AuthZEN 互換の評価入口を設けるか判断して対応する

## 動機

[AuthZEN Authorization API 1.0](https://openid.net/authorization-api-1-0-final-specification-approved/) は最終仕様となり、第三者の PEP が IdMagic へ接続しやすくなる可能性がある。
しかし Evaluation API は `subject` を必須とし、提示トークンから主体を導く REQ-OAUTH2-062 の判断と衝突し得る。
任意の主体を問える入口は権限の探索につながるので、互換の形が安全に成り立つかを先に確かめる。

## 対象範囲

- 判断を最初に行う。トークンを `subject` の属性で渡す形が仕様に適合するか、実在の PEP が受け入れるかを確かめる。
- 成り立つ場合、既存の評価へ委譲する薄い入口。

## 対象外

- 自己申告の主体を信じる評価。
- 既存の `POST /resource-access/evaluate` の廃止。

## 設計

入口は主体を提示トークンからだけ導き、要求の `subject` と一致しなければ不許可にする案を第一候補にする。
この形で適合を主張できず、接続の要望もなければ、再検討の条件を `## 要約` に書いて中止する。

## 計画

1. 仕様と実在の PEP を確かめて採否を決める。
2. 採用時だけ TypeSpec と入口を追加する。

## タスク

- [ ] T001 [Decision] 適合の可否と要望を確かめ、採否を確定する。
- [ ] T002 [Spec] 入口の仕様と標準の採用表を更新する。
- [ ] T003 [Acceptance] 他者の `subject` の拒否、既存の評価との一致の RED を作る。
- [ ] T004 [App] 実装する。

## 検証

- 採用した場合に `mise run test-go-package -- ./backend/shared/http/server_http` と `mise run check-published-api-compat` を実行する。
- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>` とする。

## リスク

互換のために主体の検証を緩めると、権限の探索を許す。
適合より主体の検証を優先する。
