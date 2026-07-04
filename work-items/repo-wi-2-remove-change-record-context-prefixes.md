---
id: repo-wi-2-remove-change-record-context-prefixes
title: "Work Item と ADR のコンテキストプレフィクスを廃止する"
created_at: 2026-07-04
authors: [tn]
status: pending
risk: medium
---

# Motivation
このリポジトリでは work item と ADR の実質的な所有境界が単一であり、`repo-` と `idp-` を分けても利用者に有益な判断材料を与えていない。
意味の弱いプレフィクスを維持すると、採番、検索、説明、起票時の判断が複雑になり、変更記録の運用コストが上がる。
単一の ID 名前空間に揃えることで、変更記録をより単純に扱えるようにする。

# Scope
- Work Item と ADR の命名規則を、コンテキストプレフィクスなしの単一名前空間へ変更する方針を決める。
- 既存の `idp-wi-*` / `repo-wi-*` と `idp-ADR-*` の扱いを決める。
  - 既存 ID を履歴互換のため据え置き、新規のみ新形式にするか。
  - 既存 ID とファイル名も一括移行するか。
- 変更する場合は、RA/SCL の変更記録フォーマット、検証ツール、関連ドキュメント、起票スキルの前提を更新する。
- リポジトリ内の既存参照が壊れないように、参照更新または互換方針を明記する。
- `spec/scl.yaml` の仕様意味は変更しない想定。プロダクト機能・振る舞いに影響する場合のみ SCL-first で更新する。

# Out of Scope
- プロダクト機能の追加や変更。
- Git 履歴の書き換え。
- RA/SCL の変更記録以外の ID 体系の再設計。

# Verification
- `just yaml-check-work-items`
- `just check-ids`
- 変更後に該当する場合は、RA/SCL ツールの単体テストまたは `just yaml-check`
- 既存参照を更新する場合は、`rg 'idp-wi-|repo-wi-|idp-ADR-|repo-ADR-'` で残存参照を確認する。

# Risk Notes
既存 ID を一括変更すると、過去の work item、ADR、コミットメッセージ、ドキュメント内参照が壊れる可能性がある。
履歴互換を優先して既存 ID を据え置く場合は、新旧形式が併存する移行期間が発生するため、検証ツールと起票手順が両形式を正しく扱う必要がある。
