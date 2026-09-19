// Command idmagic-gateway-routes checks the reference frontend gateway's route
// allowlists against the assembled router (REQ-SYSTEM-021). Run it through
// `mise run check-gateway-routes`.
//
// The gateway decides which paths reach the Go API with hand-written allowlists.
// When one drifts from the route table, the gateway's final handle serves the SPA
// instead, so the missing route answers 200 with `text/html` rather than failing.
// That is why the drift needs a check of its own.
package main

import (
	"flag"
	"fmt"
	"os"

	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "gateway-routes:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("idmagic-gateway-routes", flag.ContinueOnError)
	caddyfile := flags.String("caddyfile", "frontend/Caddyfile", "path of the reference gateway configuration")
	viteConfig := flags.String("vite-config", "frontend/vite.config.ts", "path of the development gateway configuration")
	if err := flags.Parse(args); err != nil {
		return err
	}

	caddySource, err := os.ReadFile(*caddyfile)
	if err != nil {
		return err
	}
	viteSource, err := os.ReadFile(*viteConfig)
	if err != nil {
		return err
	}

	findings := httpadapter.CheckGatewayAllowlists(string(caddySource), string(viteSource))
	if len(findings) > 0 {
		for _, finding := range findings {
			fmt.Fprintln(os.Stderr, "  "+finding)
		}
		return fmt.Errorf("%d gateway allowlist finding(s); the reference gateway does not match the assembled route table", len(findings))
	}

	fmt.Printf("ok  %s and %s carry every route the assembled router requires\n", *caddyfile, *viteConfig)
	return nil
}
