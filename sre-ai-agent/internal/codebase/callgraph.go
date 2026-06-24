package codebase

type CallGraph struct {
	idx *Index
}

func NewCallGraph(idx *Index) *CallGraph {
	return &CallGraph{idx: idx}
}

func (cg *CallGraph) BFS(start string, depth int) []Function {
	visited := make(map[string]bool)
	var result []Function
	queue := []string{start}

	for level := 0; level <= depth && len(queue) > 0; level++ {
		next := make([]string, 0)
		for _, id := range queue {
			if visited[id] {
				continue
			}
			visited[id] = true

			fn, ok := cg.idx.Functions[id]
			if !ok {
				continue
			}
			if id != start {
				result = append(result, fn)
			}

			for _, callee := range fn.Calls {
				if !visited[callee] {
					next = append(next, callee)
				}
			}
		}
		queue = next
	}

	return result
}
