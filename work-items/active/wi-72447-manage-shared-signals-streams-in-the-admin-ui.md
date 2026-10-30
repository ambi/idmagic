---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-31
priority: p3
depends_on: [wi-16152-redeliver-dead-lettered-security-events]
change_kind: feature
affected_spec:
  - { path: docs/modules/sharedsignals/features/stream.md, requirement: REQ-SHAREDSIGNALS-008 }
  - { path: docs/modules/sharedsignals/features/stream.md, requirement: REQ-SHAREDSIGNALS-009 }
  - { path: docs/modules/sharedsignals/features/stream.md, requirement: REQ-SHAREDSIGNALS-011 }
  - { path: docs/modules/sharedsignals/features/transmitter.md, requirement: REQ-SHAREDSIGNALS-013 }
  - { path: spec/modules/sharedsignals/models.tsp, symbol: SsfStream }
---

# 管理画面で SSF ストリームを管理し、配信不能の配送をやり直せるようにする

## 動機

SSF ストリームの登録から配送の確認までは管理 API しかなく、`frontend/src` に SharedSignals の管理画面がない。
受信側の障害で配送が `dead_letter` になっても、管理者は API を直接呼ばなければ気付けず、[[wi-16152-redeliver-dead-lettered-security-events]] で加えたやり直しも実行できない。
失効が外部へ伝わらないまま残るのを防ぐには、管理者が状況を見て操作できる画面が要る。

## 対象範囲

- ストリームの一覧と詳細。向き、状態、購読するイベントの種類と監査イベントの種類（`audit_event_types`）を示す。
- 送信側のストリームの登録と更新で、監査イベントの種類（全種別の `*` を含む）を選ぶ。自動無効化されたストリームの再有効化を示す。
- 送信側と受信側のストリームの登録、購読するイベントの種類の更新、無効化と再有効化、削除。
- 送信側のストリームの配送の一覧。状態、試行の回数、最後のエラー、`jti`、作成時刻、配送時刻を示す。
- `dead_letter` の配送のやり直し。実行前に件数を確認させ、実行後に戻した件数を示す。無効化したストリームでは実行できないことを示す。
- ストリームの詳細の読み取りに、シークレットを除く設定を加える管理 API の変更。

## 対象外

- 配送先、audience、受信側の信頼の設定の変更。更新の API が `event_types` しか受け付けないので、別の項目で扱う。
- 配送一件ごとのやり直し。
- 配送の一覧のページネーションとサーバー側の絞り込み。件数が画面の支障になると分かってから別の項目で扱う。
- システムコンソールからの閲覧（wi-494 の対象外の記述を変えない）。

## 設計

### 管理 API の不足

`GetSsfStream` の応答 `SsfStream` は、配送先、audience、`max_delivery_attempts`、`trusted_issuer` などの設定を返さない。
このままでは、管理者は詳細の画面でどの受信側のストリームかを見分けられない。
詳細の応答へ設定を加え、`delivery_authorization` は値を返さず、設定の有無だけを返す。
一覧の応答は変えない。

### 画面

配置と文言は、既存の管理画面の一覧と詳細の形（`frontend/src/features/admin-*`）に合わせる。
`jwks` はインラインの値を編集可能な形で再表示せず、`jwks_uri` か インラインかの別を示す。

## 計画

1. 詳細の応答を TypeSpec と要件へ加え、実装する。
2. 画面を実装する。

## タスク

- [ ] T001 [Spec] 詳細の応答に加える設定と、シークレットを返さないことを要件と TypeSpec に書く。
- [ ] T002 [App] 詳細の応答を実装する。
- [ ] T003 [Acceptance] 登録、配送の確認、やり直しの流れを UI の受け入れテストで RED にする。
- [ ] T004 [UI] 画面を実装する。
- [ ] T005 [Verify] 変更を検証する。

## 検証

- `mise run check-spec`
- `mise run typecheck-ui`、`mise run lint-ui`、`mise run test-ui-e2e-file`。e2e はサンドボックスの外で実行する。
- `mise run verify`

## リスク

- `delivery_authorization` が応答や画面へ漏れると、受信側へ SET を偽って送れる。応答の本文に値が現れないことをテストで確かめる。
- やり直しは多数の SET を一度に送ることがある。実行前に件数を示して誤操作を防ぐ。
