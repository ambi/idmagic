---
depends_on: [wi-163-fleet-scale-capacity-and-horizontal-scaling-architecture]
status: pending
authors: ["tn"]
risk: critical
created_at: 2026-07-10
priority: p3
group: scale-availability
change_kind: operations
affected_spec:
  - { path: docs/modules/system/features/operations.md, requirement: REQ-SYSTEM-002 }
  - { path: docs/modules/system/features/operations.md, requirement: REQ-SYSTEM-001 }
---

# 単一の障害で認証を止めない分散配置を構成し、障害試験と縮退の順序の試験で確かめる

## 動機

IdMagic が止まると、認証を委ねたすべてのシステムのログインが止まる。
起票時はこの項目が可用性の全体設計を担っていたが、その後 [wi-588](../done/wi-588-reliability-and-performance-design.md) が[可用性設計](../../docs/design/reliability/availability.md)、[スケーリング・負荷設計](../../docs/design/performance/scaling.md)、[スキーマ管理の拡張と縮小](../../docs/design/data/schema-management.md#拡張と縮小)を書き、[wi-396](../done/wi-396-prioritize-login-under-saturation.md) が負荷に応じた入場制御を実装した。
Probe、PodDisruptionBudget、再試行、サーキットブレーカー、アラートの不足は、wi-613 から wi-618 が一つずつ扱う。

設計の側に残るのは、可用性設計が「単一障害時に無停止であることは、どの障害単位についても検証していない」と書く点である。
ノードとゾーンへの分散配置の制約は未実装であり、汎用 Kubernetes の PostgreSQL（CloudNativePG のクラスター）は設計上の想定に留まる。
[キャパシティ設計のロードシェディング順序](../../docs/design/performance/capacity.md#ロードシェディング順序)も、容量を超えたときに本当にその順序で絞るかを試していない。

現在は未リリースであり、実利用の負荷も本番の実環境もないので、優先度は p3 とする。

## 対象範囲

- `idmagic-api`、`idmagic-worker`、`idmagic-frontend` に、ノードとゾーンへの分散配置の制約（topology spread constraints）を構成ファイルで置く。
- 汎用 Kubernetes のプロファイルで、PostgreSQL の待機系への自動の切り替えを構成する。
- 複数ノードのローカルクラスターで、Pod、ノード、PostgreSQL の primary をそれぞれ一つ止める障害試験を行い、ログインとトークンの発行が続くことと、切り替えにかかった時間を記録する。
- 容量を人為的に超えさせ、ロードシェディング順序のとおりに受け付けを絞ることと、テナント境界、認証と認可、再送防止、監査の完全性を弱めないことを確かめる。
- 確かめた結果に合わせて、可用性設計の実装状況の区分を改める。

## 対象外

- マルチリージョンのアクティブとアクティブ。リージョンの喪失はバックアップからの復旧で受ける。
- データ層の分割、読み取りレプリカ、接続プーラー。wi-164 が扱う。
- Probe、PodDisruptionBudget、再試行のばらつき、鍵管理サービスのサーキットブレーカー、アラート。wi-613 から wi-618 が扱う。
- 本番のクラウドでの障害試験。初回リリースの対象のデプロイ構成が決まってから扱う。

## 検証

- `mise run check-k8s`
- 障害試験と縮退の順序の試験の手順と結果。ローカルクラスターの起動と後片付けを含めて再現できる形で残す。

## リスク

ローカルクラスターでの障害試験は、マネージドサービスの切り替え時間を表さない。
結果を本番の RTO の根拠にせず、構成が単一障害で止まらないことの確認に限って使う。
