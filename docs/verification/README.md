# 検証設計

システム要求を満たしたと判断する証拠の種類、実行境界、合否条件への入口を持つ。
テストコード、CI の実行結果、作業項目の RED と変更耐性の証拠はそれぞれの正本に残し、この文書へ結果を複製しない。

## 担当範囲

| 問い | 正本 |
| --- | --- |
| どの要求をどの証拠で満たしたと判断するか | この文書と子文書 |
| テストの水準、境界、実物にする依存、テストダブルの選び方 | [テスト方針](../development/testing.md) |
| 変更ごとの RED、主要ユースケース、変更耐性の証拠 | [仕様先行の開発ワークフロー](../development/specification-first-workflow.md)と各作業項目 |
| 目標値と測定境界 | [品質要求](../requirements/quality.md) |
| 証拠を作るコマンドと CI の構成 | `mise.toml` と CI 定義 |
| 検証をいつ、誰が実施するか | [保守](../operations/maintenance.md)と[サービス管理](../operations/service-management.md) |
| 事象の最中に実施する手順 | [運用手順](../runbooks/) |

検証設計は「単体テスト設計」や「E2E テスト設計」に相当する文書を持たない。
テストの水準は、何を確かめるかではなく、どの公開境界と依存を実物にするかで選ぶ設計であり、[テスト方針](../development/testing.md)が水準ごとの目的と境界を定めている。
同じ観点の正本を二つ置くと、片方だけが更新される。
検証設計が持つのは、要求の側から見てどの証拠を必要とするかであり、その証拠をどう書くかではない。

## 子文書

| 文書 | 責務 |
| --- | --- |
| [システム受入れ設計](system-acceptance.md) | 機能受入れ、品質受入れ、可用性と復旧の試験 |
| [セキュリティ検証設計](security.md) | 脅威と制御の対応、許可しない操作の確認、依存と供給鎖の監査 |

## 要求の群ごとの証拠

行は要求の群に対応する。
個々の `REQ-*` とテストの対応はテストの `//spec:covers` が持ち、この表は持たない。
一行ずつ並べるとテストの増減のたびに表が古くなり、しかも同じ対応が二か所に載る。

「実行環境」は、その証拠を作るために何を用意するかを表す。
「リポジトリ内」はこのリポジトリにあるものだけで実行できること、「実環境が必要」は本番または本番相当の環境がなければ実行できないこと、「手段が未定」は証拠の作り方をまだ決めていないことを指す。

| 要求 | 証拠 | 実行 | 合否条件 | 実行環境 |
| --- | --- | --- | --- | --- |
| 正本文書と TypeSpec の規範の形式、ID、参照、生成物 | 静的解析 | `mise run check-spec`、`mise run verify-spec` | 形式、ID の一意性、参照の解決、再生成した成果物との一致がすべて成立する | リポジトリ内 |
| Context の規範シナリオ `REQ-*` | 単体、アダプター統合、受け入れ | `mise run test-go-race`、`mise run test-ui-unit` | 当該 ID を `//spec:covers` で名指すテストが通る | リポジトリ内 |
| [全体の標準仕様](../standards.md)と Context の標準仕様の規範 ID | 上記に加えて被覆の検査 | `mise run check-coverage-debt-ratchet`、`mise run report-coverage-debt` | テストを持たない規範 ID を新たに増やさない | リポジトリ内 |
| 公開契約の OpenAPI、経路、状態コード、イベント語彙 | 契約と実装の差分検査 | `mise run check-contract-drift`、`mise run check-generated-contract`、`mise run check-status-drift`、`mise run check-event-contract`、`mise run check-api-compat` | 宣言と実装に差が無く、公開済みの契約を壊す変更が無い | リポジトリ内 |
| 利用者経路とブラウザー固有の振る舞い | E2E | `mise run test-ui-e2e` | 正式な入口から最終効果までが実配線で成立する | リポジトリ内 |
| WCAG 2.2 のアクセシビリティ規範 | 単体、E2E | `mise run test-ui-unit`、`mise run test-ui-e2e` | 当該規範 ID を名指すテストが通る | リポジトリ内 |
| 脅威モデルの `THREAT-NNN` と許可しない操作 | セキュリティ検証 | `mise run check-security-controls`、`mise run report-security-test-gaps` | 許可しないと宣言した制御にテストがあり、許可しなかったときに状態の変更、トークンの発行、通知のいずれも残らない | リポジトリ内。脅威ごとの手段は[セキュリティ検証設計](security.md)が示す |
| 依存と供給鎖 | 依存監査、シークレット走査、静的解析 | `mise run audit-dependencies`、`mise run audit-go-reachability`、`mise run check-vulnerability-suppressions`、`mise run lint-repo`、CI の CodeQL | 未処理の既知脆弱性と、理由または期限を欠く抑制が無い | リポジトリ内。成果物の来歴の確認は手段が未定 |
| `SLO-*` を判定する監視資産 | 運用検証 | [宣言的なファイルの検証](#宣言的なファイルの検証)が定める | 同じ節が定める | リポジトリ内 |
| `SLO-*` の達成 | 稼働中の指標の評価 | 監視基盤が記録した指標を[品質要求](../requirements/quality.md#測定境界)の母集団で集計する | 30 日の移動窓で目標を満たす | 実環境が必要 |
| `CAP-*` のキャパシティ受入れ | 負荷試験 | 想定ワークロードのデータを投入し、負荷生成器でピークを与える。`mise run k6-smoke` は経路の疎通だけを確かめる | ウォームアップ後の 60 分間、必要リクエスト率で対応する SLO を満たす | 実環境が必要 |
| 可用性の障害単位ごとの検知と切り替え | 障害試験、フェイルオーバー試験 | [システム受入れ設計](system-acceptance.md#可用性と復旧の試験)が定める | 同じ節の成功条件を満たす | 実環境が必要 |
| 復旧手順の成立 | 復元試験 | `mise run restore-drill` | 空の対象へ戻し、復元後の確認一覧をすべて満たす | リポジトリ内。Docker と PostgreSQL クライアントが要る |
| 復旧目標の RPO と RTO | 本番相当のバックアップからの復元 | [システム受入れ設計](system-acceptance.md#可用性と復旧の試験)が定める | 測った経過時間と失ったデータの範囲が目標に収まる | 実環境が必要 |
| デプロイ構成、スキーマ、生成した参照 | 運用検証 | [宣言的なファイルの検証](#宣言的なファイルの検証)が定める | 同じ節が定める | リポジトリ内 |

## 宣言的なファイルの検証

デプロイと運用に使うファイルは、実行して初めて意味が決まる。
書き間違えても、プロダクトのコードを対象とするテストは失敗しない。
対象は `infra/k8s/` の Kubernetes マニフェスト、`infra/docker/` の Docker Compose 構成、Prometheus の規則と収集エージェントの構成と Grafana のダッシュボード、`infra/schema/postgres.sql`、そして実装から生成する `CONFIGURATION.md` と `ROUTE_PRIORITY.md` である。
これらの合否は、テストの成功ではなく、書いた内容と現実の一致で判断する。

| 対象 | 実行 | 合否条件 |
| --- | --- | --- |
| Kubernetes のオーバーレイ | `mise run check-k8s -- dev`、`mise run check-k8s -- prod` | 双方のオーバーレイがレンダリングでき、結果が API スキーマに厳密に適合する |
| Docker Compose の開発スタック | `mise run check-compose` | 構成が解決でき、参照する環境変数とイメージが揃う |
| Prometheus の規則、収集エージェントの構成、ダッシュボード、監視のオーバーレイ | `mise run check-monitoring` | 規則と構成が構文として妥当で、ダッシュボードが JSON として読め、監視のオーバーレイがレンダリングできる |
| 監視資産が名指す `SLO-*` | `mise run check-slo-references` | 規則とダッシュボードが宣言済みの SLO ID だけを名指す |
| PostgreSQL スキーマの収束 | `mise run check-schema` | 空のデータベースへ適用したあと、再適用が差分を出さない |
| 固定した第三者イメージ | `mise run check-container-images -- <image...>` | 宣言したダイジェストまたはタグが解決できる |
| 負荷試験スクリプト | `mise run check-k6` | スクリプトが解析でき、宣言したシナリオと閾値を読み出せる |
| 生成した設定リファレンス | `mise run check-config-reference` | 設定スキーマから再生成した内容がコミット済みの文書と一致する |
| 生成した経路の優先順位 | `mise run check-route-reference` | ルーティングの実装から再生成した内容がコミット済みの文書と一致する |

いずれもリポジトリ内で実行でき、Kubernetes、Docker Compose、監視資産、イメージ、負荷試験スクリプトの検証は Docker を要する。
どれを集約ゲートと CI に入れ、どれを変更者が手元で走らせるかは[継続的インテグレーション](../development/continuous-integration.md#ci-で検証しないもの)が持つ。

この検証が示すのは、ファイルが意図どおり解釈されることまでである。
資源の割り当て、スケジューリング、ネットワークポリシー、アラートが意図した事象で発報するかは、[システム受入れ設計](system-acceptance.md#可用性と復旧の試験)の障害試験が引き受ける。
