---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-10
priority: p1
depends_on: []
change_kind: tooling
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  reason: "未公開期間の開発検査と告知要求を変える tooling 変更であり、製品の利用者向け差分を生まない。"
  references: []
initial_context:
  specification: []
  typespec: []
  source:
    - mise.toml
    - .github/workflows/idmagic-ci.yaml
    - tools/check/src/check-api-compat.ts
    - tools/check/src/registry.ts
    - tools/check/src/check-work-items.ts
    - tools/check/src/documentation-impact.ts
    - tools/check/schemas/work-item.schema.json
    - tools/workspace/src/workspace.ts
  tests:
    - tools/check/src/documentation-impact.test.ts
    - tools/check/src/check-work-items.test.ts
    - tools/check/src/api-compat.test.ts
    - tools/check/src/runner.test.ts
    - tools/check/src/mise-config.test.ts
  stop_before_reading: [backend, frontend/src, infra]
spec_impact:
  kind: none
  reason: "未リリース期間の開発に課す告知と互換性検査の条件を変える。現在の HTTP 応答、認証と認可、永続状態、イベントと通知の製品契約は維持する。"
---

# 未リリース期間の告知と旧版互換性を日常開発の必須条件から外す

## 動機

現在は未リリースであり、当面も未リリースを継続する。
それでも API の互換性検査は標準検査と CI に含まれ、新機能にはリリース文書を要求する。
2026-10-10 の調査では、リリース文書は 73 ファイルあった。
公開済みの利用者契約がない段階で、設計変更と告知の維持へ費用を払っている。
将来のリリース基盤の整備は p3 だが、現在の開発を重くする条件の解除は p1 とする。

## 対象範囲

- 公開済みの契約の有無に応じた API 互換性ゲート、告知、成熟度の告知証拠の適用条件。
- 未リリース期間の通常変更では、現在の仕様への整合検査と作業記録だけで完了できる規約。
- 既存の告知断片の保存と探索の扱い、初回公開時にベースラインと必要な告知を確定する手順。
- mise、CI、作業項目の検査、リリース文書、エージェント手順の同期。

## 対象外

- 現在の仕様への整合、認可、テナント分離、永続化の検証を弱めること。
- 実データの保全や実在する外部消費者の契約まで、未リリースという理由で免除すること。
- 本番の配備と将来の更新検証は[本番準備](../active/wi-55532-derive-production-readiness-from-deployment-evidence.md)、[更新の事前判定](../active/wi-449-deployment-update-compatibility-preflight.md)、[混在版検証](../active/wi-450-mixed-version-release-acceptance.md)が扱う。

## 設計

初回公開前と、守るべき公開済み契約が存在する段階を区別する。
初回公開前は旧版互換性と利用者向け告知を必須判定から外し、必要な差分調査は任意で実行できるようにする。
機能の成熟度や既存データの保持条件そのものは変更しない。
既存の告知断片を、未公開なのに公開済み履歴として扱わない。

着手時に公開状態の一次情報と、初回公開へ切り替える地点を決める。
手書きの免除一覧や変更ごとの承認欄は作らず、同じ条件を各検査から使う。
規約へ注意書きだけを追加して必須ゲートを維持する案は、開発費用が変わらないため採らない。

公開状態の一次情報を `spec/release-state.json` の `phase`（`unpublished` または `published`）とする。
起票の未リリース宣言と、リリース用 CI がまだ無い開発手順を根拠に `unpublished` で開始する。
利用者への確認で、追加の実利用契約はないとの回答を得た。
実データの保持と移行検査は公開状態によらず維持し、外部消費者の契約が生じたら初回公開と同じ切り替えを行う。

検査内部の `readReleasePhase(snapshot): Promise<ReleasePhase>` が状態を読み、欠落や不正値を拒否する。
集約検査と CI は `check-published-api-compat` からこの判断を使う。
任意の差分調査用 `check-api-compat` は公開状態によらず比較する。
work item の文書検査にも同じ状態を渡し、未公開では最低告知水準を `none` とする。
明示的に選んだ告知断片は従来どおり検査する。
成熟度の昇格では主要ユースケース、セキュリティ確認、互換性または移行の結果を維持し、告知文書のパスだけを公開後に要求する。
バックエンドのモジュール、公開パッケージ、テーブル所有者と実行時の構成は変えない。

初回公開の準備で現行 OpenAPI をベースラインとして固定し、同じ準備コミットで `phase: published` に切り替える。
既存の告知断片とベースラインは未公開の下書きとして保存し、初回公開の機能一覧と移行条件から必要な文書を選び直す。
以後は未公開へ戻さない運用とし、公開済み契約を保護する。

## 計画

1. 着手時の公開契約と、維持すべき実利用環境を確認する。
2. 適用条件を決め、検査とエージェント手順を同期する。
3. 初回公開前と公開後を検証し、既存の告知とベースラインの扱いを記録する。

## タスク

- [x] T001 [Design] 公開状態、例外となる実利用契約、初回公開時の切り替えを決める。
- [x] T002 [Tooling] 未リリース期間の必須ゲートと告知要求を縮小する。
  - 受け入れ境界の代替検査: `mise run test-tools-file -- check/src/check-api-compat.test.ts`。未公開の削除で `Expected: true / Received: false` を観測し、条件付きゲートへの変更後は GREEN。規範 ID は N/A（tooling）。
  - 単体検査: `mise run test-tools-file -- check/src/documentation-impact.test.ts`。未公開でも `removal_notice` と成熟度の告知パスを要求する失敗を観測し、公開状態の適用後は GREEN。
  - workspace の検査と CI 配線: `check/src/check-work-items.test.ts`、`check/src/mise-config.test.ts` を `test-tools-file` で実行し、状態欠落の拒否と条件付きゲートの呼び出しを確認した。
- [x] T003 [Docs] 規約、手順、既存の告知の扱いを同期する。
  - 開発手順、work item 形式、CI、API 設計、エージェント手順へ公開状態の参照と初回公開の切り替えを反映した。
  - `mise run check-spec`、`mise run check-boundaries`、`mise run check-links`、`mise run check-agent-guidance`、`mise run check-command-map` は成功。
- [x] T004 [Verify] 現在仕様の検証を維持し、公開後の互換性違反を検出できることを確かめる。
  - 公開状態を常に `unpublished` とする故障注入で、公開後の API 削除と告知不足を検出するテストが失敗した。復元後に両ファイルの全テストが成功した。
  - 集約検査へ任意の比較もすべて含める旧前提を、registry と CLI の両方から除き、既存の受け入れ fixture に公開状態を追加した。
  - 七つの視点で、判断の共通化、snapshot への作用の集約、検査の選択と判定の責務、公開前後のテスト、現在契約検査の維持を確認した。
  - `mise run verify` は最終実行で成功。Go の race テスト、tooling の 954 テスト、UI 単体の 709 テストを含む。

## 検証

- 初回公開前の意図した API 変更は、現在の仕様と実装が一致すれば旧版互換性だけを理由に拒否されない。
- 公開済み契約がある入力では、破壊的変更と必要な告知の欠落を拒否する。
- 未リリースという条件で、現在の契約検査や実データの保全条件を外さない。
- mise run test-tools-file、mise run check-work-items、mise run check-links、mise run verify を実行する。
- 受け入れ RED の代替検査は `mise run test-tools-file -- check/src/check-api-compat.test.ts` とし、公開前後の workspace 入力で集約検査と任意検査の結果を比較する。製品の受け入れ境界は N/A（tooling の変更）。
- 単体 RED は `mise run test-tools-file -- check/src/documentation-impact.test.ts` で未公開の機能追加と成熟度の告知パス省略を確認する。規範 ID は N/A（製品要件を変えない）。
- 同じテスト群へ公開後の拒否、状態の欠落と不正値、現在契約検査の構成維持を含める。公開状態の配線を外すフォールト注入で変更耐性を確認する。

## リスク

公開後も免除が残ると、利用者の契約を保護できない。
初回公開への切り替えを明示して検証し、公開状態の変更を見落とさない。

## 完了

- **完了日**: 2026-10-10
- **要約**:
  `mise run spec-diff -- main` は規範仕様の変更なしと報告した。
  初回公開前の通常変更は、現在仕様との整合検査と作業記録で完了できるようにした。
  API 互換性と告知の最低水準は共通の公開状態に応じて適用し、任意の既存バージョン比較を残した。
  成熟度の昇格では告知パスだけを公開後に要求し、主要ユースケース、セキュリティとデータ保持の証拠を維持した。
  既存の告知断片を下書きとして保存し、初回公開の準備コミットでベースラインと公開状態を確定する手順を同期した。
- **受け入れ RED の証拠**:
  - **テスト**: `mise run test-tools-file -- check/src/check-api-compat.test.ts` の「未公開の API 削除を集約ゲートで拒否せず、任意の比較では検出する」。
  - **要件**: N/A: 製品の受け入れ境界を変えない tooling 変更であり、実際の registry と workspace を通る代替検査を使った。
  - **観測した失敗**: 未公開の API 削除が集約ゲートで拒否され、`Expected: true / Received: false` になった。
  - **検出できる理由**: 同じ API 差分に対する集約ゲートと任意の比較の結果を比較し、公開状態による適用の配線を検査する。
- **単体 RED の証拠**:
  - **テスト**: `mise run test-tools-file -- check/src/documentation-impact.test.ts` の「未公開では機能追加、廃止と破壊的変更に告知断片を要求しない」と「未公開の成熟度昇格は告知パスだけを省略できる」。
  - **要件**: N/A: 製品要件を変えず、公開条件による文書検査の内部判断を変更する。
  - **観測した失敗**: 未公開でも `documentation_impact none is weaker than inferred removal_notice` と、昇格の `release_note` および告知パスの欠落が所見になった。
  - **検出できる理由**: 同じ機能差分と昇格の証拠を公開前後で評価し、告知パスだけを省略する判断とセキュリティ確認の維持を区別する。
- **変更耐性の結果**:
  `readReleasePhase` が公開済み入力にも `unpublished` を返す故障を注入した。
  `check-api-compat.test.ts` の公開済み API 削除のテストと、`check-work-items.test.ts` の同じ機能追加を公開後は告知不足で拒否するテストが、いずれも `Expected: false / Received: true` で失敗した。
  故障を復元し、両ファイルの全テストが成功した。
  状態の欠落、不正な JSON と未知の値も拒否するテストを通した。
  Go コードは変更していないため、Go のミューテーションテストは適用しない。
- **検証結果**:
  - `mise run verify` - 最終実行で成功。初回は fixture と集約の旧前提に加え、サンドボックスの待受ポート拒否と Go lint の読込失敗があった。fixture と集約を修正し、サンドボックス外の再実行で残ったバージョン表記も修正した。
  - `mise run check-spec`、`mise run check-boundaries` - 成功。
  - `mise run check-work-items`、`mise run check-links`、`mise run check-agent-guidance`、`mise run check-command-map` - 成功。
  - `mise run test-tools-file` で変更した検査のテスト、既存の受け入れテスト、文書品質検査を実行 - 成功。
  - `mise run typecheck-tools`、`mise run lint-tools`、`mise run lint-go` - 成功。
  - `mise run test-ui-e2e` - N/A: 製品コードとブラウザーへ届く経路は変更していない。
  - `mise run spec-diff -- main` - 規範仕様の変更なし。
