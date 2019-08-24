package metadata

import (
	"context"
	"fmt"
	"strconv"
)

// 这个包设置的很巧妙

// 定义context.WithValue的value类型
type MD map[string]interface{}

// Packages that define a Context key should provide type-safe accessors
// for the value stored using that key
// 定义context.WithValue的key类型，不可导出，类型安全
type mdKey struct{}

// Len returns the number of items in md
// 返回当前value的长度
func (md MD) Len() int {
	return len(md)
}

// 复制md数据
func (md MD) Copy() MD {
	return Join(md)
}

// 根据传入的map创建MD
func New(m map[string]interface{}) MD {
	md := MD{}
	for k, val := range m {
		md[k] = val
	}
	return md
}

// 多个md合并成一个md
func Join(mds ...MD) MD {
	out := MD{}
	for _, md := range mds {
		for k, v := range md {
			out[k] = v
		}
	}
	return out
}

// Pairs returns an MD formated by the mapping of key, value ...
func Pairs(kv ...interface{}) MD {
	if len(kv)%2 == 1 {
		panic(fmt.Sprintf("metadata: Pairs got the odd number of input pairs for metadata: %d", len(kv)))
	}
	md := MD{}
	var key string
	for i, s := range kv {
		if i % 2 == 0 {
			key = s.(string)
			continue
		}
		md[key] = s
	}
	return md
}

// NewContext creates a new context with md attached
// 根据传入的md创建一个context
func NewContext(ctx context.Context, md MD) context.Context {
	return context.WithValue(ctx, mdKey{}, md)
}

// FromContext returns the incoming metadata in ctx if it exists
// returned MD should not be modified. Writing to it may cause races
// Modification should be made to copies of the returned MD

// 从context获取已经设置的metadata
// 返回的MD不能被修改，如果向MD写入数据可能会引起竟态
// 如果必须要修改必须调用copy获取一个MD的副本,                    这点很重要
func FromContext(ctx context.Context) (md MD, ok bool) {
	md, ok = ctx.Value(mdKey{}).(MD)
	return
}

// WithContext return no deadline context and retain metadata
// 从给定的ctx中获取metadata，从而生成一个新的context
func WithContext(ctx context.Context) context.Context {
	md, ok := FromContext(ctx)
	if ok {
		nmd := md.Copy()
		return NewContext(context.Background(), nmd)
	}
	return context.Background()
}

// 从context根据给定的key获取string类型数据
func String(ctx context.Context, key string) string {
	// 首先从context中获取mdKey类型的数据，因为mdKey类型是不可到处、被保护的所以不用担心不通过metadata包进行的修改
	// @todo 怎么防止被多多协程修改
	md, ok := ctx.Value(mdKey{}).(MD)
	if !ok {
		return ""
	}
	str, _ := md[key].(string)
	return str
}

// 从context中获取int64类型的数据，根据指定的key
func Int64(ctx context.Context, key string) int64 {
	md, ok := ctx.Value(mdKey{}).(MD)
	if !ok {
		return 0
	}
	i64, _ := md[key].(int64)
	return i64
}

// 从context中获取bool类型的数据
func Bool(ctx context.Context, key string) bool {
	md, ok := ctx.Value(mdKey{}).(MD)
	if !ok {
		return false
	}
	switch md[key].(type) {
	case bool:
		return md[key].(bool)
	case string:
		ok, _ = strconv.ParseBool(md[key].(string))
		return ok
	default:
		return false
	}
}

// Value get value from metadata in context return nil if not found
func Value(ctx context.Context, key string) interface{} {
	md, ok := ctx.Value(mdKey{}).(MD)
	if !ok {
		return nil
	}
	return md[key]
}
