---
status: accepted
authors: [tn]
created_at: 2026-07-17
---

# ADR-115: SCL 規範要素を四層 workspace 追跡グラフで直接検証する

## コンテキスト

SCL、Architecture module、実行可能な check、CI evidence は個別に存在するが、対応関係と
source revision に対する証跡の新鮮さを workspace 単位で検証できなかった。ADR-114 は
context-qualified SCL element reference を定義したが、外層の realization と evidence の所有境界は
未決定だった。

## 決定

`SCL normative element -> Architecture realization -> declared verification -> execution evidence`
の四層グラフを `verification/manifest.yaml` から構築する。SCL に assurance 台帳や
逆向き link は追加せず、ADR-103 / ADR-114 の局所所有と direct reference を維持する。

宣言済み check と実行結果は分離し、`verification/evidence.yaml` は source revision、実行時刻、
result、artifact 参照を持つ。coverage は selector ごとに realization、最小 check 数、許容
evidence kind を検査する。既存 debt は owner、reason、expires_at が必須の baseline で期限付き
許容とし、strict gate は新規 drift と期限切れ baseline を失敗させる。

## 却下した代替案

- SCL に assurance obligation を複製する案: 所有要素の contract と二重の規範になる。
- テスト名や description から対象を推測する案: 名称変更に弱く、保証意味を機械的に決められない。
- 成功した check の存在だけを coverage とする案: 古い revision の成功を現在の証跡と誤認する。
- 無期限の ignore リスト: debt が恒久化し strict gate の意味を失う。

## 影響

- `tools/ra/spec/scl.yaml` の `interfaces.CheckTraceability`、`models.TraceabilityReport`、
  `objectives.TraceabilityDeterminism` がこの決定を仕様化する。
- `tools/yaml-check/spec/scl.yaml` の `interfaces.CheckYaml` は manifest/evidence schema と Work Item direct
  reference の検査を含む。
- CI は source revision ごとの evidence を生成する必要があり、bootstrap 期間の stale evidence は
  期限付き baseline として report される。
