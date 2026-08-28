package gofr

import (
	"context"
	"os"
	"time"

	"github.com/neo532/gofr/registry"
	"github.com/neo532/gofr/transport"
	"github.com/neo532/gokit/logger"
)

// Option configures the App.
type Option func(o *options)

type options struct {
	id            string
	name          string
	versionGit    string // git/build version
	schemaVersion string // proto schema hash, from generated registry.pb.go
	metadata      map[string]string
	group         string
	protocol      string
	weight        int

	ctx  context.Context
	sigs []os.Signal

	stopTimeout time.Duration
	logger      logger.ILogger
	servers     []transport.Server

	// Before and After funcs
	beforeStart []func(context.Context) error
	afterStart  []func(context.Context) error
	beforeStop  []func(context.Context) error
	afterStop   []func(context.Context) error

	enableUpgrader bool
	pidFile        string

	registrar        registry.Registrar
	registrarTimeout time.Duration
	readyTimeout     time.Duration
}

func ID(id string) Option {
	return func(o *options) { o.id = id }
}

func Name(name string) Option {
	return func(o *options) { o.name = name }
}

// VersionGit 设置注册实例的 git/build 版本（-ldflags 注入），供部署身份与
// 观测使用。它不驱动网关的反射缓存失效——那由 SchemaVersion 承担，
// 因为 git 版本在无 schema 变更的重启时也会变化。
func VersionGit(v string) Option {
	return func(o *options) { o.versionGit = v }
}

// SchemaVersion 设置注册实例的 proto schema 哈希（来自生成的 registry.pb.go）。
// 网关据此判断后端 schema 是否变更，决定是否重新 dump 反射面。
func SchemaVersion(v string) Option {
	return func(o *options) { o.schemaVersion = v }
}

func Metadata(md map[string]string) Option {
	return func(o *options) { o.metadata = md }
}

// Group sets the registration group the instance registers under. Empty
// defaults to registry.DefaultGroup; a developer sets their own name so local
// instances override the shared pool (see registry.Discovery.WithGroup).
func Group(g string) Option {
	return func(o *options) { o.group = g }
}

// Protocol sets the caller protocol this instance advertises (e.g. "rpcx"),
// letting consumers pick the matching endpoint without per-service config.
func Protocol(p string) Option {
	return func(o *options) { o.protocol = p }
}

// Weight sets the load-balancing weight for this instance; 0 means default.
func Weight(w int) Option {
	return func(o *options) { o.weight = w }
}

func Context(ctx context.Context) Option {
	return func(o *options) { o.ctx = ctx }
}

func Signal(sigs ...os.Signal) Option {
	return func(o *options) { o.sigs = sigs }
}

func StopTimeout(d time.Duration) Option {
	return func(o *options) { o.stopTimeout = d }
}

func Logger(l logger.ILogger) Option {
	return func(o *options) { o.logger = l }
}

func Server(srv ...transport.Server) Option {
	return func(o *options) { o.servers = append(o.servers, srv...) }
}

// Registrar sets the service registry the App registers into on startup.
func Registrar(reg registry.Registrar) Option {
	return func(o *options) { o.registrar = reg }
}

// RegistrarTimeout bounds the synchronous Register/Deregister calls.
func RegistrarTimeout(d time.Duration) Option {
	return func(o *options) { o.registrarTimeout = d }
}

// ReadyTimeout bounds how long the App waits for servers to signal ready.
func ReadyTimeout(d time.Duration) Option {
	return func(o *options) { o.readyTimeout = d }
}

func BeforeStart(fn func(context.Context) error) Option {
	return func(o *options) { o.beforeStart = append(o.beforeStart, fn) }
}

func AfterStart(fn func(context.Context) error) Option {
	return func(o *options) { o.afterStart = append(o.afterStart, fn) }
}

func BeforeStop(fn func(context.Context) error) Option {
	return func(o *options) { o.beforeStop = append(o.beforeStop, fn) }
}

func AfterStop(fn func(context.Context) error) Option {
	return func(o *options) { o.afterStop = append(o.afterStop, fn) }
}

func EnableUpgrader() Option {
	return func(o *options) { o.enableUpgrader = true }
}

func PIDFile(file string) Option {
	return func(o *options) { o.pidFile = file }
}
