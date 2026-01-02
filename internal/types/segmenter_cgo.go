//go:build cgo

package types

import "github.com/yanyiwu/gojieba"

func init() {
	Jieba = gojieba.NewJieba()
}
