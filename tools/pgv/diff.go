package main

import "sort"

func ComputeDiff(base IR, head IR, baseHash, headHash string) IRDelta {
	delta := IRDelta{
		SchemaVersion:   "pgv.ir.delta.v1",
		IRSchemaVersion: IrSchemaVersion,
		BaseHash:        baseHash,
		HeadHash:        headHash,
		Status:          "PASS",
	}

	var added, removed, unchanged []IRNode
	var moved []Move

	for id, oldNode := range base.Nodes {
		newNode, exists := head.Nodes[id]
		if !exists {
			removed = append(removed, oldNode)
			continue
		}
		if oldNode.Path != newNode.Path {
			moved = append(moved, Move{
				NodeID:   id,
				FromPath: oldNode.Path,
				ToPath:   newNode.Path,
			})
		} else {
			unchanged = append(unchanged, newNode)
		}
	}

	for id, newNode := range head.Nodes {
		if _, exists := base.Nodes[id]; !exists {
			added = append(added, newNode)
		}
	}

	sortIRNodes(added)
	sortIRNodes(removed)
	sortIRNodes(unchanged)
	sortMoves(moved)

	delta.Added = added
	delta.Removed = removed
	delta.Moved = moved
	delta.Unchanged = unchanged
	delta.Summary = Summary{
		AddedCount:     len(added),
		RemovedCount:   len(removed),
		MovedCount:     len(moved),
		UnchangedCount: len(unchanged),
	}

	return delta
}

func sortIRNodes(nodes []IRNode) {
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].ID == nodes[j].ID {
			return nodes[i].Path < nodes[j].Path
		}
		return nodes[i].ID < nodes[j].ID
	})
}

func sortMoves(moves []Move) {
	sort.Slice(moves, func(i, j int) bool {
		if moves[i].NodeID == moves[j].NodeID {
			return moves[i].FromPath < moves[j].FromPath
		}
		return moves[i].NodeID < moves[j].NodeID
	})
}
