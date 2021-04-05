package docker2dot

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"dockerdot/dockerfile2llb"

	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/client/llb/imagemetaresolver"
	"github.com/moby/buildkit/solver/pb"
	digest "github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
)

func Docker2Dot(df []byte) ([]byte, error) {
	caps := pb.Caps.CapSet(pb.Caps.All())

	st, img, err := dockerfile2llb.Dockerfile2LLB(
		context.Background(),
		df,
		dockerfile2llb.ConvertOpt{
			MetaResolver: imagemetaresolver.Default(),
			LLBCaps:      &caps,
		},
	)

	if err != nil {
		return nil, err
	}

	_ = img

	def, err := st.Marshal()
	if err != nil {
		return nil, err
	}

	ops, err := loadLLB(def)
	if err != nil {
		return nil, err
	}

	var b bytes.Buffer
	writeDot(ops, &b)
	result := b.Bytes()

	return result, nil
}

type llbOp struct {
	Op         pb.Op
	Digest     digest.Digest
	OpMetadata pb.OpMetadata
}

func loadLLB(def *llb.Definition) ([]llbOp, error) {
	var ops []llbOp
	for _, dt := range def.Def {
		var op pb.Op
		if err := (&op).Unmarshal(dt); err != nil {
			return nil, errors.Wrap(err, "failed to parse op")
		}
		dgst := digest.FromBytes(dt)
		ent := llbOp{Op: op, Digest: dgst, OpMetadata: def.Metadata[dgst]}
		ops = append(ops, ent)
	}
	return ops, nil
}

func writeDot(ops []llbOp, w io.Writer) {
	fmt.Fprintln(w, "digraph {")
	defer fmt.Fprintln(w, "}")
	for _, op := range ops {
		name, shape := attr(op.Digest, op.Op)
		fmt.Fprintf(w, "  %q [label=%q shape=%q];\n", op.Digest, name, shape)
	}

	for _, op := range ops {
		for i, inp := range op.Op.Inputs {
			label := ""
			if eo, ok := op.Op.Op.(*pb.Op_Exec); ok {
				for _, m := range eo.Exec.Mounts {
					if int(m.Input) == i && m.Dest != "/" {
						label = m.Dest
					}
				}
			}
			fmt.Fprintf(w, "  %q -> %q [label=%q];\n", inp.Digest, op.Digest, label)
		}
	}
}

func attr(dgst gigest.Digest, op pb.Op) (string, string) {
	switch op := op.Op.(type) {
	case *pb.Op_Source:
		return op.Source.Identifier, "ellipse"
	case *pb.Op_Exec:
		return strings.Join(op.Exec.Meta.Args, " "), "box"
	case *pb.Op_Build:
		return "build", "box3d"
	case *pb.Op_File:
		names := []string{}

		for _, action := range op.File.Actions {
			var name string
			switch act := action.Action.(type) {
			case *pb.FileAction_Copy:
				name = fmt.Sprintf("copy{src=%s, dest=%s}", act.Copy.Src, act.Copy.Dest)
			case *pb.FileAction_Mkfile:
				name = fmt.Sprintf("mkfile{path=%s}", act.Mkfile.Path)
			case *pb.FileAction_Mkdir:
				name = fmt.Sprintf("mkdir{path=%s}", act.Mkdir.Path)
			}
			names = append(names, name)
		}
		return strings.Join(names, ","), "note"
	default:
		return dgst.String(), "plaintext"
	}
}

func getCustomString(actions []*pb.FileAction) string {
	for _, v := range actions {
		switch action := v.Action.(type) {
		case *pb.FileAction_Copy:
			return fmt.Sprintf("copy src='%v' dest='%v'", action.Copy.Src, action.Copy.Dest)
		case *pb.FileAction_Mkfile:
			return fmt.Sprintf("mkfile %+v", action.Mkfile.Path)
		case *pb.FileAction_Mkdir:
			return fmt.Sprintf("mkdir: %+v\n", action.Mkdir.Path)
		case *pb.FileAction_Rm:
			return fmt.Sprintf("rm: %+v\n", action.Rm.Path)
		default:
			return ""
		}
	}
	return ""
}
