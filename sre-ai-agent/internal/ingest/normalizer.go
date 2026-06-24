package ingest

import (
	"fmt"
	"io"
)

type MultiParser struct {
	parsers []Parser
}

func NewMultiParser() *MultiParser {
	return &MultiParser{
		parsers: []Parser{
			&NginxAccessParser{},
			&NginxErrorParser{},
			&AppJSONParser{},
		},
	}
}

func (mp *MultiParser) ParseAll(r io.Reader) ([]LogEntry, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}

	var all []LogEntry
	for _, p := range mp.parsers {
		entries, err := p.Parse(readerFromBytes(data))
		if err != nil {
			continue
		}
		all = append(all, entries...)
	}
	return all, nil
}

func readerFromBytes(b []byte) io.Reader {
	return &byteReader{data: b, pos: 0}
}

type byteReader struct {
	data []byte
	pos  int
}

func (r *byteReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
