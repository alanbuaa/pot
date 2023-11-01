package types

/*
Tx 用于定义交易一系列结构与操作
*/

type Tx []byte

func (t Tx) Validate() bool {
	return true
}
