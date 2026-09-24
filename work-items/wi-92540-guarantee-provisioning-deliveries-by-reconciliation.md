---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-24
priority: p2
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/domain/scenarios.feature.md, requirement: REQ-PLATFORM-003 }
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-003 }
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-004 }
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-005 }
---

# Provisioning の配信を、書き込み時の捕捉ではなく定期的な照合で保証する

## 動機

`REQ-PLATFORM-003` は「配信行は発火元の変更と同時にコミットまたはロールバックする」と定めている。
[[wi-22350-capture-provisioning-deliveries-in-the-mutation-transaction]] は、管理 API の経路でこれをトランザクションの作用範囲によって実現しようとした。
しかし、その過程で次のことが分かり、取り消した。

| 観察 | 内容 |
| --- | --- |
| 捕捉が経路ごと抜けている | LifecycleWorkflow による有効化と無効化（`idgovernance/usecases/lifecycle_workflow_dispatcher.go`）、SCIM による取り込み（`sourcing/scim/usecases/users.go`）、CSV インポート、利用者自身のプロフィール変更は、User を書き換えても配信行を作らない。障害がなくても、毎回確実に下流とずれる |
| 書き込み時の保証は漏れる | 記録の正を書き換える経路は今後も増える。経路ごとに捕捉を正しく呼ぶ前提は、新しい経路が増えるたびに破れる |
| 横断的な仕組みの量が見合わない | 原子性のために、全保存先を context のトランザクションへ参加させる仕組みと、memory 側の書き戻しの仕組みが必要になった。それでも上の漏れには効かない |
| 同一 DB の外は囲めない | Valkey に置く状態や外部の呼び出しは、どのみちトランザクションに入らない |

Provisioning の配信は状態ベースである。
下流へ送る属性は配信の時点の User から読み、`RemoteResourceLink` が下流に何があるかを記録している。
そのため、あるべき状態と下流へ反映済みの状態を突き合わせれば、取りこぼしの原因を問わずに回収できる。
Jobs が「配送は少なくとも 1 回、冪等性はハンドラーの責務」とし、dispatcher が未投入の配信行を拾い直すのと同じ考え方である。

## 対象範囲

- `REQ-PLATFORM-003` を、書き込み時の原子性ではなく、照合による最終的な反映の保証へ改める。
- 接続ごとに、スコープ内の User のあるべき状態と、下流へ反映済みの状態を突き合わせ、差分に対して `ProvisioningDelivery` を作る照合を実装し、定期ジョブとして走らせる。
- 書き込み時の捕捉（`notifyProvisioning`）は、反映の遅延を短くする近道として best-effort のまま残す。
- `tools/check/example-coverage-debt.json` の `EX-PLATFORM-003-01` と `EX-PLATFORM-003-02` の `blocked_by` を本項目へ付け替え、改訂後の具体例をテストで裏付けて台帳から外す。
- `docs/domain/provisioning/internals.md` の「同一トランザクションでの配信記録」と、`docs/domain/structure.md` のライフサイクルの通知の記述を現状に合わせる。

## 対象外

- 捕捉が抜けている各経路（LifecycleWorkflow、SCIM の取り込み、CSV インポート、プロフィール変更）への個別の捕捉の追加。照合が回収するので、遅延が問題になる経路が分かってから別の項目で扱う。
- Group の配信の照合。まず User で照合の形を確かめてから、同じ形で広げる。
- 配信の実行（`worker` による下流への反映）の信頼性。既存の Jobs の再試行が担う。

## 設計

### 照合の単位

照合は接続ごとに行う。
あるべき状態は、接続のスコープ（`all_users` または割り当て済みの User）と各 User の状態（有効、無効、削除予約、削除）から決まる。
反映済みの状態は、`RemoteResourceLink` と、その User について最後に確定した配信の `source_version` から読む。

| あるべき状態 | 反映済みの状態 | 作る配信 |
| --- | --- | --- |
| スコープ内で有効 | リンクなし | `create` |
| スコープ内 | リンクあり、User の版が最後に確定した配信より新しい | `update`、または状態に応じた無効化や再有効化 |
| スコープ外、または削除済み | リンクあり | 接続の `DeprovisionPolicy` に従う無効化または削除 |
| 上記以外 | — | なし |

割り当て解除は割り当ての行が消えるので、書き込み時の捕捉がないと差分として見えるのは「スコープ外なのにリンクがある」という形だけである。
上の表の 3 行目がこれを拾う。

### 既存の Full Resync との関係

管理者が手動で走らせる Full Resync（`REQ-PROVISIONING-013`、`usecases.StartFullResync`）は、スコープ内の全対象へ `update` の配信を一律に作る。
照合との違いは次の 2 点である。

- 照合は変わっていない対象を飛ばし、差分だけに配信を作る。
- 照合はスコープ外になったのにリンクが残っている対象へ、無効化または削除を作る。Full Resync はこれを作らない。

対象の選び方（`all_users` なら全 User、それ以外なら割り当て）は共通なので、照合の計算を実装したら、Full Resync をその計算の「差分を問わず全対象を送る」形として作り直せる。
Full Resync の完了の追跡は [[wi-22987-track-full-resync-completion]] が扱い、本項目とは独立に進められる。

### 冪等性

照合が作る配信も、既存の冪等キー `(tenant, connection, source_type, source_id, source_version)` を使う。
`source_version` には User の版を使い、書き込み時の捕捉と照合が同じ変更に対して同じ版を使うようにする。
これで、書き込み時の捕捉が成功した変更を照合が二重に配信することはない。
現在の捕捉は `now.UnixNano()` を版にしているので、User の `updated_at` から導く形へ揃える必要がある。

### 規範の改訂案

`REQ-PLATFORM-003` の規則文を次の趣旨へ改める。

- 記録の正の変更は、有効な接続を持つ下流へ、照合の周期以内に配信される。
- 書き込み時の捕捉の失敗や、発火元の経路が捕捉を呼ばないことは、この保証を破らない。

具体例は次の形へ改める。

- `EX-PLATFORM-003-01`: 変更が書き込み時に捕捉されれば、配信が作られて `succeeded` になる。
- `EX-PLATFORM-003-02`: 書き込み時の捕捉が失敗しても、次の照合で配信が作られる。
- 新しい具体例: 捕捉を呼ばない経路で状態が変わっても、次の照合で配信が作られる。

### 採用しない代替案

- **書き込み時の原子性（wi-22350 の方式）。** 経路ごとの漏れに効かず、全保存先を作用範囲へ参加させる仕組みが要る。取り消した実装の設計は wi-22350 に残っている。
- **すべての経路へ捕捉を追加して回る。** 今ある漏れは塞げるが、新しい経路が増えるたびに同じ漏れが再発する。
- **User の変更イベントを outbox に積み、それを中継する。** 配信行そのものが outbox の役割を果たしており、経路ごとに積み忘れる問題は変わらない。

## 計画

着手時に次の問いを解決する。どれも作るものを変える。

1. 照合の周期と、1 回の照合で読む量の上限。全件の走査にするか、`updated_at` の高水位線からの差分走査と、低頻度の全件走査を組み合わせるか。
2. 「最後に確定した配信の版」を配信行から都度求めるか、`RemoteResourceLink` に最後に反映した版を持たせるか。
3. 誤削除ガード（`REQ-PROVISIONING-011`）と照合の関係。照合が大量の無効化や削除を一度に作るとき、既存の閾値による隔離をそのまま効かせるか。
4. 照合を `worker` の定期ジョブにするか、`idmagic-batch` のコマンドにするか。
5. Full Resync を照合の計算の上へ作り直すことを本項目に含めるか、別の項目にするか。

## タスク

- [ ] T001 [Spec] `REQ-PLATFORM-003` と関連する Provisioning の具体例を改訂し、`mise run check-spec` を通す。
- [ ] T002 [Acceptance] 捕捉を呼ばない経路で User を無効化しても配信が作られないことを RED として観測する。
- [ ] T003 [App] 照合の計算（あるべき状態と反映済みの状態から作る配信の集合）を純粋な関数として実装する。
- [ ] T004 [App] 捕捉と照合の `source_version` を User の版に揃える。
- [ ] T005 [Adapter] 照合を定期ジョブとして組み立てる。
- [ ] T006 [Docs] Provisioning の内部設計と構造の記述を現状に合わせ、被覆台帳を更新する。
- [ ] T007 [Verify] 変更を検証する。

## 検証

- `mise run check-spec`
- `mise run test-go-changed`
- `mise run verify`

## リスク

- **照合が大量の配信を一度に作る。** 初回の照合や長い停止の後は差分が大きい。誤削除ガードの閾値と、1 回に作る配信の上限で抑える。
- **照合と書き込み時の捕捉が同じ変更を二重に配信する。** `source_version` を揃えて冪等キーで重複を落とす。揃えるまでは二重配信が起こり得るので、T004 を照合の有効化より先に行う。
- **照合の走査が DB を圧迫する。** 接続ごと、テナントごとに分割し、周期と 1 回の読み取り量を設定で制限する。
