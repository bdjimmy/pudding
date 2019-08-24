package pudding

import (
	"regexp"
)

// IRouter http router framework interface
type IRouter interface {
	IRoutes
	Group(string, ...HandlerFunc) *RouterGroup
}

// IRoutes http router interface
type IRoutes interface {
	// 注册中间件
	UseFunc(...HandlerFunc) IRoutes
	Use(...Handler) IRoutes

	// 注册http请求处理函数
	Handler(string, string, ...HandlerFunc) IRoutes
	HEAD(string, ...HandlerFunc) IRoutes
	GET(string, ...HandlerFunc) IRoutes
	POST(string, ...HandlerFunc) IRoutes
	PUT(string, ...HandlerFunc) IRoutes
	DELETE(string, ...HandlerFunc) IRoutes
}

type RouterGroup struct {
	// 中间件、最后一个是http处理函数
	Handlers []HandlerFunc
	// 请求path的基础路径
	basePath string
	// http 基础引擎
	engine   *Engine
	// 标记是否是root节点的RouterGroup
	root     bool
	// 方法的配置, 拥有公共方法配置的放到一个group中
	baseConfig *MethodConfig
}

// RouterGroup 实现了IRouter
var _ IRouter = &RouterGroup{}

// UseFunc adds middleware to the group
func (group *RouterGroup) UseFunc(middleware ...HandlerFunc) IRoutes {
	group.Handlers = append(group.Handlers, middleware...)
	return group.returnObj()
}

// Use adds middleware to the group
func (group *RouterGroup) Use(middleware ...Handler) IRoutes {
	for _, m := range middleware {
		group.Handlers = append(group.Handlers, m.ServeHTTP)
	}
	return group.returnObj()
}

// Group create a new router group
func (group *RouterGroup) Group(relativePath string, handlers ...HandlerFunc) *RouterGroup {
	return &RouterGroup{
		Handlers: group.combineHandlers(handlers),
		basePath: group.calculateAbsolutePath(relativePath),
		engine:   group.engine,
		root:     false,
	}
}

// SetMethodConfig is used to set config on specified method
func (group *RouterGroup) SetMethodConfig(config *MethodConfig) *RouterGroup {
	group.baseConfig = config
	return group
}

// BasePath router group base path
func (group *RouterGroup) BasePath() string {
	return group.basePath
}

// Handle registers a new request handle and middleware with the given path and method
// 为请求注册中间件和处理函数
// The last handler should be the real handler,
// the other ones should be middleware that can and should be shared among different routes
// 最后一个函数必须是http请求处理函数， 其他的是路由中间件， 并且可以在不同路由中共享
func (group *RouterGroup) Handler(httpMethod, relativePath string, handlers ...HandlerFunc) IRoutes {
	if matches, err := regexp.MatchString("^[A-Z]+$", httpMethod); !matches || err != nil {
		panic("http method " + httpMethod + " is not valid")
	}
	return group.handle(httpMethod, relativePath, handlers...)
}

// HEAD is a shortcut for router.Handle("HEAD", path, handle)
func (group *RouterGroup) HEAD(relativePath string, handlers ...HandlerFunc) IRoutes {
	return group.Handler("HEAD", relativePath, handlers...)
}

// HEAD is a shortcut for router.Handle("GET", path, handle)
func (group *RouterGroup) GET(relativePath string, handlers ...HandlerFunc) IRoutes {
	return group.Handler("GET", relativePath, handlers...)
}

// HEAD is a shortcut for router.Handle("POST", path, handle)
func (group *RouterGroup) POST(relativePath string, handlers ...HandlerFunc) IRoutes {
	return group.Handler("POST", relativePath, handlers...)
}

// HEAD is a shortcut for router.Handle("PUT", path, handle)
func (group *RouterGroup) PUT(relativePath string, handlers ...HandlerFunc) IRoutes {
	return group.Handler("PUT", relativePath, handlers...)
}

// HEAD is a shortcut for router.Handle("DELETE", path, handle)
func (group *RouterGroup) DELETE(relativePath string, handlers ...HandlerFunc) IRoutes {
	return group.Handler("DELETE", relativePath, handlers...)
}

func (group *RouterGroup) handle(httpMethod, relativePath string, handlers ...HandlerFunc) IRoutes {
	absolutePath := group.calculateAbsolutePath(relativePath)
	injections := group.injections(relativePath)
	handlers = group.combineHandlers(injections, handlers)
	// 想mux中注册回调函数
	group.engine.addRoute(httpMethod, absolutePath, handlers...)
	if group.baseConfig != nil {
		group.engine.SetMethodConfig(absolutePath, group.baseConfig)
	}
	return group.returnObj()
}

// combineHandlers 合并多个handlerFunc
func (group *RouterGroup) combineHandlers(handlerGroups ...[]HandlerFunc) []HandlerFunc {
	finalSize := len(group.Handlers)
	for _, handlers := range handlerGroups {
		finalSize += len(handlers)
	}
	if finalSize >= int(_abortIndex) {
		panic("too many handlers")
	}
	mergedHandlers := make([]HandlerFunc, finalSize)
	// 复制当前已有的handler
	copy(mergedHandlers, group.Handlers)
	position := len(group.Handlers)
	// 挨个合并新增的handler
	for _, handlers := range handlerGroups {
		copy(mergedHandlers[position:], handlers)
		position += len(handlers)
	}
	return mergedHandlers
}

// calculateAbsolutePath 计算绝对路径
func (group *RouterGroup) calculateAbsolutePath(relativePath string) string {
	return joinPaths(group.basePath, relativePath)
}

// 返回engine或者group
func (group *RouterGroup) returnObj() IRoutes {
	// 根节点
	if group.root {
		return group.engine
	}
	return group
}

// injections 根据路径，按照自上而下的正则匹配合适的中间件函数, 需要注意的是匹配成功后就会返回处理函数
func (group *RouterGroup) injections(relativePath string) []HandlerFunc {
	absPath := group.calculateAbsolutePath(relativePath)
	for _, injection := range group.engine.injections {
		if !injection.pattern.MatchString(absPath) {
			continue
		}
		return injection.handlers
	}
	return nil
}
