// Command idmagic-kerberos は配備した信頼ファイルを検査する。AD への接続や認証の成功は検査しない。
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ambi/idmagic/backend/authentication"
	"github.com/ambi/idmagic/backend/idmanagement"
)

func main() {
	path := flag.String("check-file", "", "Kerberos configuration JSON file")
	flag.Parse()
	if *path == "" {
		fmt.Fprintln(os.Stderr, "--check-file is required")
		os.Exit(2)
	}
	users := idmanagement.NewMemoryModule().UserRepo
	module := authentication.NewMemoryModule(users)
	fingerprint, err := module.ConfigureNegotiate(*path, users)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("Trust files are valid; configuration SHA-256: %s\nAD reachability and browser authentication have not been tested.\n", fingerprint)
}
