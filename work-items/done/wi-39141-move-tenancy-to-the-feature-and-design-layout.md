---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-03
priority: p1
depends_on: [wi-21670-move-identity-management-to-the-feature-and-design-layout]
change_kind: docs
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  reason: Tenancy の仕様文書の配置と書式だけを変える。製品の利用者が観測する振る舞い、API、設定は変わらない。
  references: []
initial_context:
  specification:
    - SPECIFICATION_FORMAT.md
    - docs/domain/tenancy/README.md
    - docs/domain/tenancy/glossary.md
    - docs/domain/identity-management/README.md
    - docs/domain/identity-management/roles/README.md
    - docs/domain/identity-management/design/README.md
    - docs/domain/identity-management/design/decisions.md
    - docs/domain/identity-management/design/architecture.md
    - docs/design/security/authorization.md
  typespec: []
  source:
    - backend/tenancy/module.go
    - backend/tenancy/ports/quota_repository.go
    - backend/shared/http/support_http/tenant_middleware.go
    - tools/check/src/specification-rules.ts
    - tools/check/src/spec-diff.ts
    - tools/check/legacy-spec-layout.json
    - tools/check/relocated-spec-paths.json
  tests: []
  stop_before_reading: [frontend]
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

### 着手時に確定した方針

| 論点 | 決定 | 理由 |
| --- | --- | --- |
| 機能群の階層 | 置かない。8 つの機能ノードは現在のパスのまま Context の直下に残し、機能群は Context の README の機能の索引で表す | wi-87813 で IdManagement が機能群を廃止しており、見本の形に合わせる。パスが変わらないので、移動に伴う参照の書き換えが `scenarios.feature.md` から `README.md` への変更だけで済む |
| 旧形式の規則の本文 | 本文の行は変えずに移す。`Primary actor:` だけの規則（21 件）は、行為者を機能仕様の概要へ移し、要件文を書かない | 規則の内容と書き方の変更は wi-71372 が EARS 形式の書き直しとして扱う。ここで要件文を書くと、その作業を先取りし、照合で忠実さを確かめられなくなる |
| 欄の名前 | `理由` の欄は `判断` へ改め、機能の `decisions.md` へのリンクは判断の本文で置き換える | `check-spec` が機能仕様で `理由` を拒否する。機能の `decisions.md` は新しい形式に置けない |
| 操作ごとの要約表 | 書かない | 現行の `SPECIFICATION_FORMAT.md` が要約表を廃止した |
| 操作とスコープの対応 | 写さない | TypeSpec の `x-api-token-scopes` と権限の注釈の言い換えである |
| ルートの判断のうちシステム文書が扱うもの | 写さない | 有効な呼び出し元、同一テナント、`Origin` と CSRF の検証は認可設計が、生成する ID の UUID 型はデータベース設計が扱う |
| 実装と食い違う内部設計 | コードに合わせて書き、食い違いを `design/risks.md` に載せる | wi-21670 と同じ扱い。Soft の上限の加算と警告、上限の使用量の照合ジョブはコードにない |

## 計画

1. 機能群の README と、機能ノードの仕様本文と付録を作る。
2. 判断と仕組みの説明を振り分け、`design/` を作る。
3. リンクを直し、旧形式の一覧から tenancy を外す。

## タスク

- [x] T001 [Spec] 機能群と機能ノードを新しい構造へ移す。
  N/A: 製品の規範 ID はない。8 つの機能ノードに仕様本文 `README.md` と付録 `examples.feature.md` を置き、`lifecycle/states.md` の表を状態遷移の節へ移した。付録は作業用のスクリプトで旧ファイルから見出しの段だけを変えて作り、要件の本文は手で移した。`mise run check-spec` が通る。
- [x] T002 [Spec] 判断と仕組みの説明を振り分け、`design/` を作る。
  N/A: 製品の規範 ID はない。ルートと機能ノードの `decisions.md` と `internals.md` を、モデルと要件の判断の欄、セキュリティ上の考慮の節、機能の `design.md`（解決、設定、ブランド設定、通知テンプレート）、`design/`（アーキテクチャ、判断 7 件、データ、リスク）に振り分けた。`mise run check` の用語の検査が「既定」と「容量」を RED として検出し、直して GREEN にした。旧内部設計のうちコードと食い違う記述（存在しない `support_url` と `legal_url`、制御面のルートの接頭辞、Soft の上限、使用量の照合ジョブ）はコードに合わせて書き、後の二つを `design/risks.md` に載せた。
- [x] T003 [Spec] 流入するリンクと、未完了の work item のパスを直す。
  N/A: 製品の規範 ID はない。`mise run check-work-items` が旧パスの `affected_spec` 104 件を、`mise run check-links` がリリース文書 3 件の旧パスへのリンク 10 件を RED として検出した。未完了の work item 15 件のパスを新しい `README.md` へ直し、完了した work item は `tools/check/relocated-spec-paths.json` で読み替えた。システム文書 2 件のリンクと、未完了の work item の本文が名指す旧ファイルも直した。
- [x] T004 [Verify] 検査と spec-diff で、規則が変わっていないことを確かめる。
  `mise run spec-diff` は main に対する規範の変更を報告しない。作業用の照合の道具で、main の旧ファイルと例 104 件の 431 ステップ、要件 43 件の題名、本文 128 行、状態表を照合し、差は `理由` を `判断` へ改めた 3 行だけだった。`mise run check`、`mise run render-docs`、`mise run verify` が通る。

## 検証

- `mise run check-spec`、`mise run check`
- `mise run spec-diff` で、REQ と EX の削除と変更が 0 件であることを確かめる。
- 生成した HTML で、移した各ページを確かめる。

## リスク

テナントの解決とリソース上限は、セキュリティの境界に関わる。
判断を注記へ移すときに、拒否の条件を説明する文を落とさないよう、移行の前後で文を照合する。

## 完了

- **Completed At**: 2026-10-04
- **Summary**:
  `mise run spec-diff` は、main に対する規範仕様の差分を報告しない。
  Tenancy の仕様と設計を、IdManagement と同じ機能仕様と内部設計の構造へ移した。
  8 つの機能ノード（ライフサイクル、解決、連携エンドポイント、設定、ブランド設定、属性スキーマ、通知テンプレート、リソース上限）は Context の直下に置いたままとし、機能群（テナント、テナントの設定）は Context の README の機能の索引で表した。
  Context の README は、責務と境界、5 つの Aggregate のモデル、公開する契約（テナントの解決、`TenantRepository`、`QuotaRepository`、属性スキーマのポート、ブランド設定の読み取り、ドメインイベント）、機能の索引からなる。
  設計は、アーキテクチャ、データ、リスクと、重要な判断 7 件（Supporting への分類、UUID と realm の鍵、パスの接頭辞による解決、一つの正規ロケーション、2 段の管理の認可、Hard と Soft の上限、外装と属性スキーマの独立）からなる。
  要件の題名、本文、例、状態表は変えていない。`Primary actor:` だけの要件 21 件は要件文を持たないまま移し、EARS 形式の書き直しに委ねた。
  旧内部設計のうちコードと食い違う記述は、コードに合わせて書いた。Soft の上限が働かないことと、使用量のカウンターを照合する仕組みがないことは `design/risks.md` に載せた。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-work-items`、`mise run check-links`
  - **Requirement**: N/A: 文書の配置と書式の移行であり、製品の規範を変えない。
  - **Observed Failure**: 旧形式のファイルを移した直後、`check-work-items` は完了した作業項目と未完了の作業項目の `affected_spec` 104 件が旧パスで解決できないと報告し、`check-links` はリリース文書 3 件の 10 件のリンク切れを報告した。
  - **Detection Reason**: 受け入れ境界は該当しない。規則と文書を移すと旧パスを指す参照は解決できなくなり、二つの検査がそれを一件ずつ検出する。
- **Unit RED Evidence**:
  - **Test**: `mise run check`（用語の検査）、作業用の照合の道具
  - **Requirement**: N/A: 文書の配置と書式の移行であり、製品の規範を変えない。
  - **Observed Failure**: 用語の検査が `design/decisions.md` の「既定」と「容量」を拒否した。照合の道具は、`理由` を `判断` へ改めた REQ-TENANCY-028、032、036 の本文を差として報告した。
  - **Detection Reason**: `spec-diff` は形式の移行では題名だけを比べるので、本文と例の欠落は照合の道具で検出する。道具が検出できることは、次の故障の注入で確かめた。
- **Change-Resistance Results**:
  文書の移行であり、変異の対象となるコードの変更はない。
  照合の道具に二つの故障を注入した。EX-TENANCY-024-02 のステップの発行者からポートを消すと、例のステップの差として報告した。REQ-TENANCY-037 の本文から「新しい作成だけを拒否する」を消すと、本文の差として報告した。どちらも戻した後、差は既知の 3 行だけに戻った。
- **Verification Results**:
  - `mise run check-spec` - 成功
  - `mise run check` - 成功
  - `mise run check-links` - 成功
  - `mise run render-docs` - 成功（機能地図、要件一覧、未決事項、状態図を生成したページで確かめた）
  - `mise run spec-diff` - 規範の変更なし
  - `mise run verify` - 成功
