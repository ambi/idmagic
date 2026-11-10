---
status: pending
authors: [tn]
risk: high
reversibility: irreversible
created_at: 2026-11-11
priority: p1
depends_on: [wi-96973-guide-entra-federation-setup-and-verify-applied-settings]
change_kind: feature
affected_spec:
  - { path: docs/modules/ws-federation/features/relying-party.md, requirement: REQ-WSFEDERATION-001 }
  - { path: spec/modules/ws-federation/main.tsp, symbol: IdMagic.WsFederation.Operations.DeleteWsFedRelyingParty }
---

# Entra 側の切り戻しを確認してからドメイン単位でフェデレーションを解除する

## 動機

Entra 向けの RP は複数のドメインを一つに保持するが、現在の一覧の削除は RP 全体を消す。
この操作は Entra 側の設定を解除しないため、IdMagic へ認証を送る外部設定を残したまま信頼設定を失う。
一つのドメインの解除が他のドメインのサインインも止める操作になっている。

## 対象範囲

- ドメイン単位の解除を管理 API と管理画面に追加し、Entra 専用 RP を汎用の削除で消す操作を拒否する。
- 現在の外部設定とローカル設定を保存し、Managed への切り替えまたは保存済みの以前の IdP への復元を案内する。
- Graph から読み戻した対象ドメインの結果を確認した後、対象だけをローカルの構成から外す。
- 残るドメインの issuer、sourceAnchor、署名鍵、passive/active 認証を保ち、最後のドメインを外す場合の RP の後始末を定義する。

## 対象外

- IdMagic からの Graph 自動変更、Managed 認証のためのパスワード同期や認証方式の構築。
- 一般の WS-Fed RP、Application 所有の RP、Entra テナントやドメインそのものの削除。

## 設計

WsFederation の所有する解除操作で、対象ドメインと設定の版、切り戻し先、外部状態の確認を受け取る。
他ドメインの構成から対象の一件だけを除く計算と、認可、競合検査、保存、監査を分ける。
観測値の取り込みと鮮度の扱いは [[wi-96973-guide-entra-federation-setup-and-verify-applied-settings]] を使う。
観測が IdMagic を指したまま、未確認、別テナント、別ドメイン、設定の版違いの場合はローカル設定を残す。
再実行、対象なし、最後のドメイン、同時更新、監査イベントの結果を要件にする。

外部変更とローカル保存を一つの原子的な操作とは扱わない。
Entra の切り戻しだけ成功した場合は、残っている IdMagic 設定から再開できるようにする。
ローカル削除を先に行う方式と、確認ダイアログだけで RP 全体を消す方式は採らない。
既存の汎用 DELETE による Entra 専用 RP の削除は互換性が変わるため、拒否と専用操作への案内を TypeSpec と仕様へ明記する。
鍵は共有されるので、この解除で破棄しない。

## 計画

1. 外部状態の許容条件と再試行、汎用 DELETE の拒否、ドメインを除いた後の保存規則を仕様と TypeSpec に定める。
2. Microsoft 公式資料で Managed への切り替えと以前の federation configuration の復元手順を確認し、バックアップからの復旧を含めて runbook にする。
3. 専用 API、確認画面、切り戻しコマンドと読み戻し、削除後の遷移を実装する。

## タスク

- [ ] T001 [Spec] ドメイン単位の解除と RP 全体の誤削除の拒否を定める。
- [ ] T002 [Acceptance] 二つのドメインのうち一つを解除しても他方で署名とサインインが成立する試験を先に置く。
- [ ] T003 [App] 所有者の解除操作、競合検査、監査と再試行を実装する。
- [ ] T004 [UI] 対象と影響範囲、外部切り戻し、確認、ローカル解除を案内する。
- [ ] T005 [Verify] 部分失敗、最後のドメイン、権限不足と汎用削除の迂回を検証する。

## 検証

- `mise run test-go-package -- ./backend/wsfederation/...`：二ドメインの一方だけの解除、他方の発行、汎用 DELETE の拒否、確認不足と競合後の無変更を確かめる。
- `mise run test-ui-e2e-file -- <追加する Entra 解除の spec>`：外部確認前に削除できず、失敗後も設定が残り、再訪問して再開できることを確かめる。
- `mise run check-contract-drift`、`mise run check-published-api-compat`、`mise run verify-changed`：契約の変更と実装の影響範囲を検証する。
- WI-79 の検証テナントで、切り戻し先でのサインインと、残るドメインのサインインを独立に確認する。

## リスク

Managed に戻すだけでは、利用者がクラウド認証できる保証にならない。
切り戻し先で使う認証と復旧用の管理者を検証用テナントで確かめる。
外部の信頼設定を変える操作を含むため不可逆と分類し、作業中に以前の設定を保持する。
手動観測後の外部変更との競合は完全には防げず、確認時刻と残る制約を示す。
