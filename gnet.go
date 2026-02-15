package main

import (
	"context"
	"io"
	"net"
	"net/url"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"

	"gnet/internal/gfd"
	"gnet/pkg/buffer/ring"
	errorx "gnet/pkg/errors"
	"gnet/pkg/logging"
	"gnet/pkg/math"
)

// Action is an action that occurs after the completion of an event.
type Action int

const (
	// None indicates that no action should occur following an event.
	None Action = iota

	// Close closes the connection.
	Close
	Shutdown
)

type Engine struct {
	eng *engine
}

func (e Engine) Validate() error {
	if e.eng == nil || len(e.eng.listeners) == 0 {
		return errorx.ErrEmptyEngine
	}
	if e.eng.isShutdown() {
		return errorx.ErrEngineInShutdown
	}
	return nil
}

func (e Engine) CountConnections() (count int) {
	if e.Validate() != nil {
		return -1
	}

	e.eng.eventLoops.iterate(func(_ int, el *eventloop) bool {
		count += int(el.countConn())
		return true
	})
	return
}

func (e Engine) Register(ctx context.Context) (<-chan RegisteredResult, error) {
	if err := e.Validate(); err != nil {
		return nil, err
	}

	if e.eng.eventLoops.len() == 0 {
		return nil, errorx.ErrEmptyEngine
	}

	c, ok := FromNetConnContext(ctx)
	if ok {
		return e.eng.eventLoops.next(c.RemoteAddr()).Enroll(ctx, c)
	}

	addr, ok := FromNetAddrContext(ctx)
	if ok {
		return e.eng.eventLoops.next(addr).Register(ctx, addr)
	}

	return nil, errorx.ErrInvalidNetworkAddress
}

func (e Engine) Dup() (fd int, err error) {
	if err := e.Validate(); err != nil {
		return -1, err
	}

	if len(e.eng.listeners) > 1 {
		return -1, errorx.ErrUnsupportedOp
	}

	for _, ln := range e.eng.listeners {
		fd, err = ln.dup()
	}

	return
}

func (e Engine) DupListener(network, addr string) (int, error) {
	if err := e.Validate(); err != nil {
		return -1, err
	}

	for _, ln := range e.eng.listeners {
		if ln.network == network && ln.address == addr {
			return ln.dup()
		}
	}

	return -1, errorx.ErrInvalidNetworkAddress
}

func (e Engine) Stop(ctx context.Context) error {
	if err := e.Validate(); err != nil {
		return err
	}

	e.eng.shutdown(nil)

	ticker := time.NewTicker(shutdownPollInterval)
	defer ticker.Stop()
	for {
		if e.eng.isShutdown() {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

/*
type asyncCmdType uint8

const (
	asyncCmdClose = iota + 1
	asyncCmdWake
	asyncCmdWrite
	asyncCmdWritev
)

type asyncCmd struct {
	fd  gfd.GFD
	typ asyncCmdType
	cb  AsyncCallback
	param any
}


func (e Engine) AsyncWrite(fd gfd.GFD, p []byte, cb AsyncCallback) error {
	if err := e.Validate(); err != nil {
		return err
	}

	return e.eng.sendCmd(&asyncCmd{fd: fd, typ: asyncCmdWrite, cb: cb, param: p}, false)
}


func (e Engine) AsyncWritev(fd gfd.GFD, batch [][]byte, cb AsyncCallback) error {
	if err := e.Validate(); err != nil {
		return err
	}

	return e.eng.sendCmd(&asyncCmd{fd: fd, typ: asyncCmdWritev, cb: cb, param: batch}, false)
}


func (e Engine) Close(fd gfd.GFD, cb AsyncCallback) error {
	if err := e.Validate(); err != nil {
		return err
	}

	return e.eng.sendCmd(&asyncCmd{fd: fd, typ: asyncCmdClose, cb: cb}, false)
}


func (e Engine) Wake(fd gfd.GFD, cb AsyncCallback) error {
	if err := e.Validate(); err != nil {
		return err
	}

	return e.eng.sendCmd(&asyncCmd{fd: fd, typ: asyncCmdWake, cb: cb}, true)
}
*/

type Reader interface {
	io.Reader
	io.WriterTo
	Next(n int) (buf []byte, err error)
	Peek(n int) (buf []byte, err error)
	Discard(n int) (discarded int, err error)
	InboundBuffered() int
}

type Writer interface {
	io.Writer
	io.ReaderFrom
	SendTo(buf []byte, addr net.Addr) (n int, err error)
	Writev(bs [][]byte) (n int, err error)
	Flush() error
	OutboundBuffered() int
	AsyncWrite(buf []byte, callback AsyncCallback) (err error)
	AsyncWritev(bs [][]byte, callback AsyncCallback) (err error)
}

type AsyncCallback func(c Conn, err error) error

type Socket interface {
	Fd() int
	Dup() (int, error)
	SetReadBuffer(size int) error
	SetWriteBuffer(size int) error
	SetLinger(secs int) error
	SetKeepAlivePeriod(d time.Duration) error
	SetKeepAlive(enabled bool, idle, intvl time.Duration, cnt int) error
	SetNoDelay(noDelay bool) error
}

type Runnable interface {
	Run(ctx context.Context) error
}

type RunnableFunc func(ctx context.Context) error

func (fn RunnableFunc) Run(ctx context.Context) error {
	return fn(ctx)
}

type RegisteredResult struct {
	Conn Conn
	Err  error
}

type EventLoop interface {
	Register(ctx context.Context, addr net.Addr) (<-chan RegisteredResult, error)
	Enroll(ctx context.Context, c net.Conn) (<-chan RegisteredResult, error)
	Execute(ctx context.Context, runnable Runnable) error
	Schedule(ctx context.Context, runnable Runnable, delay time.Duration) error
	Close(Conn) error
}

type Conn interface {
	Reader
	Writer
	Socket
	Context() (ctx any)
	EventLoop() EventLoop
	SetContext(ctx any)
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	Wake(callback AsyncCallback) error
	CloseWithCallback(callback AsyncCallback) error
	Close() error
	SetDeadline(time.Time) error
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
}

type (
	EventHandler interface {
		OnBoot(eng Engine) (action Action)
		OnShutdown(eng Engine)
		OnOpen(c Conn) (out []byte, action Action)
		OnClose(c Conn, err error) (action Action)
		OnTraffic(c Conn) (action Action)
		OnTick() (delay time.Duration, action Action)
	}

	BuiltinEventEngine struct{}
)

func (*BuiltinEventEngine) OnBoot(_ Engine) (action Action) {
	return
}

func (*BuiltinEventEngine) OnShutdown(_ Engine) {
}

func (*BuiltinEventEngine) OnOpen(_ Conn) (out []byte, action Action) {
	return
}

func (*BuiltinEventEngine) OnClose(_ Conn, _ error) (action Action) {
	return
}

func (*BuiltinEventEngine) OnTraffic(_ Conn) (action Action) {
	return
}

func (*BuiltinEventEngine) OnTick() (delay time.Duration, action Action) {
	return
}

var MaxStreamBufferCap = 64 * 1024

func createListeners(addrs []string, opts ...Option) ([]*listener, *Options, error) {
	options := loadOptions(opts...)

	logger, logFlusher := logging.GetDefaultLogger(), logging.GetDefaultFlusher()
	if options.Logger == nil {
		if options.LogPath != "" {
			logger, logFlusher, _ = logging.CreateLoggerAsLocalFile(options.LogPath, options.LogLevel)
		}
		options.Logger = logger
	} else {
		logger = options.Logger
		logFlusher = nil
	}
	logging.SetDefaultLoggerAndFlusher(logger, logFlusher)

	logging.Debugf("default logging level is %s", logging.LogLevel())

	if options.LockOSThread && options.NumEventLoop > 10000 {
		logging.Errorf("too many event-loops under LockOSThread mode, should be less than 10,000 "+
			"while you are trying to set up %d\n", options.NumEventLoop)
		return nil, nil, errorx.ErrTooManyEventLoopThreads
	}

	if options.EdgeTriggeredIOChunk > 0 {
		options.EdgeTriggeredIO = true
		options.EdgeTriggeredIOChunk = math.CeilToPowerOfTwo(options.EdgeTriggeredIOChunk)
	} else if options.EdgeTriggeredIO {
		options.EdgeTriggeredIOChunk = 1 << 20
	}

	rbc := options.ReadBufferCap
	switch {
	case rbc <= 0:
		options.ReadBufferCap = MaxStreamBufferCap
	case rbc <= ring.DefaultBufferSize:
		options.ReadBufferCap = ring.DefaultBufferSize
	default:
		options.ReadBufferCap = math.CeilToPowerOfTwo(rbc)
	}
	wbc := options.WriteBufferCap
	switch {
	case wbc <= 0:
		options.WriteBufferCap = MaxStreamBufferCap
	case wbc <= ring.DefaultBufferSize:
		options.WriteBufferCap = ring.DefaultBufferSize
	default:
		options.WriteBufferCap = math.CeilToPowerOfTwo(wbc)
	}

	var hasUDP, hasUnix bool
	for _, addr := range addrs {
		proto, _, err := parseProtoAddr(addr)
		if err != nil {
			return nil, nil, err
		}
		hasUDP = hasUDP || strings.HasPrefix(proto, "udp")
		hasUnix = hasUnix || proto == "unix"
	}

	goos := runtime.GOOS
	if options.ReusePort &&
		(options.Multicore || options.NumEventLoop > 1) &&
		(goos != "linux" && goos != "dragonfly" && goos != "freebsd") {
		options.ReusePort = false
	}

	if options.ReusePort && hasUnix {
		options.ReusePort = false
	}

	if hasUDP {
		options.ReusePort = true
		options.EdgeTriggeredIO = false
	}

	listeners := make([]*listener, len(addrs))
	for i, a := range addrs {
		proto, addr, err := parseProtoAddr(a)
		if err != nil {
			return nil, nil, err
		}
		ln, err := initListener(proto, addr, options)
		if err != nil {
			return nil, nil, err
		}
		listeners[i] = ln
	}

	return listeners, options, nil
}

func Run(eventHandler EventHandler, protoAddr string, opts ...Option) error {
	listeners, options, err := createListeners([]string{protoAddr}, opts...)
	if err != nil {
		return err
	}
	defer func() {
		for _, ln := range listeners {
			ln.close()
		}
		logging.Cleanup()
	}()
	return run(eventHandler, listeners, options, []string{protoAddr})
}

func Rotate(eventHandler EventHandler, addrs []string, opts ...Option) error {
	listeners, options, err := createListeners(addrs, opts...)
	if err != nil {
		return err
	}
	defer func() {
		for _, ln := range listeners {
			ln.close()
		}
		logging.Cleanup()
	}()
	return run(eventHandler, listeners, options, addrs)
}

var (
	allEngines sync.Map

	shutdownPollInterval = 500 * time.Millisecond
)

func Stop(ctx context.Context, protoAddr string) error {
	var eng *engine
	if s, ok := allEngines.Load(protoAddr); ok {
		eng = s.(*engine)
		eng.shutdown(nil)
		defer allEngines.Delete(protoAddr)
	} else {
		return errorx.ErrEngineInShutdown
	}

	if eng.isShutdown() {
		return errorx.ErrEngineInShutdown
	}

	ticker := time.NewTicker(shutdownPollInterval)
	defer ticker.Stop()
	for {
		if eng.isShutdown() {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func parseProtoAddr(protoAddr string) (string, string, error) {

	// This is for cases like this: udp://[ff02::3%lo0]:9991
	protoAddr = strings.ReplaceAll(protoAddr, "%", "%25")

	if runtime.GOOS == "windows" {
		if strings.HasPrefix(protoAddr, "unix://") {
			parts := strings.SplitN(protoAddr, "://", 2)
			if parts[1] == "" {
				return "", "", errorx.ErrInvalidNetworkAddress
			}
			return parts[0], parts[1], nil
		}
	}

	u, err := url.Parse(protoAddr)
	if err != nil {
		return "", "", err
	}

	switch u.Scheme {
	case "":
		return "", "", errorx.ErrInvalidNetworkAddress
	case "tcp", "tcp4", "tcp6", "udp", "udp4", "udp6":
		if u.Host == "" || u.Path != "" {
			return "", "", errorx.ErrInvalidNetworkAddress
		}
		return u.Scheme, u.Host, nil
	case "unix":
		hostPath := path.Join(u.Host, u.Path)
		if hostPath == "" {
			return "", "", errorx.ErrInvalidNetworkAddress
		}
		return u.Scheme, hostPath, nil
	default:
		return "", "", errorx.ErrUnsupportedProtocol
	}
}

func determineEventLoops(opts *Options) int {
	numEventLoop := 1
	if opts.Multicore {
		numEventLoop = runtime.NumCPU()
	}
	if opts.NumEventLoop > 0 {
		numEventLoop = opts.NumEventLoop
	}
	if numEventLoop > gfd.EventLoopIndexMax {
		numEventLoop = gfd.EventLoopIndexMax
	}
	return numEventLoop
}
