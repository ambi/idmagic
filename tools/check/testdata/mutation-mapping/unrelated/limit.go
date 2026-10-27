package unrelated

import "github.com/ambi/idmagic/tools/check/testdata/mutation-mapping/target"

// 逆依存には入るが、このパッケージのテストはこの関数を通らない。
func WithinLimit(value, maximum int) bool {
	return target.WithinLimit(value, maximum)
}
