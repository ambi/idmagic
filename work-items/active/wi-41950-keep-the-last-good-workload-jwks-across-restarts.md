---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-11-11
priority: p3
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/modules/workloadidentity/features/attestation-exchange.md, requirement: REQ-WORKLOADIDENTITY-012 }
---

# 最後に取得できた信頼設定の鍵を、プロセスの再起動後も猶予の期間だけ使えるようにする

## 動機

アテステーションの交換は、`jwks_uri` の取得に失敗したとき、最後に取得できた鍵の集合を 24 時間まで使う。
その集合は各プロセスのメモリにしかないため、プロセスを再起動した直後に取得元へ届かなければ、猶予なしに `jwks_unavailable` で拒否する（attestation-exchange.md の信頼性の節）。
SPIRE の Bundle Endpoint の障害と、IdMagic のローリング更新や自動スケールが重なると、すべての Agent が同時にトークンを得られなくなる。
SPIFFE Federation の仕様も、バンドルを取得できない間は手元のバンドルを使い続けることを前提にしている。

## 対象範囲

- 取得に成功した鍵の集合と取得時刻を、信頼設定ごとに永続化する。
- 取得に失敗し、メモリに鍵がないとき、永続化した鍵を、取得から 24 時間以内に限って使う。
- 永続化した鍵を使ったことを、監査または運用の指標で区別できるようにする。

## 対象外

- 24 時間の猶予の長さの変更。
- OAuth2 のクライアントの `jwks_uri` の取得。

## 設計

保存先は、信頼設定のテーブルの列（`jwks_cached` と既存の `jwks_cached_at`）を第一候補とする。
管理者による JWKS の再取得（REQ-WORKLOADIDENTITY-008）がすでに `jwks_cached_at` を記録するので、同じ時刻の意味を交換時の取得にも広げる。
交換のたびに書くと書き込みが増えるため、取得した内容が変わったときか、前回の保存から一定の時間が過ぎたときだけ書く。

採らない案：共有キャッシュ（Redis など）に置く。依存を一つ増やし、障害の点も増える。

公開鍵だけを保存するので、保存時の暗号化は求めない。
信頼設定を削除または無効化したとき、保存した鍵も使わない。

## 計画

1. 書き込みの頻度と保存先を確定する。
2. 仕様の信頼性の節と REQ-WORKLOADIDENTITY-012 を更新し、実装する。

## タスク

- [ ] T001 [Spec] 信頼性の節と要件を更新する。
- [ ] T002 [Acceptance] 再起動後に取得元へ届かなくても、24 時間以内なら交換できることを RED にする。
- [ ] T003 [App] 保存と読み出しを実装する。
- [ ] T004 [Verify] 検証を実行する。

## 検証

- `mise run test-go-package -- ./backend/workloadidentity/...`（DB を使うテストを含むため、サンドボックスの外で実行する）。
- `mise run check-schema`。
- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>` とする。

## リスク

- 鍵を失効させたい場面で、古い鍵を 24 時間使い続ける。管理者が信頼設定を無効化するか、インラインの `jwks` に切り替えれば止められることを運用文書に書く。
