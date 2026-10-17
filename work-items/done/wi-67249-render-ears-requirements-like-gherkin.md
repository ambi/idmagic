---
status: completed
---

# 生成サイトで EARS の要件文を、受け入れ例の Gherkin と同じように構文で色分けして表示する

`mise run spec-diff` は main に対して規範仕様の変更なしと報告する。
生成サイトの機能仕様のページで、要件文を EARS の区切りの語で節に分け、受け入れ例の Gherkin と同じ見た目の名札（Where、While、When、If、Then）を付けて節ごとに表示する。
主体と区切りの語は表示から外し、拒否の節（If）は色で見分けられるようにした。原稿の書き方と検査は変えていない。
生成サイトの英文の検査は、名札を文の一部として読まない。

全文は `git log -1 -- work-items/active/wi-67249-render-ears-requirements-like-gherkin.md` が示すコミットの親、またはそのコミットで読む。
