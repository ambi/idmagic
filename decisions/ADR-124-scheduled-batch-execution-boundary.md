---
status: accepted
authors: [tn]
created_at: 2026-07-18
supersedes: [ADR-099]
---

# ADR-124: 定期的な横断処理を外部スケジュールの one-shot batch に分離する

## コンテキスト
`idmagic-worker` は tenant-owned durable Job をリース付きで並列処理するため、
負荷に応じて水平スケールする。一方、retention sweep と署名鍵 lifecycle は
全テナントを横断して低頻度に一度だけ評価する運用処理であり、worker replica
ごとに常駐 ticker を持たせると実行数が replica 数へ連動する。

署名鍵専用の常駐プロセスまたは専用 one-shot バイナリを増やす案も、タスクが
増えるたびに composition root と配布物が増える。Kubernetes CronJob、cron、
systemd timer はスケジュール、再試行、履歴、タイムアウトを既に所有するため、
アプリケーションが同じ機能を再実装する理由もない。

## 決定
外部スケジューラから起動され、一つの処理を実行して終了する共通
`idmagic-batch` executable を設ける。処理はサブコマンドとして選択し、
retention sweep と signing-key lifecycle を別々に実行できるようにする。

通常の `idmagic-worker` は durable Jobs の claim と handler 実行だけを所有し、
periodic maintenance goroutine を持たない。batch composition root は各 bounded
context の use case を呼ぶだけで、retention policy と鍵 lifecycle の規則は
それぞれ Authentication/Audit と SigningKeys に残す。

Kubernetes では同一コンテナイメージからタスクごとに個別 CronJob を構成する。
各処理は外部の同時起動制御に加えて、条件付き更新または PostgreSQL advisory
lock により冪等性を持つ。スケジューラを利用できない環境は cron または
systemd timer から同じ executable を起動する。

## 却下した代替案
- `idmagic-worker` 内の periodic goroutine: worker の水平スケールと横断 batch
  の実行数が結合し、leader election または全処理の分散排他が必要になる。
- 常駐 cron worker: missed-run、leader election、cron 解釈、再試行を自前で
  所有し、外部スケジューラと責務が重複する。
- タスクごとの executable: 配布物と composition root が処理数に比例して増える。
- tenant-owned Jobs queue: retention はテナント横断であり、tenant_id 必須の
  Job invariant に例外を持ち込む。

## 影響
- `idmagic-worker` から retention ticker を除去する。
- `backend/cmd/idmagic-batch` と `retention-sweep` /
  `signing-key-lifecycle` サブコマンドを追加する。
- Kubernetes はサブコマンドごとに CronJob、同時実行制御、期限、再試行を設定する。
- `SigningKeys.objectives.SigningKeyLifecycle` と lifecycle scenario は、
  スケジュール方法に依存せず one-shot 評価から実現される。
- ADR-099 の retention sweep 配置だけを置き換え、durable Job の
  at-least-once、lease、retry、drain の決定は維持する。
