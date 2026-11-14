---
status: cancelled
---

# 破壊的な操作の確認を `window.confirm` から画面のダイアログへ移す

中止した。
2026-11-14 の棚卸しで、`frontend/src` に `window.confirm` の呼び出しは残っていなかった。
5 か所は、テナント画面の分割（c482e706c）などの変更の中で、共通の確認ダイアログ `ConfirmActionDialog` へ置き換わっている。
手組みのダイアログを共通部品へ揃える残りの作業は wi-73600 が扱う。
