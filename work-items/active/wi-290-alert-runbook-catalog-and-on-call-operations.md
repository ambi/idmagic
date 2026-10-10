---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-07-25
priority: p1
depends_on: []
change_kind: operations
affected_spec:
  - { path: spec/contexts/system/main.tsp, symbol: IdMagic.Contract.MetricsExposition }
---

# アラート用 runbook の共通構成と索引を完成させる

## Motivation

本項目の作成後に運用文書とアラート参照の整備が進み、2026-09-06 時点では Kubernetes と Docker の 11 アラートすべてが `runbook_url` を持ち、参照先の存在は `mise run check-slo-references` で検査される。

`docs/runbooks/` にはトークン、ログイン、ジョブ、入場制御などの文書もあり、「runbook が 1 本しかなく、アラートから辿れない」という当初の前提は解消済みである。

残っているのは、runbook の構成と入口が統一されていないことである。

とくに三つのジョブアラートが共有する `async-jobs.md` は日常運用の説明であり、発火条件、最初の確認、緩和、確認、エスカレーションを順に辿る他のアラート用 runbook と形が異なる。

また `docs/runbooks/README.md` がなく、アラート名から文書を探す索引と、今後の文書が守る共通構成を検査する仕組みがない。

本項目は既存の参照を作り直さず、オンコールが各アラートから一貫した手順へ進める状態を完成させる。

## Scope

- 現在の 11 アラートと `runbook_url` を棚卸しし、Kubernetes と Docker の対応差分がないことを固定する。
- `docs/runbooks/README.md` にアラート名、影響、参照先を載せた索引と、アラート用 runbook の必須構成を記録する。
- `async-jobs.md` の日常運用情報を維持したまま、三つのジョブアラートについて発火条件、最初の確認、緩和、復旧確認、エスカレーションを辿れる構成へ直す。
- 入場制御を含む既存の共有 runbook が複数アラートの差を明確に扱えるか確認し、不足する分岐だけを追加する。
- アラート用 runbook に必須見出しと実行可能な確認手順があることを検査し、`mise` の監視検査へ接続する。
- リポジトリの文書入口から runbook 索引へ辿れるようにする。

## Out of Scope

- `runbook_url` の新設と参照先存在検査。すでに実装済みである。
- アラート閾値またはサービス目標の見直し。
- 新しいメトリクスやアラートの追加。
- バックアップ、復旧、リリース差し戻しの手順変更。
- オンコール担当者やページャ製品の設定。
- runbook に記した調査先の製品機能を実装すること。

## Design

アラート用 runbook の共通構成は「発火条件」「最初に確認すること」「緩和」「確認」「エスカレーション」とする。

一つの文書を複数アラートが共有する場合は、アラート名ごとの発火条件と分岐を冒頭で示し、読者が自分のアラートに無関係な手順を先に読まない構成にする。

検査は見出しの存在と参照整合性を保証する。

コマンドや問い合わせの意味までは構文で保証できないため、各手順はリポジトリで提供する `mise` タスクまたは対象環境で実行する明示的なコマンドを示し、レビューで観測対象と期待結果を確認する。

## Tasks

- [ ] T001 [Inventory] 11 アラートの参照先、共有関係、文書構成を一覧にする。
- [ ] T002 [Docs] `docs/runbooks/README.md` に索引とアラート用 runbook の共通構成を記録する。
- [ ] T003 [Runbook] `async-jobs.md` を、三つのジョブアラートから初動と緩和へ進める構成に更新する。
- [ ] T004 [Runbook] 既存の共有 runbook を確認し、アラートごとの分岐と復旧確認に残る不足を補う。
- [ ] T005 [Acceptance RED] 必須見出しの欠落と Kubernetes、Docker 間の参照差分を検出する失敗例を固定する。
- [ ] T006 [Tooling] runbook の必須構成と両環境の参照整合性を監視検査へ追加する。
- [ ] T007 [Docs] リポジトリの文書入口から runbook 索引へリンクする。
- [ ] T008 [Verify] 各 runbook の確認手順を実行し、標準検証を通す。

## Verification

- `mise run check-monitoring`
- `mise run check-slo-references`
- `mise run check-k8s dev`
- `mise run check-k8s prod`
- `mise run check-work-items`
- `mise run verify`
- 必須見出しを一つ外した固定具と、Kubernetes または Docker の参照をずらした固定具が検査に失敗することを確認する。

## Risk Notes

リスクは medium であり、実行時の製品動作は変えない。

最大の危険は、形式だけをそろえて障害時に実行できない手順を残すことである。

各確認手順には観測対象と期待する結果を記し、実行できない環境依存の操作は前提条件と必要な権限を明示する。
