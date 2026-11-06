---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-11-07
priority: p2
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/requirements/standards.md, requirement: WCAG22-LABELS-ERRORS }
  - { path: docs/modules/system/features/http-api.md, requirement: REQ-SYSTEM-028 }
---

# Zod でフォームの変換、検証、エラー表示を共通化する

## 動機

[ライブラリ選定の見直し](../done/wi-90339-review-library-adoption.md)で確認した画面に、次の不足がある。

| 箇所 | 振る舞いと影響 |
| --- | --- |
| ユーザー作成 | ユーザー名が空白だけの場合、HTML の required を通った後、submit 内の trim で空と判定して説明なく return する |
| アプリケーション作成 | trim 後の空判定で送信を無効化するが、入力欄に `aria-invalid` を付けていない。編集画面には付いている |
| サインインポリシーの CIDR | `parseNetworkCIDRs` は行分割、trim、空行除去だけで、不正な CIDR を画面で検証しない |
| 文字列長 | HTML の `maxLength` は UTF-16 単位で数え、サーバーはコードポイントで数える（REQ-SYSTEM-028）。絵文字などで入力側が先に制限する |

フォームは 61 個あり、`.trim()` が 203 回散らばり、`aria-invalid` を付けるのは 9 ファイルだけである。
不足はこの散らばりから生じている。
フロントエンドは文字列の状態を API のリクエストへ変換し、型の付いた結果とフィールドごとのエラーを同時に作る必要がある。
TypeScript は実行時に型を持たないため、一つの宣言から変換、検査、エラー、型推論を得られるスキーマライブラリを使う。

## 対象範囲

- Zod（通常の API）を runtime の依存に追加する。
- フォームの値の変換、検証、フィールドごとのエラー、`aria-invalid`、送信の無効化、エラー表示を共通の部品にする。
- 上の表の不足を直し、残りのフォームを順に移す。

## 対象外

- API の応答の形の実行時検証。応答は同じ TypeSpec から作るサーバーが返し、ずれは契約検査で防ぐ。
- バックエンドの検証の変更。

## 設計

### ライブラリの選択

Zod、Zod Mini、Valibot を比べた。
三つとも runtime の依存を持たない。
Zod Mini は Zod と同じパッケージで、利点はバンドルの小ささだけなので、公式の推奨に従い通常の API を使う。

| 観点 | Zod 4 | Valibot 1.5 |
| --- | --- | --- |
| 過去の脆弱性 | メールアドレスの正規表現の ReDoS（CVE-2023-4316） | 絵文字の正規表現の ReDoS（CVE-2025-66020） |
| 実行時のコード生成 | object の解析を `new Function` で生成する。`jitless` で止められる | なし |
| 利用者の規模 | はるかに大きい | 小さい |
| 公開元 | 開発ツール（TanStack の生成ツール、knip、shadcn CLI）を通じて既に信頼している | 新しく増える |

差は小さく、次の順で決めた。
新しく信頼する相手を増やさない。
ランタイムのコードは多くの目で検査されているほうを選ぶ。
既に保守している依存に寄せる。
開発用にあることは、ランタイムの脆弱性の影響が別であるため、主な理由にはしない。
Anthropic と OpenAI の SDK はどれも Zod に合わせているが、利用者のスキーマを受け取る互換性のためであり、補強にとどめる。

### 採用の条件

- SPA の起動時に `z.config({ jitless: true })` を設定する。
  このアプリの CSP は `unsafe-eval` を許さず、生成は拒否されるが、判定のための呼び出しが CSP 違反として報告されるためである。
- 正規表現を使う組み込みの形式検証（email、emoji など）は原則として使わない。
  使う場合は理由を書く。
  CIDR とコードポイント数は `refine` に自前の関数を渡して検査する。
- 版を固定し、開発ツールが使う Zod と同じ版にそろえる。
- スキーマは `frontend/src/lib/` の共通の場所に置き、画面から直接 `zod` を import しない。
  両者は Standard Schema に対応するため、将来の移行を小さくできる。

`aria-invalid` とエラーの文章は [WCAG22-LABELS-ERRORS](../../docs/requirements/standards.md) と [UI 設計の検証の分担](../../docs/design/application/user-interface.md#検証の分担)に従う。
自動のアクセシビリティ検査の導入は [wi-292](wi-292-wcag22-accessibility-conformance-and-automated-checks.md) が扱う。

## 計画

1. 主要ユースケースの証拠を計画する。
2. 共通の部品と `jitless` の設定を作り、上の表の 4 件を RED から直す。
3. 残りのフォームを機能ごとに移す。

## タスク

- [ ] T001 [Acceptance] 4 件の不足を観測する境界で RED を確かめる。
- [ ] T002 [UI] Zod の設定と共通の部品を作る。
- [ ] T003 [UI] 4 件を直す。
- [ ] T004 [UI] 残りのフォームを移す。
- [ ] T005 [Verify] CSP 違反が報告されないことと、変更を検証する。

## 検証

- `mise run test-ui-unit` で変換と検証の関数を確かめる。
- `mise run test-ui-e2e` で送信の無効化、`aria-invalid`、エラー表示を確かめる。
- `mise run audit-dependencies` と `mise run check-ui-dependencies` で依存を確かめる。

## リスク

Zod が runtime に入り、脆弱性がブラウザーで効くようになる。
入力の大半は管理者自身が打ち込んだ値だが、編集画面はサーバーの既存の値も検証する。
組み込みの正規表現を避け、版を固定して監査する。
