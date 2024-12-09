package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

// checkGoLicenses runs `go-licenses check . --disallowed_types=forbidden,notice,reciprocal,restricted`
// in the specified directory and captures stdout and stderr.
func checkGoLicenses(dir string, disallowed []string) (map[string]bool, error) {
	var foundViolations = make(map[string]bool, 0)

	cmd := exec.Command("go-licenses", "check", ".", "--disallowed_types="+strings.Join(disallowed, ","))
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// log.Printf("Error running go-licenses in %s\nStdout: %s\nStderr: %s\n", dir, stdout.String(), stderr.String())
		lines := strings.Split(stderr.String(), "\n")
		for i := 0; i < len(lines); i++ {
			if strings.Contains(lines[i], "Reciprocal license type") ||
				strings.Contains(lines[i], "Forbidden license type") ||
				strings.Contains(lines[i], "Notice license type") ||
				strings.Contains(lines[i], "Restricted license type") {
				fmt.Println("Found violation in", dir, ":", lines[i])
				foundViolations[lines[i]] = true
			}
		}
	}

	log.Printf("Successfully checked licenses in %s\n", dir)
	return foundViolations, nil
}

// generateGoLicensesReport runs `go-licenses report . --template=../../../scripts/check_licenses/template.tpl`
// in the specified directory.
func generateLicensesReport(dir string) error {
	if err := os.WriteFile("template.tpl", []byte(template), 0644); err != nil {
		return err
	}
	defer os.Remove("template.tpl")

	cmd := exec.Command("go-licenses", "report", ".", "--template=template.tpl")
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		log.Printf("Error running go-licenses in %s\n%s\n", dir, stderr.String())
		return err
	}

	if err := os.MkdirAll("docs", 0755); err != nil {
		return err
	}
	file, err := os.Create("docs/_licenses.md")
	if err != nil {
		return err
	}

	_, err = io.Copy(file, &stdout)
	return err
}

// findGoModDirs searches recursively for directories containing a go.mod file starting from rootDir.
func findGoModDirs(rootDir string) ([]string, error) {
	var dirs []string
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Name() == "go.mod" {
			dirs = append(dirs, filepath.Dir(path))
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return dirs, nil
}

func main() {
	g := new(errgroup.Group)
	g.SetLimit(10)

	var dirs []string

	cmd := cobra.Command{
		Use: "license",
	}
	var disallowedTypesPF *[]string

	checkCmd := &cobra.Command{
		Use:          "check",
		SilenceUsage: true,
		Args:         cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if dirs, err = findGoModDirs(args[0]); err != nil {
				return fmt.Errorf("Error finding go.mod directories: %v\n", err)
			}
			if len(dirs) == 0 {
				return errors.New("no directories with go.mod found")
			}

			for _, dir := range dirs {
				g.Go(func() error {
					log.Printf("Checking licenses in directory: %s\n", dir)
					violations, err := checkGoLicenses(dir, *disallowedTypesPF)
					if err != nil {
						return fmt.Errorf("Error checking licenses: %v\n", err)
					}
					if len(violations) > 0 {
						return fmt.Errorf("Found %d violations in %s\n", len(violations), dir)
					}
					return nil
				})
				return g.Wait()
			}

			return nil
		},
	}
	disallowedTypesPF = checkCmd.PersistentFlags().StringArray(
		"disallowed_types",
		[]string{"forbidden", "restricted"},
		"disallowed license types (allowed values: forbidden, notice, reciprocal, restricted, unknown)",
	)

	reportCmd := &cobra.Command{
		Use:  "report",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if dirs, err = findGoModDirs(args[0]); err != nil {
				return fmt.Errorf("Error finding go.mod directories: %v\n", err)
			}
			if len(dirs) == 0 {
				return errors.New("no directories with go.mod found")
			}

			for _, dir := range dirs {
				g.Go(func() error {
					log.Printf("Creating report of licenses in directory: %s\n", dir)
					if err := generateLicensesReport(dir); err != nil {
						log.Fatalf("Error generating report: %v\n", err)
					}
					return nil
				})
				return g.Wait()
			}

			return nil
		},
	}

	cmd.AddCommand(checkCmd)
	cmd.AddCommand(reportCmd)

	cmd.Execute()
}
