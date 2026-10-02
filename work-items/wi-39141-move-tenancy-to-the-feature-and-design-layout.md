---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-03
priority: p1
depends_on: [wi-21670-move-identity-management-to-the-feature-and-design-layout]
change_kind: docs
spec_impact:
  kind: none
  reason: "Tenancy の文書の置き場所と書式だけを変える。REQ-TENANCY の各規則と EX の ID、例のステップ、規則が定める値、状態遷移の表は変えない。"
---

# Tenancy の仕様と設計を、機能仕様と内部設計の構造へ移す

## 動機

Tenancy の文書には、IdManagement と同じ問題がある。
ルートの `decisions.md` には、認可、キー、分類の判断が箇条書きで並ぶ。
`internals.md` は、テナントの識別子の一節だけである。
機能ノードの `decisions.md` と `internals.md` では、同じ判断と理由が重ねて書かれている（ブランド設定の項目の制限、上限を変更する権限など）。

wi-21670 で IdManagement の移行を終え、その見本で構造を確定させた後に、Tenancy を同じ手順で移す。

## 対象範囲

- 機能群と機能ノードを次の構成にする。

  | 機能群 | 機能 |
  | --- | --- |
  | テナント | lifecycle、resolution、integration-endpoints |
  | テナントの設定 | settings、branding、attribute-schema、notification-template、quota |

- ルートの `decisions.md` と `internals.md` を、機能ノードの注記、`design/decisions.md`、`design/<concept>.md` へ振り分ける。
- 機能ノードの `decisions.md` と `internals.md` を、機能仕様の **判断** の注記と、機能の `design.md` へ移す。
  重ねて書かれた判断と理由は、一か所にまとめる。
- `backend/tenancy` の構成要素を棚卸しし、`design/README.md` の構成表と設計視点の網羅表を埋める。
- 移した文書を指すリンクを直す。

## 対象外

- 規則の内容の変更。
- テナント分離の規則そのもの。
  これは `docs/design/security/authorization.md` が扱う。

## 設計

`design/decisions.md` へ書く判断の候補は、次のとおりである。

- テナントのキーを、不変な UUID と可変な realm に分ける。
- テナントの解決は、パスの接頭辞をデフォルトとする。
- テナントの正規ロケーションを一つにする。
- この Context を `Supporting` に分類する。
- 管理の認可を、所属テナント内の `admin` と、テナントを越える `system_admin` の 2 段にする。
- リソース上限を Hard と Soft に分ける。

構成表の軸は、wi-21670 で確定した形に合わせる。
Tenancy のコードは、機能スライスを持たない一つの層構成（`backend/tenancy/{domain,ports,usecases,handlers_http,db_postgres,db_memory}`）である。
そのため構成表の行は、機能スライスではなく機能ノードにする。

## 計画

1. 機能群の README と、機能ノードの仕様本文と付録を作る。
2. 判断と仕組みの説明を振り分け、`design/` を作る。
3. リンクを直し、旧形式の一覧から tenancy を外す。

## タスク

- [ ] T001 [Spec] 機能群と機能ノードを新しい構造へ移す。
- [ ] T002 [Spec] 判断と仕組みの説明を振り分け、`design/` を作る。
- [ ] T003 [Spec] 流入するリンクと、未完了の work item のパスを直す。
- [ ] T004 [Verify] 検査と spec-diff で、規則が変わっていないことを確かめる。

## 検証

- `mise run check-spec`、`mise run check`
- `mise run spec-diff` で、REQ と EX の削除と変更が 0 件であることを確かめる。
- 生成した HTML で、移した各ページを確かめる。

## リスク

テナントの解決とリソース上限は、セキュリティの境界に関わる。
判断を注記へ移すときに、拒否の条件を説明する文を落とさないよう、移行の前後で文を照合する。
