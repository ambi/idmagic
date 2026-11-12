---
status: pending
authors: [tn]
risk: medium
created_at: 2026-11-06
priority: p1
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/authorization/features/model.md, requirement: REQ-AUTHORIZATION-011 }
---

# 認可モデル未作成時のテンプレート公開試験の失敗を解消する

## 動機

`AdminAuthorizationModelEditPage` の `publishes the document viewer template when no model exists yet` は、確認操作の後に `Publish` ボタンを取得できず失敗する。
[[wi-32030-separate-openldap-tests-from-default-go-verification]] の影響検査で見つかり、対象ファイルだけの実行でも再現する。
同項目はフロントエンドを変更していない。

## 対象範囲

- モデル未作成時にテンプレートを選び、差分を確認して公開する経路と、試験の非同期処理の待機を照合する。
- 画面の振る舞いと試験のどちらに原因があるかを特定し、確認と公開の証拠を保って修正する。

## 対象外

- OpenLDAP 試験の実行区分。
- 原因を確かめずに試験をスキップすることと、公開の表明を削ること。

## 設計

既存の `REQ-AUTHORIZATION-011` を判断の起点にする。
確認画面の到達、最新モデルが 404 の場合の扱い、公開可能な状態を観測し、画面の不具合か待機不足かを切り分けてから修正範囲を決める。
待機時間の固定値を増やす案は、待つ条件を表現できないため採らない。

### 2026-11-13 の調査

- 対象ファイルだけの実行は 5 回続けて成功し、通常の実行では再現しない。
- 試験の `fetch` の差し替えで、最新モデルの読み直し（404 の応答）を 20 ms 遅らせると、毎回 `Unable to find an accessible element with the role "button" and name "Publish"` で失敗する。
- 試験は確認の操作の後に `findByText(/document/)` で待つ。テンプレートを選んだ時点で編集欄に `document` が現れるので、この待機は確認の表示を待たずに戻る。読み直しが先に終わったときだけ確認の表示と `Publish` が現れる。
- 同じファイルの別の試験は、同じ形の競合を b79cc061f（確認の見出しの role を待つ形への変更）で直している。この試験には同じ修正が入っていない。

画面は最新を読み直してから差分を示すので（REQ-AUTHORIZATION-011）、`Publish` が読み直しの後に現れるのは仕様どおりである。
したがって原因は試験の待機の条件である見込みが高い。
修正は、確認の表示にだけ現れる要素（見出しの role）を待つ形にし、POST の内容の表明を保つ。

## 計画

1. 20 ms の遅延で失敗を再現し、修正後は遅延があっても成功することを確かめる。遅延は確認の手段であり、試験に残さない。
2. 同じファイルのほかの試験に、確認の表示より先に現れる文字列を待つ箇所が残っていないかを調べる。
3. 対象ファイルと UI 検証を通す。

## タスク

- [ ] T001 [Acceptance] 遅延の注入で失敗を再現し、画面が読み直しの後に `Publish` を示すことを確かめる。
- [ ] T002 [App] 試験の待機の条件を、確認の表示に固有の要素へ直す。
- [ ] T003 [Verify] 遅延を注入した状態と通常の状態の両方で対象試験を通し、UI 検証を通す。

## 検証

- `mise run test-ui-unit-file -- src/features/admin-authorization/AdminAuthorizationModelEditPage.test.tsx`：単独での再現と修正の確認。
- `mise run verify-ui`：画面と試験への影響の確認。

## リスク

待機不足だと決めつけて表明を弱めると、公開できない画面の不具合を隠す。
期待する画面状態と POST の内容を保って原因を判定する。
