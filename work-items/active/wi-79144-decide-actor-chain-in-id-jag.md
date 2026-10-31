---
status: pending
authors: [tn]
risk: medium
created_at: 2026-10-31
priority: p3
depends_on: [wi-38172-verify-id-jag-interoperability-with-an-external-idp]
change_kind: feature
affected_spec:
  - { path: docs/modules/oauth2/features/token.md, requirement: REQ-OAUTH2-066 }
  - { path: docs/modules/oauth2/features/token.md, requirement: REQ-OAUTH2-070 }
---

# ID-JAG で Agent の行為者を保つか判断して対応する

## 動機

固定した `draft-ietf-oauth-identity-assertion-authz-grant-05` が処理を定めないので、IdMagic は ID-JAG の両ホップで `actor_token` と `act` を拒否する。
その結果、企業管理のアプリ間接続を通ると、誰のためにどの Agent が動いたかを資源側のトークンで表せない。
標準が固まらない段階で独自の拡張を入れると相互運用を損なう懸念があり、採否を判断する必要がある。

## 対象範囲

- 判断を最初に行う。draft の改訂、MCP の Enterprise-Managed Authorization、Okta の実装での扱いを確かめる。
- 標準または実装の合意が確認できた場合、行為者を保つ発行と償還。

## 対象外

- 標準の裏付けがない独自のクレームを既定で発行すること。

## 設計

既定は現状の拒否の維持である。
draft が定めるか、実在の製品との互換が確認できたときだけ、その形に合わせて `act` を受け入れる。
資源側の監査の不足を、IdMagic の発行側の監査との相関で補えるかも比べる。
条件を満たさなければ、再検討の条件を `## 要約` に書いて中止する。

## 計画

1. 標準と実装を調べて採否を決める。
2. 採用時だけ規範を改め、実装する。

## タスク

- [ ] T001 [Decision] 標準と実装を調べ、採否を確定する。
- [ ] T002 [Spec] REQ-OAUTH2-066、REQ-OAUTH2-070 の周辺を更新する。
- [ ] T003 [Acceptance] 連鎖の保持、停止した Agent の拒否の RED を作る。
- [ ] T004 [App] 実装する。

## 検証

- 採用した場合に `mise run test-go-package -- ./backend/oauth2/internal/token/usecases` と `mise run verify-changed -- <着手時の基準リビジョン>` を実行する。

## リスク

draft の改訂で形が変わると、公開後の変更が難しい。
対応する版を固定して広告する。
