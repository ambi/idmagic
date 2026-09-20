# ドメイン設計文書

この文書は、`docs/domain/` に収めたドメイン設計文書の入口である。
IdMagic を Bounded Context へ分け、各 Context が何を意味し、どう振る舞い、内部をどう作るかを定める。
要求をどの機構で満たすかは[設計文書](../design/README.md)で定め、ここは扱わない。

## システム全体

Context を跨いで固定されるもの。
どの Context を読む前でも成り立つ。

| 文書 | 定めるもの |
| --- | --- |
| [用語集](glossary.md) | Context を跨いで意味が固定される語 |
| [全体の標準仕様](standards.md) | システム全体として準拠する外部規範 |
| [構造](structure.md) | Context の内部構造、Context 間イベント、リポジトリの配置 |
| [システム横断シナリオ](scenarios.feature.md) | 一つの Context では起こせない振る舞いの規範シナリオ |

## Bounded Context

Context の一覧と Subdomain の区分は[論理アーキテクチャ](../architecture/logical.md#context-map)に記録する。
ここに一覧を複製すると、Context を足したときに片方が古くなる。

各 Context のディレクトリには同じ 7 文書を置く。

| 文書 | 定めるもの |
| --- | --- |
| `README.md` | その Context の責務と境界 |
| `glossary.md` | その Context でだけ意味が定まる語、および全体の語をその Context へ狭めた定義 |
| `standards.md` | その Context が準拠する外部規範と、規範からの逸脱 |
| `states.md` | 状態と遷移の表。状態遷移図はこの表から生成する |
| `decisions.md` | その Context に閉じた設計判断とその理由 |
| `internals.md` | 現在の内部設計と機構 |
| `scenarios.feature.md` | 規範シナリオ。`REQ-<CONTEXT>-NNN` と `EX-<CONTEXT>-NNN-NN` を宣言する |

モデル、API、認証機構の契約は `spec/contexts/<context>/` の TypeSpec が定める。
散文の側に同じ契約を書かない。
