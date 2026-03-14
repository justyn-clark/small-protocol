package commands

import "strings"

type recoverableStrictIssues struct {
	OrphanProgress      bool
	LegacyRuntimeLayout bool
}

func (r recoverableStrictIssues) Any() bool {
	return r.OrphanProgress || r.LegacyRuntimeLayout
}

func detectRecoverableStrictIssues(output checkOutput) recoverableStrictIssues {
	issues := recoverableStrictIssues{}
	if output.Lint.Status != "failed" {
		return issues
	}
	for _, message := range output.Lint.Errors {
		if strings.Contains(message, "strict invariant S2 failed") {
			issues.OrphanProgress = true
		}
		if strings.Contains(message, "strict invariant S4 failed") && (strings.Contains(message, ".small/archive/") || strings.Contains(message, ".small/runs/")) {
			issues.LegacyRuntimeLayout = true
		}
	}
	return issues
}
