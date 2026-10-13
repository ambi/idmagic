---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-13
priority: p2
depends_on: [wi-47518-align-shared-document-placement-and-design-records]
change_kind: docs
spec_impact:
  kind: none
  reason: "図の表現と選択基準を改善する。IdMagic と外部当事者の責任分界、実行単位の配置と通信、認証と認可の結果、永続状態、ドメインイベント、外向きの通知、REQ と TypeSpec の契約は維持する。"
---

# 読者の問いに合わせて Mermaid の図を選び、システム境界の既存図を改善する

## 動機

Mermaid には C4 などの表現があるが、既存文書の図は主に flowchart、シーケンス図、ER 図である。
システム境界の図も flowchart で、人、システム、ゲートウェイなどの役割を同じ箱で表している。
表現する対象に合う図種を使えば、読者が責任分界を読み取るための負担を減らせる。

図種や図数を増やすことは目的にしない。
既存図の置換、維持、削除を比較し、追加情報が不要な場合は図も説明も増やさない。
変更する理由がなければ、既存図をそのまま保つ。

## 対象範囲

- [システム境界](../../docs/design/architecture/system-boundary.md)の既存 flowchart を C4Context に置き換え、人、IdMagic、外部システムと利用関係を区別する。
- [設計フォーマット](../../docs/formats/design-format.md)へ、図の必要性、図種、粒度を選ぶ短い基準を追加する。
- 変更した図を生成サイトで確認し、必要な要素、関係、日本語ラベルと読みやすさを検証する。

## 対象外

- C4 の全階層、全図種の導入、図数や採用図種数の達成目標。
- runtime.md や logical.md への新規図の追加、既存図と同じ内容を描く別図の併設。
- 下の調査で維持と判断した配置図、ネットワーク図、処理図、ER 図、生成状態図の変更。
- 図の例集、Mermaid 構文の解説、新しい図の台帳、独立した ADR の作成。
- Mermaid の更新、テーマと描画器の変更、新しいアイコン配信基盤の導入。
- 製品の責任分界、モジュール構成、依存、実行方式、プロトコルと規範仕様の変更。
- 過去の作業項目にある図の改稿。

## 設計

### 起票時の調査と採否

2026-10-13 に docs/ の Mermaid ブロックを検索したところ、手書き図は flowchart 12 図、sequenceDiagram 4 図、erDiagram 6 図だった。
これとは別に、描画器は状態遷移表から stateDiagram-v2 を生成している。
各文書の図と周囲の説明を確認し、次のように判断した。

| 対象 | 確認した内容 | 判断と理由 |
| --- | --- | --- |
| [システム境界](../../docs/design/architecture/system-boundary.md) | 人、連携アプリケーション、IdMagic、外部の提供者が同じ箱で描かれ、辺には関係の説明がない | 既存の一図を C4Context に置換する。人とシステムを区別し、辺に利用、連携、運用の目的を付けることで、責任表を読む前に関係を把握できる |
| [共通トポロジー](../../docs/design/architecture/deployment.md#共通トポロジー) | API、Worker のレーン、Batch、Seed、スキーマ適用、起動主体と通信を一図で示す | flowchart を維持する。C4Container は実行単位の分担に適するが、この図は起動と配置の関心も含む。変換だけでは理解の改善が明確でなく、別の Container 図を足すと同じ構成の二重管理になる |
| [デプロイプロファイル](../../docs/design/architecture/deployment.md#デプロイプロファイル) | Compose の観測経路、Kubernetes の Service、Deployment、HPA、PDB、NetworkPolicy と、未作成要素の破線がある | flowchart を維持する。C4Deployment や architecture-beta への統一より、通信、制御、未作成の区別を保つ方が、この文書の問いに合う |
| [プラットフォーム](../../docs/design/infrastructure/platform.md#google-cloud) | リソース階層、production の構成、リリースの順序を別々に示す | 維持する。階層、共通プロジェクトの利用、工程順序は異なる関心であり、一つの C4 図へまとめない。配置図と重なる部分だけを理由に、共有リソースや管理関係まで消さない |
| [ネットワーク](../../docs/design/infrastructure/network.md) | キャッシュと拒否の分岐、上流 IdP の戻り、トークン取得の順序、ネットワーク区画を示す | flowchart と sequenceDiagram を維持する。処理分岐と通信順序には既存の表現が合っており、C4Dynamic への置換や境界図の追加は不要 |
| IdManagement の[CSV](../../docs/modules/identity-management/design/csv-transfer.md#プレビューと適用)、[エクスポート](../../docs/modules/identity-management/data-export/README.md#アーキテクチャ)、[監査](../../docs/modules/identity-management/design/audit-events.md#ドメインイベントの配信点) | プレビューと適用、ジョブと成果物、保存後のイベント配信を示す | シーケンス図を維持する。処理順序と確定の前後が読者の関心であり、静的な構成図へ変えるとその情報を失う |
| [データベース](../../docs/design/data/database.md#スキーマの読み方) | カラムを省き、テーブルと外部キーの多重度を示す | ER 図を維持する。classDiagram への置換やカラムの追記は、目的に合わず SQL の複製を増やす |
| [生成の関係](../../docs/development/local-development.md#生成の関係)と生成状態図 | 前者は入力と生成物、後者は規範の状態遷移表を図示する | 維持する。C4 の対象ではなく、状態図を手書きすると規範表と二重管理になる |

今回の採用は既存図一つの置換とし、新しい図は追加しない。
維持する図に、今回の検討結果を説明する文章は追記しない。
この比較と不採用の理由は本作業項目に保つ。

### C4Context に描く範囲

IdMagic を一つのシステムとして描き、利用者と管理者、運用者、連携アプリケーション、上流の IdP とアイデンティティ源、下流の送信先、外部のデータベース基盤と運用基盤を結ぶ。
既存の本文と責任表を根拠に、各関係の目的を日本語で付ける。
ブラウザー、ゲートウェイ、API プロセスなどの内部経路はこの俯瞰図へ描き込まず、既存のランタイム、配置、ネットワーク設計への参照を使う。
図の「利用する」関係は物理的な直接接続を意味しないことを、図の導入で短く示す。
ゲートウェイの担い手と PostgreSQL の具体的な配置については、既存の本文と表にある責任分界を保つ。

同じ図の flowchart 版を併設しない。
責任表は当事者ごとの義務を扱うため保つが、ノード一覧や関係一覧を別の表として足さない。
Context 図を描くために、モジュールや実行単位の境界を再設計しない。

### 図種の選択基準と表示環境

[設計フォーマット](../../docs/formats/design-format.md)に、読者が知りたい関係を本文や表だけで十分に理解できるなら図を増やさないこと、既存図を改善できる場合は置換することを短く定める。
システムと外部当事者には C4、分岐には flowchart、通信順序には sequenceDiagram、関係と多重度には erDiagram、状態には既存の表から生成する状態図を選ぶ基準を置く。
図種ごとの採用義務や、全ページへの図の設置義務は作らない。
C4Container の Container は Docker のコンテナとは区別し、既存の[実行単位の語彙](../../docs/formats/documentation-guide.md#47-architecturedeploymentmdとdesigninfrastructure)に従う。

起票時の tools/package.json は Mermaid 12.1.0 を指定している。
描画器は Mermaid の全ブロックを mermaid.parse で検査し、生成サイトでは securityLevel: strict で描画する。
専用の C4 導入処理を足す前提は置かない。
[Mermaid の C4 文書](https://mermaid.js.org/syntax/c4.html)は C4 を実験的とし、レイアウトの自動調整とテーマ対応にも制約を示しているため、構文検査の成功だけで表示品質を判断しない。
[C4 の公式方針](https://c4model.com/diagrams)も価値のある階層だけを使うとしており、[Context](https://c4model.com/diagrams/system-context)と[Container](https://c4model.com/diagrams/container)の粒度を区別する。

置換後の表示で日本語の欠け、関係ラベルの重なり、配色による判別の悪化が解消できなければ、既存 flowchart のまま人と関係を明示する改善へ戻す。
図の追加や描画器の拡張で置換を成立させない。
この採否条件は表示の検証で確定し、比較結果を本作業項目へ残す。

## 計画

1. [wi-47518](../done/wi-47518-align-shared-document-placement-and-design-records.md)の文書と用語の整理を先に取り込む。同じ設計文書とフォーマットを扱うため、その整理後の状態に置換を適用する。
2. 着手時に対象図、責任表、Mermaid のバージョンを再確認し、起票時の判断を変える差分だけを確認する。
3. 設計フォーマットへ必要性と図種の選択基準を書き、システム境界の一図を置き換える。
4. 構文と参照を検査し、生成サイトで既存図との比較を終える。採否条件を満たさなければ同じ一図の flowchart を改善する。
5. 実際の採否と検証結果を記録し、文書変更として完了する。

## タスク

- [ ] T001 [Docs] 起票時の対象と判断を先行項目の完了後の文書へ照合し、着手時の文脈を記録する。
- [ ] T002 [Docs] 図の必要性、置換と維持、図種と粒度の選択基準を設計フォーマットへ追加する。
- [ ] T003 [Docs] システム境界の既存図を C4Context に置換し、責任分界と関係の意味を保つ。
- [ ] T004 [Verify] 日本語ラベル、配色、通常表示と拡大表示を既存図と比較し、C4 の採否条件に従って最終図を確定する。
- [ ] T005 [Verify] 参照と形式、Mermaid の構文、生成サイトを検査し、図が増えていないことと規範の維持を確認する。
- [ ] T006 [Records] 実際の採否、検証結果と完了を記録し、作業項目を移動する。

## 検証

- 起票時は `mise run check-work-items` と `mise run check-links` で、作業項目の形式、依存と参照を確認する。
- 実装時も `mise run check-work-items`、`mise run check-links` を実行し、変更した記録と文書の参照を検証する。
- `mise run check-rendered-docs` で文書サイトの生成、日本語の文章検査、Mermaid の構文検査を通す。
- `mise run render-docs` でサイトを再生成し、`mise run serve-docs` で提供したページをブラウザーで確認する。localhost の待受けとブラウザーの起動はサンドボックス外で実行する。
- 明色と暗色、通常表示と拡大表示、デスクトップ幅と狭い画面幅で、ラベルの欠けや重なり、関係の判別、ページ内での収まりを確認する。
- 旧図と最終図で、当事者、責任分界、利用と連携の関係を照合する。内部経路を省いた部分は既存の担当文書へたどれることを確認する。
- 差分で図数の増加、同じ内容の二図の併設、規範本文や TypeSpec の変更がないことを確認する。
- Markdown だけの変更として[振る舞いを保つ変更](../../docs/development/specification-first-workflow.md#振る舞いを保つ変更)を適用し、RED と故障注入のためだけのテストは作らない。

## リスク

C4 の箱を使うだけでは、粒度と関係の意味が曖昧なままになる。
責任分界を読むための Context 図に絞り、技術的な経路は担当する既存文書へ委ねる。
表示上の制約で読みにくくなる可能性は、既存図との比較と採否条件で扱う。
先行項目で文書や責任の説明が変わるため、起票時の本文を写すのでなく、着手時の現行文書を図の根拠にする。
