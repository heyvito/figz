package decoder

import (
	"archive/zip"
	"bytes"
	"cmp"
	"compress/flate"
	"fmt"
	"github.com/heyvito/figz/fig"
	"github.com/heyvito/gokiwi"
	"io"
	"os"
	"slices"
)

type Document struct {
	Version uint32
	Root    *fig.NodeChange
	Blobs   []*fig.Blob
}

func Decode(path string) (*Document, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("unable to open file: %w", err)
	}
	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("unable to stat file: %w", err)
	}
	if stat.Size() < 8 {
		return nil, fmt.Errorf("file size too small: %d", stat.Size())
	}

	header := make([]byte, 8)
	n, err := file.Read(header)
	if err != nil {
		return nil, fmt.Errorf("unable to read header: %w", err)
	}
	if n < 8 {
		return nil, fmt.Errorf("short read reading header: %d. Expected 8", n)
	}

	_, err = file.Seek(0, 0)
	if err != nil {
		return nil, fmt.Errorf("unable to seek file: %w", err)
	}

	if header[0] == 'P' && header[1] == 'K' {
		return decodeJamFromZip(file, stat.Size())
	} else if string(header) == "fig-jam." {
		data, err := io.ReadAll(file)
		if err != nil {
			return nil, fmt.Errorf("unable to read fig-jam: %w", err)
		}
		return decodeFigJam(data)
	} else {
		return nil, fmt.Errorf("unsupported file format")
	}
}

func decodeJamFromZip(file *os.File, size int64) (*Document, error) {
	r, err := zip.NewReader(file, size)
	if err != nil {
		return nil, fmt.Errorf("unable to open jam file: %w", err)
	}
	ok := false
	uncompressedSize := uint64(0)
	for _, v := range r.File {
		if v.Name == "canvas.fig" {
			uncompressedSize = v.UncompressedSize64
			ok = true
			break
		}
	}
	if !ok {
		return nil, fmt.Errorf("unable to locate internal canvas from jam file")
	}

	jam, err := r.Open("canvas.fig")
	if err != nil {
		return nil, fmt.Errorf("unable to open canvas from jam file: %w", err)
	}
	data := make([]byte, uncompressedSize)
	readSize, err := jam.Read(data)
	if err != nil {
		return nil, fmt.Errorf("unable to decompress jam canvas: %w", err)
	}
	if uint64(readSize) != uncompressedSize {
		return nil, fmt.Errorf("divergent read and uncompressed size: expected %d, found %d", uncompressedSize, readSize)
	}

	return decodeFigJam(data)
}

func decodeFigJam(data []byte) (*Document, error) {
	header := data[0:8]
	if string(header) != "fig-jam." {
		return nil, fmt.Errorf("invalid header; expected 'fig-jam.', got %s", header)
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
		return nil, fmt.Errorf("invalid chunk size; expected at least 2, got %d", len(chunks))
	}

	zr := flate.NewReader(bytes.NewReader(chunks[1]))
	encodedData, err := io.ReadAll(zr)
	if err != nil {
		return nil, fmt.Errorf("error reading first chunk data: %v", err)
	}

	struc, err := fig.DecodeMessage(gokiwi.NewBuffer(encodedData))
	if err != nil {
		return nil, fmt.Errorf("error decoding message chunk: %v", err)
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

	return &Document{
		Version: version,
		Root:    nodes["0:0"],
		Blobs:   blobs,
	}, nil
}
