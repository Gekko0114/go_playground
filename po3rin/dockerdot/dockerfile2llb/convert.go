package dockerfile2llb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"strconv"
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
					if isScratch {
						d.state = llb.Scratch()
					} else {
						d.state = llb.image(d.stage.BaseName, dfCmd(d.stage.SourceCode), llb.Platform(*platform), opt.ImageResolveMode, llb.WithCustomName(prefixCommand(d, "FROM "+d.stage.BaseName, opt.PrefixPlatform, platform)))
					}
					d.platform = platform
					return nil
				})
			}(i, d)
		}
	}

	if err := eg.Wait(); err != nil {
		return nil, nil, err
	}

	buildContext := &mutableOutput{}
	ctxPaths := map[string]struct{}{}

	for _, d := range allDispatchStates.states {
		if !isReachable(target, d) {
			continue
		}
		if d.base != nil {
			d.state = d.base.state
			d.platform = d.base.platform
			d.image = clone(d.base.image)
		}

		if _, ok := shell.BuildEnvs(d.image.Config.Env)["PATH"]; !ok {
			d.image.Config.Env = append(d.image.Config.Env, "PATH="+system.DefaultPathEnv)
		}

		for _, env := range d.image.Config.Env {
			k, v := parseKeyValue(env)
			d.state = d.state.AddEnv(k, v)
		}
		if d.image.Config.WorkingDir != "" {
			if err = dispatchWorkdir(d, &instructions.WorkdirCommand{Path: d.image.Config.WorkingDir}, false, nil); err != nil {
				return nil, nil, err
			}
		}
		if d.image.Config.User != "" {
			if err = dispatchUser(d, &instructions.UserCommand{User: d.image.Config.User}, false); err != nil {
				return nil, nil, err
			}
		}

		d.state = d.state.Network(opt.ForceNetMode)

		opt := dispatchOpt{
			allDispatchStates: allDispatchStates,
			metaArgs:          opt.MetaArgs,
			shlex:             shlex,
			sessionID:         opt.SessionID,
			buildContext:      llb.NewState(buildContext),
			proxyEnv:          proxyEnv,
			cacheIDNamespace:  opt.CacheIDNamespace,
			buildPlatforms:    platformOpt.buildPlatforms,
			targetPlatform:    platformOpt.targetPlatform,
			extraHosts:        opt.ExtraHosts,
			copyImage:         opt.OverrideCopyImage,
			llbCaps:           opt.LLBCaps,
		}
		if opt.copyImage == "" {
			opt.copyImage = DefaultCopyImage
		}

		if err = dispatchOnBuild(d, d.image.Config.OnBuild, opt); err != nil {
			return nil, nil, err
		}

		for _, cmd := range d.commands {
			if err := dispatch(d, cmd, opt); err != nil {
				return nil, nil, err
			}
		}

		for p := range d.ctxPaths {
			ctxPaths[p] = struct{}{}
		}
	}

	if len(opt.Labels) != 0 && target.image.Config.Labels == nil {
		target.image.Config.Labels = make(map[string]string, len(opt.Labels))
	}
	for k, v := range opt.Labels {
		target.image.Config.Labels[k] = v
	}

	opts := []llb.LocalOption{
		llb.SessionID(opt.SessionID),
		llb.ExcludePatterns(opt.Excludes),
		llb.SharedKeyHint(opt.ContextLocalName),
		WithInternalName("load build context"),
	}

	if includePatterns := normalizeContextPaths(ctxPaths); includePatterns != nil {
		opts = append(opts, llb.FollowPaths(includePatterns))
	}

	bc := llb.Local(opt.ContextLocalName, opts...)
	if opt.BuildContext != nil {
		bc = *opt.BuildContext
	}
	buildContext.Output = bc.Output()

	defaults := []llb.ConstraintsOpt{
		llb.Platform(platformOpt.targetPlatform),
	}

	if opt.LLBCaps != nil {
		defaults = append(defaults, llb.WithCaps(*opt.LLBCaps))
	}
	st := target.state.SetMarshalDefaults(defaults...)

	if !platformOpt.implicitTarget {
		target.image.OS = platformOpt.targetPlatform.OS
		target.image.Architecture = platformOpt.targetPlatform.Architecture
		target.image.Variant = platformOpt.targetPlatform.Variant
	}

	return &st, &target.image, nil
}

func metaArgsToMap(metaArgs []instructions.KeyValuePairOptional) map[string]string {
	m := map[string]string{}

	for _, arg := range metaArgs {
		m[arg.Key] = arg.ValueString()
	}
	return m
}

func toCommand(ic instructions.Command, allDispatchStates *dispatchStates) (command, error) {
	cmd := command{Command: ic}
	if c, ok := ic.(*instructions.CopyCommand); ok {
		if c.From != "" {
			var stn *dispatchState
			index, err := strconv.Atoi(c.From)
			if err != nil {
				stn, ok = allDispatchStates.findStateByName(c.From)
				if !ok {
					stn = &dispatchState{
						stage:        instructions.Stage{BaseName: c.From},
						deps:         make(map[*dispatchState]struct{}),
						unregistered: true,
					}
				}
			} else {
				stn, err = allDispatchStates.findStateByIndex(index)
				if err != nil {
					return command{}, err
				}
			}
			cmd.sources = []*dispatchState{stn}
		}
	}

	if ok := detectRunMount(&cmd, allDispatchStates); ok {
		return cmd, nil
	}
	return cmd, nil
}

type dispatchOpt struct {
	allDispatchStates *dispatchStates
	metaArgs          []instructions.KeyValuePairOptional
	buildArgValues    map[string]string
	shlex             *shell.Lex
	sessionID         string
	buildContext      llb.State
	proxyEnv          *llb.ProxyEnv
	cacheIDNamespace  string
	targetPlatform    specs.Platform
	buildPlatforms    []specs.Platform
	extraHosts        []llb.HostIP
	copyImage         string
	llbCaps           *apicaps.CapSet
}

func dispatch(d *dispatchState, cmd command, opt dispatchOpt) error {
	if ex, ok := cmd.Command.(instructions.SupportsSingleWordExpansion); ok {
		err := ex.Expand(func(word string) (string, error) {
			return opt.shlex.ProcessWordWithMap(word, toEnvMap(d.buildArgs, d.image.Config.Env))
		})
		if err != nil {
			return err
		}
	}

	var err error
	switch c := cmd.Command.(type) {
	case *instructions.MaintainerCommand:
		err = dispatchMaintainer(d, c)
	case *instructions.EnvCommand:
		err = dispatchEnv(d, c)
	case *instructions.RunCommand:
		err = dispatchRun(d, c, opt.proxyEnv, cmd.sources, opt)
	case *instructions.WorkdirCommand:
		err = dispatchWorkdir(d, c, true, &opt)
	case *instructions.AddCommand:
		err = dispatchCopy(d, c.SourceAndDest, opt.buildContext, true, c, c.Chown, opt)
		if err == nil {
			for _, src := range c.Sources() {
				if !strings.HasPrefix(src, "http://") && !strings.HasPrefix(src, "https://") {
					d.ctxPaths[path.Join("/", filepath.ToSlash(src))] = struct{}{}
				}
			}
		}

	case *instructions.LabelCommand:
		err = dispatchLabel(d, c)
	case *instructions.OnBuildCommand:
		err = dispatchOnbuild(d, c)
	case *instructions.CmdCommand:
		err = dispatchCmd(d, c)
	case *instructions.EntryPointCommand:
		err = dispatchEntrypoint(d, c)
	case *instructions.HealthCheckCommand:
		err = dispatchHealthCheck(d, c)
	case *instructions.ExposeCommand:
		err = dispatchExpose(d, c, opt.shlex)
	case *instructions.UserCommand:
		err = dispatchUser(d, c, true)
	case *instructions.VolumeCommand:
		err = dispatchVolume(d, c)
	case *instrcutions.ShellCommand:
		err = dispatchShell(d, c)
	case *instructions.ArgCommand:
		err = dispatchArg(d, c, opt.metaArgs, opt.buildArgValues)
	case *instructions.CopyCommand:
		l := opt.buildContext
		if len(cmd.sources) != 0 {
			l = cmd.sources[0].state
		}
		err = dispatchCopy(d, c.SourceAndDest, l, false, c, c.Chown, opt)
		if err == nil && len(cmd.sources) == 0 {
			for _, src := range c.Sources() {
				d.ctxPaths[path.Join("/", filepath.ToSlash(src))] = struct{}{}
			}
		}
	default:
	}
	return err
}
