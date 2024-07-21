package main

import (
	"bytes"
	"cmp"
	"compress/flate"
	"encoding/binary"
	"figz/fig"
	"figz/tikz"
	"github.com/heyvito/gokiwi"
	"slices"

	//"figz/fig"
	"fmt"
	"io"
	"os"
)

type View struct {
	buffer []byte
	cursor int
}

var enc = binary.LittleEndian

func (v *View) Uint32(at int) uint32 {
	return enc.Uint32(v.buffer[at:])
}

type Document struct {
	Version uint32
	Root    *fig.NodeChange
	Blobs   []*fig.Blob
}

func main() {
	path := "/Users/vitosartori/Downloads/fml/canvas.fig"
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}

	header := data[0:8]
	if string(header) != "fig-jam." {
		panic("Invalid header")
	}

	v := View{buffer: data}
	version := v.Uint32(8)
	var chunks [][]byte
	offset := 12

	for offset < len(data) {
		chunkSize := v.Uint32(offset)
		offset += 4
		chunks = append(chunks, data[offset:offset+int(chunkSize)])
		offset += int(chunkSize)
	}

	if len(chunks) < 2 {
		panic("Not enough chunks")
	}

	//fmt.Printf("Version: %d\n", version)
	//fmt.Printf("Chunk count: %d\n", len(chunks))

	zr := flate.NewReader(bytes.NewReader(chunks[1]))
	encodedData, err := io.ReadAll(zr)
	if err != nil {
		panic("Failed decoding chunk 1: " + err.Error())
	}

	struc, err := fig.DecodeMessage(gokiwi.NewBuffer(encodedData))
	if err != nil {
		panic(err)
	}

	nodeChanges, blobs := struc.NodeChanges, struc.Blobs
	nodes := map[string]*fig.NodeChange{}

	for _, node := range nodeChanges {
		nodes[fmt.Sprintf("%d:%d", node.Guid.SessionId, node.Guid.LocalId)] = node
	}

	for _, node := range nodeChanges {
		if node.ParentIndex != nil {
			sessionID, localID := node.ParentIndex.Guid.SessionId, node.ParentIndex.Guid.LocalId
			parent, ok := nodes[fmt.Sprintf("%d:%d", sessionID, localID)]
			if ok {
				parent.Children = append(parent.Children, node)
			}
		}
	}

	for _, node := range nodeChanges {
		if node.Children != nil {
			slices.SortFunc(node.Children, func(a, b *fig.NodeChange) int {
				return cmp.Compare(b.ParentIndex.Position, a.ParentIndex.Position)
			})
		}
	}

	for _, node := range nodes {
		node.ParentIndex = nil
	}

	doc := &Document{
		Version: version,
		Root:    nodes["0:0"],
		Blobs:   blobs,
	}

	str := tikz.NewCompiler(doc.Root.Children[1], nil)
	fmt.Println(str)
}
