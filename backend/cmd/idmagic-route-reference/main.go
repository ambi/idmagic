// Command idmagic-route-reference writes the operator-facing route priority
// reference (ROUTE_PRIORITY.md) from the assembled router and the priority
// classification the admission middleware applies, or checks that the tracked
// file still matches it (REQ-SYSTEM-019). Run it through
// `mise run generate-route-reference` and `mise run check-route-reference`.
package main

import (
	"flag"
	"fmt"
	"os"

	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "route-reference:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("idmagic-route-reference", flag.ContinueOnError)
	output := flags.String("output", "ROUTE_PRIORITY.md", "path of the generated route priority reference")
	check := flags.Bool("check", false, "fail instead of writing when the file is out of date")
	if err := flags.Parse(args); err != nil {
		return err
	}

	generated := httpadapter.RenderPriorityClassReference()

	if !*check {
		if err := os.WriteFile(*output, []byte(generated), 0o644); err != nil { //nolint:gosec // operator-facing documentation
			return err
		}
		fmt.Printf("ok  wrote %s\n", *output)
		return nil
	}

	current, err := os.ReadFile(*output)
	if err != nil {
		return err
	}
	if string(current) != generated {
		return fmt.Errorf("%s is out of date; run `mise run generate-route-reference`", *output)
	}
	fmt.Printf("ok  %s matches the assembled router and its priority classification\n", *output)
	return nil
}
