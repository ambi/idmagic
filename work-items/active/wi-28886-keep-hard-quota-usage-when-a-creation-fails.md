---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-11-06
priority: p2
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/tenancy/features/quota.md, requirement: REQ-TENANCY-013 }
  - { path: docs/modules/tenancy/features/quota.md, requirement: REQ-TENANCY-049 }
---

# 作成に失敗したリソースが Hard Quota の使用量を占め続けないようにする

## 動機

[wi-10874](../done/wi-10874-inventory-promises-kept-by-every-entry-path.md) の棚卸しで、Hard Quota の作成の経路が、保存の前に使用量を加算し、その後の検証や保存の失敗で加算を戻さないことが分かった。

| リソース | 加算する箇所 | 加算の後に失敗しうるもの |
| --- | --- | --- |
| `groups` | `CreateGroup` | `group.Validate`、`GroupRepo.Save` の一意性の衝突 |
| `agents` | `CreateAgent` | `agent.Validate`、`AgentRepo.Save` |
| `applications` | `CreateApplication` | `ValidateApplication`、保存 |
| `oauth2_clients` | `RegisterClient`（管理 API と動的登録） | `redirect_uris` の欠落、`private_key_jwt` の鍵、`Validate`、保存 |
| `ssf_streams` | ストリームの作成 | 検証と保存 |
| `consents` | 同意の HTTP ハンドラー、Agent への付与 | 保存 |

動的登録の不正なメタデータは、外部のクライアントが繰り返し送れる。
失敗のたびに使用量が一つ残り、再集計もないため、上限に達すると正当な作成まで拒否される。
`users` だけは [REQ-TENANCY-049](../../docs/modules/tenancy/features/quota.md) の再集計が次の実行でずれを回収する。
[REQ-TENANCY-013](../../docs/modules/tenancy/features/quota.md) は上限の超過で拒否したときに使用量を変えないと定めるが、ほかの理由で作成に失敗したときを定めていない。
規範の取りこぼしであり、作れなかったリソースが使用量を占める現在の振る舞いは上限の目的に反する。

## 対象範囲

- 作成に失敗したリソースが、`users` 以外の Hard Quota の使用量を占め続けないようにする。
- 失敗した作成と使用量の関係を規範に書く。

## 対象外

- `active_sessions` と `active_jobs`。それぞれ [wi-32983](wi-32983-end-login-sessions-through-one-operation.md) と [wi-83761](wi-83761-release-the-active-job-quota-on-every-job-cancellation.md) が扱う。
- User の削除が同意の使用量を減らさない欠陥。[wi-51164](wi-51164-release-the-consent-quota-when-a-user-is-deleted.md) が扱う。

## 設計

### 所有の形

三つの候補を比べる。

| 候補 | 内容 | 判断の材料 |
| --- | --- | --- |
| A：同じ確定の単位 | 加算と保存を一つのトランザクションにする | 保存先と `tenant_usages` が同じ DB にある。モジュールごとの確定の単位を変える範囲が広い |
| A：補償 | 加算した操作が、失敗の経路で減算する | 失敗の経路をすべて覆う必要があり、取りこぼしの形が残る |
| 再集計 | REQ-TENANCY-049 と同じく、保持期限バッチが実際の件数で使用量を置き換える | 所有者ごとの件数の測定が要る。ずれは次の実行まで残る |

着手時に、作成と削除の両方のずれを一つの仕組みで回収できる再集計を第一に比べる。
同意の加算は HTTP ハンドラーにあるので、どの候補でも OAuth2 の同意のユースケースへ移す。
2026-11-13 時点で、Agent への代行の付与の加算は、すでに同意のユースケース（`backend/oauth2/internal/consent/usecases/agent_delegations.go`）にある。

Application の削除は、削除した件数を保存と同じトランザクションで減算している（`backend/application/internal/db_postgres/deletion.go`）。
「同じ確定の単位」の候補を比べるとき、この前例を作成の側へ広げられるかを確かめる。

## タスク

- [ ] T001 [Spec] 作成に失敗したときの使用量を規範に書く。
- [ ] T002 [Test] Group の名前の衝突と動的登録の不正なメタデータの後に使用量が残ることを RED にする。
- [ ] T003 [App] 選んだ方式を各リソースへ入れる。

## 検証

- `mise run test-go-package -- ./backend/tenancy/...`
- 選んだ方式が触れるモジュールの `mise run test-go-package`。
- `mise run check-spec`

## リスク

- 再集計を採る場合、件数を数えるモジュールから Tenancy への入力の向きを、REQ-TENANCY-049 の User と同じにする。
