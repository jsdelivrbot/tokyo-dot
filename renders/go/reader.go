package main

import (
	"fmt"
	"io"
	"log"
	"regexp"
	"strconv"
	"strings"

	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/encoding"
	edot "gonum.org/v1/gonum/graph/encoding/dot"
	"gonum.org/v1/gonum/graph/formats/dot"
	"gonum.org/v1/gonum/graph/formats/dot/ast"
	"gonum.org/v1/gonum/graph/simple"
)

func newTrainGraph() *trainGraph {
	return &trainGraph{DirectedGraph: simple.NewDirectedGraph()}
}

type trainGraph struct {
	*simple.DirectedGraph
}

func (g *trainGraph) StationNode(sid string) *station {
	for _, n := range g.DirectedGraph.Nodes() {
		s := n.(*station)
		if sid == s.StationID() {
			return s
		}
	}
	return nil
}

func (g *trainGraph) Weight(s, d int64) (float64, bool) {
	if !g.HasEdgeBetween(s, d) {
		return 0, false
	}
	e := g.Edge(s, d).(*edge)
	return e.time, true
}

func (g *trainGraph) NewNode() graph.Node {
	return &station{Node: g.DirectedGraph.NewNode()}
}

func (g *trainGraph) NewEdge(from, to graph.Node) graph.Edge {
	return &edge{Edge: g.DirectedGraph.NewEdge(from, to)}
}

func (g *trainGraph) String() string {
	return fmt.Sprintf("%d platforms", len(g.Nodes()))
}

var quotedLabel = regexp.MustCompile(`"(?P<content>.*?)"`)

type edge struct {
	graph.Edge
	time   float64
	oneWay bool
	cars   []string
}

func (e *edge) SetAttribute(attr encoding.Attribute) error {
	if attr.Key == "len" {
		time, err := strconv.ParseFloat(attr.Value, 32)
		if err != nil {
			return err
		}
		e.time = time
	} else if attr.Key == "return" && attr.Value == "no" {
		e.oneWay = true
	} else if attr.Key == "label" {
		value := quotedLabel.ReplaceAllString(attr.Value, "${content}")
		if value != "-" && value != "車内" {
			transfer := strings.SplitN(value, "|", 2)
			dir := strings.SplitN(transfer[0], ";", 2)
			if len(dir) == 2 {
				fmt.Println("transfer car by direction")
			}
			e.cars = strings.Split(dir[0], ",")
		}
	}
	return nil
}

type station struct {
	graph.Node
	stationID string
}

func (s *station) String() string {
	return s.stationID
}

func (s *station) StationID() string {
	return s.stationID
}

func (s *station) SetDOTID(id string) {
	s.stationID = id
}

func convertToTrainGraph(graph *ast.Graph) *trainGraph {
	dst := newTrainGraph()
	if err := edot.Unmarshal([]byte(graph.String()), dst); err != nil {
		log.Fatal(err)
	}
	for _, e := range dst.Edges() {
		// Make edges two-way unless specified individually:
		if f := e.(*edge); !f.oneWay {
			// Return edge already exists in graph:
			if dst.HasEdgeFromTo(f.To().ID(), f.From().ID()) {
				continue
			}
			r := dst.NewEdge(f.To(), f.From()).(*edge)
			r.time = f.time
			dst.SetEdge(r)
		}
	}
	return dst
}

func readGraph(f io.Reader) *trainGraph {
	g, err := dot.Parse(f)
	if err != nil {
		log.Fatal(err)
	}
	return convertToTrainGraph(g.Graphs[0])
}
