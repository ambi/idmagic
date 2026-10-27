package buildfailure

const width = 1 * 2

// 乗算を除算へ変えると配列の長さが負になり、テストを実行する前に構築が失敗する。
var _ [width - 1]byte

func Value() int {
	return width
}
