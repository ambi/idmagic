---
status: completed
authors: [tn]
risk: low
created_at: 2026-07-18
depends_on: []
change_kind: tooling
spec_impact:
  kind: none
  reason: >-
    SCL の検証 schema (tools/yaml-check)、RA/SCL メタドキュメント
    (SPECIFICATION_CORE_LANGUAGE.md)、新規 ADR、implement-work-item skill を変更する
    プロセス改善であり、app-level の spec/*.yaml が宣言する規範要件・契約は変更しない。
---

# 実装が仕様の素朴な期待より狭い場合の開示義務を RA/SCL に組み込む

## Motivation

wi-238 (SCIM inbound filter/pagination) の実装では、`backend/scim/domain/filter.go` が
RFC 7644 §3.4.2.2 filter grammar のサブセット(属性・演算子 allowlist)だけを実装した。
この allowlist 化自体は wi-238 の `## Plan` に事前明記された正当な決定だったが、その内側で
生じたさらなる実装レベルの制約(`gt`/`ge`/`lt`/`le` が構文として認識されるが実質使用不能、
複数値属性の複合フィルタ未対応 等)は、work item のどの構造化フィールドにも現れず、完了報告
(work item の `completion.summary` およびユーザーへの最終チャット報告)でも自己申告されず、
ユーザーが直接質問して初めて判明した。

原因は次の3点:

1. SCL の `standards.requirements[].adoption` が `required | optional | excluded` の3値しかなく、
   「標準を部分的にしか採用しない」を表す構造化フィールドがない。`excluded` と違い `partial` 相当の
   決定は自由記述の interface description に埋もれるしかなく、`yaml-check` で検証できない。
2. `implement-work-item` skill の ADR 作成トリガーが「非自明な設計判断があれば」という完全に
   裁量任せの一文のみで、具体的な基準がない。
3. `implement-work-item` skill の完了処理手順に、ユーザー向け最終報告へ限界事項を含めるという
   指示が一切ない。

`ARCHITECTURE.md` の複雑度 budget/debt (budgets 超過は無条件で fail、`debts` に `ceiling` 付きで
明示宣言しない限り通らない) という既存の前例と同型の設計——超過・narrowing 自体は禁止しないが、
必ず構造化フィールドで明示宣言させ、宣言がなければ機械的に fail する——を standards adoption にも
適用する。

## Scope

- `tools/yaml-check/schemas/scl-v3.schema.json`: `Standard.requirements[].adoption` に `partial` を
  追加し、`reason` 必須条件を `excluded` だけでなく `excluded | partial` に拡張する。
- `SPECIFICATION_CORE_LANGUAGE.md` §2.1 (68–114行) の requirement field 表を `partial` に合わせて更新する。
- `decisions/ADR-121-scope-narrowing-disclosure-obligation.md` を新規作成する。
- `.agents/skills/implement-work-item/SKILL.md` (`.claude/skills` はこれへのシンボリックリンク) の
  手順1・項目2 (ADR トリガー)、§1.1 (test-first の規律)、「3. 完了処理」を更新する。

## Out of Scope

- fuzz/property test を全 untrusted-input parser に一律義務化すること(費用対効果が悪いとして
  ユーザーが明示的に却下。文法が複雑・高リスクと判断される場合のみの判断基準にとどめる)。
- wi-238 の SCL (`spec/contexts/scim.yaml`) への遡及適用(`adoption: partial` の追記)。
  過去の完了済み決定は書き換えず、この機構は今後の work item から適用する。
- ADR/開示義務そのものの完全自動検証(機械強制するのは `adoption: partial/excluded` の
  `reason` 必須化のみ。それ以外は ADR-119 と同じ self-attest 方式)。

## Plan

- SCL schema の `enum`/`if-then` 拡張は、既存の `excluded` + `reason` パターンをそのまま踏襲する
  (新フィールドは増やさない)。
- ADR-121 は `ADR_FORMAT.md` の型(コンテキスト/決定/却下した代替案/影響)に従う。却下した代替案には
  「AI の裁量に委ねる」(今回機能しなかった)と「fuzz/property test の一律義務化」(ユーザーが
  費用対効果を理由に却下)を明記する。
- `implement-work-item` skill の変更は3箇所: (1) ADR トリガーを具体的な箇条書きに置換、
  (2) test-first 節に fuzz/property test 検討の一文を追加(リスクベース、義務ではない)、
  (3) 完了処理の先頭に開示手順を追加。

## Tasks

- [x] T001 [Tooling] `tools/yaml-check/schemas/scl-v3.schema.json` に `adoption: partial` を追加し、
      `reason` 必須条件を拡張する。`SPECIFICATION_CORE_LANGUAGE.md` §2.1 を同期する。
- [x] T002 [Docs] ADR-121 を作成する(開示義務・ADR トリガー・完了報告義務・リスクベースの
      fuzz/property test 判断基準を決定として記録)。
- [x] T003 [Tooling] `implement-work-item` skill の ADR トリガー・test-first 節・完了処理を
      ADR-121 に合わせて更新する。
- [x] T004 [Verify] `just yaml-check`、`just check-ids` を実行し、既存の全 `spec/contexts/*.yaml` /
      `spec/scl.yaml` が新 schema で引き続き通ることを確認した(全 244 work item、360 record id、
      21 SCL ファイル、ARCHITECTURE.md、traceability manifest/evidence すべて green)。

## Verification

- `just yaml-check`
- `just check-ids`

## Risk Notes

低リスク。振る舞い変更を伴わないプロセス・ドキュメント変更であり、既存の `spec/*.yaml` は
`adoption` に新しい許容値が増えるだけで後方互換(既存の `required`/`optional`/`excluded` は
そのまま有効)。唯一の懸念は、`if/then` 条件の書き方を誤ると既存の `excluded` 検証まで壊す
リグレッションを招くことで、`just yaml-check-scl` で全 SCL ファイルへの回帰を確認する。

## Completion

- **Completed At**: 2026-07-18
- **Summary**:
  `tools/yaml-check/schemas/scl-v3.schema.json` の `Standard.requirements[].adoption` に `partial`
  を追加し、`reason` 必須条件を `excluded` から `excluded | partial` へ拡張した。
  `SPECIFICATION_CORE_LANGUAGE.md` §2.1 を同期し、`partial` の意味と ADR-121 へのリンクを追記した。
  `decisions/ADR-121-scope-narrowing-disclosure-obligation.md` を新規作成し、(1) SCL の
  `adoption: partial` 機構、(2) ADR 作成トリガーを「非自明だと思ったら」という裁量任せから
  具体基準(標準の adoption を partial/excluded にした、または work item の Motivation/Scope/
  タイトルが示唆する範囲より狭い実装をした)へ置換、(3) 完了報告への開示義務(「全部グリーン」
  だけでの完了報告を禁止)、(4) fuzz/property test はリスクベースの判断基準にとどめ全
  untrusted-input parser への一律義務化はしない、の4点を決定として記録した。
  `.agents/skills/implement-work-item/SKILL.md` (`.claude/skills` はこれへのシンボリックリンク)
  の ADR トリガー・test-first 節・完了処理の3箇所を ADR-121 に合わせて更新した。
- **Affected Guarantees State**:
  今後の work item は、SCL の標準 adoption を `partial`/`excluded` にする際に `reason` を
  `yaml-check` で機械強制される。ADR 作成義務と完了報告への開示は self-attest
  (ADR-119 と同方式)であり、機械強制ではない。過去に完了した work item(wi-238 を含む)の
  `completion` 履歴・関連 SCL は遡及的に変更していない。
- **Verification Results**:
  - `just yaml-check-scl` — passed(全21 SCL ファイル、新 schema で既存 `excluded` 検証を含め
    後方互換を確認)
  - `just yaml-check` — passed(work-items 244件、record id 360件、ARCHITECTURE.md、
    traceability manifest/evidence すべて green)
  - `just check-ids` — passed(ADR-121 の番号衝突なし)
- **Out of Scope として意図的に対応しなかったこと**(ADR-121 が定める開示義務の自己適用):
  - fuzz/property test を全 untrusted-input parser に一律義務化すること
    (ユーザーが費用対効果を理由に明示的に却下)。
  - wi-238 の `spec/contexts/scim.yaml` への `adoption: partial` の遡及適用
    (過去の完了済み決定は書き換えない方針のため、今回は行わなかった)。
