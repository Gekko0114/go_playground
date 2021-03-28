package shared

import (
	"grpc/proto"

	"golang.org/x/net/context"
)

type GRPCClient struct{ client proto.KVClient }

func (m *GRPCClient) Put(key string, value []byte) error {
	_, err := m.client.Put(context.Background(), &proto.PutRequest{
		Key:   key,
		Value: value,
	})
	return err
}

func (m *GRPCClient) Get(key string) ([]byte, error) {
	resp, err := m.client.Get(context.Background(), &proto.GetRequest{
		Key: key,
	})

	if err != nil {
		return nil, err
	}

	return resp.Value, nil
}

type GRPCServer struct {
	Impl KV
}

func (m *GRPCServer) Put(
	ctx context.Context,
	req *proto.PutRequest) (*proto.Empty, error) {
	return &proto.Empty{}, m.Impl.Put(req.Key, req.Value)
}

func (m *GRPCServer) Get(
	ctx context.Context,
	req *proto.GetRequest) (*proto.GetResponse, error) {
	v, err := m.Impl.Get(req.Key)
	return &proto.GetResponse{Value: v}, err
}
