---
status: completed
archived_from: 178a3e4607e66b875ccf0b9ea3f9a104b0e279ae
---

# Application が宣言する具体例 32 件にテストを対応付け、被覆台帳から外す

`mise run spec-diff` は `main` に対して `no normative specification change` を返した。
Application が宣言する具体例のうち台帳に残っていた 30 件を実装と照合し、実装と一致する 25 件を `//spec:covers` を持つテストへ結び付けて被覆台帳から外した。
既存テストへのディレクティブ追加だけで済んだのは 4 件で、残りは SAML 詳細の SSO URL と SLO URL、作成応答が運ぶ平文シークレット、上書きを持たない隣の Application に適用され続けるデフォルトポリシー、MFA 未登録人数の表示といった観測の補強か、`backend/shared/http/testing_stack` と Application の管理 API 上に書いた新しいテストを要した。
実装と一致しない 5 件は [[wi-625-application-assignment-accepts-any-subject]]、[[wi-626-align-cross-tenant-application-read-refusals]]、[[wi-628-implement-desired-state-application-assignment]] に切り出し、台帳へ `blocked_by` と `finding` を残した。
readiness pass の途中で `EX-APPLICATION-010-01` を実装なしと誤って測ったが、`unenrolled_user_count` が管理 API から `AdminSignInPolicyPage` まで通っていることを確認して消化側へ戻し、いったん起票した切り出し先を取り下げた。

全文は `git show 178a3e4607e66b875ccf0b9ea3f9a104b0e279ae:<このファイルのパス>` で読む。
