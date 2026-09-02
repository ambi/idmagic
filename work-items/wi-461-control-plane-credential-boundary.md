---
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-09-03
change_kind: feature
priority: p2
depends_on: []
affected_spec:
  - { path: docs/contexts/tenancy/scenarios.md, requirement: REQ-TENANCY-011 }
  - { path: docs/contexts/tenancy/scenarios.md, requirement: REQ-TENANCY-003 }
  - { path: spec/contexts/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.SetTenantEndpointStyle }
  - { path: spec/contexts/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.DisableTenant }
  - { path: spec/contexts/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.EnableTenant }
---

# 破壊的な制御面操作を API アクセストークンから到達不能にし、対話セッション限定にする

## Motivation

いま、制御面の操作の中で最も破壊的な 2 つが、最も弱い資格情報から到達できる。

| 操作 | 宣言 | 何が起きるか |
| --- | --- | --- |
| `SetTenantEndpointStyle` | `tenants:write` (`spec/contexts/tenancy/main.tsp:241`) | issuer が変わり発行済みトークンの `iss` 検証が壊れ、WebAuthn RP ID が変わり既存のパスキーがすべて無効になり、Cookie スコープが変わり進行中のセッションが切れる |
| `DisableTenant` | `tenants:write` (`spec/contexts/tenancy/main.tsp:173`) | そのテナントの `/authorize`、`/token`、ログインがすべて `invalid_request` になる |

一方、参照しかしない `ListTenantDataKeyHealth` は `interactive_session` を宣言し (`spec/contexts/data-keys/main.tsp:31`)、どのトークンからも到達できない。**強度の順序が逆になっている。**

`docs/authorization.md` は操作を対話セッション限定にする理由を 2 つ挙げている。既存のスコープ語彙に対応がない場合と、操作自体が権限の昇格経路になる場合である。テナントの無効化と正規ロケーションの切り替えはどちらにも当たらないので、現在の規則からはこの宣言が出てこない。しかし対象テナントの利用者全員を締め出し、既に配った資格情報の前提を壊す操作を、失効まで有効な bearer token 1 本で実行できてよい理由もない。**規則の側に「破壊的な制御面操作」という軸がないことが原因である。**

`docs/contexts/tenancy/decisions.md` はこの到達経路を認識したうえで記録している。「制御面の操作は `admin:tenants_manage` の条件を満たす利用者が自身のテナントで発行したトークンから到達しうるので、`tenants:*` にも到達経路がある」。到達しうることは書かれているが、破壊的な操作までそれでよいかは判断されていない。

なお、制御面の操作へ到達するには制御面テナントに所属する `system_admin` であることが要るため、これは誰でも踏める経路ではない。塞ぐのは、**正規の運用者が発行した長命なトークンが漏れたとき、あるいは自動化が誤ったときに、テナントを停止できてしまう**ことである。

## Scope

- `SetTenantEndpointStyle`、`DisableTenant`、`EnableTenant` を `interactive_session` 限定にする。
- `docs/authorization.md` の「対話セッション限定の操作」に 3 つ目の理由 (破壊的な制御面操作) を書き、どの操作がそれに当たるかの判断基準を示す。
- `docs/contexts/tenancy/decisions.md` の記述を、`tenants:*` が届く範囲と届かない範囲が分かる形に更新する。
- `REQ-TENANCY-011` に、API アクセストークンからは到達できないことを規範として書く。テナントの無効化には現在シナリオがないため、`REQ-TENANCY-003` の `ALT` として同じ規範を置く。
- 管理コンソールの該当操作が対話セッションで動き続けることを確認する。

## Out of Scope

- 制御面操作への step-up と再認証。管理ポータル側に step-up の機構がなく (`frontend/src/components/StepUpDialog.tsx` はアカウントポータルだけが使う)、導入は認証機構そのものの変更になる。本 work item は到達できる資格情報の種類を決めるだけで、対話セッションの強度は変えない。step-up を入れる判断は別の work item が持つ。
- `CreateTenant`、`UpdateTenant`、`UpdateTenantQuota` を対話セッション限定にすること。テナントの払い出しと上限調整は自動化に正当な用途があり、失敗しても対象テナントの利用者を締め出さない。
- `tenants:disable` のような新しいスコープ値を作ること。却下の理由は Design に書く。
- テナント横断のヘルス参照の認可欠陥。[[wi-460-cross-tenant-health-control-plane-membership]] が持つ。
- テナント横断操作の UI 集約。[[wi-462-control-plane-console-single-entry]] が持つ。
- API アクセストークンの寿命、失効、発行時の承認。トークン一般の性質であり、制御面に固有の判断ではない。

## Design

### 選んだ規則

**対象テナントの利用者が到達できなくなる制御面の操作は、対話セッション限定とする。** 判定は破壊性であり、読み書きの別ではない。この軸を `docs/authorization.md` の「対話セッション限定の操作」へ 3 つ目の理由として加える。

現在この基準に当たるのは `SetTenantEndpointStyle`、`DisableTenant`、`EnableTenant` の 3 つである。`EnableTenant` は復旧側だが、`DisableTenant` と対にして同じ強度に置く。片方だけを人間に閉じると、止められないのに動かせるという非対称ができ、停止と再開が別の資格情報で行われた記録が残る。

`CreateTenant`、`UpdateTenant`、`UpdateTenantQuota` は `tenants:write` のままとする。テナントの払い出しは請求や契約の系から自動化する用途が実在し、失敗しても既存テナントの利用者は締め出されない。上限の変更は `REQ-TENANCY-013` により作成を拒否しうるが、既に発行した資格情報の前提は壊さない。

### 却下した選択肢

- **`tenants:disable` のような新しいスコープ値を作り、自動化からの停止を許す。** `jobs:cancel` という粒度の先例はあるが、停止を自動化したい要求がまだない。スコープ値は `ApiTokenScope` の公開語彙なので、増やしたあとに減らすのは難しい。要求が現れてから追加する方が安い。
- **何もせず、`tenants:write` のままにする。** 到達経路があること自体は文書化済みなので、隠れた欠陥ではない。しかし記録されているのは「到達しうる」ことだけで、破壊的な操作までそれでよいという判断は下されていない。判断しないまま置くと、単一プロセスが判断なのか偶然なのか読み取れないのと同じ状態が残る。
- **`tenants:*` を持つトークンの発行自体を禁じる。** 参照系 (`tenants:read`) には在庫把握という正当な用途があり、まとめて塞ぐと必要なものまで消える。
- **管理コンソールからの呼び出しだけを許す判定を入れる。** 到達元のクライアントで判定することになり、`docs/authorization.md` の「UI での非表示は認可判定ではない」と同じ誤りを、サーバー側で犯すことになる。資格情報の種類で決める。

### 影響の範囲

`x-api-token-scopes` を `interactive_session` に変えると、API アクセストークンからの到達はフェイルクローズで塞がる (`docs/authorization.md` の対応づけ)。管理コンソールはブラウザーのログインセッションで呼ぶため影響を受けない。したがって実装の変更は宣言側に閉じる見込みだが、着手時に強制の経路が宣言だけを見ていることを確認する。`mise run check-admin-scopes` が全 admin operation の宣言と `ApiTokenScope` 語彙の対応を検査するので、宣言の変更がその検査を通ることを確かめる。

**これは公開契約の破壊的変更である。** `tenants:write` のトークンでテナントの停止や正規ロケーションの切り替えを自動化している利用者がいれば動かなくなる。着手時に `documentation_impact` を `upgrade_note` として宣言し、代替 (制御面テナントの管理者が管理コンソールから実行する) を書く。

## Plan

1. `docs/authorization.md` に 3 つ目の理由を書き、判断基準を定める。基準を先に決めないと、対象の 3 操作が恣意的な選択に見える。
2. `docs/contexts/tenancy/decisions.md` を更新し、`tenants:*` が届く操作と届かない操作を書き分ける。
3. `REQ-TENANCY-011` と `REQ-TENANCY-003` に規範を追加する。
4. TypeSpec の `x-api-token-scopes` を 3 操作について変更し、doc に理由を書く。
5. 受け入れ境界で RED を確認する。`tenants:write` のトークンが現在は 3 操作へ到達できることを観測してから塞ぐ。
6. 管理コンソールからの実行が変わらないことを確認する。
7. アップグレードノートを書く。
8. 検査を通す。

## Tasks

- [ ] T001 [Spec] `docs/authorization.md` に破壊的な制御面操作の理由と判断基準を追加する。
- [ ] T002 [Spec] `docs/contexts/tenancy/decisions.md` を更新する。
- [ ] T003 [Spec] `REQ-TENANCY-011` と `REQ-TENANCY-003` に規範を追加する。
- [ ] T004 [Spec] `SetTenantEndpointStyle`、`DisableTenant`、`EnableTenant` の `x-api-token-scopes` を `interactive_session` にする。
- [ ] T005 [Acceptance] `tenants:write` のトークンが 3 操作へ到達できることを HTTP 境界で観測し、RED を確認する。
- [ ] T006 [App] 強制の経路が宣言どおりに塞ぐことを確かめ、必要なら実装を合わせる。
- [ ] T007 [App] 管理コンソールからの実行が変わらないことを確認する。
- [ ] T008 [Docs] アップグレードノートを書く。
- [ ] T009 [Verify] 検査を通す。

## Verification

- `mise run check-admin-scopes`
- `mise run check-api-compat`
- `mise run test-go-race`
- `mise run check-spec`
- `mise run check-ids`
- `mise run verify`

## Risk Notes

リスクは high。公開契約から能力を取り除く変更であり、誤ると自動化を壊すか、逆に塞いだつもりで塞げていない。

**塞ぎ漏れが最も危ない誤りである。** 宣言を変えただけで強制が変わらなければ、契約は厳しく見えるのに到達経路は残る。`docs/authorization.md` の対応づけはフェイルクローズだが、それに依存していることを受け入れテストで確かめる。宣言の差分を読むだけで完了としない。

**自動化を壊す方向の影響は、こちらから見えない。** リポジトリの中に `tenants:write` でテナントを停止する呼び出し元はないが、外部の利用者は分からない。アップグレードノートで代替を明示する。

**判断基準が曖昧だと、次の操作で同じ議論を繰り返す。** 「破壊的」を「対象テナントの利用者が到達できなくなる」と定義し、`docs/authorization.md` に書く。定義を書かずに 3 操作だけを列挙しない。

`reversibility` は reversible。宣言を戻せば元の到達経路に戻り、外部が保存した値の意味は変わらない。ただし戻すまでの間、既存の自動化は動かない。新しいスコープ値を足す案を却下したのは、そちらが公開語彙を増やして取り消せなくなるためである。
