---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# 製品横断の標準 7 行にテストを対応付け、所有するパッケージを決めて台帳から外す

`mise run spec-diff` は `no normative specification change against main` を返す。規範の変更は無い。
差分は `docs/domain/standards.md` の 8 行のうち、名指しを持たなかった 7 行に対する被覆の状態である。
7 行すべてがその行の `Statement` を区別できる入力と観測を持つテストを得て
`tools/check/standards-coverage-debt.json` から消え、台帳は 104 件から 97 件になった。
新設したテストは Go 4 件、フロントエンドの単体 4 件、ブラウザー E2E 2 件の計 10 件で、
製品コードは 1 行も変わっていない。
この文書だけの落とし穴だった「担い手が複数ある行」は `GDPR-ERASURE` の 1 行だけで、
IdManagement と Authentication の 2 つのパッケージに注記と観測を置いた。
**欠陥を 2 件見つけ、どちらも切り出した。** Purge の cascade が WebAuthn 資格情報と
リカバリコードを消していないこと（[[wi-513-purge-leaves-webauthn-credentials-and-recovery-codes]]）と、
監査記録の保持について `docs/domain/audit/decisions.md` の決定（7 年、削除の入口なし）と
実装（365/90/30 日、`DeleteOlderThan` を一括処理から公開）が食い違っていること
（[[wi-514-audit-retention-decision-and-sweep-disagree]]）である。**どちらも行そのものは
満たしたままなので、7 行の消化は成立している。** 前者は `GDPR-ERASURE` が名指す資格情報の
一部が消え残るという不足で、行が言う「消去する」に穴があるという意味では欠陥だが、
対応付けたテストは実際に消えている 5 種を観測しており、名指しは実在の検証に付いている。
後者は行が言う「定義済みの期間」自体は成立しており、食い違うのはその期間が Audit Context の
決定と一致しない点である。
**測ったことが 2 つある。** 1 つは E2E ハーネスのキー送出で、名前付きキーは編集コマンドへ
写像されるため走査も活性化も起きず、生の文字だけが実キーになる（Design の該当節）。
もう 1 つはフォーカス表示の算出で、`transition` の途中で読むと透明なリングが返る。
後者は最初の測定で「サインインボタンにフォーカス表示が無い」という誤った結論を出しかけた。
**共有サーバーの状態も 1 度踏んだ。** キーボードの試験が最初に同意を「許可」で終えたため、
同じ利用者とクライアントの同意レコードが残り、あとに走る `authorize-golden-path.spec.ts` から
同意の段が消えて落ちた。E2E は 1 実行 1 サーバーで、`tests/e2e/setup.ts` が「各 spec は自分が
操作する対象を自分で用意し、共有フィクスチャを書き換えないこと」と書いているとおりである。
「拒否」で終えるよう変えた。拒否は `AuthorizationRequest` を `Rejected` にするだけで同意
レコードを残さない。許可の側は走査で届くところまでを観測に残している。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
