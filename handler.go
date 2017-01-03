package lambdaproxy

import (
	"encoding/json"
	"fmt"
	"github.com/apex/go-apex"
	"log"
	"net/http"
	"strings"
)

// Handler handles Lambda events.
type Handler interface {
	Handle(*Context) error
}

// HandlerFunc implements Handler.
type HandlerFunc func(*Context) error

func (h HandlerFunc) Handle(ctx *Context) error {
	return h(ctx)
}

// Handle executes a series of handler functions that work on the lambda
// event. The execution of a set of handler is assigned a single context,
// using which objects can be shared across handlers.
func Handle(h ...HandlerFunc) {
	apex.HandleFunc(func(event json.RawMessage, ctx *apex.Context) (interface{}, error) {
		req := new(Request)
		err := json.Unmarshal(event, req)
		if err != nil {
			log.Printf("[ERROR] lambdaProxy.Handle: unable to unmarshal lambda event json: %s", err)
			return &response{
				StatusCode: http.StatusInternalServerError,
				Body:       http.StatusText(http.StatusInternalServerError),
				Headers:    EmptyHeaders,
			}, nil
		}

		resp := &response{
			StatusCode: http.StatusNoContent,
			Body:       "",
			Headers:    EmptyHeaders,
		}
		hCtx := &Context{Request: req}
		return execHandlerFuncs(resp, hCtx, h...)
	})
}

type Route struct {
	HTTPMethod   string
	ResourcePath string
	HandlerFuncs []HandlerFunc
}

func routeKey(method, path string) string {
	return strings.ToLower(fmt.Sprintf("%s:%s", strings.TrimSpace(method), strings.TrimSpace(path)))
}

type router struct {
	routes          map[string]*Route
	notFoundHandler HandlerFunc
}

func (r *router) Add(method, resourcePath string, h ...HandlerFunc) *router {
	r.routes[routeKey(method, resourcePath)] = &Route{
		HTTPMethod:   method,
		ResourcePath: resourcePath,
		HandlerFuncs: h,
	}
	return r
}

func (r *router) GET(resourcePath string, h ...HandlerFunc) *router {
	return r.Add("get", resourcePath, h...)
}

func (r *router) PUT(resourcePath string, h ...HandlerFunc) *router {
	return r.Add("put", resourcePath, h...)
}

func (r *router) POST(resourcePath string, h ...HandlerFunc) *router {
	return r.Add("post", resourcePath, h...)
}

func (r *router) DELETE(resourcePath string, h ...HandlerFunc) *router {
	return r.Add("delete", resourcePath, h...)
}

func (r *router) HEAD(resourcePath string, h ...HandlerFunc) *router {
	return r.Add("head", resourcePath, h...)
}

func (r *router) SetNotFoundHandler(h HandlerFunc) *router {
	r.notFoundHandler = h
	return r
}

func (r *router) Serve() {
	apex.HandleFunc(func(event json.RawMessage, ctx *apex.Context) (interface{}, error) {
		req := new(Request)
		err := json.Unmarshal(event, req)
		if err != nil {
			log.Printf("[ERROR] lambdaProxy.Router.Serve: unable to unmarshal lambda event json: %s", err)
			return &response{
				StatusCode: http.StatusInternalServerError,
				Body:       http.StatusText(http.StatusInternalServerError),
				Headers:    EmptyHeaders,
			}, nil
		}

		resp := &response{
			StatusCode: http.StatusNoContent,
			Body:       "",
			Headers:    EmptyHeaders,
		}
		hCtx := &Context{Request: req}
		rt, ok := r.routes[routeKey(req.HTTPMethod, req.Resource)]
		if !ok || len(rt.HandlerFuncs) == 0 {
			return execHandlerFuncs(resp, hCtx, r.notFoundHandler)
		}
		return execHandlerFuncs(resp, hCtx, rt.HandlerFuncs...)
	})
}

func execHandlerFuncs(resp *response, hctx *Context, h ...HandlerFunc) (interface{}, error) {
	for _, hf := range h {
		err := hf.Handle(hctx)
		if err != nil {
			if he, ok := err.(*HTTPError); ok {
				return &response{
					StatusCode: he.Code,
					Body:       he.Message,
					Headers:    EmptyHeaders,
				}, nil
			}

			log.Printf("[ERROR] lambdaProxy.Handle: error processing function handler: %s", err)
			return &response{
				StatusCode: http.StatusInternalServerError,
				Body:       http.StatusText(http.StatusInternalServerError),
				Headers:    EmptyHeaders,
			}, nil
		}

		if hctx.response != nil {
			resp = hctx.response
		}
	}
	return resp, nil
}

func notFoundHandler(c *Context) error {
	b, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("DumpContext: %s", err)
	}

	log.Printf("[DEBUG] Context: \n%s", string(b))
	return c.NoContent(http.StatusNotFound)
}

func NewRouter() *router {
	return &router{routes: make(map[string]*Route), notFoundHandler: notFoundHandler}
}
