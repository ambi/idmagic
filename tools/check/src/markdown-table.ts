/**
 * Markdown の表の一行をセルに分ける。両端の区切りは省いてもよい。CEL のガードが
 * 論理和を `\|\|` と書くように、エスケープした区切りはセルを終えない。
 */
export function tableRowCells(line: string): string[] {
  return line
    .trim()
    .replace(/^\|/, '')
    .replace(/(?<!\\)\|$/, '')
    .replaceAll('\\|', '\u0000')
    .split('|')
    .map((cell) => cell.trim().replaceAll('\u0000', '\\|'))
}
