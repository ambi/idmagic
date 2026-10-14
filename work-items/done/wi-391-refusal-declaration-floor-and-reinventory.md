---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# 状態を変える操作が返す拒否をシナリオに必ず宣言させ、検出漏れしていた拒否を洗い直す

「宣言された拒否にテストがあるか」しか問えなかった検査に、**何を宣言すべきかを問う規則 (R4)** を足した。契約が状態を変える操作へ 403 を宣言しているなら、その拒否がいつ起きるのかをシナリオが言わなければならない。突き合わせの鍵は両側にすでに書かれているエラー型なので、シナリオの書式は変えていない。
意味の差は 4 つある。
第一に、**誰がその操作を実行できないかが、6 コンテキストで初めて規範になった**。R4 は導入時点で `authorization`・`identity-governance`・`sharedsignals`・`workloadidentity` の `AccessDeniedError`、`authentication` の `MfaEnrollmentNotAllowedError`、`oauth2` の `OAuthAccessDeniedError` を落とした。いずれも実装は拒否していたが、規範のどこにも書かれていなかった。5 件を新しいシナリオとして書き起こし、1 件 (REQ-AUTHENTICATION-018) は既存のシナリオが「登録 API も利用できない」とだけ書いていたのでエラー型を明示した。落ちる件数が 6 件で収まったので、**許可リストは作っていない**。R4 は例外を持たない。
第二に、**その要求が現存の欠陥を 1 件暴いた**。宣言にはテストが要り、拒否のテストは「触らなかったもの」を assert する。そう書いた `authorization` のテストが落ち、非管理者による `PutAuthorizationModel` が 403 を返しながらモデルの版を作っていることが分かった。原因は `WriteAdminAccessError` が応答を書いた結果 (`nil`) を返していたことで、その戻り値を返す `requireAuthorizationAdmin` / `requireWorkflowAdmin` の 2 つのヘルパーが拒否を呼び出し元へ伝えていなかった。`identity-governance` では 403 の本文に続けてワークフロー一覧がそのまま書き出されていた。`ErrAdminAccessRefused` (`ErrResponseWritten` を包む) を返させて塞いだ。**wi-390 が塞いだ形が、writer を 1 段挟んだところに残っていた。**
第三に、**最低要件の粒度を測ったうえで選んだ**。コンテキスト単位で拒否を 1 つ要求する案は、今日どこも落ちないので採らなかった。操作単位で要求する案は本来欲しい保証だが、403 を宣言する変更操作 177 件のうちシナリオが操作名に言及しているのは 37 件で、残り 140 件を許可リストに固定してから 1 件ずつ剥がすことになる。エラー型単位を採り、参照系の拒否で変更操作の義務が満たせてしまう限界を Design に残した。`signing-keys` がその実例なので、R4 の判定とは別に鍵の回転と無効化の拒否 (REQ-SIGNINGKEYS-011) を補った。
第四に、**137 件の負債が「読む順番」を持つようになった**。`mise run report-refusal-debt` が、拒否を assert するテストの候補から named / nearby / none に三分する。分類そのものは機械に決めさせない。着手時点は named 65・nearby 68・none 3 で、`signing-keys` の 5 件を確認したところ全て「テストは実在し注記だけが無い」だったので注記を付け、許可リストは 137 → 131 に縮んだ。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
