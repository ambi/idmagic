---
status: pending
authors: [tn]
risk: high
created_at: 2026-10-30
priority: p1
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/oauth2/standards.md, requirement: RFC8693-SUBJECT-TOKEN }
  - { path: docs/modules/oauth2/features/token.md, requirement: REQ-OAUTH2-045 }
  - { path: docs/modules/oauth2/features/token.md, requirement: REQ-OAUTH2-048 }
---

# トークン交換で元トークンの鍵所有と再束縛の許可を検証する

## 動機

トークン仕様の実現方式は、subject token の送信者制約を交換後にも引き継ぐと述べる。
静的調査では `ExchangeToken` が元トークンの `SenderConstraint` を照合せず、今回の `ProofJKT` または `ProofX5TS256` だけで新しい `cnf` を作る経路を確認した。
ハンドラーの一般的な DPoP 検証は新しい鍵の所有を示すが、元トークンの鍵を所有することは示さない。
これは未再現の懸念であり、実証済みの脆弱性とは扱わない。

候補の再現経路は、交換を許可されたクライアントが K1 に束縛した token を K1 の所有証明なしに提示し、K2 の proof で再束縛するか、一般 profile で proof を省略して制約を落とす経路である。
grant、scope、resource、存在する場合の `may_act` を満たす正常な入力を対照にして確認する。

## 対象範囲

- `subject_token` と `actor_token` の送信者制約と交換権限を、クライアント認証と区別して検証する。
- 同じ鍵の継承、許可された委譲先の鍵への再束縛、制約の欠落と不一致を区別する。
- 元の所有者の証明を省略できる信頼モデルがある場合は、対象の委譲ポリシーと監査を明示し、暗黙の再束縛にしない。
- DPoP と mTLS、未制約 token、外部 workload token を仕様上区別し、拒否後に token を発行しない。

## 対象外

- 保護リソースでの `ath` 検証の作り直し、鍵管理基盤の新設。
- すべての再委譲に同一鍵を永久に強制する独自規則。
- [[wi-378-user-visible-agent-delegation-grants]] の本人による委譲付与と撤回、外部 AS との ID-JAG。

## 設計

再現を先に行い、仕様の「引き継ぎ」が意味する保証と実装の差を確定する。
RFC 8693 は委譲先の新しい鍵への再束縛自体を一律に禁じないため、仕様の根拠を確認せず token endpoint に保護資源用の `ath` を要求しない。
新しい要件の候補は、元 token の利用権限と鍵所有の条件を満たさない交換が、制約の除去または無許可の再束縛によって成功しないことである。

主要型は既存 `SenderConstraint` と、検証済みの交換証明を表す最小の入力とする。
計算の候補は `ValidateExchangeBinding(subjectConstraint, actorConstraint, verifiedProof, delegationPolicy)` であり、許可する出力制約か拒否理由を返す。
署名、証明の replay 記憶、現在のポリシー取得、発行と監査は明示した作用としてユースケースが編成する。
一回だけの検証結果を分岐で使い回し、グラントごとに弱い別実装を持たない。

境界は OAuth2 内のままを第一候補とし、外部の PEP API への依存を増やさない。
新しい公開ポートが必要になれば D1/D3 に従い、既存の認証入力の拡張と専用ポートを比較する。
具体的な証明方式、同一鍵と委譲先鍵の条件、制約の異種変換、外部 attestation への適用は、実装前に標準と現実の利用側を確認して確定する。

移行では既存の正当な交換を対照にし、証明が必要なクライアントへ変更を告知する。
ロールバックが必要な場合は問題の交換経路を停止し、所有証明を省略する緩和へ戻さない。

## 計画

1. 正式な token endpoint で K1/K2、proof なし、mTLS、actor token の経路を再現する。
2. 再現結果から要件差分と証明方式を確定し、実現方式の曖昧な「引き継ぎ」を整理する。
3. 拒否と正常な再束縛を同じ判定へ接続し、発行と監査の作用を検証する。

## タスク

- [ ] T001 [Diagnosis] 未再現の経路を受け入れ境界で確認し、既存テストが検出していない故障を特定する。
- [ ] T002 [Spec] 元 token の利用条件、再束縛、制約を保持する意味、失敗時の無作用を規範化する。
- [ ] T003 [Acceptance] 不正な鍵変更と制約除去を拒否する RED と、正当な交換の対照を作る。
- [ ] T004 [App] 証明入力の検証と出力制約の決定を接続し、拒否後に発行しないようにする。
- [ ] T005 [Verify] DPoP、mTLS、actor token、証明 replay、並行要求、監査への鍵素材の非露出を検証する。

## 検証

- `mise run check-spec`：規範と実現方式の整合を確認する。
- `mise run test-go-package -- ./oauth2/...`：K1 所有、K2 不一致、proof 欠落、許可された再束縛を検証する。
- `mise run test-go-package -- ./oauth2/internal/handlers_http`：正式な HTTP 配線で拒否後の発行件数がゼロであることと、正常対照の `cnf` を確認する。
- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>` とする。HTTP を使う検証はサンドボックス外で実行する。
- 必要な規範 ID と予定した主要ユースケースは実装前に登録し、同一鍵の比較を外した故障がテストで検出されることを確かめる。

## リスク

無許可の再束縛は送信者制約を無意味にする一方、一律の同一鍵強制は正当な委譲先への交換を壊す。
所有証明と委譲許可を区別し、標準と対象ユースケースから条件を定める。
調査段階の静的懸念を前提に実装を固定せず、最初の再現と規範の確認で判断する。
