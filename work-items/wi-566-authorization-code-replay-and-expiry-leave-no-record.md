---
depends_on: []
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-12
priority: p2
change_kind: bugfix
spec_impact: { kind: none, reason: "EX-OAUTH2-005-07 と EX-OAUTH2-005-08 は既に宣言済みである。実装をその宣言へ合わせる作業であり、規範は動かない。" }
affected_spec:
  - { path: docs/contexts/oauth2/scenarios.feature.md, requirement: REQ-OAUTH2-005 }
---

# 認可コードの再交換と期限切れが、宣言どおりの記録を残さない

## Motivation

[[wi-559-back-oauth2-remaining-examples-with-tests]] が REQ-OAUTH2-005 を消化する過程で、実装が具体例の `Then` を 2 つ満たしていないことを測った。

**`EX-OAUTH2-005-07`：** 同じ認可コードを 2 回交換したとき、具体例は「発行ファミリーのトークンがすべて失効する」「`RefreshTokenReuseDetected` が発行される」「`TokenRevoked` が発行される」の 3 つを言う。実測では発行されたイベントは `[AccessTokenIssued AuthorizationCodeRedeemed RefreshTokenIssued]` だけで、いずれも 1 回目の交換によるものだった。2 回目の交換は失効の記録だけを残し、通知を 1 つも出していなかった。

このうち `RefreshTokenReuseDetected` は wi-559 が直した。`refresh_tokens.go` が refresh トークン再利用の検出でまったく同じ組（`RevokeFamily` の直後に `emit`）を持っており、認可コード再提示の経路だけがその発行を欠いていた。既存の写像の欠落だったので、局所の修正として `revokeReplayedFamily` へ寄せた。

**残るのは `TokenRevoked` である。** これは局所では済まない。`RefreshTokenStore.RevokeFamily(ctx, familyID) error` は失効させた token を返さないので、token ごとの `TokenRevoked{TokenID, Reason}` を組み立てる材料が呼び出し側に無い。port の署名を変えるか、失効を通知する別の口を設けるかを決める必要がある。

**`EX-OAUTH2-006-02` も同じ欠落を持つ。** wi-559 が `/token` の入口から測った。ローテーション済みの旧 refresh トークンを再使用すると、`invalid_grant` で拒否され、記録は `Revoked` になり、family も失効し、`RefreshTokenReuseDetected` も発行される。出ないのは `TokenRevoked` だけである。原因は同じ `RevokeFamily` の署名なので、認可コード側と一緒に直る。片方だけ直すと、また片方が黙る。

**`EX-OAUTH2-005-08`：** 発行から 60 秒を超えた認可コードの交換は `invalid_grant` で拒否される（ここは正しい）。しかし具体例が言う「認可コードの状態は Expired になる」が起きない。実測では記録の状態は `issued` のままだった。

`spec.AuthorizationCodeRecordState` は `expired` を持ち、`authorization_code_machine.go` は `{issued, Expire, expired}` の遷移を宣言している。**遷移を実行する製品コードが 1 行も無い。** 宣言だけがあって、誰もその状態へ動かさない。

## Scope

- `EX-OAUTH2-005-07` と `EX-OAUTH2-006-02` の `TokenRevoked` を発行できるようにする。`RevokeFamily` が失効させた token を報告する形にするか、別の口を設けるかを決め、Design に理由を書く。`db_memory` と `db_postgres` の双方を揃える。
- `EX-OAUTH2-005-08` の `expired` 遷移を実装する。**誰がいつ遷移させるかを先に決める。** 交換の拒否時に遅延で書くのか、掃除の job が持つのかで、監査に残る時刻の意味も、期限切れのまま提示されなかったコードの扱いも変わる。決めた理由を Design に書く。
- 3 件すべてについて、`tools/check/example-coverage-debt.json` の当該行から `blocked_by` と `finding` を外し、id を名指すテストを対応付けて台帳から削除する。
- 認可コード再提示の経路と refresh トークン再利用の経路が、同じ状況で同じイベントを出すことを 1 つのテストで固定する。片方だけ直すと、また片方が黙る。

## Out of Scope

- REQ-OAUTH2-005 のほかの具体例。[[wi-559-back-oauth2-remaining-examples-with-tests]] が持つ。
- `RefreshTokenReuseDetected` の発行。wi-559 が済ませた。
- 認可コードの TTL そのものの変更。60 秒は SCL の不変条件である。
- 監査イベントの一覧や保持期間の変更。

## Verification

- `mise run check-spec` が、`EX-OAUTH2-005-07`、`EX-OAUTH2-005-08`、`EX-OAUTH2-006-02` を台帳から外した状態で通る。
- `mise run test-go-package -- ./backend/oauth2/token/usecases`
- `mise run test-go-package -- ./backend/oauth2/token/db_postgres`
- `mise run verify`

## Risk Notes

- **失効の記録と通知を、片方だけで満足する。** この欠陥がそもそも「失効はするが黙っている」形だった。テストは保存先の状態とイベントの双方を読む。
- **`expired` 遷移を交換の拒否時にだけ書いて、済んだことにする。** それでは一度も提示されなかった期限切れコードは永久に `issued` のまま残る。決めた方式がその場合に何を残すかを、Design に書いて観測する。
- **`RevokeFamily` の署名変更が、失効そのものを弱める。** 返り値を足す変更は `db_postgres` の実装にも及ぶ。失効したことと、失効した token を報告することを、別々に観測する。
- **`_ = deps.RefreshStore.RevokeFamily(...)` がエラーを捨てている。** 現状は失効に失敗しても拒否だけ返る。署名を触るときに、この握り潰しを続けるかどうかも決める。
