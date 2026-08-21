// Command idmagic-config-reference writes the operator-facing
// ConfigurationReference (CONFIGURATION.md) from the startup Config
// definition, or checks that the tracked file still matches it
// (REQ-SYSTEM-017). Run it through `mise run generate-config-reference` and
// `mise run check-config-reference`.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ambi/idmagic/backend/cmd/internal/bootstrap"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "config-reference:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("idmagic-config-reference", flag.ContinueOnError)
	output := flags.String("output", "CONFIGURATION.md", "path of the generated configuration reference")
	check := flags.Bool("check", false, "fail instead of writing when the file is out of date")
	if err := flags.Parse(args); err != nil {
		return err
	}

	generated, err := bootstrap.RenderConfigReference()
	if err != nil {
		return err
	}

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
		return fmt.Errorf("%s is out of date; run `mise run generate-config-reference`", *output)
	}
	fmt.Printf("ok  %s matches the startup configuration definition\n", *output)
	return nil
}
