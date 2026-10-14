---
status: completed
archived_from: bb8b8175200d963c2b68946eff1c6aa4d4a57935
---

# 設計文書の用語を定着した外来語へ揃え、表記ゆれをなくす

設計文書の用語が、指す英語の概念へ一対一で戻せるようになった。
deployment、runtime、platform、secret、capacity、observability、admission control、guidelines、load shedding order、sizing formula、reference workload profile の各概念が、それぞれ「デプロイ／デプロイメント」「ランタイム」「プラットフォーム」「シークレット」「キャパシティ」「オブザーバビリティ」「アドミッションコントロール」「ガイドライン」「ロードシェディング順序」「サイジング計算式」「リファレンスワークロードプロファイル」という一つの表記を持つ。
「秘密鍵」「秘密情報」「保存容量」「容量超過」「実行時に」のように別概念を指す共起は残っており、規則表がその理由を持つ。
意味の復元できなかった「訓練」は箇所ごとに確定した。`docs/design/architecture/deployment.md` の「ローカル開発と訓練」は「ローカル開発と復旧ドリル」になり、これは `infra/backup/restore-drill.sh` が `infra/docker/docker-compose.dev.yaml` を使い捨てプロジェクトとして起動している事実に基づく。
題名が「ガイドライン」になった 2 文書はファイル名も `api-guidelines.md`、`design-guidelines.md` へ改めた。`rules` と `guidelines` の二つの英語が同じ文書を指す状態を残さないためである。
`mise run spec-diff` は REQ-SIGNINGKEYS-012 を挙げる。EX-SIGNINGKEYS-012-01 の Given 1 行が「配備の `PERSISTENCE`」から「デプロイ先の `PERSISTENCE`」になったためで、id、条件、要求する結果はいずれも同じである。
再発は `mise run check-terminology` が拒否する。免除は `tools/check/src/terminology.ts` の規則表が持つ理由付きの共起だけで、ファイル単位の免除は無い。

全文は `git show bb8b8175200d963c2b68946eff1c6aa4d4a57935:<このファイルのパス>` で読む。
