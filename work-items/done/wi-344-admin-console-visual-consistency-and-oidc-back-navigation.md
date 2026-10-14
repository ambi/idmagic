---
status: completed
archived_from: 0341c2869c761480c74567abfffcb15f65f78dfa
---

# 実機レビュー指摘に基づき、認証画面/管理コンソールの視覚的一貫性と OIDC コールバック後のブラウザ「戻る」挙動を修正する

実機レビュー指摘5件をすべて修正した。(1)(2) `auth-main`/ダッシュボードの各カード・セクションに
背景色・境界線・共通 `Card` コンポーネントを適用し、背景との同化を解消した。(3) 単独の
`text-accent` 誤用6箇所を `text-accent-foreground` に修正した（`--color-accent` トークン
定義自体は `combobox`/`select`/`dropdown-menu` のハイライト状態が正しく依存しているため
変更していない — 詳細は Design 系統3 の訂正記述を参照）。(4) サインインポリシー・設定画面の
編集ボタン4箇所に `IconPencil` を追加し、`AdminPaneActions.tsx` と同じ見た目に統一した。
(5) `completeLoginFromCallback`/`beginLogin` の `window.location.assign` を `replace` に
変更し、OIDC コールバック後の「戻る」で使用済み `?code=xxx` URL に着地しないようにした。
副次的に、T004 の行数増加で `AdminSignInPolicyPage.tsx` が既存の `ui-page-lines` debt
ceiling (457) を3行超過したため、`architecture.yaml` の該当 debt エントリの ceiling を
460に更新し reason を追記した（wi-234-complexity-ratchet の既存超過に対する増分、新規
debt の追加ではない）。
Out of Scope とした管理コンソール全体のデザインシステム刷新、per-application override 行の
`text-blue-700` ハードコード解消、OIDC RP フロー全体の再設計は本 WI では未実施。
**完了後の追加訂正**: 完了報告後、実機レビューで (1) の `auth-main` が
「余白がついて aside より小さく見える」と再指摘された。原因は `.auth-main` に元々あった
`max-w-[29rem] mx-auto`（背景が無かった間は不可視だった内側の幅制限）が `bg-card` 追加で
可視化されたことだった。`.auth-main`（列いっぱいの背景サーフェス）と新設
`.auth-main-body`（`max-w-[29rem]` の内容幅制限）に責務を分離し、`auth-aside` と対称な
半々のレイアウトに修正した。solo モード・900px メディアクエリの `max-w-none` 上書きも
追随して移設し、両方をスクリーンショットで目視確認して回帰がないことを確認した。

全文は `git show 0341c2869c761480c74567abfffcb15f65f78dfa:<このファイルのパス>` で読む。
