---
id: wi-132-frontend-presentation-logic-separation-and-validation-test
title: フロントエンドにおけるプレゼンテーションコンポーネントとロジックの分離および単体テスト追加
created_at: 2026-07-05
authors: [tn]
status: pending
risk: low
---

# Motivation
React フロントエンドの各画面（Page.tsx）がルーター（@tanstack/react-router）や API クライアント、ドラッグ＆ドロップなどの副作用ライブラリと密結合しているため、画面全体としてのユニットテスト（コンポーネントテスト）の記述が困難であった。
テスト容易性を高め、フォームバリデーションや表示ロジックの正しさを高速な単体テストで網羅するために、コンテナ・プレゼンタパターンの適用によるロジックと UI の分離（対策A）およびバリデーションロジックのユーティリティ化（対策B）を行う必要がある。

# Scope
- `ui/src/features/` 配下の密結合な Page.tsx から、ビジネスロジックやフォーム表示・バリデーションを担当する純粋なプレゼンテーションコンポーネントを切り出す。
- フォームの入力バリデーションロジックを純粋な関数として `src/lib/` などのユーティリティ、あるいはカスタム Hooks に切り出す。
- 切り出された Presentation コンポーネント、ユーティリティ、およびカスタム Hooks に対する Vitest/React Testing Library 単体テストを追加する。

# Out of Scope
- Page.tsx コンポーネント自体のテスト（E2Eでカバーされるため）。
- E2E テストシナリオの変更。

# Verification
- `just verify-ui`
- `just test-ui-unit`
- 追加されたテストが正常にパスし、カバレッジが向上することを確認する。

# Risk Notes
既存コンポーネントの構造分割（コンポーネントの切り出し）のみであり、画面の挙動自体は変更しない。ただし、ルーティングや API 呼び出しとの接続が壊れないよう、慎重に結合を確認する必要がある。
