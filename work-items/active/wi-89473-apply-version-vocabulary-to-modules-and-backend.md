---
status: pending
authors: ["tn"]
risk: low
reversibility: reversible
created_at: 2026-10-31
priority: p3
depends_on: []
change_kind: docs
affected_spec:
  - { path: docs/modules/identity-management/features/dynamic-group.md, requirement: REQ-IDMANAGEMENT-023 }
  - { path: docs/modules/identity-management/features/dynamic-group.md, requirement: REQ-IDMANAGEMENT-066 }
  - { path: docs/modules/identity-management/features/dynamic-group.md, requirement: REQ-IDMANAGEMENT-068 }
  - { path: docs/modules/identity-management/features/dynamic-group.md, requirement: REQ-IDMANAGEMENT-069 }
  - { path: docs/modules/system/features/startup-configuration.md, requirement: REQ-SYSTEM-016 }
  - { path: docs/modules/system/features/startup-configuration.md, requirement: REQ-SYSTEM-017 }
  - { path: docs/modules/tenancy/features/branding.md, requirement: REQ-TENANCY-034 }
---

# version の表記の検査を docs/modules と backend へ広げる

## 動機
[文書の表記](../../docs/formats/documentation-guide.md)は version を「版」と書かないと定める。
`tools/check/repository/documentation.repository.ts` の検査は `docs/modules/` と `backend/` を「別の作業項目が扱う」として除いているが、担当する記録がなく、両方に「版」が残っている。
検査がないので、新しい記述でも同じ表記が増える。

## 対象範囲
- `docs/modules/` と `backend/` の version を指す「版」を「バージョン」へ置き換える。
- `tools/check/repository/documentation.repository.ts` の検査対象へ両方を加える。

## 対象外
- コメント以外のコードの変更。
- version 以外の語彙の検査。

## 設計
「版」には version でないものがあるので、一括で置き換えず、文脈を読んで分類する。

| 表記 | 意味 | 扱い |
| --- | --- | --- |
| 規則の版、鍵の版、モデルの版 | version | 「バージョン」へ置き換える |
| `PostgreSQL 版`、`メモリ版`、`memory 版` | アダプターの実装の別 | 維持する |
| 第 N 版 | revision | 維持する。既存の検査の除外に合わせる |

検査は実装の別を指す表記を語形で除外する。
除外の規則は検査のテストの近くに理由と一緒に書く。

`affected_spec` は起票時に「版」を含む要件である。
見出しや例の文言だけが変わる要件は `mise run spec-diff` で確かめて追記する。

## 計画
1. 残りの箇所を分類して置き換える。
2. 検査の対象を広げ、除外の規則を足す。

## タスク
- [ ] T001 [Docs] `docs/modules/` を置き換える。
- [ ] T002 [Docs] `backend/` のコメントを置き換える。
- [ ] T003 [Check] 検査の対象を広げ、実際の「版」を一時的に残して失敗することを確かめる。

## 検証
- `mise run check -- main`
  - reason: 語彙の検査と、要件の文言の変更と宣言の一致を確かめるため。
- `mise run lint-go`、`mise run test-go-changed`
  - reason: Go のコメントの変更で整形とディレクティブが壊れないことを確かめるため。

## リスク
要件の見出しのアンカーが変わるので、見出しへのリンクは `mise run check-links` で確かめる。
実装の別を指す表記を誤って置き換えると、意味が変わる。
