package ast

import (
	"bytes"
	"github.com/onflow/cadence/runtime/common"
)

type Comments struct {
	Leading  []Comment
	Trailing []Comment
}

type Comment struct {
	source []byte
}

func NewComment(memoryGauge common.MemoryGauge, source []byte) Comment {
	// TODO(preserve-comments): Track memory usage
	return Comment{
		source: source,
	}
}

var blockCommentDocStringPrefix = []byte("/**")
var blockCommentStringPrefix = []byte("/*")
var lineCommentDocStringPrefix = []byte("///")
var lineCommentStringPrefix = []byte("//")
var blockCommentStringSuffix = []byte("*/")

func (c Comment) Multiline() bool {
	return bytes.HasPrefix(c.source, blockCommentStringPrefix)
}

func (c Comment) Doc() bool {
	if c.Multiline() {
		return bytes.HasPrefix(c.source, blockCommentDocStringPrefix)
	} else {
		return bytes.HasPrefix(c.source, lineCommentDocStringPrefix)
	}
}

// Text without opening/closing comment characters /*, /**, */, //
func (c Comment) Text() []byte {
	withoutPrefixes := cutOptionalPrefixes(c.source, [][]byte{
		blockCommentDocStringPrefix, // check before blockCommentStringPrefix
		blockCommentStringPrefix,
		lineCommentDocStringPrefix, // check before lineCommentStringPrefix
		lineCommentStringPrefix,
	})
	return cutOptionalSuffixes(withoutPrefixes, [][]byte{
		blockCommentStringSuffix,
	})
}

func cutOptionalPrefixes(input []byte, prefixes [][]byte) (output []byte) {
	output = input
	for _, prefix := range prefixes {
		cut, _ := bytes.CutPrefix(output, prefix)
		output = cut
	}
	return
}

func cutOptionalSuffixes(input []byte, suffixes [][]byte) (output []byte) {
	output = input
	for _, suffix := range suffixes {
		cut, _ := bytes.CutSuffix(output, suffix)
		output = cut
	}
	return
}
