package handlers_http

import (
	"strings"
	"testing"
	"unicode/utf8"
)

//spec:covers REQ-WSFEDERATION-001: コマンドの値を PowerShell のリテラルとして保存し、引用符、変数、改行を実行されるコードにしないこと。
func TestEntraPowerShellCommandsPreserveLiteralValues(t *testing.T) {
	value := "urn:O'Brien:‘left’‚base‛:$env:HOME;$(throw 'bad')`\nnext"
	settings := map[string]string{"IssuerUri": value, "SigningCertificate": "base64=="}
	commands := entraPowerShellCommands("o'brien.example", settings)
	for _, command := range []string{commands.Create, commands.Update} {
		for _, want := range []string{
			"-DomainId 'o''brien.example'",
			"-IssuerUri 'urn:O''Brien:‘‘left’’‚‚base‛‛:$env:HOME;$(throw ''bad'')`\nnext'",
			"-SigningCertificate 'base64=='",
		} {
			if !strings.Contains(command, want) {
				t.Fatalf("command lacks literal %q: %s", want, command)
			}
		}
	}
	if commands != entraPowerShellCommands("o'brien.example", settings) {
		t.Fatal("commands must be deterministic")
	}
}

//spec:covers REQ-WSFEDERATION-001: 一つの単一引用符リテラルから脱出せず、元の文字列を PowerShell に渡すこと。
func FuzzPowerShellLiteral(f *testing.F) {
	for _, value := range []string{"", "contoso.com", "O'Brien", "‘’‚‛'", "';throw 'oops'", "$env:HOME;$(throw 1)`\n日本語\r\n\""} {
		f.Add(value)
	}
	f.Fuzz(func(t *testing.T, value string) {
		if !utf8.ValidString(value) {
			t.Skip("HTTP JSON が返す文字列は UTF-8 である")
		}
		literal := []rune(powerShellLiteral(value))
		if len(literal) < 2 || literal[0] != '\'' || literal[len(literal)-1] != '\'' {
			t.Fatalf("not a single-quoted literal: %q", string(literal))
		}
		var decoded strings.Builder
		// PowerShell の引用符の走査に従い、内側の引用符は必ず二文字を消費する。
		for i := 1; i < len(literal)-1; i++ {
			ch := literal[i]
			if ch == '\'' || ch >= '\u2018' && ch <= '\u201b' {
				if i+1 >= len(literal)-1 || literal[i+1] != ch {
					t.Fatalf("literal escapes into code: %q", string(literal))
				}
				i++
			}
			decoded.WriteRune(ch)
		}
		if decoded.String() != value {
			t.Fatalf("decoded %q, want %q", decoded.String(), value)
		}
	})
}
