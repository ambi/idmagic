---
status: pending
authors: [tn]
risk: high
reversibility: irreversible
created_at: 2026-10-31
priority: p1
depends_on: [wi-11085-enforce-mcp-tool-calls-for-unmodified-servers]
change_kind: feature
affected_spec:
  - { path: docs/modules/oauth2/features/token.md, requirement: REQ-OAUTH2-056 }
  - { path: docs/modules/oauth2/features/token.md, requirement: REQ-OAUTH2-048 }
  - { path: docs/modules/oauth2/features/token.md, requirement: REQ-OAUTH2-060 }
---

# 送信者制約を保ったままゲートウェイが資源宛のトークンへ差し替える

## 動機

[Ping](https://developer.pingidentity.com/identity-for-ai/integrations/idai-authorizing-gemini-on-behalf-of-users-with-ping.html) と [Okta](https://www.okta.com/blog/product-innovation/okta-agent-gateway-runtime-security/) のゲートウェイは、許可の後に資源宛のトークンへ交換して転送する。
IdMagic で同じ構成を DPoP の Agent に適用すると拒否される見込みである。
REQ-OAUTH2-056 は元のトークンと同じ鍵の証明を求めるが、その鍵を持つのは Agent であり、ゲートウェイではないからである。
仕様はこの付け替えを「委譲の許可を発行する仕組みがないため受け付けない」としており、その仕組みが必要になった。

## 対象範囲

- まず現在の拒否を再現し、下流で元のトークンをそのまま受ける構成と比べて、差し替えの要否を確かめる。
- 登録したゲートウェイが、元のトークンの範囲内で、資源宛の短命なトークンを自身の鍵へ束縛して得る手段。
- 結果の `act` に Agent とゲートウェイの連鎖を残し、失効を既存の `agent_chain` で判定する。

## 対象外

- 送信者制約を落として Bearer にする方式。盗んだトークンの付け替えを許す。
- 非 OAuth の下流資格情報（[[wi-62382-decide-non-oauth-downstream-credentials-in-the-vault]]）。
- `actor_token` の鍵の照合を一般の交換へ広げること。

## 設計

根拠の候補を T001 で比較する。

| 候補 | 利点 | 懸念 |
| --- | --- | --- |
| 評価の結果に一回限りの交換の許可を載せる（第一候補） | 判断と交換が一対一になり、許可された操作だけに限れる | 評価の応答と状態が増える |
| Agent が事前に委譲の許可を得る | 所有の意思が明示される | Agent の改修が要り、無改修の価値が下がる |
| ゲートウェイの登録だけを根拠に許す | 単純 | 盗んだトークンをゲートウェイへ持ち込めば付け替えられる |

承認済みの操作は交換の派生を `approval_required` とする REQ-OAUTH2-065 を維持し、評価を交換より前に行う。
`may_act` の発行経路と REQ-OAUTH2-060 の付与の確認がゲートウェイの行為者をどう扱うかは T001 の再現で確かめる。

## 計画

1. 再現と比較を行い、差し替えの要否と根拠を決める。不要と判断すれば理由を要約に書いて中止する。
2. REQ-OAUTH2-056 の判断を改め、TypeSpec を更新する。
3. 盗んだトークンの持ち込みを拒否試験の中心に置く。

## タスク

- [ ] T001 [Decision] 拒否を正式な `/token` で再現し、要否と根拠を確定する。
- [ ] T002 [Spec] 交換の条件と結果の連鎖を仕様へ書く。
- [ ] T003 [Acceptance] 正当な差し替え、許可なしの交換、別の操作への流用、失効後の拒否の RED を観測する。
- [ ] T004 [App] 交換と MCP ゲートウェイを実装する。
- [ ] T005 [Verify] 深さ上限、本人の撤回、mTLS、再利用を検証する。

## 検証

- `mise run test-go-package -- ./backend/oauth2/internal/token/usecases`。
- `mise run test-go-package -- ./backend/shared/http/server_http`：DPoP の Agent から下流までを観測する。
- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>` とする。

## リスク

送信者制約の保証を弱める変更であり、誤ると盗んだトークンの再束縛を許す。
公開後は交換の形を変えにくいので、脅威モデルを更新して実装する。
