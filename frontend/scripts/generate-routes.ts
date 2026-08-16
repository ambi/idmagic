// generate-routes は src/routes 以下のファイルから src/routeTree.gen.ts を作り直す。
//
// Vite のプラグインが dev と build で同じことをするが、`bun run build` は Vite を起動する前に
// 型検査を走らせる。新しいルートファイルを足した直後は、それを宣言するツリーが書かれる前に
// 型検査が落ちるため、生成だけを単体で回せる口でこの循環を切る。
import { Generator, getConfig } from '@tanstack/router-generator'
import path from 'node:path'

const root = path.resolve(import.meta.dir, '..')
const config = await getConfig({ target: 'react', autoCodeSplitting: true }, root)
await new Generator({ config, root }).run()
console.log(`generated ${path.relative(root, config.generatedRouteTree)}`)
