package shared

import (
	"net/rpc"
)

type RPCClient struct{ client *rpc.Client }

func (m *RPCClient) Put(key string, value []byte) error {
	var resp interface{}
	return m.client.Call("Plugin.Put", map[string]interface{}{
		"key":   key,
		"value": value,
	}, &resp)
}

func (m *RPCClient) Get(key string) ([]byte, error) {
	var resp []byte
	err := m.client.Call("Plugin.Get", key, &resp)
	return resp, err
}

type RPCServer struct {
	Impl KV
}

func (m *RPCServer) Put(args map[string]interface{}, resp *interface{}) error {
	return m.Impl.Put(args["key"].(string), args["value"].([]byte))
}

func (m *RPCServer) Get(key string, resp *[]byte) error {
	v, err := m.Impl.Get(key)
	*resp = v
	return err
}
