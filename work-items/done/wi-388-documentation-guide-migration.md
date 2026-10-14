---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# 仕様の正本を種類ごとのファイルへ分割し、DOCUMENTATION_GUIDE の構成へ移行する

仕様の正本を、1 Context 1 ファイルから種類ごとのファイルへ移した。規範的な内容は動かしていない。`just spec-diff` は全工程を通して規範的差分を報告せず、シナリオ、Standards の行、遷移の行、TypeSpec の宣言はいずれも移行前と同一である。
6086 行の `SPECIFICATION.md` 22 個が、`README.md` / `glossary.md` / `standards.md` / `states.md` / `decisions.md` / `internals.md` / `scenarios.feature.md`（Context）と `README.md` / `structure.md` / `api-rules.md` / `observability.md` / `deployment.md` / `persistence.md` / `authorization.md`（ルート）になった。`Design` は寿命で分けた。理由を持つ判断は `decisions.md` の一覧に、コードから復元できない機構の説明は `internals.md` の散文になり、`Internal Interfaces` や `Design Decisions` のような観点名の見出しは消えた。
仕様が得たものが 3 つある。1 つ目は状態の表である。全 22 の状態機械が `| State | Kind | Meaning |` を持ち、初期状態がちょうど 1 つであること、遷移表の `From` と `To` が表に現れることを機械検査する。従来の `Initial: X Terminal: Y` の 1 行では、状態の集合も各状態の意味も書けなかった。2 つ目は `docs/authorization.md` である。主体の種類、スコープの名前空間、対話セッション限定の規則とその 2 つの理由、テナント境界を 1 か所に集約した。従来は 21 Context の `Authorization boundary` に同じ規則が散っていた。3 つ目は通知テンプレートのカタログの移動である。`Database design policy` の下にあったが、これは通知機能の製品仕様であって永続化の方針ではないので、`NotificationTemplate` を所有する `Tenancy` の `internals.md` へ移した。
ツール側では、`documentKind` がリポジトリ相対パスから適用する文法を決め、`validateDocument` がそれで分岐する。移行中は両構成を受け入れ、完了後に旧構成の受け入れを外した。`spec-diff` は状態機械を、それを載せるファイルではなく所有する Context で識別するようになったので、機械がファイル間を移動しても遷移の変更として報告しない。表のセルはエスケープされた `|` を保持する。`UserLifecycle` の purge ガードが CEL の論理和を `\|\|` と書いており、これを分割していた既存の不具合が状態の表の検査で表面化した。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
