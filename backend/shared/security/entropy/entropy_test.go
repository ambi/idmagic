package entropy_test

import (
	"bytes"
	"testing"

	"github.com/ambi/idmagic/backend/shared/security/entropy"
)

func TestFixedReadReturnsTheGivenBytesInOrder(t *testing.T) {
	src := entropy.Fixed([]byte{1, 2, 3, 4})
	first := make([]byte, 3)
	if err := src.Read(first); err != nil {
		t.Fatalf("Read: %v", err)
	}
	second := make([]byte, 1)
	if err := src.Read(second); err != nil {
		t.Fatalf("second Read: %v", err)
	}
	if !bytes.Equal(first, []byte{1, 2, 3}) || !bytes.Equal(second, []byte{4}) {
		t.Fatalf("Read = %v then %v, want [1 2 3] then [4]", first, second)
	}
}

func TestFixedReadFailsWhenTheBytesRunOut(t *testing.T) {
	src := entropy.Fixed([]byte{1, 2})
	if err := src.Read(make([]byte, 3)); err == nil {
		t.Fatal("Read past the fixed bytes succeeded, want an error")
	}
}

// Int は上限未満の値だけを受け入れ、上限以上の値を引いたら次の値を引き直す。
func TestFixedIntRejectsValuesAtOrAboveTheBound(t *testing.T) {
	// 上限 20 は 5 ビットで表すので、各バイトの下位 5 ビットを候補にする。0x1f（31）は上限以上なので捨てる。
	src := entropy.Fixed([]byte{0x1f, 0x03})
	got, err := src.Int(20)
	if err != nil {
		t.Fatalf("Int: %v", err)
	}
	if got != 3 {
		t.Fatalf("Int(20) = %d, want 3", got)
	}
}

func TestCryptoReadFillsTheBuffer(t *testing.T) {
	buf := make([]byte, 32)
	if err := entropy.Crypto().Read(buf); err != nil {
		t.Fatalf("Read: %v", err)
	}
	if bytes.Equal(buf, make([]byte, 32)) {
		t.Fatal("Read left the buffer zeroed")
	}
}
