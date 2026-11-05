---
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-11-06
priority: p1
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/tenancy/features/quota.md, requirement: REQ-TENANCY-036 }
  - { path: docs/modules/authentication/features/session.md, requirement: REQ-AUTHENTICATION-035 }
---

# ログインセッションの終了を一つの操作に集め、`active_sessions` の使用量を実際の件数に保つ

## 動機

[wi-10874](../done/wi-10874-inventory-promises-kept-by-every-entry-path.md) の棚卸しで、ログインセッションを終える経路のうち、`active_sessions` の使用量を減らすものが一部だけであることが分かった。
加算は `SessionManager.CreateWithPending` がログインのたびに行い、本番の組み立ては `QuotaRepo` を必ず渡す。

| 経路 | 呼ぶ操作 | 使用量の減算 | `SessionEnded` |
| --- | --- | --- | --- |
| 本人と管理者による失効、OIDC のログアウト | `sessions.go` のユースケース | あり | あり |
| SAML の SLO、WS-Fed のサインアウト | `SessionManager.Revoke` が `Store.Revoke` を直接呼ぶ | なし | なし |
| 主体が無効になったセッション | `SessionAuthentications.EndSession` が `Store.Revoke` を直接呼ぶ | なし | なし |
| 期限切れの掃除 | `DeleteExpiredBatch` | なし | なし |
| User の完全削除、受信した SET | `DeleteAllForSub` | なし | なし |

多くのセッションは明示的なログアウトではなく期限切れで終わる。
使用量は増え続け、再集計もないため、既定の 50,000 に達したテナントは誰もログインできなくなる。
[REQ-TENANCY-036](../../docs/modules/tenancy/features/quota.md) は削除したリソースの減算を求めるので、これは違反である。
`SessionEnded` の欠落は、[REQ-AUTHENTICATION-035](../../docs/modules/authentication/features/session.md) がプロトコルによらずサーバーの側の失効を求めるのに対し、記録の有無を定めていない規範の取りこぼしである。

## 対象範囲

- 期限切れ、完全削除、受信した SET を含む全経路で、`active_sessions` の使用量が `Active` のセッションの件数からずれ続けないようにする。
- SAML の SLO、WS-Fed のサインアウト、主体が無効になったセッションの終了を、使用量と `SessionEnded` を伴う一つの操作へ通す。
- 終了の経路と `SessionEnded` の有無を規範に書く。

## 対象外

- 既存のテナントでずれた使用量の一度きりの補正。下の再集計を採れば次の実行で回収される。
- セッションの上限値の変更。

## 設計

### 所有の形

二つの候補を比べる。

| 候補 | 内容 | 判断の材料 |
| --- | --- | --- |
| A | 終了を Authentication のセッションのユースケース一つに置き、`SessionStore.Revoke` と削除の系統を `authentication/internal` の外から呼べなくする。掃除と一括削除は消した件数を減算する | 掃除は `DELETE ... LIMIT` の件数をテナントごとに数える必要がある |
| A と再集計 | 終了の操作は A に集め、使用量は REQ-TENANCY-049 と同じく保持期限バッチが `Active` のセッションの件数で置き換える | 期限切れは時刻の経過で起き、削除の前は件数に入り続ける。加算と保存の間のずれも回収できる |

着手時に二つを比べ、期限切れが掃除の前に使用量を占め続ける時間を許せるかで選ぶ。
`SessionManager.Revoke` と `EndSession` は、終了のユースケースを呼ぶ形に置き換える。

## タスク

- [ ] T001 [Spec] 終了の経路ごとの使用量と `SessionEnded` を規範に書く。
- [ ] T002 [Test] SLO と期限切れの後に使用量が減らないことを RED にする。
- [ ] T003 [App] 終了を一つのユースケースへ集め、直接の `Store.Revoke` を消す。
- [ ] T004 [App] 選んだ方式で掃除と一括削除の使用量を保つ。

## 検証

- `mise run test-go-package -- ./backend/authentication/internal/session/usecases`
- `mise run test-go-package -- ./backend/tenancy/...`
- `mise run check-spec`

## リスク

- 再集計を採る場合、件数の数え方を Authentication が Tenancy へ渡す方向は、REQ-TENANCY-049 の User と同じ依存の向きにする。
- 公開範囲を狭めると、テストの組み立てが `Store.Revoke` を直接呼ぶ箇所が壊れる。
