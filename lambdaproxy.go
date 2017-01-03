package lambdaproxy

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/imdario/mergo"
	"log"
	"os"
)

func init() {
	log.SetOutput(os.Stderr)
}

type response struct {
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
}

type Request struct {
	RequestContext        RequestContext    `json:"requestContext"`
	HTTPMethod            string            `json:"httpMethod"`
	Path                  string            `json:"path"`
	Resource              string            `json:"resource"`
	Headers               map[string]string `json:"headers"`
	QueryStringParameters map[string]string `json:"queryStringParameters"`
	PathParameters        map[string]string `json:"pathParameters"`
	StageVariables        map[string]string `json:"stageVariables"`
	Body                  string            `json:"body"`
}

type RequestContext struct {
	ResourceID   string   `json:"resourceId"`
	APIID        string   `json:"apiId"`
	ResourcePath string   `json:"resourcePath"`
	HTTPMethod   string   `json:"httpMethod"`
	RequestID    string   `json:"requestId"`
	AccountID    string   `json:"accountId"`
	Identity     Identity `json:"identity"`
	Stage        string   `json:"stage"`
}

type Identity struct {
	APIKey                        string `json:"apiKey"`
	UserArn                       string `json:"userArn"`
	CognitoAuthenticationType     string `json:"cognitoAuthenticationType"`
	Caller                        string `json:"caller"`
	UserAgent                     string `json:"userAgent"`
	User                          string `json:"user"`
	CognitoIdentityPoolID         string `json:"cognitoIdentityPoolId"`
	CognitoIdentityID             string `json:"cognitoIdentityId"`
	CognitoAuthenticationProvider string `json:"cognitoAuthenticationProvider"`
	SourceIP                      string `json:"sourceIp"`
	AccountID                     string `json:"accountId"`
}

type HTTPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (he *HTTPError) Error() string {
	return fmt.Sprintf("code=%d, message=%s", he.Code, he.Message)
}

func NewHTTPError(code int, message string) *HTTPError {
	he := &HTTPError{Code: code, Message: http.StatusText(code)}
	if message != "" {
		he.Message = message
	}
	return he
}

var EmptyHeaders = map[string]string{}

type Context struct {
	Request  *Request  `json:"request"`
	response *response `json:"response"`
}

func (ctx *Context) Bind(m interface{}) error {
	err := json.Unmarshal([]byte(ctx.Request.Body), m)
	if err != nil {
		return fmt.Errorf("lambdaproxy: unable to bind body to struct: %s", err)
	}

	err = mergo.Map(m, ctx.Request.QueryStringParameters)
	if err != nil {
		return fmt.Errorf("lambdaproxy: unable to bind query params to struct: %s", err)
	}
	return nil
}

func (ctx *Context) QueryParam(n string) string {
	return ctx.Request.QueryStringParameters[n]
}

func (ctx *Context) Param(n string) string {
	return ctx.Request.PathParameters[n]
}

func (ctx *Context) StageVar(n string) string {
	return ctx.Request.StageVariables[n]
}

func (ctx *Context) Header(n string) string {
	return ctx.Request.Headers[n]
}

func (ctx *Context) NoContent(status int) error {
	return ctx.status(status, "", EmptyHeaders)
}

func (ctx *Context) String(status int, body string) error {
	return ctx.status(status, body, EmptyHeaders)
}

func (ctx *Context) JSON(status int, body interface{}) error {
	return ctx.status(status, body, EmptyHeaders)
}

func (ctx *Context) status(status int, body interface{}, headers map[string]string) error {
	b, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("lambdaproxy: unable to convert body to json: %s", err)
	}
	ctx.response = &response{
		StatusCode: status,
		Headers:    headers,
		Body:       string(b),
	}
	return nil
}

func (ctx *Context) Continue() error {
	return nil
}
