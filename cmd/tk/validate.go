package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/jacksmith/tk/internal/cli"
	"github.com/jacksmith/tk/internal/ops"
	"github.com/jacksmith/tk/internal/storage"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Check data integrity",
	Long: `Check all projects for data integrity issues.

Checks for:
- Orphan blockers (references to non-existent items)
- Dependency cycles
- Duplicate IDs
- Invalid ID formats
- Missing required fields

Use --fix to auto-repair fixable issues (removes orphan references).`,
	RunE: runValidate,
}

var validateFix bool

func init() {
	validateCmd.Flags().BoolVar(&validateFix, "fix", false, "auto-repair fixable issues")
	rootCmd.AddCommand(validateCmd)
}

func runValidate(cmd *cobra.Command, args []string) error {
	s, err := storage.Open(".")
	if err != nil {
		return err
	}

	if validateFix {
		return runValidateAndFix(s)
	}
	return runValidateOnly(s)
}

func runValidateOnly(s *storage.Storage) error {
	errors, err := ops.Validate(s)
	if err != nil {
		return err
	}

	if len(errors) == 0 {
		fmt.Println(cli.Green("No issues found."))
		return nil
	}

	fmt.Printf("Found %d issue(s):\n\n", len(errors))

	for _, e := range errors {
		typeStr := formatValidationErrorType(e.Type)
		fmt.Printf("%s %s: %s\n", e.ItemID, typeStr, e.Message)
		if len(e.Details) > 0 {
			fmt.Printf("  %s\n", strings.Join(e.Details, " → "))
		}
	}

	// Exit with error code if there are issues
	os.Exit(1)
	return nil
}

func runValidateAndFix(s *storage.Storage) error {
	// First report current issues
	errors, err := ops.Validate(s)
	if err != nil {
		return err
	}

	if len(errors) == 0 {
		fmt.Println(cli.Green("No issues found."))
		return nil
	}

	fmt.Printf("Found %d issue(s). Attempting to fix...\n\n", len(errors))

	// Apply fixes
	fixes, err := ops.ValidateAndFix(s)
	if err != nil {
		return err
	}

	if len(fixes) > 0 {
		fmt.Println("Fixes applied:")
		for _, f := range fixes {
			fmt.Printf("  %s: %s\n", f.ItemID, f.Description)
		}
		fmt.Println()
	}

	// Re-validate to show remaining issues
	remainingErrors, err := ops.Validate(s)
	if err != nil {
		return err
	}

	if len(remainingErrors) == 0 {
		fmt.Println(cli.Green("All fixable issues resolved."))
		return nil
	}

	fmt.Printf("Remaining issues (%d) that cannot be auto-fixed:\n\n", len(remainingErrors))
	for _, e := range remainingErrors {
		typeStr := formatValidationErrorType(e.Type)
		fmt.Printf("%s %s: %s\n", e.ItemID, typeStr, e.Message)
		if len(e.Details) > 0 {
			fmt.Printf("  %s\n", strings.Join(e.Details, " → "))
		}
	}

	os.Exit(1)
	return nil
}

func formatValidationErrorType(t ops.ValidationErrorType) string {
	switch t {
	case ops.ValidationErrorOrphanBlocker:
		return cli.Yellow("[orphan]")
	case ops.ValidationErrorCycle:
		return cli.Red("[cycle]")
	case ops.ValidationErrorDuplicateID:
		return cli.Red("[duplicate]")
	case ops.ValidationErrorInvalidID:
		return cli.Red("[invalid-id]")
	case ops.ValidationErrorMissingRequired:
		return cli.Red("[missing]")
	case ops.ValidationErrorInvalidPriority:
		return cli.Red("[priority]")
	default:
		return fmt.Sprintf("[%s]", t)
	}
}
