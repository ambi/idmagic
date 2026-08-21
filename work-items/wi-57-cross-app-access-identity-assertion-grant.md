---
status: pending
authors: ["tn"]
risk: high
created_at: 2026-06-22
priority: p2
depends_on: [wi-50-token-exchange-delegation-actor-chain, wi-56-mcp-authorization-server]
change_kind: feature
affected_spec:
  - { path: spec/contexts/oauth2/scenarios.md, requirement: REQ-OAUTH2-048 }
  - { path: spec/contexts/oauth2/models.tsp, symbol: TokenRequest }
---

# Cross-App Access (Identity Assertion Authorization Grant) でエージェントのアプリ間アクセスを仲介する

## Motivation

企業内のエージェントが、あるアプリ (アプリ A、MCP クライアント) から別のアプリや MCP サーバー (アプリ B) のデータへ、ユーザーごとの個別の同意画面を介さずにアクセスしたい。これを IdP が仲介し、企業が一元的に可視化・統制する標準が Identity Assertion Authorization Grant (`draft-ietf-oauth-identity-assertion-authz-grant`、Okta の Cross-App Access) である。IdP が信頼するアイデンティティアサーション (`id_token` など) を、アプリ B 向けのアクセストークンへ交換する。

Okta は 2026 年を通してクライアント側・SAML リソースアプリ側・SAML 要求側の実装ガイドを継続して公開しており、企業向けの MCP 認可の中心になりつつある。IdMagic はトークン交換 ([[wi-50-token-exchange-delegation-actor-chain]]) と MCP 認可サーバー ([[wi-56-mcp-authorization-server]]) を備えているため、アサーションを起点としたアプリ間・エージェントから MCP サーバーへの付与を仲介できる位置にいる。これによりアプリごとの再同意を排し、エージェントのアプリ間アクセスを IdP が集中管理 (付与・可視化・失効) できる。

**プロトコルの形 (2026-08-08 の監査 [[wi-322-mcp-authorization-spec-repin-2026-07-audit]] による訂正)。** ID-JAG は単一のトークン交換プロファイルではなく **2 ホップ**である。

1. クライアント → IdP (IdMagic が IdP 役のとき): RFC 8693 トークン交換。`subject_token` はアイデンティティアサーション (`id_token` など)、`requested_token_type` は `urn:ietf:params:oauth:token-type:id-jag` で ID-JAG を発行する。
2. クライアント → 宛先アプリの認可サーバー (IdMagic が宛先の認可サーバー役のとき): **RFC 7523 の JWT Bearer グラント** (`grant_type=urn:ietf:params:oauth:grant-type:jwt-bearer`、`assertion` に ID-JAG) でアクセストークンを発行する。これはトークン交換ではない。

IdMagic は現状 RFC 7523 をクライアント認証 (`client_assertion`) にしか用いておらず、**認可グラントとしての JWT Bearer は未対応**である。この `grant_type` 自体を新設する必要がある。

**標準の状態 (2026-08 時点)。** `draft-ietf-oauth-identity-assertion-authz-grant` は Standards Track の Internet-Draft であり、まだ RFC 化されていない。実装は draft の改訂に追従する前提で行う。

## Scope

- 対象とする draft の改訂を固定し、アイデンティティアサーションの受理条件 (信頼する発行者と受け手) を確定する。
- アプリ A からアプリ B への許可関係 (どのクライアントがどのリソースを要求できるか) の登録モデルを設ける。所有は `Application` Context とする。
- ID-JAG の発行 (IdP 役) を、既存のトークン交換の `requested_token_type` プロファイルとして追加する。
- ID-JAG の償還 (宛先の認可サーバー役) を、新設する RFC 7523 JWT Bearer グラントとして追加する。`spec/contexts/oauth2/models.tsp` の `TokenRequest` と、クライアントが宣言できる `grant_types` の双方に反映する。
- アサーションの検証を厳格にする。`iss` / `sub` / `aud` / `exp` / `iat` / `jti`、ユーザーの認可、発信元のクライアントとエージェント、委譲チェーンを持つものだけを受理し、登録済みの発信元鍵と宛先リソースのポリシーで検証する。アサーションの受け手は IdMagic の交換エンドポイント、要求されるリソースは宛先アプリとして分ける。
- `jti` の再送窓、短い有効期限、発信元と宛先のテナント一致、発信元アプリの許可リスト、ユーザーの同意または企業ポリシー、`authorization_details` の縮小をすべて満たす場合だけ、宛先を受け手とするトークンを発行する。
- 交換の結果は既存の `act` チェーンへ発信元アプリとエージェントを加える。アサーションの本文とユーザーのデータは保存しない。
- 管理コンソールにアプリ間アクセス許可の付与・一覧・取消を追加する。
- durable な設計判断は `spec/contexts/oauth2/decisions.md` と `spec/contexts/application/decisions.md` へ、機構の説明は各 `internals.md` へ、本変更固有の分析と却下した代替案は本ファイルの `## Design` へ置く。

## Out of Scope

- 外部 IdP (Okta など) との相互運用の認証取得。
- MCP 認可サーバーの基盤そのもの。[[wi-56-mcp-authorization-server]] が前提として完了済みである。
- エンドユーザー個別の同意フロー。本 work item は企業管理によるアプリ間の付与を対象とする。
- SAML を起点とするアプリ間アクセス。Okta は 2026-07 に SAML リソースアプリと SAML 要求側の手順を公開したが、まず OIDC の経路を通す。

## Design

未定。着手時に次を確定して本節に記録する。

- 追従する draft の改訂番号。実装時点の最新を固定し、改訂で壊れる箇所を明示する。
- 許可関係の粒度。クライアント対リソースで足りるか、ユーザーまたはエージェントの単位まで下ろすか。
- ID-JAG の有効期限と `jti` 再送窓の長さ。短くすると時計のずれに弱くなり、長くすると再送の窓が広がる。

## Plan

- 2 ホップのうち、まず IdP 役 (ID-JAG の発行) を既存のトークン交換へプロファイルとして足す。宛先の認可サーバー役 (JWT Bearer グラントの新設) は独立して進められる。
- JWT Bearer グラントの新設は、既存の `client_assertion` 検証と経路を共有しない。クライアント認証と認可グラントは別の判断であり、混ぜると片方の緩和がもう片方へ漏れる。
- 偽造・受け手違い・期限切れ・再送のテストを先に書く。

## Tasks

- [ ] T001 [Design] 追従する draft の改訂、アサーションのプロファイル、ポリシーの所有、トークン交換への対応付けを確定し `## Design` に記録する。
- [ ] T002 [Spec] アサーションのモデル、企業管理のアプリ間ポリシー、交換のインターフェースと誤り・イベント・制約・シナリオを追加して再生成する。
- [ ] T003 [Application] 発信元から宛先へのリソース許可リスト、ユーザーとエージェントのポリシー、鍵のメタデータと管理ユースケースを実装する。
- [ ] T004 [OAuth2] アサーション検証器、`jti` 再送記憶、`subject_token` の振り分け、`authorization_details` と `act` チェーンの縮小を既存のトークン交換へ追加する。
- [ ] T005 [OAuth2] RFC 7523 JWT Bearer を認可グラントとして新設し、ID-JAG の償還経路を実装する。
- [ ] T006 [UI] アプリ間ポリシーの管理と、必要な場合の同意画面に発信元・宛先・エージェント・操作を表示する。
- [ ] T007 [Verify] 偽造・受け手違い・期限切れ・再送のアサーション、テナント越え、ポリシー取消、委譲の深さ、宛先の受け手を通しで検証する。

## Verification

- `just verify-spec`
- `just test-go`
  - reason: アサーションの検証、許可ポリシーの照合 (未許可は拒否)、受け手の限定、取消後の拒否、`jti` 再送の境界。
- `just verify-ui`
- `just verify`
- 手動: アプリ A のアサーションでアプリ B 向けトークンを取得し、許可ポリシー外のアプリが拒否され、ポリシーを取り消すと以後拒否されることを確認する。

## Risk Notes

リスクは high。アプリ間アクセスの自動仲介は、許可関係の検証が緩いと横方向の権限拡大に直結する。アサーションの発行者・受け手・有効期限を厳格に検証し、管理者が登録した許可ポリシーに合致する場合だけ交換する。迷ったら拒否する側へ倒す。

draft 段階の仕様であることもリスクである。追従する改訂を `## Design` で固定し、改訂で壊れうる箇所を実装時点で明示する。RFC 化されていない以上、相互運用の主張は控える。
