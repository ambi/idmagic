---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-13
priority: p3
depends_on: []
change_kind: bugfix
spec_impact: { kind: none, reason: "契約と具体例が宣言している拒否の形に、実装が従っていない。規範は動かさず実装を合わせる。" }
affected_spec:
  - { path: docs/domain/identity-management/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-009 }
---

# Agent 管理 API が、契約の宣言に無い 404 と 409 で拒否する

## Motivation

`EX-IDMANAGEMENT-009-04` は、別テナントの `client_id` を Agent にバインドする要求について「エラー "InvalidRequestError"」を要求する。実装が返すのは 404 の `client_not_found` である。

```
status=404 body={"type":"urn:idmagic:error:client_not_found","title":"Client not found","status":404,...}
```

食い違いは具体例 1 件にとどまらない。TypeSpec が宣言する Agent 管理 API 10 個の応答は、いずれも 200/201/204、400、401、403、422 だけである。

| operation | 宣言している status |
| --- | --- |
| RegisterAgent | 201, 400, 403, 422 |
| GetAgent / ListAgents | 200, 400, 403 |
| UpdateAgent | 200, 400, 403, 422 |
| DisableAgent / EnableAgent / KillAgent / DeleteAgent | 204, 400, 403 |
| BindAgentCredential / UnbindAgentCredential | 204, 400, 403 |

一方 `writeAdminAgentError` (`backend/idmanagement/agent/handlers_http/admin_agent_handler.go`) は 5 つの拒否を宣言に無い status で書く。

| エラー | 書いている status | code |
| --- | --- | --- |
| `ErrAgentNotFound` | 404 | `agent_not_found` |
| `ErrAgentClientNotFound` | 404 | `client_not_found` |
| `ErrAgentNameConflict` | 409 | `agent_name_conflict` |
| `ErrAgentKilled` | 409 | `agent_killed` |
| `ErrAgentClientBound` | 409 | `agent_client_already_bound` |

`mise run check-status-drift` は現状 0 finding で通る。342 operation のうち全体を読めているのは 83 個で、Agent の 10 個は「一部だけ読めた」側に入っているためである。検査が見ていない範囲でずれが育った形なので、直したうえで検査が届くようにするか、届かない理由を記録に残す。

[[wi-540-back-identity-management-examples-with-tests]] が `EX-IDMANAGEMENT-009-04` を消化しようとして測り、台帳へ残した。

## Scope

- Agent 管理 API の拒否を、契約が宣言する status へ揃える。**宣言の側を実装へ合わせるのではなく、実装を宣言へ合わせる。** 404 と 409 を新たに宣言すると、`agent_not_found` が「Agent の存在」を越境した呼び出し元へ漏らす形になる。403 と 404 を撃ち分ける管理 API は、テナントの分離を status で破る。
- `EX-IDMANAGEMENT-009-04` を `tools/check/example-coverage-debt.json` から外し、`backend/idmanagement/handlers_http/scenario_examples_test.go` の
  `TestBindingAForeignTenantsCredentialLeavesTheAgentUnbound` が具体例 id を名指すようにする。同テストは既に「拒否されること」と「関連付けが残らないこと」を観測しており、足りないのは拒否の形だけである。
- `mise run check-status-drift` が Agent の 10 operation を全体まで読めるようにするか、読めない理由を記録する。

## Out of Scope

- Agent 以外の管理 API が宣言外の status を書いていないかの全数調査。本項目は Agent の 10 operation に閉じる。
- シナリオと具体例の書き換え。
- 拒否の `code` (`agent_not_found` など) の改名。status だけを揃える。

## Verification

- `mise run check-spec`
- `mise run check-status-drift`
- `mise run test-go-package -- ./backend/idmanagement/handlers_http`
- `mise run test-go-package -- ./backend/idmanagement/agent/handlers_http`

## Risk Notes

- **status だけを書き換えて、拒否の理由を運ぶ経路を壊す。** 呼び出し元は `type` の URN 接尾辞で分岐する。status を 400 へ揃えても `code` は保つ。
- **404 を 400 へ寄せると、存在しない Agent と権限の無い Agent が同じ応答になる。** それは意図した側である。テナント境界を status で漏らさないことが、この揃えの目的である。
