package pudding

import (
	"context"
	"github.com/bdjimmy/pudding/metadata"
	"github.com/bdjimmy/pudding/utils"
	"github.com/pkg/errors"
	"log"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"github.com/bdjimmy/pudding/middleware/perf"
)

const (
	defaultMaxMemory = 32 << 20 // 32 MB
)

var (
	// Engine 实现了IRouter
	_ IRouter = &Engine{}
)

func init() {

}

// Handler responds to an HTTP request.
type Handler interface {
	ServeHTTP(c *Context)
}

// HandlerFunc http request handler function
type HandlerFunc func(c *Context)

// ServeHTTP calls f(ctx)
func (f HandlerFunc) ServeHTTP(c *Context) {
	f(c)
}

// ServerConfig is the pudding server config model
type ServerConfig struct {
	NewWork      string         `dsn:"network"`
	Address      string         `dsn:"address"`
	TimeOut      utils.Duration `dsn:"timeout"`
	ReadTimeOut  utils.Duration `dsn:"query.readTimeout"`
	WriteTimeOut utils.Duration `dsn:"query.writeTimeout"`
}

// MethodConfig is the pudding server's methods config model
type MethodConfig struct {
	Timeout utils.Duration
}

// Engine
type Engine struct {
	RouterGroup

	//lock 保护conf变量
	lock sync.RWMutex
	conf *ServerConfig

	// 监听的地址
	address string

	// http mux router
	// http 多路复用路由
	mux *http.ServeMux
	// store *http.Server
	// 原子保留http的server指针
	server atomic.Value

	// metastore is the path as key and the metadata of this path as value
	metastore map[string]map[string]interface{}

	// RWMutex 保护methodConfigs变量
	pcLock        sync.RWMutex
	methodConfigs map[string]*MethodConfig

	// 保留通过正则注册公共的中间件
	injections []injection
}

// injection
type injection struct {
	pattern  *regexp.Regexp
	handlers []HandlerFunc
}

// Start listen and serve pudding engine by given DSN
func (engine *Engine) Start() error {
	conf := engine.conf
	l, err := net.Listen(conf.NewWork, conf.Address)
	if err != nil {
		errors.Wrapf(err, "pudding: listen tcp: %s", conf.Address)
		return err
	}

	log.Printf("pudding: start http listen addr: %s", conf.Address)
	server := &http.Server{
		ReadHeaderTimeout: time.Duration(conf.ReadTimeOut),
		WriteTimeout:      time.Duration(conf.WriteTimeOut),
	}
	// 启动一个协程进行管理
	go func() {
		if err := engine.RunServer(server, l); err != nil {
			// 服务器主动退出
			if errors.Cause(err) == http.ErrServerClosed {
				log.Print("pudding: server closed")
				return
			}
			panic(errors.Wrapf(err, "pudding: engine.ListenServer(%+v, %+v)", server, l))
		}
	}()

	return nil
}

// New returns a new blank Engine instance without any middleware attached
func New() *Engine {
	engine := &Engine{
		RouterGroup: RouterGroup{
			Handlers: nil,
			basePath: "/",
			root:     true,
		},
		address: "",
		conf: &ServerConfig{
			TimeOut: utils.Duration(time.Second),
		},
		mux:           http.NewServeMux(),
		metastore:     make(map[string]map[string]interface{}),
		methodConfigs: make(map[string]*MethodConfig),
		injections:    make([]injection, 0),
	}
	engine.RouterGroup.engine = engine
	// Note add prometheus monitor location
	// Note start pprof
	perf.StartPerf()
	return engine
}

// NewServer returns a new blank Engine instance without any middleware attached
func NewServer(conf *ServerConfig) *Engine {
	//if conf == nil{
	//	if !flag.Parsed() {
	//		fmt.Fprintf(os.Stderr, "pudding: please call flag.Parse() before Init warden server, some configure may not effect \n")
	//	}
	//	conf = parseDSN(_httpDSN)
	//}else{
	//	fmt.Fprintf(os.Stderr, "pudding: config will deprecated, argument will be ignored. please use -http flag or HTTP env to configure http server \n ")
	//}

	engine := &Engine{
		RouterGroup: RouterGroup{
			Handlers: nil,
			basePath: "/",
			root:     true,
		},
		address:       "",
		mux:           http.NewServeMux(),
		metastore:     make(map[string]map[string]interface{}),
		methodConfigs: make(map[string]*MethodConfig),
	}
	if err := engine.SetConfig(conf); err != nil {
		panic(err)
	}
	engine.RouterGroup.engine = engine
	// Note add prometheus monitor location
	// Note start pprof
	perf.StartPerf()
	return engine
}

// DefaultServer returns Engine instance with some middleware already attached
func DefaultServer(conf *ServerConfig) *Engine {
	engine := NewServer(conf)
	// 默认使用某些中间件
	//engine.Use()
	return engine
}

func Default() *Engine {
	engine := New()
	//engine.Use()
	return engine
}

// SetMethodConfig is used to set config on specified path
func (engine *Engine) SetMethodConfig(path string, mc *MethodConfig) {
	engine.pcLock.Lock()
	engine.methodConfigs[path] = mc
	engine.pcLock.Unlock()
}

// 添加请求路由, handlers 为中间件和具体的请求处理函数
func (engine *Engine) addRoute(method, path string, handlers ...HandlerFunc) {
	if path[0] != '/' {
		panic("pudding: path must begin with '/' ")
	}
	if method == "" {
		panic("pudding: HTTP method can not be empty ")
	}
	if len(handlers) == 0 {
		panic("pudding: there must be at least one handler")
	}
	// 初始化path的meta数据
	if _, ok := engine.metastore[path]; !ok {
		engine.metastore[path] = make(map[string]interface{})
	}
	// 向多路复用器中注册函数, 每个请求都会创建一个context
	engine.mux.HandleFunc(path, func(w http.ResponseWriter, req *http.Request) {
		c := &Context{
			Context:  nil,
			engine:   engine,
			index:    -1,
			handlers: nil,
			Keys:     nil,
			method:   "",
			Error:    nil,
		}

		c.Request = req
		c.Writer = w
		c.handlers = handlers
		c.method = method

		// 注册自己的处理函数
		engine.handleContext(c)
	})
}

func (engine *Engine) SetConfig(conf *ServerConfig) (err error) {
	if conf.TimeOut < 0 {
		return errors.New("pudding: config timeout must > 0")
	}
	if conf.NewWork == "" {
		conf.NewWork = "tcp"
	}
	// 加锁，防止设置
	engine.lock.Lock()
	engine.conf = conf
	engine.lock.Unlock()

	return
}

func (engine *Engine) methodConfig(path string) *MethodConfig {
	engine.pcLock.RLock()
	mc := engine.methodConfigs[path]
	engine.pcLock.RUnlock()
	return mc
}

func (engine *Engine) handleContext(c *Context) {
	var cancel func()
	req := c.Request
	// 解析http请求的数据, switch的用法
	switch {
	case strings.Contains(req.Header.Get("Content-Type"), "multipart/form-data"):
		c.Request.ParseMultipartForm(defaultMaxMemory)
	default:
		c.Request.ParseForm()
	}

	// get derived timeout from http request header, compare with the engine configured, and use the minimum one
	// 从http头部获取请求的超时时间，并和配置中的超时时间比对，最终设置小的那个超时时间
	engine.lock.RLock()
	tm := time.Duration(engine.conf.TimeOut)
	engine.lock.RUnlock()
	// the method config is preferred
	// engine.conf.Timeout < engine.methodConfig.timeout, 每个方法也可以自定义超时时间
	if pc := engine.methodConfig(req.URL.Path); pc != nil {
		tm = time.Duration(pc.Timeout)
	}
	// 从http的请求头中获取客户端超时时间， 和服务端配置的超时时间比对
	if ctm := timeout(req); ctm > 0 && tm > ctm {
		tm = ctm
	}
	// 设置metadata
	md := metadata.MD{
		metadata.RemoteIP:   remoteIP(req),
		metadata.RemotePort: remotePort(req),
		metadata.Caller:     caller(req),
		metadata.Mirror:     mirror(req),
	}
	ctx := metadata.NewContext(context.Background(), md)
	if tm > 0 {
		c.Context, cancel = context.WithTimeout(ctx, tm)
	} else {
		c.Context, cancel = context.WithCancel(ctx)
	}
	// 这个地方需要注意， 所有中间件执行完会调用取消函数
	// 所以， 如果后台执行一定要调用NewContext或者FromContext，否则后台任务会被自动取消
	defer cancel()
	c.Next()
}

// Router return a http.Handler for using http.ListenAndServe() directly
// 从engine中返回http.Handler给http.ListenAndServe直接使用
func (engine *Engine) Router() http.Handler {
	return engine.mux
}

// Server is used to load stored http server
// 获取http.Server
func (engine *Engine) Server() *http.Server {
	s, ok := engine.server.Load().(*http.Server)
	if !ok {
		return nil
	}
	return s
}

// Shutdown the http server without interrupting active connections
// 关闭Server, 不中断活动连接
func (engine *Engine) ShutDown(ctx context.Context) error {
	server := engine.Server()
	if server == nil {
		return errors.New("pudding: no server")
	}
	return errors.WithStack(server.Shutdown(ctx))
}

//// UseFunc attaches a global middleware to the router.
//// ie. the middleware attached though UseFunc() will be include in the handlers chain for every single request
//// Even 404, 405, static files...
//// For example, this is the right place for a logger or error management middleware
//func (engine *Engine) UseFunc(middleware ...HandlerFunc) IRoutes {
//	engine.RouterGroup.UseFunc(middleware...)
//	return engine
//}
//
//// Use
//func (engine *Engine) Use(middleware ...Handler) IRoutes {
//	engine.RouterGroup.Use(middleware...)
//	return engine
//}

// Ping is used to set the general HTTP ping handler
func (engine *Engine) Ping(handler HandlerFunc) {
	engine.GET("/monitor/ping", handler)
}

// Register is used to export metadata to discovery
// 返回已经注册的方法，用于服务发现
func (engine *Engine) Register(handler HandlerFunc) {
	engine.GET("/register", handler)
}

// Run attaches the router to a http.Server and starts listening and serving HTTP requests
// It is a shortcut for http.ListenAndServe(addr, router)
// Note: this method will block the calling goroutine indefinitely unless an error happens
// 启动http服务，并且设置路由，调用者会被阻塞
func (engine *Engine) Run(addr ...string) (err error) {
	address := resolveAddress(addr)
	server := http.Server{
		Addr:    address,
		Handler: engine.mux,
	}
	engine.server.Store(server)
	if err = server.ListenAndServe(); err != nil {
		err = errors.Wrapf(err, "listenAndServe addr: %v", addr)
	}
	return
}

// RunServer will serve and start listening HTTP requests by given server and listener
func (engine *Engine) RunServer(server *http.Server, l net.Listener) (err error) {
	server.Handler = engine.mux
	engine.server.Store(server)
	if err = server.Serve(l); err != nil {
		err = errors.Wrapf(err, "listen server: %+v/%+v", server, l)
		return
	}
	return
}

func (engine *Engine) metadata() HandlerFunc {
	return func(c *Context) {
		//c.JSON(engine.metastore, nil)
	}
}

// Inject 正则注册中间件
func (engine *Engine) Inject(pattern string, handlers ...HandlerFunc) {
	engine.injections = append(engine.injections, injection{
		pattern:  regexp.MustCompile(pattern),
		handlers: handlers,
	})
}
