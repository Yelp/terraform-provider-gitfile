package gitfile

import (
	"fmt"
	"strconv"
	"strings"
)

type handle struct {
	kind string
	hash int
	path string
}

func (h *handle) String() string {
	return fmt.Sprintf("%s %d %s", h.kind, h.hash, h.path)
}

func parseHandle(s string) *handle {
	splits := strings.SplitN(s, " ", 3)
	var hash int
	var err error
	if len(splits) < 3 {
		panic(fmt.Sprintf("Could not split handle into 3 parts: '%s'", s))
	}
	if hash, err = strconv.Atoi(splits[1]); err != nil {
		panic(err)
	}
	return &handle{
		kind: splits[0],
		hash: hash,
		path: splits[2],
	}
}
