---
status: pending
authors: [tn]
risk: high
created_at: 2026-10-30
priority: p2
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/oauth2/standards.md, requirement: CIMD00-URL-SHAPE }
  - { path: docs/modules/oauth2/standards.md, requirement: CIMD00-FETCH }
  - { path: docs/modules/oauth2/standards.md, requirement: CIMD00-STRUCTURE }
  - { path: docs/modules/oauth2/standards.md, requirement: CIMD00-CACHE }
---

# CIMD の取得と入力検証を draft 02 の必須条件に合わせる

## 動機

MCP の対象改訂は [[wi-322-mcp-authorization-spec-repin-2026-07-audit]] で 2026-07-28 に更新済みだが、CIMD は draft-00 を参照する。
[draft-02](https://www.ietf.org/archive/id/draft-ietf-oauth-client-id-metadata-document-02.html) の3節は単独の `.` と `..` のパス要素を禁止し、5節は認可サーバーによる HTTP リダイレクトの自動追従を禁止する。
現行の URL 判定に dot segment の拒否はなく、CIMD の HTTP クライアントは `MaxRedirects: 3` を使う。
200 OK 限定、userinfo と fragment の拒否、取得 URL と文書内 `client_id` の完全一致は既に実装されており、同じ機能を追加し直さない。

## 対象範囲

- draft-02 を固定した差分表を作り、義務の主語、MUST、SHOULD、MAY と現在の採用を分ける。
- single-dot/double-dot のパス要素と、HTTP リダイレクトの自動追従を拒否する。
- shared secret と private key metadata の扱いを4.1節と照合する。未知フィールドを消す前に、禁止された既知の項目が検証から消えないかを確認する。
- エラー文書をキャッシュしない MUST、HTTP cache header に従う SHOULD と、metadata 変更時の同意と信頼の扱いを区別して現在の方針を明示する。
- -00 の規範 ID の扱いと、必要な新規または変更要件を仕様へ反映する。

## 対象外

- CIMD を経由した `private_key_jwt` の新規提供。MCP と現行プロファイルで MAY である能力の未実装だけを不適合とは扱わない。
- DCR の削除、MCP 全体の再実装、CIMD 文書を代理発行するサービス。
- 未知の JSON フィールドをすべて拒否する独自規約。
- 4.1節の文書発行者の MUST NOT を、根拠なく認可サーバーの MUST reject と読み替えること。

## 設計

仕様差分の中心は、CIMD の URL と取得先の同一性を、正規化やリダイレクトで緩めないことである。
元の `client_id` 文字列を比較し、`:443` を省略した URL と同一視しない既存処理を維持する。
dot segment の percent-encoding、空の path、query の扱いは仕様の MUST と SHOULD を区別し、実装前に期待例を確定する。

URL 判定と文書検証は OAuth2 の CIMD 機能内に閉じる。
共有 `safehttp` 全体の redirect 方針を変える案は、JWKS など別用途へ波及するため採らず、CIMD の呼び出し設定または局所アダプターで拒否する案を第一候補とする。
モジュールの公開範囲と保存所有者は変えない。
主要型は既存 `clientIDMetadataDocument` と URL の検証結果を用い、`ValidateClientIdentifierURL` と文書の検証を計算、HTTP 取得と cache を作用として保つ。

秘密素材については文書を公開する側の禁止と、認可サーバーが受け取る側の検証責任を記録する。
秘密を保持、ログ出力、再公開しないことは製品の境界として確定し、受理拒否の条件は規範の根拠と互換性から実装前に決める。
拡張メタデータを一律拒否せず、公開鍵を秘密鍵と誤判定しない。
8.4節の metadata 変更時の対処は選択可能なポリシーであり、一律の同意取消を標準必須として追加しない。

移行では redirect を使う client に正規の metadata URL への変更を案内する。
既存の正当な文書は同じ client ID のまま使う。
ロールバックが必要なら問題の CIMD 解決を止め、redirect 追従や内部アドレスへの取得を復活させない。

## 計画

1. 固定した draft の差分と現在の取得、URL、cache、文書検証を対照する。
2. 仕様の採用表と必須条件、選択方針を更新してから局所修正する。
3. 他用途の safehttp を変えずに、正式な CIMD 解決経路から拒否を確認する。

## タスク

- [ ] T001 [Design] 義務の主語、dot segment の表現、禁止素材の扱い、cache と信頼更新の方針を確定する。
- [ ] T002 [Spec] draft を再ピン留めし、必須条件と推奨条件、現在の制限を採用表へ反映する。
- [ ] T003 [Acceptance] redirect の二段目へ一度もアクセスしないことと、dot segment の拒否の RED を作る。
- [ ] T004 [App] CIMD の局所設定と検証を直し、エラー文書を cache に保存しない。
- [ ] T005 [Verify] 正常200、全 redirect 状態、文書不一致、秘密素材、公開鍵、未知拡張、cache、SSRF の回帰を確認する。

## 検証

- `mise run check-spec`：採用する draft と規範 ID の解決を確認する。
- `mise run test-go-package -- ./oauth2/...`：URL と JSON の検証、正常文書と拒否、cache の振る舞いを確認する。
- `mise run test-go-package -- ./shared/security/safehttp/...`：CIMD の変更が共通の取得処理を緩めていないことを確認する。
- 正式な認可入口の配線は `mise run test-go-one -- <所有パッケージ> <CIMD受け入れテスト>` で確認し、取得先の記録から redirect 二段目への無作用を観測する。具体的な既存テストを着手時に選ぶ。
- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>` とする。HTTP を使う検証はサンドボックス外で実行する。

## リスク

URL の正規化と redirect を許すと、client ID と取得先の同一性が曖昧になる。
逆に未知フィールドの一律拒否は、draft が許す拡張を壊す。
採用表の強さを正確に読み、義務を超える製品方針は理由と互換性を明示する。
