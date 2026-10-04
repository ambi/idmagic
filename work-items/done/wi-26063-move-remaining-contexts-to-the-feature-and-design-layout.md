---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-03
priority: p2
depends_on: [wi-21670-move-identity-management-to-the-feature-and-design-layout, wi-39141-move-tenancy-to-the-feature-and-design-layout]
change_kind: docs
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  reason: 残りの Context の仕様文書の配置と書式だけを変える。製品の利用者が観測する振る舞い、API、設定は変わらない。
  references: []
initial_context:
  specification:
    - SPECIFICATION_FORMAT.md
    - docs/domain/tenancy/README.md
    - docs/domain/tenancy/design/README.md
    - docs/domain/tenancy/design/architecture.md
    - docs/domain/tenancy/design/decisions.md
    - docs/domain/tenancy/quota/README.md
  typespec: []
  source:
    - tools/check/src/feature-nodes.ts
    - tools/check/src/check-specification-rules.ts
    - tools/check/legacy-spec-layout.json
    - tools/check/feature-node-debt.json
    - tools/check/relocated-spec-paths.json
  tests: []
  stop_before_reading: [frontend]
spec_impact:
  kind: none
  reason: "残りの Context の文書の置き場所と書式だけを変える。各 Context の REQ と EX の ID、例のステップ、規則が定める値、状態遷移の表、標準仕様の表は変えない。"
---

# 残りの Context の仕様と設計を、機能仕様と内部設計の構造へ移す

## 動機

wi-43431 は、新旧の二つの形式を Context ごとに切り替える仕組みを入れる。
IdManagement（wi-21670）と Tenancy（wi-39141）を移した後も、19 の Context が旧形式のまま `tools/check/legacy-spec-layout.json` に残る。
二つの形式が並ぶ間は、読み手は Context ごとに読み方を変えなければならない。
検査と生成器も、二つの経路を保守し続けることになる。

## 対象範囲

- `legacy-spec-layout.json` に残る Context を、wi-21670 と wi-39141 で確定した手順で移す。
- セキュリティ統制の検査（`tools/check/src/check-security-controls.ts`）が固定のパスで読む `docs/domain/api-tokens/scenarios.feature.md` を、api-tokens を移すときに新しい置き場所へ改める。

## 対象外

- 規則の内容の変更。
- 旧形式の経路を検査と生成器から取り除き、`legacy-spec-layout.json` を削除すること。wi-53794 へ分けた。
- 仕様文書の木の用語（「機能ノード」）と例の付録の名前（`examples.feature.md`）の見直し。wi-99596 が扱う。
- 移行の途中で見つけた、コードと旧文書の食い違いの修正。各 Context の `design/risks.md` に載せ、修正は別の work item に委ねる。

## 設計

移行の手順は、wi-21670 と wi-39141 の完了時点の手順に従う。

### 着手時に確定した方針

| 論点 | 決定 | 理由 |
| --- | --- | --- |
| 分割 | 19 の Context の移行は一つの記録で行い、旧形式の経路の除去（当初の T002）だけを wi-53794 へ分ける | Context ごとに記録を分けると、準備の読み込み、frontmatter、設計、完了の記録を 19 回繰り返す。経路の除去はツールのテストの書き直しが主で、文書の移行とレビューの観点が異なり、独立して受け入れられる |
| 機能ノードの切り方 | コードに機能スライスがあればそれに合わせる（Authentication、OAuth2、Sourcing）。ない Context は、操作と状態機械のまとまりで切る。機能群の階層は置かず、Context の README の機能の索引で表す | 現行の規約は「すべての機能スライスに機能ノードを置く」を検査する。機能群を置かないのは IdManagement と Tenancy の見本に合わせる |
| 旧形式の規則の本文 | 本文の行は変えずに移す。`Primary actor:` だけの規則は、行為者を機能仕様の概要へ移し、要件文を書かない | 規則の書き直しは wi-35451 が扱う。照合で忠実さを確かめられるようにする |
| 廃止した規則 | 見出しと本文を機能仕様へ移し、付録からは外す | 付録の `Rule` には一つ以上の例を求められる |
| 旧 `decisions.md` | 代替案が本文から読み取れる判断は `design/decisions.md` へ、それ以外はモデルの判断の欄か、セキュリティ上の考慮の節へ振り分ける。操作とスコープの対応の言い換えは写さない | IdManagement と Tenancy と同じ扱い |
| 旧 `internals.md` | 一つの機能の仕組みは機能の `design.md` へ、複数の機能にまたがる仕組みは `design/<concept>.md` か話題の文書へ、ファジングの記述は `design/verification.md` へ移す | 現行の規約の話題の語彙に従う |
| 実装と食い違う内部設計 | コードを読んで確かめ、コードに合わせて書き、食い違いを `design/risks.md` に載せる | wi-21670 と wi-39141 と同じ扱い |
| 見出しのアンカー | 判断の見出しに読点と「2.0」のような点を含めない | 生成する HTML のアンカーと、リンクの検査のアンカーが食い違う |

## 計画

1. 規模の小さい Context から順に移し、Context ごとに照合と `mise run check` を通してチェックポイントを作る。
2. api-tokens を移すときに、セキュリティ統制の検査の固定のパスを改める。
3. OAuth2 を移すときに、機能スライスの負債の一覧を空にする。

## タスク

- [x] T001 [Spec] 残りの Context を移す。
  N/A: 製品の規範 ID はない。19 の Context（claim-mapping、data-keys、seeding、workloadidentity、ws-federation、audit、signing-keys、api-tokens、sharedsignals、identity-governance、saml、sourcing、authorization、jobs、application、provisioning、system、authentication、oauth2）を、機能仕様、例の付録、`design/` からなる形式へ移した。Context ごとに `mise run check` が通る。`mise run check` の用語の検査が「既定」と「容量」を 4 回 RED として検出し、`check-rendered-docs` が読点と「2.0」を含む見出しのアンカーの不一致を 2 回検出し、直して GREEN にした。
- [x] T002 [Tooling] セキュリティ統制の検査の固定のパスを改める。
  N/A: 製品の規範 ID はない。旧パスのままでは `mise run check-repository` が `ENOENT: docs/domain/api-tokens/scenarios.feature.md` で失敗し、新しいパス `docs/domain/api-tokens/authentication/examples.feature.md` で、拒否の宣言の件数が移行前と同じ 63 件と 95 件に戻ることを確かめた。
- [x] T003 [Spec] 流入するリンク、Go と TypeScript のコメント、未完了の work item のパスを直す。
  N/A: 製品の規範 ID はない。`mise run check-work-items` と `mise run check-links` が旧パスの参照を RED として検出した。未完了の work item の `affected_spec` は要件 ID ごとの移動先へ書き換え、完了した work item は `tools/check/relocated-spec-paths.json` で読み替えた。生成物 `ROUTE_PRIORITY.md` とそれを書く Go の文字列も新しいパスへ直した。
- [x] T004 [Verify] 検査と spec-diff で、規則が変わっていないことを確かめる。
  `mise run spec-diff` は main に対する規範の変更を報告しない。作業用の照合の道具で、19 の Context の要件 263 件の題名と本文、例の見出し 606 件とそのステップ、状態表 21 個を main の旧ファイルと照合し、すべて一致した。

## 検証

- `mise run check`
- `mise run spec-diff` で、REQ と EX の削除と変更が 0 件であることを確かめる。

## リスク

旧形式の経路を取り除く前に移行し漏れた Context があると、検査が失敗する。
経路を取り除くのは、`legacy-spec-layout.json` が空になった後に限る。

## 完了

- **Completed At**: 2026-10-04
- **Summary**:
  `mise run spec-diff` は、main に対する規範仕様の差分を報告しない。
  残りの 19 の Context を、IdManagement と Tenancy と同じ機能仕様と内部設計の構造へ移し、旧形式のまま残る Context はなくなった（`tools/check/legacy-spec-layout.json` の一覧は空である）。
  各 Context の README は責務と境界、モデル、公開する契約、機能の索引からなり、規則は機能ノードの機能仕様へ、例は付録へ、判断と仕組みは機能の `design.md` と `design/`（アーキテクチャ、重要な判断、話題ごとの文書、横断的概念）へ振り分けた。
  Authentication、OAuth2、Sourcing はコードの機能スライスに合わせて機能ノードを置き、`tools/check/feature-node-debt.json` の未対応のスライスは空になった。Authentication のルートの規則はログインとアカウントポータルの機能ノードへ移した。
  要件の題名、本文、例、状態表は変えていない。`Primary actor:` だけの要件は要件文を持たないまま移し、書き直しは wi-35451 に委ねた。
  旧内部設計のうちコードと食い違う記述はコードに合わせて書き、食い違いを各 Context の `design/risks.md` に載せた。主なものは、DataKeys の DEK のローテーションと無効化と破棄を呼ぶ経路がないこと、Seeding、IdGovernance、Sourcing が記録の正を持つ Context の Repository を直接書くこと、Audit の 7 年の保持の目標と保持期間の掃引の食い違い、SharedSignals の失効エポックの反応が `worker` のイベントに働かないこと、SigningKeys の XML の署名の鍵が自動で回らないことである。
  旧形式の経路の除去は wi-53794 へ、用語とファイルの名前の見直しは wi-99596 へ分けた。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-work-items`、`mise run check-links`、`mise run check-repository`
  - **Requirement**: N/A: 文書の配置と書式の移行であり、製品の規範を変えない。
  - **Observed Failure**: 旧形式のファイルを移すたびに、未完了と完了した work item の `affected_spec` とリリース文書のリンクが旧パスで解決できないと報告された。api-tokens を移した時点で、セキュリティ統制の検査が `ENOENT: docs/domain/api-tokens/scenarios.feature.md` で失敗した。
  - **Detection Reason**: 受け入れ境界は該当しない。規則と文書を移すと旧パスを指す参照は解決できなくなり、参照の検査がそれを一件ずつ検出する。
- **Unit RED Evidence**:
  - **Test**: `mise run check`（用語の検査と生成した HTML のアンカーの検査）、作業用の照合の道具
  - **Requirement**: N/A: 文書の配置と書式の移行であり、製品の規範を変えない。
  - **Observed Failure**: 用語の検査が「既定」と「容量」を、生成した HTML の検査が読点と「2.0」を含む見出しのアンカーを拒否した。照合の道具は、Authentication の REQ-AUTHENTICATION-009 の本文の行が付録に残っていることを差として報告した。
  - **Detection Reason**: `spec-diff` は形式の移行では題名だけを比べるので、本文と例の欠落は照合の道具で検出する。
- **Change-Resistance Results**:
  文書の移行であり、変異の対象となるコードの変更はない。
  照合の道具は、Tenancy の移行の前のリビジョン（ff305686 の親）に対して実行し、wi-39141 が報告した既知の差 3 件だけを検出することで、検出の能力を確かめてから使った。
- **Verification Results**:
  - `mise run check` - 成功
  - `mise run spec-diff` - 規範の変更なし
  - 作業用の照合の道具 - 19 の Context の要件 263 件、例の見出し 606 件とそのステップ、状態表 21 個がすべて一致
  - `mise run verify` - 成功（初回は `lint-tools` が `relocated-spec-paths.json` の整形で失敗し、`mise run format-tools` で直した）
