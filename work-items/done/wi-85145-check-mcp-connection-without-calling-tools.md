---
status: completed
---

# 業務ツールを実行せずに MCP の接続を確認し、不足する設定を案内する

`docs/modules/oauth2/features/admin-access.md` に REQ-OAUTH2-094 を追加し、管理コンソールが Agent の権限の材料で MCP 接続の確認項目と不足の案内を示すことを定めた。HTTP 契約は変えていない。

確認は管理コンソールが既存の管理 API の値だけで行い、新しい複合 API は作らなかった。判定は `frontend/src/features/admin-agents/mcpConnectionCheck.ts` の純粋関数に置き、権限の材料の画面が接続先ごとに登録、割当、トークンの上限、一覧の取得の四項目を示す。カタログは経路の loader が接続先ごとに既存の tool-catalog API で集め、取得の失敗は未到着と区別する。

IdMagic から MCP サーバーへ届く確認（直接の一覧取得、認証）は、現在の REQ-OAUTH2-094 の判断で置かないとした。公開された MCP サーバーへの取得の経路を作る wi-38469 の対象範囲へ、これらの項目の追加と判断の改訂を移した。業務ツールの実行は行わず、確認は保存値の読み取りだけである。

検証は `verify-changed` で check、lint-repo、verify-ui が通った。Go、tools、audit は差分が届かず対象外。主要ユースケースは `AdminAgentAccessPage.test.tsx` の画面テストと `mcpConnectionCheck.test.ts` の単体テストで固定した。
