package usecases

import "testing"

// FuzzParseAuthorizationDetails は、受理した authorization_details が空でないことを表明する。
//
// 空の配列を成功として返すと、呼び出し側の「詳細が無いので検証不要」という経路へ流れ込み、
// 登録済み type に対する検証 (ValidateAuthorizationDetails) を素通りする。
func FuzzParseAuthorizationDetails(f *testing.F) {
	f.Add("")
	f.Add("   ")
	f.Add(`[]`)
	f.Add(`[{"type":"payment"}]`)
	f.Add(`{"type":"payment"}`)
	f.Add(`[[[[[[[[[[1]]]]]]]]]]`)
	f.Add(`[{"type":"payment","locations":["https://api.example"]}]`)

	f.Fuzz(func(t *testing.T, raw string) {
		if len(raw) > 64*1024 {
			return
		}
		details, err := ParseAuthorizationDetails(raw)
		if err != nil {
			if details != nil {
				t.Fatalf("ParseAuthorizationDetails returned %d details together with an error", len(details))
			}
			return
		}
		// 空文字と空白のみは「指定なし」として nil を返す正常系。
		if details == nil {
			return
		}
		if len(details) == 0 {
			t.Fatalf("ParseAuthorizationDetails accepted an empty detail list from %q", raw)
		}
	})
}
