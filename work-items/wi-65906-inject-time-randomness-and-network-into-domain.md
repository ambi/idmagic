---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-09
priority: p3
depends_on: []
change_kind: refactor
spec_impact: { kind: none, reason: "domain のパッケージが自分で得ていた現在時刻、乱数、ネットワークの値を、usecases から引数で受け取る形へ変えるだけである。生成する識別子と期限の規則、HTTP の応答、発行するドメインイベント、永続状態は変えない。" }
---

# domain のパッケージが現在時刻、乱数、ネットワークを直接使わないようにする

## 動機

`docs/domain/structure.md` は、`domain` で `time.Now`、`crypto/rand`、`math/rand`、OS、ネットワークを直接使わないと定めている。
`tools/check/boundary-debt.json` には、この規則の違反（`domain-effect`）が 20 件ある。

| Context | ファイル | 使っているもの |
| --- | --- | --- |
| Authentication | `federation/domain/models.go` | `net`、`time.Now` |
| Authentication | `trusteddevice/domain/trusted_device.go` | `crypto/rand` |
| IdManagement | `group/domain/group_csv.go` | `net` |
| OAuth2 | `approval/domain/approval_request.go`、`authorization/domain/authorization_code.go`、`device/domain/device_authorization.go`、`token/domain/refresh_token.go` | `crypto/rand`、`time.Now` |
| OAuth2 | `client/domain/cimd.go`、`client/domain/client.go`、`domain/mcp_resource_server.go` | `net`、`time.Now` |
| Provisioning | `domain/connection.go` | `net` |
| SAML | `domain/authnrequest.go` | `net`、`time.Now` |
| Seeding | `domain/plan.go` | `net` |

時刻と乱数を内部で得る関数は、テストで値を固定できず、期限や識別子の性質を決定的に検証できない。

## 対象範囲

- `time.Now` と `crypto/rand` を使う関数を、現在時刻と乱数（またはトークンの生成器）を引数で受け取る形へ変え、`usecases` から渡す。
- `net` の使用を、ファイルごとに分類する。
  URL や IP アドレスの構文の検証のように入力だけで決まる計算に `net` の型と関数を使っているものは、検査が誤って作用とみなしているかを確かめる。
  名前解決や接続を行っているものは `ports` へ移す。
- 解消した違反 ID を `tools/check/boundary-debt.json` から消す。

## 対象外

- 識別子や期限の生成規則の変更。

## 設計

`net` の扱いは、着手時に分類の結果で決める。
`net.ParseIP` や `net/url` の構文解析は決定論的な計算であり、`structure.md` が禁じる「ネットワークへの直接アクセス」ではない。
分類の結果、構文解析しか使っていないファイルが多ければ、検査の規則を「`net` の import」から「名前解決と接続の関数の呼び出し」へ狭めるほうが規則の意図に合う。
その場合は検査の変更をこの項目に含め、規則の文言と検査を一致させる。

## タスク

- [ ] T001 [Inventory] `net` を使う 7 ファイルを、構文解析だけか、名前解決や接続を行うかで分類する。
- [ ] T002 [App] 現在時刻と乱数を引数で受け取る形へ変える。
- [ ] T003 [App] 分類に従い、`net` の使用を `ports` へ移すか、検査を狭める。
- [ ] T004 [Tooling] 解消した違反 ID を `tools/check/boundary-debt.json` から消す。
- [ ] T005 [Verify] 変更を検証する。

## 検証

- `mise run check-boundaries`
- `mise run verify`

## リスク

- 乱数の供給元を引数にすると、本番の配線で弱い乱数を渡す誤りがありうる。
  本番の配線では `crypto/rand` を使う生成器だけを渡し、`math/rand` を受け付けない型にする。
