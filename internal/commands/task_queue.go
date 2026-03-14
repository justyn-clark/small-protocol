package commands

import "strings"

func nextActionableTaskIDs(tasks []PlanTask, limit int) []string {
	taskStatuses := make(map[string]string, len(tasks))
	for _, task := range tasks {
		taskStatuses[strings.TrimSpace(task.ID)] = normalizePlanStatus(task.Status)
	}

	actionable := []string{}
	for _, task := range tasks {
		if normalizePlanStatus(task.Status) != "pending" {
			continue
		}

		depsSatisfied := true
		for _, depID := range task.Dependencies {
			depStatus, exists := taskStatuses[depID]
			if !exists || depStatus != "completed" {
				depsSatisfied = false
				break
			}
		}
		if !depsSatisfied {
			continue
		}

		actionable = append(actionable, strings.TrimSpace(task.ID))
		if limit > 0 && len(actionable) >= limit {
			break
		}
	}

	return actionable
}

func nextRunnableTaskID(plan *PlanData) string {
	if plan == nil {
		return ""
	}
	if current := firstTaskIDByStatus(plan, "in_progress"); current != "" {
		return current
	}
	actionable := nextActionableTaskIDs(plan.Tasks, 1)
	if len(actionable) > 0 {
		return actionable[0]
	}
	return ""
}

func preferredPlanTaskID(plan *PlanData) string {
	if plan == nil {
		return ""
	}
	if taskID := nextRunnableTaskID(plan); taskID != "" {
		return taskID
	}
	if blocked := firstTaskIDByStatus(plan, "blocked"); blocked != "" {
		return blocked
	}
	return firstTaskIDByStatus(plan, "pending")
}
