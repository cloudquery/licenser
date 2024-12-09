package main

import (
	"os"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/stretchr/testify/require"
)

func TestCheck(t *testing.T) {
	cmd := rootCmd()
	cmd.SetArgs([]string{"check", "."})
	err := cmd.Execute()
	require.NoError(t, err)
}

func TestReport(t *testing.T) {
	cmd := rootCmd()
	cmd.SetArgs([]string{"report", "."})
	err := cmd.Execute()
	require.NoError(t, err)
	content, err := os.ReadFile("docs/_licenses.md")
	require.NoError(t, err)
	snaps.MatchSnapshot(t, string(content))
	defer os.Remove("docs/_licenses.md")
}
