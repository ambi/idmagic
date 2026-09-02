---
status: pending
authors: [tn]
risk: high
reversibility: irreversible
created_at: 2026-09-03
change_kind: bugfix
priority: p1
depends_on: [wi-467-enforce-csrf-on-tenant-quota-update]
affected_spec:
  - { path: docs/contexts/tenancy/scenarios.md, requirement: REQ-TENANCY-012 }
  - { path: docs/contexts/tenancy/scenarios.md, requirement: REQ-TENANCY-014 }
  - { path: spec/contexts/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.UpdateTenantQuota }
  - { path: spec/contexts/tenancy/models.tsp, symbol: IdMagic.Contract.TenantQuotaUpdateRequest }
---

# テナントクォータ更新経路を Tenancy 管理 API の規約へそろえる

## Motivation

`UpdateTenantQuota` は、Tenancy の制御面ハンドラーのうち唯一、共通の規約をどれも通らずに書かれている。

第一に、テナントの識別子を解決しない。

兄弟の経路はすべてパス引数を `resolveTenantByRealm` で realm から UUID へ解決するが、クォータ更新だけは受け取った文字列をそのまま `QuotaRepo.SetQuota` へ渡す。

`tenant_quotas.tenant_id` は `tenants(id)` を参照する UUID 列なので、同じパス引数が経路によって realm と UUID の二つの意味を持つ状態になっている。

管理画面もこの不整合をそのまま反映しており、属性更新と停止には `tenant.realm` を、クォータ更新には `tenant.id` を送っている。

兄弟の経路と同じ規約で呼ぶ API クライアントは、realm を送った時点でデータベース層の型エラーになる。

第二に、拒否を応答へ写像しない。

兄弟の経路は `WriteAdminAccessError` を通して 401 と 403 の Problem Details を書くが、クォータ更新は `requireSystemAdmin` が返した番兵エラーをそのまま返す。

この番兵エラーは HTTP 状態を持たないため、共通のエラーハンドラーが 500 として扱い、権限のない要求の拒否が「未処理の要求エラー」として記録される。

仕様は 403 を宣言し、この操作については 401 を宣言していないため、宣言された拒否がどちらも発生しない。

第三に、存在しないテナントに対して 404 を返さず、`writeTenantError` を通らない。

第四に、内部エラーの文字列を応答本文へ連結しており、データベースのエラーメッセージが呼び出し元へ出る。

第五に、要求本文を `support.DecodeJSON` ではなく `c.Bind` で読み、応答を `support.NoStoreJSON` ではなく `c.JSON` で書いている。

前者は未知フィールドの拒否と本文長の上限を失い、後者は管理応答に付くはずの `Cache-Control: no-store` を落とす。

そしてこの経路を名指しする HTTP テストは一つもない。

規約から外れていること自体は、[[wi-467-enforce-csrf-on-tenant-quota-update]] の CSRF 欠落と [[wi-470-record-control-plane-state-changes-in-the-audit-log]] の監査記録欠落と同じ根を持つが、識別子と応答契約の不統一はそれらのどちらでも直らない。

## Scope

- クォータ更新のテナント識別子を realm として解決し、兄弟の制御面経路と同じ意味にそろえる。
- 存在しないテナントを `writeTenantError` 経由の 404 で返す。
- 認証と認可の拒否を `WriteAdminAccessError` へ通し、宣言どおりの 401 と 403 の Problem Details にする。
- 保存失敗の応答から内部エラーの文字列を外す。
- 要求本文を `support.DecodeJSON` で読み、未知フィールドの拒否と本文長の上限を効かせる。
- 成功応答を `support.NoStoreJSON` で書き、`Cache-Control: no-store` を付ける。
- 仕様へ 401 と 404 を宣言し、パス引数 `tenant_id` が realm を指すことを doc に明示する。
- 管理画面のクォータ更新呼出しを `tenant.realm` へ変える。
- 成功、権限なし、未認証、存在しないテナント、未知フィールドを含む本文の各経路を HTTP 受け入れテストで固定する。

## Out of Scope

- クォータ更新の `Origin` と CSRF トークンの検証。
  [[wi-467-enforce-csrf-on-tenant-quota-update]] が扱う。
- クォータ更新の監査イベント発行。
  [[wi-470-record-control-plane-state-changes-in-the-audit-log]] が扱う。
- システムコンソールに要求する認証強度と再認証時刻。
  [[wi-468-system-console-privileged-session-assurance]] が扱う。
- クォータ値そのものの範囲検証、および同一テナントへの同時更新の調停。
- 上限の適用規則、Hard と Soft の区分、利用量の集計方法を変えること。
- ほかの管理 API のパス引数命名を一括で見直すこと。

## Design

パス引数の意味は realm にそろえる。

制御面のテナント経路は、作成、属性更新、正規ロケーション切替、停止、再開のすべてが realm を受け取っており、多数派に合わせるほうが、公開済みの URL 形と管理画面の遷移の両方に対して破壊が小さい。

逆に全経路を UUID へそろえる案は採らない。

realm は URL 上でテナントを表すための識別子であり、UUID を要求すると、運用者は上限を調整するためだけに一覧を引いて内部 ID を控える必要が生じる。

UUID も受け付けて両方を解決する案も採らない。

同じ位置に二つの識別子空間を許すと、realm が UUID の形をとる将来のテナントで曖昧になり、拒否の理由も「存在しない realm」と「存在しない ID」に分岐して増える。

この変更は、現在 UUID を送っている呼び出し元を壊す。

壊れる呼び出し元は管理画面と、この経路だけを UUID で呼ぶよう作られた自動化に限られ、いずれも realm を持っているため、移行は送る値の差し替えで済む。

拒否の写像は、番兵エラーを直接返す形をやめ、兄弟の経路と同じ `WriteAdminAccessError` の一本道にする。

共通のエラーハンドラー側で番兵エラーを 403 へ写像する案は採らない。

その写像を増やすと、ハンドラーが拒否を書かずに処理を続ける実装が検査を通ってしまい、応答だけが正しく副作用が残る誤りを見えなくする。

順序は CSRF 検証、認証と認可、識別子解決、本文の復号、保存とする。

識別子解決を本文の復号より前に置くのは、要求本文の妥当性が対象テナントの存在を観測する手段にならないようにするためである。

## Plan

1. realm を送る要求が失敗し、権限のない要求が 500 になる現在の挙動を HTTP 境界で観測し、RED を確認する。
2. 仕様へ 401 と 404 を宣言し、パス引数の意味を doc に明示して生成物を再生成する。
3. ハンドラーを共通ヘルパーの規約へそろえ、拒否、404、内部エラー本文、復号、応答ヘッダーを直す。
4. 管理画面のクォータ更新を realm へ切り替える。
5. 成功と各拒否経路を受け入れテストで固定し、拒否時に保存ポートが呼ばれないことを確認する。
6. 状態コードと契約のドリフト検査を通す。

## Tasks

- [ ] T001 [Acceptance] realm を送る要求の失敗と、権限のない要求が 500 になる挙動を観測し、RED を確認する。
- [ ] T002 [Spec] `UpdateTenantQuota` へ 401 と 404 を宣言し、`tenant_id` が realm であることを doc に書く。
- [ ] T003 [App] 識別子解決、拒否の写像、404、内部エラー本文の除去をハンドラーへ適用する。
- [ ] T004 [App] 要求本文の復号と応答の書き出しを共通ヘルパーへそろえる。
- [ ] T005 [UI] 管理画面のクォータ更新呼出しを `tenant.realm` へ変える。
- [ ] T006 [Acceptance] 成功、未認証、権限なし、未知テナント、未知フィールドの各経路を確認する。
- [ ] T007 [Unit] 拒否経路で `QuotaRepo.SetQuota` が呼ばれないことを確認する。
- [ ] T008 [Verify] 生成物を再生成し、契約とセキュリティ制御の検査を通す。

## Verification

- `mise run test-go-race`
- `mise run test-ui-unit`
- `mise run check-status-drift`
- `mise run check-contract-drift`
- `mise run check-security-controls`
- `mise run check-api-compat`
- `mise run check-spec`
- `mise run check-work-items`
- `mise run verify`

## Risk Notes

パス引数の意味を変えるため、現在 UUID を送っている呼び出し元は変更の時点で 404 を受け取る。

管理画面の切り替えを同じ変更に含め、移行が必要な呼び出し元を変更記録へ明示する。

拒否の写像を直すと、これまで 500 として観測されていた要求が 401 と 403 になる。

監視の閾値やアラートが 5xx の発生を前提にしていないことを確認する。

未知フィールドの拒否を有効にすると、余分なキーを送っていた既存の自動化が 400 になる。

これは意図した契約であり、受け入れテストで固定する。

公開されたパス引数の意味と宣言される状態コードが変わり、外部の呼び出し元が送る値を変更する必要があるため、`reversibility` は irreversible とする。
