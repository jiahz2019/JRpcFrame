package utils

import (
	"bytes"
	"encoding/gob"
)
/*
    brief:编码数据
	param [in] data:需要编码的数据
*/
func EncodeObject(data interface{}) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(data)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
/*
    brief:解码数据
	param [in] data:需要解码的数据
*/
func DecodeObject(data []byte, to interface{}) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	return dec.Decode(to)
}