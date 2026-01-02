//go:build !cgo

package types

import "strings"

type fallbackSegmenter struct{}

func (f *fallbackSegmenter) Cut(sentence string, hmm bool) []string {
	// Simple fallback: split by whitespace
	return strings.Fields(sentence)
}

func (f *fallbackSegmenter) CutForSearch(sentence string, hmm bool) []string {
	// Simple fallback: split by whitespace
	return strings.Fields(sentence)
}

func init() {
	Jieba = &fallbackSegmenter{}
}
