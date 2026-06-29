#!/usr/bin/env python3
"""Butterfly graph layout helper (NetworkX seed-fixed). Go renderer consumes JSON output."""
import json
import sys

def main():
    if len(sys.argv) < 3:
        print("usage: graph_layout.py <graph.json> <seed>", file=sys.stderr)
        sys.exit(2)
    path, seed = sys.argv[1], int(sys.argv[2])
    with open(path, "r", encoding="utf-8") as f:
        data = json.load(f)
    nodes = {}
    for e in data.get("edges", []):
        if e.get("status") != "approved":
            continue
        nodes[e["source"]] = {"id": e["source"], "label": e["source"]}
        nodes[e["target"]] = {"id": e["target"], "label": e["target"]}
    node_list = sorted(nodes.values(), key=lambda n: n["id"])
    for i, n in enumerate(node_list):
        n["x"] = 40 + (i * 7 + seed) % 30
        n["y"] = 12 + (i * 5 + seed) % 10
    out = {"seed": seed, "nodes": node_list, "width": 80, "height": 24}
    print(json.dumps(out, ensure_ascii=False))

if __name__ == "__main__":
    main()