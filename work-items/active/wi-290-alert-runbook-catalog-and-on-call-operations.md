---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-07-25
priority: p2
group: operations
depends_on: []
change_kind: operations
affected_spec:
  - { path: spec/modules/system/main.tsp, symbol: IdMagic.Contract.MetricsExposition }
---

# アラートの runbook の索引と共通の構成を作り、参照先の存在を検査する

## 動機

Kubernetes と Docker のアラートは 12 件あり、どれも `runbook_url` で `docs/runbooks/` の文書を指す（2026-11-14 に確認）。
起票後の 2026-09-06 には `mise run check-slo-references` が参照先の存在を検査していたが、このタスクは今はなく、参照先の存在を確かめる検査もない。
アラートの名前から文書を探す索引（`docs/runbooks/README.md`）もない。

runbook の形も揃っていない。
三つのジョブのアラートが共有する `async-jobs.md` は日常の運用の説明であり、発火の条件、最初の確認、緩和、復旧の確認、エスカレーションを順に辿るほかのアラートの runbook と形が違う。
オンコールの担当者は、アラートごとに違う読み方を強いられる。

## 対象範囲

- 二つの構成ファイルのアラートと `runbook_url` を棚卸しし、Kubernetes と Docker で同じアラートが同じ runbook を指すことを固定する。
- `runbook_url` の参照先が存在することと、アラートの runbook が必須の見出し（発火の条件、最初の確認、緩和、復旧の確認、エスカレーション）を持つことを検査し、`mise run check` から実行する。
- `docs/runbooks/README.md` に、アラートの名前、影響、参照先の索引と、アラートの runbook の必須の構成を書く。リポジトリの文書の入口から索引へ辿れるようにする。
- `async-jobs.md` の日常の運用の情報を残したまま、三つのジョブのアラートを必須の構成で辿れるようにする。

## 対象外

- アラートの閾値とサービス目標の見直し。
- 新しい指標とアラートの追加。wi-617 と wi-618 が扱う。
- バックアップ、復旧、リリースの後退の手順の変更。
- オンコールの担当者とページャー製品の設定。

## 検証

- `mise run check`
- `mise run check-monitoring`
- `mise run check-links`

## リスク

必須の見出しを機械で求めると、見出しだけを置いた空の節が通る。
見出しの下に実行できる確認の手順があることは、レビューで確かめる。
