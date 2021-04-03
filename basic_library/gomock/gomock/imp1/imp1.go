package imp1

import "bufio"

type Imp1 struct{}

type ImpT int

type ForeignEmbedded interface {
	ForeignEmbeddedMethod() *bufio.Reader
	ImplicitPackage(s string, t ImpT, st []ImpT, pt *ImpT, ct chan ImpT)
}
