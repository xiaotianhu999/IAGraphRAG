package types

// Segmenter defines the interface for Chinese word segmentation
type Segmenter interface {
	Cut(sentence string, hmm bool) []string
	CutForSearch(sentence string, hmm bool) []string
}

// Jieba is a global instance of Chinese text segmentation tool
var Jieba Segmenter
