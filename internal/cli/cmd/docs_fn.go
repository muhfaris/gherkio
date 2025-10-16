package cmd

import (
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/muhfaris/gherkio/internal/runner"
	"github.com/spf13/cobra"
)

var fnCmd = &cobra.Command{
	Use:   "fn",
	Short: "Function template documentation",
	Long:  `Displays documentation for available function templates.`,
	Example: `  # List all function templates
  gherkio docs fn`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runFnDocs(cmd, args)
	},
}

func init() {
	docsCmd.AddCommand(fnCmd)
}

func runFnDocs(cmd *cobra.Command, args []string) error {
	info := runner.GetFunctionTemplateInfo()

	var names []string
	for name := range info {
		names = append(names, name)
	}
	sort.Strings(names)

	var b strings.Builder
	tw := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "  NAME\tDESCRIPTION\tEXAMPLE\tRESULT")
	fmt.Fprintln(tw, "  ----\t-----------\t-------\t------")

	for _, name := range names {
		fn := info[name]
		fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\n", name, fn.Description, fn.Example, fn.Result)
	}
	_ = tw.Flush()

	fmt.Print(b.String())
	return nil
}