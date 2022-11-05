package jrpc

import(
	"reflect"
)
type RpcError string
var NilError RpcError
var nilError = reflect.Zero(reflect.TypeOf((*error)(nil)).Elem())

func (e RpcError) Error() string {
	return string(e)
}

func ConvertError(e error) RpcError {
	if e == nil {
		return NilError
	}

	rpcErr := RpcError(e.Error())
	return rpcErr
}
