---
status: pending
authors: [tn]
risk: high
reversibility: irreversible
created_at: 2026-11-15
priority: p1
group: spec-conformance
depends_on: [wi-47962-hold-provisioning-tasks-for-disabled-connections]
change_kind: bugfix
affected_spec:
  - { path: docs/modules/provisioning/features/connection.md, requirement: REQ-PROVISIONING-002 }
  - { path: docs/modules/provisioning/features/task.md, requirement: REQ-PROVISIONING-020 }
---

# 接続削除後に古いプロビジョニング状態を残さず、既存ジョブを安全に終了させる

## 動機

wi-47962 の対象外を確認したところ、接続の削除自体は実装済みだが、関連状態の削除が保存先で異なることが分かった。
`DeleteConnection` は ConnectionRepository.Delete を呼び、削除後にイベントを発行する。
PostgreSQL は外部キーの ON DELETE CASCADE でタスク、リンク、削除予約、フル同期を消すが、メモリ実装は接続と資格情報だけを消す。
接続の ID は Application の ID なので、同じ Application に接続を作り直したとき、メモリ実装に残った古いタスクやリンクが新しい接続と結び付くおそれがある。
この再登録時の挙動はまだ実行で検証していないが、保存先による削除結果の不一致はコードから確認できる。
また、既に Jobs に投入したジョブを接続削除後にどう終了させるかは、現在の要件が定めていない。

## 対象範囲

- 接続削除時のタスク、リンク、削除予約、フル同期と関連付けの扱いを規範に定める。
- メモリと PostgreSQL で削除の結果と失敗時の結果を揃える。
- 投入済みジョブの終了方法を決め、古いタスクを新しい接続へ送らない。
- 同じ Application に接続を再登録する経路を検証する。

## 対象外

- 下流に既に作ったアカウントやグループの削除。
- 既に送信を開始した外部リクエストの取り消し。

## 設計

削除後に作り直した接続を、削除前の関連状態やジョブが利用できないことを受け入れ結果とする。
関連状態を削除するか履歴として切り離して保持するか、旧ジョブを取り消すか実行時に無操作で終了するかは、実装前に既存 IdP の事例と比較して利用者へ相談する。
接続削除と関連状態の整理を一つの確定にする必要があるかも、この結果から判断する。
モジュール間の依存、削除の組み立て地点、テーブル所有者を変える案では境界の比較を先に記録する。

## 計画

現行の削除要件、両 Repository、Jobs の実行経路と共有契約を照合し、同じ Application への再登録で古いタスクが走るかを最小の境界で確認する。
削除直前の pending、投入済みの in_flight、保持中の pending、予約、リンク、フル同期を区別して整理方針を決める。
確認した欠陥の修正と、新たに決める削除契約を分けて規範へ反映する。

## タスク

- [ ] T001 [Spec] 削除と再登録、関連状態、投入済みジョブの結果を決める。
- [ ] T002 [Test] 両保存先の不一致と古い状態の再利用を検出する RED を確認する。
- [ ] T003 [App] 所有者の削除操作へ整理を集約し、両保存先を揃える。
- [ ] T004 [Verify] 削除の成功と失敗、別テナント、再登録、古いジョブの無作用を検証する。

## 検証

- `mise run test-go-package -- ./backend/provisioning/...`：削除、再登録と両保存先の共有契約。
- `mise run check-boundaries`：削除の組み立てと依存の検査。
- `mise run check-spec` と `mise run verify-changed -- main`：規範と最終差分の検査。

## リスク

関連状態の削除は元に戻せないため、履歴と再送の必要性を先に決める。
接続だけ消えて関連状態が残る部分成功と、古いジョブが新しい資格情報を使う結果を防ぐ。
