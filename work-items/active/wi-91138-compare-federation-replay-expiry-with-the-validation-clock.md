---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-11-28
priority: p1
group: development-process
depends_on: []
change_kind: maintenance
spec_impact: { kind: none, reason: "応答の再送を拒否する規則と有効期限の長さは変えない。再送の予約が期限切れを判定する時刻を、検証に渡した時刻へそろえるだけで、HTTP 契約、永続状態の形、イベントは変えない。" }
---

# 連携の再送の予約で、期限切れを検証に渡した時刻と比べる

## 動機

0a5f4667e で追加した `TestValidateResponse_RejectsKeycloakResponsesOutsideTheirContext` は、記録から 10 分を過ぎると `a replayed response must be rejected` で失敗する。
2026-11-28 の時点で、Go の変更を含まないブランチでも `mise run test-go` が失敗した。
`main` の標準ゲートが全員の作業で失敗するため、グループの基準の p2 より上げて p1 とする。

原因は時刻の出どころの食い違いである。
`ValidateResponse` は呼び出し側から受け取った `now` で予約の期限（`now + 10 分`）を作る。
一方、`ReplayStore.Reserve` は既存の予約の期限切れを、メモリ実装では `time.Now()`、PostgreSQL 実装では SQL の `now()` と比べる。
テストは記録した `received_at`（`testdata/keycloak-saml.json`、2026-11-28T02:21Z）を `now` として渡すので、記録から 10 分を過ぎると 1 回目の予約がもう期限切れと見なされ、2 回目の予約も成功する。
本番では `now` が実時計とほぼ同じなので、再送の拒否は今も成り立つ。

## 対象範囲

- `federationports.ReplayStore.Reserve` の期限切れの判定を、呼び出し側の時刻で行う。
- メモリ実装と PostgreSQL 実装の両方を同じ判定にそろえ、呼び出し側（SAML の応答検証、Kerberos の交渉）から時刻を渡す。
- 記録の時刻を使う再送のテストが、実行した日時によらず通るようにする。
- 二つの実装が同じ判定をすることを、共有の契約テストで確かめる。

## 対象外

- 再送の予約を保持する期間（10 分）と、期限切れの行の掃除の方式。
- ほかのストレージのアダプターが実時計で判定する箇所と、それを止める規則と検査。[wi-15471](wi-15471-keep-storage-adapters-off-the-wall-clock.md) が扱い、この項目は main の `test-go` を直す最小の変更に留める。

## 設計

| 案 | 判断 |
| --- | --- |
| A. `Reserve` に判定の時刻 `now` を足し、両方の実装がそれと比べる | 採る。検証の時刻を一か所から渡せるので、テストで時刻を固定でき、実装の間の差もなくなる |
| B. テストだけを直し、記録の時刻ではなく実時計で検証する | 採らない。記録した応答の `NotOnOrAfter` などの検証が実時計では成り立たず、記録を使う意味がなくなる |
| C. テストで予約を期限で判定しない偽物に差し替える | 採らない。再送の拒否を実装の判定で確かめなくなり、メモリ実装の誤りを見逃す |

PostgreSQL 実装は `DeleteExpiredReplays` の `now()` を引数の時刻へ替える。
DB の時計とアプリケーションの時計の差に依存しなくなるが、判定の意味は変わらない。

## 計画

1. 共有の契約テストに、期限内の再予約を拒否し、期限後の再予約を受け入れることを、呼び出し側の時刻で確かめる表を足し、メモリ実装で RED を観測する。
2. ポートと二つの実装、呼び出し側を変える。
3. `TestValidateResponse_RejectsKeycloakResponsesOutsideTheirContext` が記録から 10 分を過ぎた時刻でも通ることを確かめる。

## タスク

- [ ] T001 [Test] 再送の予約の契約テストを足し、RED を観測する。
- [ ] T002 [Adapters] `Reserve` に時刻を渡し、メモリ実装と PostgreSQL 実装をそろえる。
- [ ] T003 [Verify] SAML と Kerberos の検証のテストと、PostgreSQL の契約テストを通す。

## 検証

- `mise run test-go-package -- ./backend/authentication/internal/federation/...`
- PostgreSQL の契約テストは DB を使うので、AGENTS.md の実行方法に従う。

## リスク

呼び出し側が時刻を渡し忘れると、ゼロ値の時刻と比べて予約が常に期限内と見なされ、正当な応答まで拒否する。
時刻の引数を必須の位置に置き、ゼロ値を拒否する。
