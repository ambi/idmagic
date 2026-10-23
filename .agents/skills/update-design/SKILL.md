---
name: update-design
description: モジュール、公開パッケージ、組み立て地点、テーブル所有者、全体構造、技術、実行時の構成、主要な設計規則の変更を、担当する現在状態の文書へ反映する。
---

# 現在状態の設計の更新

1. 作業項目では `mise run brief -- <work-item> --target <対象パス>` から、変更を所有する設計と規約の該当節を選ぶ。
   境界を変えるときは `--concern boundary`、作用と確定を変えるときは `--concern effects` を付ける。
   起点と追加取得は[作業文脈の導出](../../../docs/development/task-context.md#起点と追加取得)に従う。
2. [一次情報源の割り当て](../../../docs/formats/documentation-guide.md#一次情報源の割り当て)から、変更した事実を所有する最小の文書を選ぶ。
   全体の構造は[要件文書と全体設計文書](../../../docs/formats/documentation-guide.md#要件文書と全体設計文書)、モジュール内の構成は[モジュール設計文書](../../../docs/formats/documentation-guide.md#モジュール設計文書)から該当する節を読む。
3. 境界の変更では、実装前に[境界を選ぶ判断手順](../../../docs/design/application/design-guidelines.md#境界を選ぶ判断手順)で比較した記録に基づき、責務表、公開範囲、テーブル所有者と関連する索引を更新する。
   全体の索引は [docs/README.md](../../../docs/README.md)、実装構成と依存規則は[バックエンド設計](../../../docs/design/application/backend.md)へ反映する。
   コードから導けるモジュール間の依存の辺は文書へ複製しない。
4. モジュールの固有設計、例外、品質要件の担当は[設計領域と割り当ての形式](../../../docs/formats/design-format.md#設計領域)に従って更新する。
   モジュールの責務、主な流れ、素直な形から外れる構造と機能の入口は `docs/modules/<module>/README.md` へ、Aggregate の変更はその `data.md` の集約へ反映する。
   機能スライスを追加、改名、削除した場合は、対応する[仕様の木](../../../docs/formats/specification-format.md#仕様の木)と機能群の索引も更新する。
5. 信頼境界、主体、外部連携、シークレット、個人データ、後から証明する記録を変える場合は、[脅威モデル](../../../docs/design/security/threat-model.md#読み方)の該当箇所を更新する。
   `covered` への変更は、その制御を規範 ID が名指す場合に限る。
6. 作業固有の代替案、計画と履歴は作業項目に残し、現在の設計を作業記録へ委ねない。
   API 契約は TypeSpec、外から観測する振る舞いは担当する機能仕様へ反映する。
7. `mise run check-spec` と `mise run check-boundaries` を通す。

設計文書と判断の記録が現在状態を持つため、別の ADR やアーキテクチャ台帳は追加しない。
