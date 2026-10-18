---
status: completed
---

# パスワードの既定値をテナント設定と同じ公開パッケージから共有する

`spec-diff` は規範仕様の差分なしを示した。
パスワードの既定値4個を Tenancy の domain に集め、Authentication の既存の定数名をその参照として保った。
認証の判定、設定 API の応答と上書きの拒否、永続状態とイベントは変えていない。
Tenancy から Authentication への参照をなくし、循環依存の負債を53件から52件に減らした。
固定値のための注入ポートと新しい共有ライブラリは増やさず、既存テストも変更していない。
ほかの循環と属性スキーマの所有は対象外であり、wi-21934 に残る。

全文は `git log -1 -- work-items/active/wi-61185-share-password-defaults-through-tenancy.md` が示すコミットの親、またはそのコミットで読む。
