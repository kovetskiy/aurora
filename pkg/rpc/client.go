package rpc

import (
	"reflect"
	"runtime"
	"strings"

	"github.com/powerman/rpc-codec/jsonrpc2"
)

type Client struct {
	*jsonrpc2.Client
}

func NewClient(address string) *Client {
	client := jsonrpc2.NewHTTPClient(
		address,
	)

	return &Client{Client: client}
}

func (client *Client) Call(
	fn interface{},
	request interface{},
	reply interface{},
) error {
	name := getServiceMethod(fn)

	return client.Client.Call(name, request, reply)
}

func getServiceMethod(fn interface{}) string {
	fnValue := reflect.ValueOf(fn)
	fnType := fnValue.Type()

	fnKind := fnType.Kind()
	if fnKind != reflect.Func {
		panic("bug: incorrect rpc receiver passed")
	}

	receiver := runtime.FuncForPC(fnValue.Pointer()).Name()
	if strings.HasSuffix(receiver, "-fm") {
		panic("bug: func specified with closure, pass it like (*Struct).Method")
	}

	chunks := strings.Split(receiver, ".")
	if len(chunks) < 3 {
		panic(
			"bug: incorrect rpc receiver specified, " +
				"expected at least 2 dots, got: " + receiver,
		)
	}

	nameFunc := chunks[len(chunks)-1]
	nameType := chunks[len(chunks)-2]

	nameType = strings.Trim(nameType, "(*)")

	return nameType + "." + nameFunc
}
