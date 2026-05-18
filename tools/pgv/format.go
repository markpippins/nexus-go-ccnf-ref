package main

import "fmt"

func FormatHuman(delta IRDelta) string {
	s := fmt.Sprintf("PGV Diff: status=%s\n", delta.Status)
	s += fmt.Sprintf("  %d added, %d removed, %d moved, %d unchanged\n",
		delta.Summary.AddedCount, delta.Summary.RemovedCount,
		delta.Summary.MovedCount, delta.Summary.UnchangedCount)

	if len(delta.Added) > 0 {
		s += "\n  Added:\n"
		for _, n := range delta.Added {
			s += fmt.Sprintf("    + %s (%s)\n", n.ID, n.Type)
		}
	}

	if len(delta.Removed) > 0 {
		s += "\n  Removed:\n"
		for _, n := range delta.Removed {
			s += fmt.Sprintf("    - %s (%s)\n", n.ID, n.Type)
		}
	}

	if len(delta.Moved) > 0 {
		s += "\n  Moved:\n"
		for _, m := range delta.Moved {
			s += fmt.Sprintf("    ~ %s\n      from: %s\n      to:   %s\n", m.NodeID, m.FromPath, m.ToPath)
		}
	}

	return s
}
