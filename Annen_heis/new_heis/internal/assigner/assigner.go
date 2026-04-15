package assigner

import (
	"cmp"
	"slices"

	"new_heis/internal/model"
)

func Assign(table model.RequestTable, states []model.SoftState) map[model.RequestKey]int {
	out := map[model.RequestKey]int{}
	if len(states) == 0 {
		return out
	}

	states = slices.Clone(states)
	slices.SortFunc(states, func(a, b model.SoftState) int {
		return cmp.Compare(a.ID, b.ID)
	})

	for _, entry := range table.ActiveEntries() {
		if entry.Key.Kind != model.RequestKindHall {
			continue
		}
		bestID, bestCost := states[0].ID, cost(states[0], entry.Key)
		for _, state := range states[1:] {
			if c := cost(state, entry.Key); c < bestCost || c == bestCost && state.ID < bestID {
				bestID, bestCost = state.ID, c
			}
		}
		out[entry.Key] = bestID
	}
	return out
}

func cost(state model.SoftState, key model.RequestKey) int {
	n := abs(state.Floor-key.Floor) * 4

	switch state.Behavior {
	case model.BehaviorDoorOpen:
		if state.Floor == key.Floor {
			return n - 1
		}
		return n + 2
	case model.BehaviorMoving:
		if dir := dirTo(state.Floor, key.Floor); dir != model.DirectionStop && dir != state.Direction {
			n += 3
		}
		if key.Direction != model.DirectionStop && key.Direction != state.Direction {
			n++
		}
	case model.BehaviorStuck:
		n += 10_000
	}

	return n
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func dirTo(from, to int) model.Direction {
	switch {
	case to > from:
		return model.DirectionUp
	case to < from:
		return model.DirectionDown
	default:
		return model.DirectionStop
	}
}
