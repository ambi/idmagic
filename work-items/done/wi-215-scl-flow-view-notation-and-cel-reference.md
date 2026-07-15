---
status: completed
authors: [tn]
risk: high
created_at: 2026-07-16
depends_on: []
---

# SCL flows を views/sees/does 記法へ再設計し CEL root binding を形式定義する

## Motivation

[[ADR-112]] が決定した通り、`flows.transitions` のフラット配列は遷移(from/action/to)だけを
強調し、UI フロー資料として本質的な「各画面が何を表示するか(sees)」「利用者が何をするか(does)」
が埋没している。また CEL 式の root binding(`context`/`subject`/`resource`/`principal`/
`input`/`output`/`response`/`measurement`/`request`/`event`/`emitted`)は「使える root 名」
だけが定義され、その実体・形が文書化されていないため読み手が実例から類推するしかなかった。

本 work item はこの2点を、`SPECIFICATION_CORE_LANGUAGE.md` 全体の仕様書としての
形式化とあわせて実装する。

## Scope

- `spec/scl.yaml` 直下の SCL メタ仕様である `SPECIFICATION_CORE_LANGUAGE.md` の全セクション
  (§1 目的 〜 §8 versioning)
- `tools/yaml-check/schemas/scl-v3.schema.json` の `$defs.Flow`
- `tools/yaml-check/src/scl-semantics.ts` の flow 到達可能性・interface 参照解決ロジック
- `tools/scl-to-html` の `Flow` 型・`renderFlows`・関連テスト
- `tools/yaml-check/src/fixtures/scl-v3/{valid/tenancy.json, invalid/broken-flow.json}`
- `spec/contexts/{application,audit,authentication,identity-management,oauth2,saml,signing-keys,system,tenancy,ws-federation}.yaml` の `flows` セクション(32 flow 全件)
- `tools/scl-to-html/spec/scl.yaml`(scl-to-html 自身の自己記述 SCL。`flows.SpecificationNavigation`
  が旧記法を使っており、実装着手後に追加で発見したため移行対象に加える)

## Out of Scope

- `objectives`(現状の SLO 専用の形を維持し、今回は一切変更しない)
- `spec_version` の bump(`"3.0"` のまま据え置く。単一 work item 内の一括移行のため
  新旧共存期間がなく、スキーマ世代を新設する必要がない)
- `scenarios`/`authorization`/`states`/`models`/`interfaces` の実質的な仕様変更
  (`SPECIFICATION_CORE_LANGUAGE.md` 上の記述を正式なリファレンス形式に書き直すのみで、
  スキーマ・validator が強制する制約そのものは変えない)

## Plan

`views`/`FlowView`/`FlowAction` という3つの JSON Schema 定義で
`flows.<Name>.views.<ViewName>.{sees, does: [{action, does, interface?, to?, external?}]}`
を表現する([[ADR-112]] 参照)。`to`/`external` は既存どおり同時指定しない。
`to` は今まで通りバリデーション対象外の裸文字列とし、flow/ファイル横断の遷移
(例: `AccountHome` → 別ファイル定義の `AccountApplications`)を許容し続ける。

`scl-semantics.ts` の到達可能性検証は `views[key].does[].to` を辿る BFS に一般化し、
「宣言された全 `views` キーが `entry` から到達可能」であることを検証する
(view がキーとして明示化されたことによる自然な強化)。同一 view 内の `action` 重複も検出する。

32 flow の実内容移行では、各 view に紐づく実際の `interface` の `requires`/`response`
フィールド定義から `sees`(画面の表示・入力項目)と `does`(操作の説明)を起こす。
interface の裏付けが薄い view は Risk Notes に明記し、推測で埋めずレビュー対象として残す。

`SPECIFICATION_CORE_LANGUAGE.md` の書き直しは、32 flow の移行を終えた後に行う
(実データを突き合わせながら記述の裏付けを取るため)。書き直しの非交渉条件:
ドキュメントは `scl-v3.schema.json`/`scl-semantics.ts` が実際に強制する制約と
過不足なく一致すること(tooling にない制約を創作しない/tooling にある制約を書き漏らさない)。

却下した代替案: `spec_version` を `"3.1"` 等へ上げて新旧スキーマを一時共存させる案は、
2.0→3.0 移行(wi-207〜212)と異なり今回は単一 work item で一括移行するため不要と判断し却下。

## Tasks

- [x] T001 [SCL] `tools/yaml-check/schemas/scl-v3.schema.json` の `Flow` を
      `views`/`FlowView`/`FlowAction` へ再設計する。
- [x] T002 [Tool] `tools/yaml-check/src/scl-semantics.ts` の flow 到達可能性・interface 解決を
      `views` ベースに書き換え、同一 view 内の `action` 重複チェックを追加する。
- [x] T003 [Tool] `tools/yaml-check/src/fixtures/scl-v3/{valid/tenancy.json, invalid/broken-flow.json}`
      を新記法へ移行する。
- [x] T004 [Tool] `tools/scl-to-html` の `Flow` 型・`renderFlows`・`render.test.ts` を
      新記法に対応させる。あわせて `tools/scl-to-html/spec/scl.yaml` の自己記述 flow も移行する。
- [x] T005 [SCL] `spec/contexts/*.yaml`(10 ファイル、32 flow)の `flows` を新記法へ移行し、
      実内容(`sees`/`does`)を記述する。
- [x] T006 [SCL] `SPECIFICATION_CORE_LANGUAGE.md` を全セクション、
      tooling の制約と一対一対応する形式へ書き直す。
- [x] T007 [Verify] `just scl-render` で派生物(HTML/OpenAPI/JSON Schema)を再生成する。
- [x] T008 [Verify] 全検証コマンドを通す。

## Verification

- `just test-tools`
- `just typecheck-tools`
- `just yaml-check-scl`
- `just scl-render`
- `just verify-ui`
- `just yaml-check`
- `just check-ids`

## Risk Notes

- 破壊的スキーマ変更のため、T001〜T004(schema/semantics/tooling/fixture)を先に
  一貫させてから T005(実データ移行)に進む。T005 完了までは `just yaml-check-scl` が
  一時的に失敗する状態を許容する。
- 以下の view は対応する `interface` の裏付けが薄く、`sees`/`does` の記述が推測に頼っている。
  実装後に人間によるレビューを推奨する。
  - `oauth2.yaml` `DeviceAuthorization` flow の `Status` view: 専用 interface がなく、
    `SubmitBrowserDevice` の承認/拒否結果表示から推測している。
  - `system.yaml` `AdminDashboard` view: 2アクションとも interface 紐付けがない
    ナビゲーションハブであり、画面の具体的な表示項目は明言できない。
  - `identity-management.yaml` `AdminRoles` view: 唯一のアクションが `ListAdminUsers`
    interface を再利用しており、ロール一覧画面としての裏付けが薄い
    (既存 spec 側の不整合の疑いがある)。
- `SPECIFICATION_CORE_LANGUAGE.md` の全面書き直しは、tooling が実際に検証していない
  文言(例: `standards`/`context_map` の依存グラフ非循環性)を発見した場合、
  ドキュメント側を実態に合わせて調整するか、別途 tooling 側の検証追加を検討する
  (本 work item のスコープ内で対応可能な範囲に留める)。
  実装時に確認した結果、依存グラフ非循環性と `shared_kernel` 過多 warning は
  `tools/yaml-check/src/context-map.ts` で実際に検証されており、ドキュメントの記述は
  そのまま正確だった(調整不要)。

## Completion

- **Completed At**: 2026-07-16
- **Summary**:
  `flows` の記法を `entry`+`transitions` のフラット配列から、`entry`+`views`(view 名を
  キーとする map、各 view が `sees` と `does`(action の配列)を持つ)へ再設計した。
  JSON Schema(`Flow`/`FlowView`/`FlowAction`)、`scl-semantics.ts` の到達可能性 BFS
  (view キー全件が `entry` から到達可能であることを検証するよう強化)・action 重複検出・
  interface 参照解決、`scl-to-html` の型定義・レンダラー(view ごとに sees を見出しにして
  does の表を並べる形へ変更)・テストを更新し、既存 10 context ファイル・32 flow 全件と
  `scl-to-html` 自身の自己記述 SCL(1 flow)を新記法へ移行して実際の画面内容・操作内容を
  記述した。あわせて `SPECIFICATION_CORE_LANGUAGE.md` を全セクション書き直し、CEL の
  各 root binding(`context`/`subject`/`resource`/`principal`/`input`/`output`/`response`/
  `measurement`/`request`/`event`/`emitted`)の実体を新設の §5.1 で定義し、
  `interfaces` のネスト `fields` 形、`bindings.kind` 別必須 field、`states.*.guard` の
  無接頭辞例外などドキュメントの構造的な抜けを埋めた。ADR-112 でこの決定
  (ADR-103 §5 の「view 別台帳を持たない」を部分的に上書きすること含む)を記録した。
  `spec_version` は `"3.0"` のまま据え置いた(単一 work item 内の一括移行のため)。
- **Verification Results**:
  - `just test-tools` - passed (209 tests)
  - `just typecheck-tools` - passed
  - `just yaml-check-scl` - passed (19 files)
  - `just scl-render` - passed
  - `just verify-ui` - passed
  - `just yaml-check` - passed (216 work items, 323 ids, ARCHITECTURE.md)
  - `just check-ids` - passed
- **Risk Notes フォローアップ**: `oauth2.yaml` `DeviceAuthorization.Status`、
  `system.yaml` `AdminDashboard`、`identity-management.yaml` `AdminRoles` の3 view は
  対応する interface の裏付けが薄いまま `sees` を記述した。人間によるレビューを推奨する
  (特に `AdminRoles` は `ListAdminUsers` interface を再利用しており、ロール管理専用
  interface の欠落を示唆する既存 spec 側の疑問点)。
