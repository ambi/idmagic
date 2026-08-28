# 開発文書

このディレクトリは、開発者が IdMagic を変更、検証、リリースするための手順と、その判断に必要な説明を持つ。製品が何をするかは `docs/` 直下と各 Bounded Context の正本文書が持ち、Pull Request の規則はルートの [CONTRIBUTING.md](../../CONTRIBUTING.md) が持つため、ここへ複製しない。

文書は Diátaxis の分類を目安にする。学習のための手引き、目的を達成するための手順、事実を調べる参照、背景を理解する解説を同じ文書へ混ぜず、ファイル名が主な役割を示す。分類は検査せず、内容が増えたときに分割を判断するために使う。

| 読みたいこと | 文書 | 主な分類 |
| --- | --- | --- |
| 仕様先行の進め方、証拠契約、検証のはしご | [Specification-first Development Workflow](specification-first-workflow.md) | 解説、参照 |
| 開発環境、起動、ビルド、生成 | [ローカル開発](local-development.md) | 手順 |
| CI の正本と失敗時の切り分け | [継続的インテグレーション](continuous-integration.md) | 参照、解説 |
| テスト水準と実行境界 | [テスト方針](testing.md) | 参照 |
| 版付け、成果物、段階的な展開 | [リリース](release.md) | 手順 |
| 開発プロセス指標の採否 | [開発プロセスの計測](process-metrics.md) | 解説、参照 |

障害や手動の運用作業の最中に読む手順は [runbooks](../runbooks/) に置く。計画されたリリースはここで扱い、リリースを後退させる判断と実行は [リリースの後退](../runbooks/release-rollback.md) が持つ。
