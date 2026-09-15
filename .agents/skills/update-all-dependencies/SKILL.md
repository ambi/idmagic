---
name: update-all-dependencies
description: フロントエンド、バックエンド、組み込み tools、ランタイム、独立 CLI、CI Action、コンテナイメージを棚卸しし、リポジトリ全体の依存関係を利用可能な最新安定版へ更新する。依存関係や開発ツールを一括更新するときに使う。
---

# リポジトリ全体の依存関係を更新する

実行時点で利用可能な最新安定版を目標にする。
既存の版系列が prerelease の場合、または利用者が明示した場合に限って prerelease を候補に含める。
検出した更新面は、最新版へ更新したもの、すでに最新版だったもの、理由を記録して見送ったもののいずれかに分類できた時点で棚卸しを完了する。

## 更新面を棚卸しする

1. リポジトリの指示を読み、`git status --short` で利用者の変更を確認し、`mise tasks` で正規の操作を探す。
2. マニフェスト、ロックファイル、`mise.toml`、CI、Dockerfile、Compose、Kubernetes、生成器、スクリプトの shebang、入れ子のプロジェクトルートを調べる。
3. Renovate や Dependabot の設定は検出の手掛かりとして使い、設定対象外の版指定も探索する。
4. 次の所有層ごとに、宣言場所、現在版、最新安定版、更新方法、同期先、検証方法を一覧にする。

| 所有層 | 数える対象 |
| --- | --- |
| Bootstrap | `mise` の最低版、CI が導入する `mise`、開発環境を起動する Action |
| ランタイムとパッケージマネージャ | Go、Bun、Node.js などと、`packageManager`、Dockerfile、言語指令にある同じ版 |
| エコシステム依存 | Go module、frontend の dependencies と devDependencies、組み込み tools の依存関係と lockfile |
| 独立 CLI | `mise.toml` が固定する linter、generator、scanner、formatter |
| CI | GitHub Actions などの tag または commit SHA と、その版コメント |
| 実行イメージ | Dockerfile の base image、Compose と Kubernetes が参照する第三者イメージ |

アプリケーション自身の配布タグや環境ごとに注入するプレースホルダーは、第三者依存の版として数えない。
同じ製品の版が複数箇所にある場合は、一つの更新単位として扱う。

## 最新版を確定する

既存の `list-outdated-*` タスクと各パッケージマネージャの情報を起点にし、major update も候補から外さない。
必要な照会や更新に `mise` タスクが無ければ、再現可能なタスクを `mise.toml` に追加してから `mise run <task>` で実行する。
自動更新タスクの成功は、そのタスクが扱わない更新面の棚卸し完了を意味しない。

パッケージレジストリ、公式リリース、公式リポジトリを一次情報源にする。
prelease、撤回版、保守終了、アーカイブ済みのプロジェクトを区別し、単に番号が最大の版を選ばない。
同時に更新すべき製品群は互換性のある組み合わせを選ぶ。
最新版が現在のランタイムやプラットフォームをサポートしない場合は、必要な前提更新まで候補へ含める。

## 所有単位ごとに更新する

更新前に現在の検証結果を取得し、既存の失敗を更新による失敗と区別する。
次の順序を基本にし、各単位が通ってから次へ進む。

1. Bootstrap、ランタイム、パッケージマネージャを更新し、同じ版を表す宣言を同期する。
2. `mise run update-go-dependencies` など、リポジトリが持つ backend 更新タスクを実行する。
3. `mise run update-ui-dependencies` など、frontend の manifest と lockfile を更新するタスクを実行する。
4. `mise run update-tools-dependencies` など、組み込み tools の manifest と lockfile を更新するタスクを実行する。
5. 独立 CLI、CI Action、base image、第三者の実行イメージを更新する。
6. 生成物の更新が必要なら、対応する `mise` タスクだけを実行する。

Go module の major 版で module path が変わる場合は、通常の更新タスクでは上がらないため、import と設定を含む migration として扱う。
commit SHA で固定した CI Action は SHA と版コメントを同じ release に同期し、イメージや Action の固定方法はリポジトリの方針を保つ。
major update では公式 migration guide と release notes を読み、コンパイルエラーだけでなく非推奨化、既定値、設定形式、出力形式、ブラウザーや OS の要件を確認する。
必要な互換修正は同じ更新単位へ含める。
検査を通すために lint、型検査、テスト、脆弱性検査を弱めたり、警告を一括抑制したりしない。

外部 API、利用者から見える振る舞い、認証、データ形式が変わる場合は `spec-change` を先に使う。
境界、技術、ランタイム構成、設計ガイドラインが変わる場合は `update-design` も使う。
依存側の破壊的変更がプロダクトの意味を変えない限り、既存の契約を保つ互換修正を選ぶ。

## 各段階と全体を検証する

所有単位の変更直後に、その層を担当する最小の `mise` タスクを実行する。
Go は `verify-go`、frontend は `verify-ui`、組み込み tools は test、typecheck、lint の各タスク、インフラは Compose、Kubernetes、monitoring の検査を使う。
ランタイムや framework の更新がブラウザー経路へ届く場合は E2E も実行する。

最後に次を満たす。

1. `mise install` とリポジトリの setup タスクが、更新済みの宣言と lockfile から成功する。
2. `mise run verify` が成功し、必要なら `mise run verify-full` も成功する。
3. 依存関係と脆弱性の監査タスクが成功する。
4. outdated の照会を再実行し、残る候補をすべて説明できる。
5. `git diff --check` が成功し、差分に意図しない manifest、lockfile、生成物の書き換えがない。

検証失敗を環境要因と判断するときは、同じ失敗を再現し、根拠を残す。
更新できない項目は勝手に上限を追加せず、現在版、候補版、阻害要因、解除条件を報告する。

## 完了を報告する

更新した項目と互換修正、すでに最新版だった項目、見送った項目と理由、実行した検証、残るリスクを報告する。
すべての更新面を分類するまで「すべて最新版」とは報告しない。
必要な検証がすべて成功したら、利用者がコミットしないよう明示した場合を除き、`commit` Skill を使って更新と互換修正を Conventional Commits としてコミットする。
利用者の既存変更と検証未完了の変更はコミットへ含めない。
push は利用者が明示した場合だけ行う。
