---
status: completed
archived_from: bf5989cb816f22088f834db8f925588565bc90bd
---

# 共有シェルを保護しフロントエンド Lines カバレッジ40%を達成する

`AdminShell`・`SystemShell`・`AdminPaneActions` にコンポーネントテストを新設し、`StepUpDialog` に passkey (webauthn) 経路のテストを追加した。ナビゲーションの active 状態、パンくずの一段/二段表示、アカウントメニューの開閉とログアウト、権限・状態表示（busy / disabled / danger tone）、確認・キャンセルを公開 UI から検証する。特に `SystemShell.tsx` は本 work item 以前は 0% カバレッジだった。

全文は `git show bf5989cb816f22088f834db8f925588565bc90bd:<このファイルのパス>` で読む。
