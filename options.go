package main

import (
	"time"

	"gnet/pkg/logging"
)

// TCPSocketOpt is the type of TCP socket options.
type TCPSocketOpt int

const (
	TCPNoDelay TCPSocketOpt = iota
	TCPDelay
)

type Option func(opts *Options)

type Options struct {
	LB                      LoadBalancing
	ReuseAddr               bool
	ReusePort               bool
	MulticastInterfaceIndex int
	BindToDevice            string
	Multicore               bool
	NumEventLoop            int
	ReadBufferCap           int
	WriteBufferCap          int
	LockOSThread            bool
	Ticker                  bool
	TCPKeepAlive            time.Duration
	TCPKeepInterval         time.Duration
	TCPKeepCount            int
	TCPNoDelay              TCPSocketOpt
	SocketRecvBuffer        int
	SocketSendBuffer        int
	LogPath                 string
	LogLevel                logging.Level
	Logger                  logging.Logger
	EdgeTriggeredIO         bool
	EdgeTriggeredIOChunk    int
}

func loadOptions(options ...Option) *Options {
	opts := new(Options)
	for _, option := range options {
		option(opts)
	}
	return opts
}

func WithOptions(options Options) Option {
	return func(opts *Options) {
		*opts = options
	}
}

func WithMulticore(multicore bool) Option {
	return func(opts *Options) {
		opts.Multicore = multicore
	}
}

func WithLockOSThread(lockOSThread bool) Option {
	return func(opts *Options) {
		opts.LockOSThread = lockOSThread
	}
}

func WithReadBufferCap(readBufferCap int) Option {
	return func(opts *Options) {
		opts.ReadBufferCap = readBufferCap
	}
}

func WithWriteBufferCap(writeBufferCap int) Option {
	return func(opts *Options) {
		opts.WriteBufferCap = writeBufferCap
	}
}

func WithLoadBalancing(lb LoadBalancing) Option {
	return func(opts *Options) {
		opts.LB = lb
	}
}

func WithNumEventLoop(numEventLoop int) Option {
	return func(opts *Options) {
		opts.NumEventLoop = numEventLoop
	}
}

func WithReusePort(reusePort bool) Option {
	return func(opts *Options) {
		opts.ReusePort = reusePort
	}
}

func WithReuseAddr(reuseAddr bool) Option {
	return func(opts *Options) {
		opts.ReuseAddr = reuseAddr
	}
}

func WithTCPKeepAlive(tcpKeepAlive time.Duration) Option {
	return func(opts *Options) {
		opts.TCPKeepAlive = tcpKeepAlive
	}
}

func WithTCPKeepInterval(tcpKeepInterval time.Duration) Option {
	return func(opts *Options) {
		opts.TCPKeepInterval = tcpKeepInterval
	}
}

func WithTCPKeepCount(tcpKeepCount int) Option {
	return func(opts *Options) {
		opts.TCPKeepCount = tcpKeepCount
	}
}

func WithTCPNoDelay(tcpNoDelay TCPSocketOpt) Option {
	return func(opts *Options) {
		opts.TCPNoDelay = tcpNoDelay
	}
}

func WithSocketRecvBuffer(recvBuf int) Option {
	return func(opts *Options) {
		opts.SocketRecvBuffer = recvBuf
	}
}

func WithSocketSendBuffer(sendBuf int) Option {
	return func(opts *Options) {
		opts.SocketSendBuffer = sendBuf
	}
}

func WithTicker(ticker bool) Option {
	return func(opts *Options) {
		opts.Ticker = ticker
	}
}

func WithLogPath(fileName string) Option {
	return func(opts *Options) {
		opts.LogPath = fileName
	}
}

func WithLogLevel(lvl logging.Level) Option {
	return func(opts *Options) {
		opts.LogLevel = lvl
	}
}

func WithLogger(logger logging.Logger) Option {
	return func(opts *Options) {
		opts.Logger = logger
	}
}

func WithMulticastInterfaceIndex(idx int) Option {
	return func(opts *Options) {
		opts.MulticastInterfaceIndex = idx
	}
}

func WithBindToDevice(iface string) Option {
	return func(opts *Options) {
		opts.BindToDevice = iface
	}
}

func WithEdgeTriggeredIO(et bool) Option {
	return func(opts *Options) {
		opts.EdgeTriggeredIO = et
	}
}

func WithEdgeTriggeredIOChunk(chunk int) Option {
	return func(opts *Options) {
		opts.EdgeTriggeredIOChunk = chunk
	}
}
