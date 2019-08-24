package pudding

import (
	"context"
	"github.com/bdjimmy/pudding/render"
	"math"
	"net/http"
)

const (
	// 中间件函数最大个数
	_abortIndex int8 = math.MaxInt8 / 2
)

var (
	_openParen  = []byte("(")
	_closeParen = []byte(")")
)

// Context is the most important pare.
// It allows us to pass variables between middleware, manage the flow,
// validate the JSON of a request and reander a JSON response for example
type Context struct {
	context.Context

	// http请求输入输出
	Request *http.Request
	Writer  http.ResponseWriter

	// flow control, 流量控制
	index int8
	// 所有的处理函数，包括中间件、http处理函数
	handlers []HandlerFunc

	// Keys is key/value pair exclusively for the context of each request.
	Keys map[string]interface{}

	Error error

	method string
	engine *Engine
}

/******************************************/
/************** flow control **************/
/******************************************/

// Next should be used only inside middleware
// It executes the pending handlers in the chain inside the calling handler
func (c *Context) Next() {
	c.index++
	s := int8(len(c.handlers))
	for ; c.index < s; c.index++ {
		// only check method on last handler, otherwise middlewares will never be effected if request method is not matched
		// 检测最后一个注册函数是否匹配method
		if c.index == s-1 && c.method != c.Request.Method {
			code := http.StatusMethodNotAllowed
			//c.Error = ecode.MethodNotAllowed
			http.Error(c.Writer, http.StatusText(code), code)
			return
		}
		c.handlers[c.index](c)
	}
}

// Abort prevents pending handlers from being called
func (c *Context) Abort() {
	c.index = _abortIndex
}

// AbortWithStatus call `Abort()` and writes the handers with the specified status code.
func (c *Context) AbortWithStatus(code int) {
	c.Status(code)
	c.Abort()
}

// IsAborted returns true if the current context was aborted
func (c *Context) IsAborted() bool {
	return c.index >= _abortIndex
}

/******************************************/
/*********** metadata management **********/
/******************************************/

// Set is used to store a new key/value pair exclusively for this context
// It also lazy initializes c.Keys if it was not used previously
// c.Keys 延迟初始化
func (c *Context) Set(key string, value interface{}) {
	if c.Keys == nil {
		c.Keys = make(map[string]interface{})
	}
	c.Keys[key] = value
}

// Get returns the value for the given key
// If the value does not exists it returns (nil, false)
func (c *Context) Get(key string) (value interface{}, exists bool) {
	value, exists = c.Keys[key]
	return
}

/******************************************/
/***********  response rending  **********/
/******************************************/

// Status sets the HTTP response code
func (c *Context) Status(code int) {
	c.Writer.WriteHeader(code)
}

// Reader http response with http code by a reader instance
func (c *Context) Render(code int, r render.Render) {
	// 设置response的content-type
	r.WriteContentType(c.Writer)
	// lt 0 不设置 http status
	if code > 0 {
		c.Status(code)
	}
	// 是否允许设置body
	if !bodyAllowForStatus(code) {
		return
	}
	// 写上body
	if err := r.Render(c.Writer); err != nil {
		c.Error = err
		return
	}
}

// String writes the given string into the response body
func (c *Context) String(code int, format string, values ...interface{}) {
	c.Render(code, render.String{Format: format, Data: values})
}

// Redirect return a http redirect to the specific location
func (c *Context) Redirect(code int, location string) {
	c.Render(-1, render.Redirect{
		Code:     code,
		Location: location,
		Request:  c.Request,
	})
}

// Bytes writes some data into the body stream and update the http status
func (c *Context) Bytes(code int, contentType string, data ...[]byte) {
	c.Render(code, render.Data{
		ContentType: contentType,
		Data:        data,
	})
}

func (c *Context) JSON(errno int, message string, data interface{}) {
	// http code
	code := http.StatusOK
	//c.Error = err
	c.Render(code, render.JSON{
		Code:    errno,
		Message: message,
		Data:    data,
	})

}

func (c *Context) JSONMap(errno int, message string, data map[string]interface{}) {
	// http code
	code := http.StatusOK
	//c.Error = err
	data["code"] = errno
	data["message"] = message
	c.Render(code, render.MapJSON(data))

}

// 根据状态码判断是否允许设置body
func bodyAllowForStatus(status int) bool {
	switch {
	case status >= 100 && status <= 199:
	case status == 204:
	case status == 304:
		return false
	}
	return true
}
