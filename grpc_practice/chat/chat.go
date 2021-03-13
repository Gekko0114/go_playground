package chat

import (
	"fmt"
	"log"
	reflect "reflect"

	"golang.org/x/net/context"
)

type Server struct {
}

func (s *Server) SayHello(ctx context.Context, in *Message) (*Message, error) {
	log.Printf("Receive message body from client: %s", in.Body)
	response_list := []string{"Zelda", "Link", "Ganondorf"}
	result, err := contains(in.Body, response_list)
	if err != nil {
		return &Message{Body: "error"}, err
	} else if result == true {
		return &Message{Body: "The Legend of Zelda"}, nil
	}
	return &Message{Body: "Hello From the Server!" + in.Body}, nil
}

func contains(target interface{}, list interface{}) (bool, error) {
	switch list.(type) {
	default:
		return false, fmt.Errorf("%v is an unsupported type", reflect.TypeOf(list))
	case []int:
		revert := list.([]int)
		for _, r := range revert {
			if target == r {
				return true, nil
			}
		}
		return false, nil
	case []uint64:
		revert := list.([]uint64)
		for _, r := range revert {
			if target == r {
				return true, nil
			}
		}
		return false, nil
	case []string:
		revert := list.([]string)
		for _, r := range revert {
			if target == r {
				return true, nil
			}
		}
		return false, nil
	}
	return false, fmt.Errorf("processing failed")
}
