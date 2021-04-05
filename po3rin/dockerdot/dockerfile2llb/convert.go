package dockerfile2llb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/hcl/parser"
	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/solver/pb"
	"github.com/moby/buildkit/util/apicaps"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

const (
	emptyImageName          = "scratch"
	defaultContextLocalName = "context"
	historyComment          = "buildkit.dockerfile.v0"
	DefaultCopyImage        = "docker/dockerfile-copy:v0.1.9"
)

type ConvertOpt struct {
	Target            string
	MetaResolver      llb.ImageMetaResolver
	BuildArgs         map[string]string
	Labels            map[string]string
	SessionID         string
	BuildContext      *llb.State
	Excludes          []string
	IgnoreCache       []string
	CacheIDNamespace  string
	ImageResolveMode  llb.ResolveMode
	TargetPlatform    *specs.Platform
	BuildPlatforms    []specs.Platform
	PrefixPlatform    bool
	ExtraHosts        []llb.HostIP
	ForceNetMode      pb.NetMode
	OverrideCopyImage string
	LLBCaps           *apicaps.CapSet
	ContextLocalName  string
}

func Dockerfile2LLB(ctx context.Context, dt []byte, opt ConvertOpt) (*llb.State, *Image, error) {
	if len(dt) == 0 {
		return nil, nil, errors.Errorf("the Dockerfile cannot be empty")
	}
	if opt.ContextLocalName == "" {
		opt.ContextLocalName = defaultContextLocalName
	}

	platformOpt := buildPlatformOpt(&opt)

	optMetaArgs := getPlatformArgs(platformOpt)
	for i, arg := range optMetaArgs {
		optMetaArgs[i] = setKValue(arg, opt.BuildArgs)
	}

	dockerfile, err := parser.Parse(bytes.NewReader(dt))
	if err != nil {
		return nil, nil, err
	}

	shlex := shell.NewLex(dockerfile.EscapeToken)

	for _, metaArg := range metaArgs {
		if metaArg.Value != nil {
			*metaArg.Value, _ = shlex.ProcessWordWithMap(*metaArg.Value, metaArgsToMap(optMetaArgs))
		}
		optMetaArgs = append(optMetaArgs, setKVValue(metaArg.KeyValuePairOptional, opt.BuildArgs))
	}

	metaResolver := opt.MetaResolver
	if metaResolver == nil {
		metaResolver = imagemetaresolver.Default()
	}

	allDispatchStates := newDispatchStates()

	for i, st := range stages {
		name, err := shlex.ProcessWordWithMap(st.BaseName, metaArgsToMap(optMetaArgs))
		if err != nil {
			return nil, nil, err
		}
		if name == "" {
			return nil, nil, errors.Errorf("base name (%s) should not be blank", st.BaseName)
		}
		st.BaseName = name
		ds := &dispatchState{
			stage:          st,
			deps:           make(map[*dispatchState]struct{}),
			ctxPaths:       make(map[string]struct{}),
			stageName:      st.Name,
			prefixPlatform: opt.PrefixPlatform,
		}

		if st.Name == "" {
			ds.stageName = fmt.Sprintf("stage-%d", i)
		}

		if v := st.Platform; v != "" {
			v, err := shlex.ProcessWordWithMap(v, metaAArgsToMap(optMetaArgs))
			if err != nil {
				return nil, nil, errors.Wrapf(err, "failed to process arguments for platform %s", v)
			}

			p, err := platforms.Parse(v)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "failed to parse platform %s", v)
			}
			ds.platform = &p
		}
		allDispatchStates.addState(ds)

		total := 0
		if ds.stage.BaseName != emptyImageName && ds.base == nil {
			total = 1
		}

		for _, cmd := range ds.stage.Commands {
			switch cmd.(type) {
			case *instructions.AddCommand, *instructions.CopyCommand, *instructions.RunCommand:
				total++
			case *instructions.WorkdirCommand:
				if useFileOp(opt.BuildArgs, opt.LLBCaps) {
					total++
				}
			}
		}
		ds.cmdTotal = total
		if opt.IgnoreCache != nil {
			if len(opt.IgnoreCache) == 0 {
				ds.ignoreCache = true
			} else if st.Name != "" {
				for _, n := range opt.IgnoreCache {
					if strings.EqualFold(n, st.Name) {
						ds.ignoreCache = true
					}
				}
			}
		}
	}

	var target *dispatchState
	if opt.Target == "" {
		target = allDispatchStates.lastTarget()
	} else {
		var ok bool
		target, ok = allDispatchStates.findStateByName(opt.Target)
		if !ok {
			return nil, nil, errors.Errorf("target stage %s could not be found", opt.Target)
		}
	}

	for _, d := range allDispatchStates.states {
		d.commands = make([]command, len(d.stage.Commands))
		for i, cmd := range d.stage.Commands {
			newCmd, err := toCommand(cmd, allDispatchStates)
			if err != nil {
				return nil, nil, err
			}
			d.commands[i] = newCmd
			for _, src := range newCmd.sources {
				if src != nil {
					d.deps[src] = struct{}{}
					if src.unregistered {
						allDispatchStates.addState(src)
					}
				}
			}
		}
	}

	if has, state := hasCircularDependency(allDispatchStates.states); has {
		return nil, nil, fmt.Errorf("circular dependency detected on stage: %s", state.stageName)
	}

	if len(allDispatchStates.states) == 1 {
		allDispatchStates.states[0].stageName = ""
	}

	eg, ctx := errgroup.WithContext(ctx)
	for i, d := range allDispatchStates.states {
		reachable := isReachable(target, d)
		if d.base == nil {
			if d.stage.BaseName == emptyImageName {
				d.state = llb.Scratch()
				d.image = emptyImage(platformOpt.targetPlatform)
				continue
			}
			func(i int, d *dispatchState) {
				eg.Go(func() error {
					ref, err := reference.ParseNormalizedNamed(d.stage.BaseName)
					if err != nil {
						return errors.Wrapf(err, "failed to parse stage name %q", d.stage.BaseName)
					}
					platform := d.platform
					if platform == nil {
						platform = &platformOpt.targetPlatform
					}
					d.stage.BaseName = reference.TagNameOnly(ref).String()
					var isScratch bool
					if metaResolver != nil && reachable && !d.unregistered {
						prefix := "["
						if opt.PrefixPlatform && platform != nil {
							prefix += platforms.Format(*platform) + " "
						}
						prefix += "internal]"
						dgst, dt, err := metaResolver.ResolveImageConfig(ctx, BaseName, gw.ResolveImageConfigOpt{
							Platform:    platform,
							ResolveMode: opt.ImageResolveMode.String(),
							LogName:     fmt.Sprintf("%s load metadata for %s", prefix, d.stage.BaseName),
						})
						if err == nil {
							var img Image
							if err := json.Unmarshal(dt, &img); err != nil {
								return err
							}

							img.Created = nil
							if d.platform == nil && platformOpt.implicitTarget {
								p := autoDetectPlatform(img, *platform, platformOpt.buildPlatforms)
								platform = &p
							}
							d.image = img
							if dgst != "" {
								ref, err = reference.WithDigest(ref, dgst)
								if err != nil {
									return err
								}
							}
							d.stage.BaseName = ref.String()
							if len(img.RootFS.DiffIDs) == 0 {
								isScratch = true
								for _, h := range img.History {
									if !h.EmptyLayer {
										isScratch = false
										break
									}
								}
							}
						}
					}
				})
			}
		}
	}
}
