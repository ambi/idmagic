---
status: accepted
authors: [tn]
created_at: 2026-07-18
---

# ADR-121: 実装が仕様の素朴な期待より狭い場合の開示を義務化する

## コンテキスト

wi-238 (SCIM inbound filter/pagination) の実装は、`backend/scim/domain/filter.go` に
RFC 7644 §3.4.2.2 filter grammar のサブセット（属性・演算子 allowlist）だけを実装した。この
allowlist 化自体は wi-238 の `## Plan` に事前明記された正当な先行決定である。しかし、その内側で
さらに生じた実装レベルの制約——`gt`/`ge`/`lt`/`le` が構文として認識されるが実質使用不能であること、
複数値属性の複合フィルタ（bracket 構文）が未対応であること——は、work item のどの構造化フィールドにも
現れず、完了報告（work item の `completion.summary`、およびユーザーへの最終報告）でも自己申告
されなかった。ユーザーが直接質問して初めて判明した。

原因は3点に整理できる。

1. SCL の `standards.requirements[].adoption` は `required | optional | excluded` の3値のみで、
   「標準を部分的にしか採用しない」を表す構造化フィールドがない。`excluded`（`reason` MUST）と違い、
   `partial` 相当の決定は自由記述の `interfaces.*.description` に埋もれるしかなく、`yaml-check` で
   検証できない。
2. `implement-work-item` skill の ADR 作成トリガーが「非自明な設計判断があれば `new-adr` Skill で
   ADR を残す」という一文のみで、具体的な基準がない（`new-adr` skill 自体もフォーマットのみを扱い、
   トリガーは持たない）。今回はこの「非自明かどうか」の判断を実装者（AI）自身が下し、ADR を起こさな
   かった。
3. `implement-work-item` skill の完了処理手順に、ユーザー向け最終報告へ限界事項を含めるという
   指示が一切ない。「`completion` 追記 → `done/` へ移動 → commit」の3手順のみである。

`ARCHITECTURE.md` の複雑度 budget/debt は既存の類似解である: budget 超過は無条件で
`yaml-check-architecture` が fail し、`debts` に `ceiling` 付きで明示宣言しない限り通らない。
超過そのものは禁止しないが、必ず構造化フィールドで明示宣言させ、宣言がなければ機械的に fail する
という設計であり、standards adoption の部分実装にも同型で適用できる。

## 決定

**1. SCL に `adoption: partial` を追加する。**
`Standard.requirements[].adoption` の許容値を `required | optional | excluded | partial` に拡張し、
`reason` の MUST 条件を `excluded` だけでなく `excluded | partial` に拡張する（新フィールドは増やさず
既存の `reason` を再利用する）。`partial` は「requirement の一部だけを実装し、どこまでを実装したか・
していないかを `reason` に書く」ことを表す。これにより標準の部分採用は自由記述の interface
description に埋もれず、`spec/*.yaml` 内の検索可能・`yaml-check` 検証可能な構造化事実になる。

**2. ADR 作成トリガーを裁量から具体基準へ置換する。**
次のいずれかに該当する work item は ADR 作成を必須とする（「非自明だと思ったら」ではなく機械的に
チェック可能な基準）。

- SCL の `adoption` を `partial` または `excluded` にした。
- work item の Motivation / Scope / タイトルが示唆する範囲より狭い範囲だけを実装した
  （例: 「〜conformance」「RFC 準拠」を謳いながら grammar の一部しか実装しない）。

**3. 完了報告への開示を義務化する。**
work item 完了時、ユーザー向け最終報告（チャット応答）には、対象範囲に `adoption: partial/excluded`
の requirement または work item の `## Out of Scope` がある場合、それらを要約した「対応していない
こと」を明記する。「全部グリーン」だけで完了を報告することを禁止する。

**4. fuzz/property test はリスクベースの判断基準とし、一律義務化しない。**
外部の未信頼入力を解釈するコードのうち、文法が複雑（再帰・組み合わせ爆発を伴う、今回の filter
grammar のような）、または認証・認可判定に関わるなど攻撃面として高リスクと判断される場合は、
fuzz/property test の要否を明示的に検討し、Tasks か Risk Notes にその判断（採用する/しないと
その理由）を記す。単純な固定フォーマットのパーサーには適用しない。

強制方式は [ADR-119](decisions/ADR-119-test-first-discipline-for-behavior-layers.md) と同じ
**self-attest ＋ レビュー抜き取り**とする。ADR 作成義務・完了報告への開示・fuzz test 要否の検討は
実装者（AI）がスキル文書の指示に従うことに依存し、完全な自動検出はできない。ただし
`adoption: partial/excluded` の `reason` 必須化だけは `yaml-check` で機械強制される。

## 却下した代替案

- **実装者（AI）の裁量に委ねる（現状維持）**: まさにこれが今回機能しなかった。「非自明かどうか」を
  実装者自身に判定させると、実装者に都合の良い方向（省略）へバイアスがかかる。
- **全 work item に ADR 作成を義務化する**: 過重。大半の work item にはスコープの曖昧さがなく、
  ADR を書く価値がない。ADR インフレは重要な決定を埋もれさせる逆効果を招く。
- **fuzz/property test を全 untrusted-input parser に一律義務化する**: 費用対効果が悪い。
  ユーザーが明示的に却下し、文法複雑・高リスクなケースに限定する判断基準へ縮小した。
- **完了報告の開示を機械強制する（例: completion.summary に特定フィールドを必須化）**: 「対応して
  いないこと」の中身は work item ごとに自由記述であり、機械的に「十分に開示されているか」を検証する
  ことはできない。self-attest とレビュー抜き取りに委ねる。

## 影響

- `tools/yaml-check/schemas/scl-v3.schema.json`: `Standard.requirements[].adoption` に `partial` を
  追加し、`reason` 必須条件を拡張する。
- `SPECIFICATION_CORE_LANGUAGE.md` §2.1 の requirement field 表を `partial` に同期する。
- `.agents/skills/implement-work-item/SKILL.md`: 手順1・項目2（ADR トリガー）を本 ADR の具体的基準へ
  置換し、§1.1（test-first の規律）へリスクベースの fuzz/property test 判断基準を追記し、
  「3. 完了処理」の先頭に完了報告への開示手順を追加する。
- 過去に完了した work item（wi-238 を含む）の `completion` 履歴は書き換えない。本 ADR は今後の
  work item から適用する。
