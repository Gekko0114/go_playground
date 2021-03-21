package user

//https://github.com/golang/mock/tree/master/sample
//go:generate mockgen -destination mock_user/mock_user.go sample Index,Embed,Embedded

import (
	"io"

	btz "bytes"
	"hash"
	"log"
	"net"
	"net/http"

	// Two imports with the same base name.
	t1 "html/template"

	t2 "text/template"

	"sample/imp1"

	renamed2 "sample/imp2"

	. "sample/imp3"
	"sample/imp4"
)

type Index interface {
	Get(key string) interface{}
	GetTwo(key1, key2 string) (v1, v2 interface{})
	Put(key string, value interface{})

	Summary(buf *btz.Buffer, w io.Writer)
	Other() hash.Hash
	Templates(a t1.CSS, b t2.FuncMap)
	Anon(string)

	ForeignOne(imp1.Imp1)
	ForeignTwo(renamed2.Imp2)
	ForeignThree(Imp3)
	ForeignFour(imp4.Imp4)

	NillableRet() error
	ConcreteRet() chan<- bool
	Ellip(fmt string, args ...interface{})
	EllipOnly(...string)
	Ptr(arg *int)
	Slice(a []int, b []byte) [3]int

	Chan(a chan int, b chan<- hash.Hash)
	Func(f func(http.Request) (int, bool))
	Map(a map[int]hash.Hash)

	Struct(a struct{})
	StructChan(a chan struct{})
}

type Embed interface {
	RegularMethod()
	Embedded
	imp1.ForeignEmbedded
}

type Embedded interface {
	EmbeddedMethod()
}

var _ net.Addr

func Remember(index Index, keys []string, values []interface{}) {
	for i, k := range keys {
		index.Put(k, values[i])
	}
	err := index.NillableRet()
	if err != nil {
		log.Fatalf("woah! %v", err)
	}
	if len(keys) > 0 && keys[0] == "a" {
		index.Ellip("%d", 0, 1, 1, 2, 3)
		index.Ellip("%d", 1, 3, 6, 10, 15)
		index.EllipOnly("arg")
	}
}

func GrabPointer(index Index) int {
	var a int
	index.Ptr(&a)
	return a
}
