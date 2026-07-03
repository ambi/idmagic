---
id: idp-wi-22-spa-e2e-smoke-test
title: "SPA E2E スモークテストを Bun のヘッドレスブラウザ (Bun.WebView) で 1 本だけ導入する"
created_at: 2026-06-19
authors: ["tn"]
status: completed
risk: low
---
# Motivation
現状の検証層 (`go test`, `bun typecheck` / `lint` / `build`) では次の
2 つの回帰を捕まえられない:
  - TanStack Router 配下の SPA dispatcher (login / consent / device /
    error のいずれを描画するかを `meta[name="idmagic:page"]` で表明する
    層)。過去 dispatcher の分岐ミスで login URL を開いても空白画面
    になる事故があった。
  - `/authorize` → 認可コードを乗せた callback URL への遷移における
    cross-origin redirect 挙動。`fetch` の redirect モードや
    `Same-Site` 設定で `code=...` と `iss=...` の片方が落ちる事故が
    過去に発生している (RFC 9207 と整合させた経緯)。

Go 側のユニットテストは「正しい HTTP レスポンスを返す」ことは保証
するが、ブラウザの DOM が実際に描画され `fetch` が想定の挙動を取って
いることは別レイヤでしか確認できない。実ブラウザで golden path 1 本を
縛り、上記 2 領域の回帰を機械的に検知する。

ブラウザ自動化には Bun 組み込みの `Bun.WebView` を使う。macOS では
システムの WKWebView を駆動するため zero-dependency でブラウザの
ダウンロードが不要、Linux / Windows ではインストール済みの Chrome を
Chrome DevTools Protocol で駆動する。これにより外部のブラウザ自動化
フレームワークや別ブラウザの取得なしに、`bun test` だけで golden path を
実機検証できる (CLAUDE.md の Bun 優先方針とも一致)。

本 WI はテスト戦略の足場 (ランナー / fixture / 1 シナリオ) を入れる
ところまで。シナリオの拡充や consent 拒否経路、`error` page の検証は
別 WI で積み増す。

# Scope
- **decision**: 新規 ADR は不要。テスト戦略の段階導入で、normative spec は変えない。 ランナーは Bun 組み込みの `Bun.WebView` + `bun test` とし、外部の ブラウザ自動化フレームワークやブラウザバイナリの取得には依存しない (macOS は WKWebView、その他は既存 Chrome を CDP 駆動)。
- **go**: アプリ本体は変更なし。テストが `cmd/idmagic` を `PERSISTENCE=memory` / `ADDR=:8081` / `ISSUER=http://localhost:5173` で起動する。ISSUER を 5173 (ブラウザ origin) に合わせるのは、`verifyBrowserRequest` が Origin と ISSUER オリジンの一致を要求するため (README のローカル開発手順と同一構成)。
- **ui**:
  - bootstrap: SPA dispatcher の不変条件を機械検証できるよう、`ui/src/main.tsx` の bootstrap で `loadPageData()` が解決した `data.kind` (描画ページ種別の 唯一の真実源) を `<meta name="idmagic:page">` として DOM に出力する。 これにより login / consent / device / error などの分岐を assert できる。
  - layout: `idmagic/ui/tests/e2e/` 配下に `bun test` 用のスモーク (`authorize-golden-path.spec.ts`) と共有 fixture (`fixtures.ts`) を置く。 新規の npm 依存は追加しない (`Bun.WebView` は Bun 同梱)。, テスト自身が beforeAll/afterAll で次を起動・停止する:
  (a) Go API を memory mode で `:8081` に (上記 go 節の env)。
  (b) Vite dev server を `:5173` に (`/authorize`・`/api` を 8081 へプロキシ)。
  (c) demo-client の登録済み redirect_uri (`http://localhost:3000/callback`)
      を受ける最小コールバックサーバを `:3000` に (テストプロセス内の
      `Bun.serve`)。認可レスポンス URL はブラウザ側 (`view.url`) から読む。
  - scenarios: `tests/e2e/authorize-golden-path.spec.ts` を 1 本だけ追加する:
  1. `/authorize?client_id=demo-client&response_type=code&...` を
     `Bun.WebView` で開く。
  2. `meta[name="idmagic:page"]` が `login` であることと
     `input[name="username"]` の存在を assert (dispatcher 回帰防止)。
  3. `alice` / `demo-password-1234` を入力して submit。
  4. `meta[name="idmagic:page"]` が `consent` になることを assert。
  5. 「許可して続行」を押す。
  6. cross-origin redirect 後の `view.url` (= callback URL) に
     `code` / `iss` / `state` が乗っていることを assert (redirect の回帰防止)。
  - scripts: `package.json` に `"test:e2e": "bun test tests/e2e"` を追加する。 ブラウザ取得用の install スクリプトは不要 (macOS は WKWebView、 その他は既存 Chrome を検出)。
- **documentation**: `idmagic/ui/README.md` に「E2E スモーク」節を追加し、 `bun run test:e2e` のローカル実行手順 (前提は go / bun が PATH に あることのみ) を書く。, README §Phase 9 の "SPA E2E スモークテスト" 行は completion で 「実装済 (本 WI)」として除去を記録する。

# Out of Scope
- 複数ブラウザエンジンでのクロスブラウザ実行。本 WI は 1 エンジン (macOS は WKWebView) に固定する。
- consent 拒否経路、`error` page の表示確認、device flow、PAR の E2E。golden path 1 本のみ。
- visual regression / screenshot diff。
- GitHub Actions / CI へのワイヤリング。ローカル実行可能にして 完了。CI 化は別 WI とする。
- OAuth / OIDC conformance suite。これは別タスク (Phase 9 末尾)。
- Storybook / コンポーネント単位の visual テスト。

# Verification
- [object Object]
- [object Object]
- [object Object]
- [object Object]
- 手動: 意図的に dispatcher のページ種別出力 (`markPage`) を一行壊して `test:e2e` が失敗することを目視確認する (回帰検知が機能している ことの自己テスト)。

# Risk Notes
`Bun.WebView` は experimental API のため、将来のバージョンで挙動・API が
変わる可能性がある。スモーク 1 本・Bun バージョン固定の範囲で受容し、
問題が出れば fixture / helper 側で吸収する。

1 エンジンのみで開始するため、Firefox など他エンジンの SPA 挙動差は
catch できない。これは現時点での意思決定であり、主要ブラウザは
chromium / WebKit 系で十分に代表できるという判断 (本番ターゲットが
変わったら拡張する)。

ローカルで `bun run test:e2e` を CI に常駐させるかは別 WI で判断する。

# Completion
- **Completed At**: 2026-06-21
- **Summary**:
  SPA の golden path (authorize -> login -> consent -> callback) を 1 本の
  E2E スモークとして導入した。ランナーは Bun 組み込みの `Bun.WebView` +
  `bun test` で、外部のブラウザ自動化フレームワークやブラウザバイナリ取得に
  依存しない (macOS は WKWebView)。テストは Go API (memory, :8081)・Vite dev
  (:5173)・demo-client の redirect_uri を受ける最小コールバックサーバ (:3000) を
  自動起動/停止し、(1) ログイン画面で `meta[name="idmagic:page"]` が `login` で
  あること + `input[name="username"]` の存在、(2) ログイン後に `consent` へ遷移
  すること、(3) 「許可して続行」後の cross-origin redirect で callback URL に
  `code` / `iss` / `state` が乗ること、を assert する。dispatcher 不変条件を
  成立させるため `ui/src/main.tsx` に `meta[name="idmagic:page"]` 出力を追加した。
- **Verification Results**:
  - [object Object]
  - [object Object]
  - [object Object]
  - [object Object]
  - 自己テスト: main.tsx の markPage を定数に壊すと test:e2e が "timeout waiting for page kind=login" で fail することを確認 (回帰検知が 機能している)。修正後は再び pass。
