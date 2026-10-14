---
status: completed
archived_from: 2e336ae0cbe20c0f0d626ffeaabb8405c7c7d9ca
---

# shadcn/ui の Base UI 版へ移行し、Button / DropdownMenu / Label / Select のデザインを刷新する

Radix UI(`@radix-ui/react-slot` / `react-dropdown-menu` / `react-label`)から
shadcn/ui の Base UI 版へ移行した。`npx shadcn@latest init -b base`(プリセット
`vega`(標準サイズ)、`baseColor: neutral`、`iconLibrary` を idmagic 既存の
`tabler` に変更)で `components.json` を作成し、`button` / `dropdown-menu` /
`label` / `select` を生成、idmagic の規約(既存の簡易 `Select` API 維持、`DropdownMenuLabel` を
`Menu.GroupLabel` ではなく素の `<div>` に変更等)に合わせて調整した。
呼び出し側の `asChild`(実測 51 箇所・約 34 ファイル)を `render`/`nativeButton`
へ、`DropdownMenuItem` の `onSelect`(Radix 専用・Base UI には無く TypeScript でも
検出できない静かな不具合になる)を全箇所 `onClick` へ変換した。
`frontend/package.json` の `@radix-ui/*` を `@base-ui/react` に置き換えた。
実装過程で当初の調査になかった追加の破壊的変更をいくつか発見した: Base UI
Select の `role="combobox"` は content ベースの accessible name 算出をしないため
`aria-label` フォールバックを追加、`Select.Item` はマウスクリック前に
`pointerdown` が無いと選択が確定しない、Menu/Select のトリガーはキーボード
Enter/Space をネイティブ `<button>` の挙動に委ねている(happy-dom では
再現されないためテストは `fireEvent.click` に統一)。詳細は Design を参照。
実装中、並列委任した3エージェントの1つが対象外ファイルへの意図しない差分を
`git checkout --` で戻す際に他エージェント・自分自身の作業中ファイルまで
巻き込んで一部が Radix 版へ巻き戻る事故が発生した(共有作業ツリーでの並列実行が
原因)。検出後、影響ファイルをすべて手動で変換し直して収束させた。
デザイントークンは shadcn init の `neutral` ベース(中立的な黒/白/グレー)を
採用し、Cloudflare 等どのブランドにも紐づかない土台を整えた。それ以上の配色・
アクセントカラーの決定、および `Dialog`/`Toast`/`Card`/`Alert`/`Input` を含む
frontend 全体のデザイン刷新は、別 WI として起票することをユーザーに提案する
(T010、[[wi-311-evaluate-base-ui-kumo-migration]] で合意した「デザイン刷新」の
範囲は本 WI では Button/DropdownMenu/Label/Select の技術的な下地に留まる)。

全文は `git show 2e336ae0cbe20c0f0d626ffeaabb8405c7c7d9ca:<このファイルのパス>` で読む。
