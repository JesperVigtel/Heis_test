package cluster

import "new_heis/internal/model"

func MergeRequests(table model.RequestTable, clock *model.Clock, entries []model.RequestEntry) bool {
	changed := false
	for _, entry := range entries {
		clock.Observe(entry.Cell.Version)
		if model.CompareVersion(entry.Cell.Version, table.Cell(entry.Key).Version) > 0 {
			table.Set(entry.Key, entry.Cell)
			changed = true
		}
	}
	return changed
}
