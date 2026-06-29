package graph

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/d-init-d/d-research-cli/internal/kb"
)

const (
	DefaultMaxNodes = 150
	HardMaxNodes    = 300
)

type Node struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	X     float64
	Y     float64
}

type Layout struct {
	Seed   int64  `json:"seed"`
	Nodes  []Node `json:"nodes"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type Projection struct {
	Nodes []Node
	Edges []kb.Edge
}

func Project(root string, seed int64, maxNodes int, focus string, egoDepth int) (Projection, error) {
	if maxNodes <= 0 {
		maxNodes = DefaultMaxNodes
	}
	if maxNodes > HardMaxNodes {
		maxNodes = HardMaxNodes
	}
	edges, err := kb.LoadEdges(root)
	if err != nil {
		return Projection{}, err
	}
	approved := make([]kb.Edge, 0, len(edges))
	nodes := map[string]Node{}
	for _, e := range edges {
		if e.Status != "approved" {
			continue
		}
		approved = append(approved, e)
		nodes[e.Source] = Node{ID: e.Source, Label: e.Source}
		nodes[e.Target] = Node{ID: e.Target, Label: e.Target}
	}
	if focus != "" && egoDepth > 0 {
		approved, nodes = egoSubgraph(focus, egoDepth, approved, nodes)
	}
	nodeList := make([]Node, 0, len(nodes))
	for _, n := range nodes {
		nodeList = append(nodeList, n)
	}
	sort.Slice(nodeList, func(i, j int) bool { return nodeList[i].ID < nodeList[j].ID })
	if len(nodeList) > maxNodes {
		nodeList = nodeList[:maxNodes]
		keep := map[string]bool{}
		for _, n := range nodeList {
			keep[n.ID] = true
		}
		filtered := make([]kb.Edge, 0)
		for _, e := range approved {
			if keep[e.Source] && keep[e.Target] {
				filtered = append(filtered, e)
			}
		}
		approved = filtered
	}
	layout := seededLayout(nodeList, seed)
	return Projection{Nodes: layout.Nodes, Edges: approved}, nil
}

func egoSubgraph(focus string, depth int, edges []kb.Edge, nodes map[string]Node) ([]kb.Edge, map[string]Node) {
	frontier := map[string]int{focus: 0}
	for round := 0; round < depth; round++ {
		next := map[string]int{}
		for _, e := range edges {
			if d, ok := frontier[e.Source]; ok && d == round {
				next[e.Target] = round + 1
			}
			if d, ok := frontier[e.Target]; ok && d == round {
				next[e.Source] = round + 1
			}
		}
		for k, v := range next {
			frontier[k] = v
		}
	}
	keep := map[string]bool{focus: true}
	for id := range frontier {
		keep[id] = true
	}
	filtered := make([]kb.Edge, 0)
	for _, e := range edges {
		if keep[e.Source] && keep[e.Target] {
			filtered = append(filtered, e)
		}
	}
	outNodes := map[string]Node{}
	for id := range keep {
		if n, ok := nodes[id]; ok {
			outNodes[id] = n
		} else {
			outNodes[id] = Node{ID: id, Label: id}
		}
	}
	return filtered, outNodes
}

func seededLayout(nodes []Node, seed int64) Layout {
	width, height := 80, 24
	if len(nodes) == 0 {
		return Layout{Seed: seed, Width: width, Height: height}
	}
	for i := range nodes {
		angle := float64((int(seed)+i*37)%360) * 0.0174533
		radius := 10 + float64(i%8)
		nodes[i].X = 40 + radius*cosApprox(angle)
		nodes[i].Y = 12 + radius*sinApprox(angle)/2
	}
	return Layout{Seed: seed, Nodes: nodes, Width: width, Height: height}
}

func cosApprox(x float64) float64 {
	// Taylor-ish small approximation is enough for deterministic layout tests.
	x2 := x * x
	return 1 - x2/2 + x2*x2/24
}

func sinApprox(x float64) float64 {
	return cosApprox(x - 1.5708)
}

func RenderASCII(proj Projection, selected string) string {
	if len(proj.Nodes) == 0 {
		return "(trống)"
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("nodes=%d edges=%d seed-layout\n", len(proj.Nodes), len(proj.Edges)))
	for _, n := range proj.Nodes {
		mark := " "
		if n.ID == selected {
			mark = "*"
		}
		b.WriteString(fmt.Sprintf("%s[%s] (%.1f,%.1f)\n", mark, n.Label, n.X, n.Y))
	}
	return strings.TrimRight(b.String(), "\n")
}

func CachePath(metaDir string, seed int64, hash string) string {
	return filepath.Join(metaDir, fmt.Sprintf("graph-layout-%s-%d.json", hash[:8], seed))
}

func LoadOrComputeLayout(metaDir, root string, seed int64, hash string) (Layout, error) {
	path := CachePath(metaDir, seed, hash)
	if data, err := os.ReadFile(path); err == nil {
		var layout Layout
		if json.Unmarshal(data, &layout) == nil {
			return layout, nil
		}
	}
	proj, err := Project(root, seed, DefaultMaxNodes, "", 0)
	if err != nil {
		return Layout{}, err
	}
	layout := seededLayout(proj.Nodes, seed)
	data, _ := json.MarshalIndent(layout, "", "  ")
	_ = os.WriteFile(path, data, 0o644)
	return layout, nil
}