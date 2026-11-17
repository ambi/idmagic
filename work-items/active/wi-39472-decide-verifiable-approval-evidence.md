---
status: pending
authors: [tn]
risk: high
created_at: 2026-11-17
priority: p3
group: agent-governance
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/modules/oauth2/features/approval.md, requirement: REQ-OAUTH2-041 }
  - { path: docs/modules/oauth2/features/resource-access.md, requirement: REQ-OAUTH2-065 }
  - { path: docs/modules/oauth2/standards.md, requirement: RFC9396-MONOTONIC-NARROWING }
---

# 採否を判断したうえで、承認の結果を外部で検証できる署名付きの証跡にする

## 動機

IdMagic は人の承認を操作の対象と引数に束縛し、一度だけ許可する（REQ-OAUTH2-065）。
しかし、その束縛は IdMagic の評価 API の内側でしか確かめられない。
加盟店、決済網、別の企業の API は、利用者が何を承認したかを独立に検証できない。

業界は承認の証跡を持ち出せる形へ向かっている。
[FIDO Alliance](https://fidoalliance.org/fido-alliance-to-develop-standards-for-trusted-ai-agent-interactions/) は Google の AP2 と Mastercard の Verifiable Intent を受け取り、Payments WG で標準化を始めた。
[Poppy](https://personalagentprotocol.org/docs/introduction) は署名付きの承認（mandate）を次の課題にし、IETF では Delegate SD-JWT が WG 文書になった。

## 対象範囲

- 判断を最初に行う。承認済みの `authorization_details` に IdMagic が署名した JWT を発行する案、SD-JWT の委譲に載せる案、FIDO の仕様を待つ案を比べる。
- 採用した案の発行、検証の手順、失効との関係。
- 見送る場合でも、内部の承認記録を外部の形式へ写せる構造かの確認。

## 対象外

- 決済網との接続、カードのトークン化。
- 利用者の端末の鍵（パスキー）による承認内容への直接の署名。FIDO の仕様が固まってから別項目にする。

## 設計

判断の材料は、FIDO、Poppy、IETF のどれが主流になるか、承認の証跡を求める利用者の要望、一回性の保証を外部の検証者とどう分担するかである。
IdMagic が署名する証跡は「IdMagic が本人のステップアップ認証を確かめ、この内容の承認を記録した」ことを示すだけで、利用者の端末での署名と同等とは表示しない。
一回性は検証者の責任になるので、証跡には `jti` と有効期限を必須にする。
見送る場合は、再検討の条件を `## 要約` に書いて中止する。

## 計画

1. 三つの案を比べて採否を決める。
2. 採用した案だけを、仕様、受け入れテスト、実装の順に進める。

## タスク

- [ ] T001 [Decision] 調査して採否と形式を確定する。
- [ ] T002 [Spec] 採用した形式の規範と TypeSpec を書く。
- [ ] T003 [Acceptance] 改変した内容、期限切れ、撤回後の証跡の RED を作る。
- [ ] T004 [App] 実装する。

## 検証

- 採用した場合に、対象のパッケージと `mise run verify-changed -- <着手時の基準リビジョン>` を実行する。

## リスク

外部へ出した証跡は撤回できない。
有効期限を短くし、撤回の確認先を証跡に含めるかを判断の中で決める。
