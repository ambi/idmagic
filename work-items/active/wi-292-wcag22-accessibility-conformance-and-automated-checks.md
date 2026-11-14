---
status: pending
authors: [tn]
risk: medium
created_at: 2026-07-25
priority: p2
group: frontend-ui
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/requirements/standards.md, requirement: WCAG22-KEYBOARD }
  - { path: docs/requirements/standards.md, requirement: WCAG22-FOCUS }
  - { path: docs/requirements/standards.md, requirement: WCAG22-LABELS-ERRORS }
  - { path: docs/requirements/standards.md, requirement: WCAG22-STATUS }
---

# WCAG 2.2 の検査を、認証画面からアカウントポータルと管理コンソールへ広げる

## 動機

[標準要件](../../docs/requirements/standards.md)は WCAG 2.2 の 4 件（`WCAG22-KEYBOARD`、`WCAG22-FOCUS`、`WCAG22-LABELS-ERRORS`、`WCAG22-STATUS`）を `required` かつ `MUST` と宣言する。
起票時はこれを検証する手段がなかったが、その後、認証画面について検査が入った。
キーボードとフォーカスは実ブラウザーの E2E（`frontend/tests/e2e/authentication-accessibility.spec.ts`）が、ラベルとエラー、状態の通知は単体テスト（`AuthFlowAccessibility.test.tsx`）が観測する。

残るのは二つである。
一つ目に、アカウントポータルと管理コンソールの画面は、要件を参照する個別のフォームのテストを除き、同じ観点で検査されていない。
wi-73600 が扱う手組みのダイアログのように、フォーカスの閉じ込めの欠けた部品が管理コンソールに残っている。
二つ目に、自動の検査器（axe-core など）は入っておらず、検査は人が選んだ操作に限られる。

認証画面の必達は満たしたので、設計の規則への既存の画面の追従として、優先度は p2 とする。

## 対象範囲

- アカウントポータルの主な画面（ホーム、セキュリティ、セッション、同意）で、4 件の要件を同じ形の E2E と単体テストで観測する。
- 管理コンソールは、画面の共通部品（シェル、ナビゲーション、フォーム、ダイアログ、表）で 4 件を観測し、個々の画面は共通部品の利用に任せる。
- 自動の検査器を導入するかを判断する。導入する場合は、検出できる範囲と人が確かめる範囲の境界を[ユーザーインターフェース設計のアクセシビリティ](../../docs/design/application/user-interface.md#アクセシビリティ)に書き、`mise` のタスクから実行する。
- 見つけた違反のうち、共通部品の修正で直るものは直し、画面ごとの修正が要るものは起票する。

## 対象外

- WCAG 2.2 の AAA。
- テナントが設定したブランドの色のコントラストの強制。ブランド配色の検査は助言にとどめる方針を変えない。
- 支援技術の利用者を招いたユーザビリティテストと、第三者による VPAT や ACR の監査。
- メールの本文のアクセシビリティ。

## 検証

- `mise run verify-ui`
- `mise run test-ui-e2e`

## リスク

自動の検査器は WCAG の一部しか検出しない。
検査器が通ったことを適合の主張に使わず、人が確かめる項目を文書に残す。
