package httptester

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
)

type ReqBuilder struct {
	baseURL       string
	url           string
	method        string
	query         url.Values
	headers       http.Header
	noFollow      bool
	body          io.Reader
	client        *http.Client
	beforeRequest func(req *http.Request) *http.Request
	afterRequest  func(req *http.Request, res *http.Response, err error)
	context       context.Context
	onError       func(error)
}

func NewReqBuilder(baseURL string, client *http.Client, onError func(error)) *ReqBuilder {
	return &ReqBuilder{
		baseURL: baseURL,
		query:   url.Values{},
		headers: http.Header{},
		client:  client,
		onError: onError,
	}
}

func (b *ReqBuilder) Method(method string, url string) *ReqBuilder {
	b.method = method
	b.url = url
	return b
}

func (b *ReqBuilder) GET(url string) *ReqBuilder {
	return b.Method("GET", url)
}

func (b *ReqBuilder) POST(url string) *ReqBuilder {
	return b.Method("POST", url)
}

func (b *ReqBuilder) PUT(url string) *ReqBuilder {
	return b.Method("PUT", url)
}

func (b *ReqBuilder) DELETE(url string) *ReqBuilder {
	return b.Method("DELETE", url)
}

func (b *ReqBuilder) NoFollow() *ReqBuilder {
	b.noFollow = true
	return b
}

func (b *ReqBuilder) Q(args ...string) *ReqBuilder {
	keys := map[string]bool{}
	for i := 0; i < len(args)/2; i++ {
		key := args[i*2]
		value := args[i*2+1]
		if _, ok := keys[key]; ok {
			b.query.Add(key, value)
		} else {
			b.query.Set(key, value)
		}
		keys[key] = true
	}
	return b
}

func (b *ReqBuilder) Header(args ...string) *ReqBuilder {
	for i := 0; i < len(args)/2; i++ {
		b.headers.Set(args[i*2], args[i*2+1])
	}
	return b
}

func (b *ReqBuilder) Auth(auth string) *ReqBuilder {
	return b.Header("Authorization", auth)
}

func (b *ReqBuilder) Bearer(bearer string) *ReqBuilder {
	return b.Auth("Bearer " + bearer)
}

func (b *ReqBuilder) Basic(username string, password string) *ReqBuilder {
	auth := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	return b.Auth("Basic " + auth)
}

func (b *ReqBuilder) Body(reader io.Reader) *ReqBuilder {
	b.body = reader
	return b
}

func (b *ReqBuilder) Form(args ...string) *ReqBuilder {
	q := url.Values{}
	for i := 0; i < len(args)/2; i++ {
		q.Set(args[i*2], args[i*2+1])
	}
	b.Header("Content-Type", "application/x-www-form-urlencoded")
	return b.Body(strings.NewReader(q.Encode()))
}

func (b *ReqBuilder) JSON(j interface{}) *ReqBuilder {
	b.Header("Content-Type", "application/json")
	jsonBytes, err := json.Marshal(j)
	if err != nil {
		b.onError(err)
	}
	return b.Body(bytes.NewReader(jsonBytes))
}

func (b *ReqBuilder) File(fieldName string, fileName string, reader io.Reader, extra map[string]string) *ReqBuilder {
	r, w := io.Pipe()

	writer := multipart.NewWriter(w)

	go func() {
		var err error

		defer func() {
			if err == nil {
				w.Close()
			}
		}()

		for k, v := range extra {
			err = writer.WriteField(k, v)

			if err != nil {
				w.CloseWithError(err)
				return
			}
		}

		part, err := writer.CreateFormFile(fieldName, fileName)

		if err != nil {
			w.CloseWithError(err)
			return
		}

		defer writer.Close()

		_, err = io.Copy(part, reader)

		if err != nil {
			w.CloseWithError(err)
			return
		}
	}()

	b.Header("Content-Type", writer.FormDataContentType())

	return b.Body(r)
}

func (b *ReqBuilder) OnError(f func(error)) *ReqBuilder {
	b.onError = f
	return b
}

func (b *ReqBuilder) BeforeRequest(f func(req *http.Request)) *ReqBuilder {
	return b.BeforeWithRequest(func(req *http.Request) *http.Request {
		f(req)
		return req
	})
}

func (b *ReqBuilder) BeforeWithRequest(f func(req *http.Request) *http.Request) *ReqBuilder {
	b.beforeRequest = f
	return b
}

func (b *ReqBuilder) AfterRequest(f func(req *http.Request, res *http.Response, err error)) *ReqBuilder {
	b.afterRequest = f
	return b
}

func (b *ReqBuilder) Context(ctx context.Context) *ReqBuilder {
	b.context = ctx
	return b
}

func (b *ReqBuilder) Do() *Response {
	u, err := url.Parse(b.baseURL + b.url)
	if err != nil {
		b.onError(err)
		return nil
	}

	if len(b.query) > 0 {
		q := u.Query()
		for k, vs := range b.query {
			for _, v := range vs {
				q.Add(k, v)
			}
		}

		u.RawQuery = q.Encode()
	}

	ctx := b.context
	if ctx == nil {
		ctx = context.Background()
	}

	req, err := http.NewRequestWithContext(ctx, b.method, u.String(), b.body)
	if err != nil {
		b.onError(err)
		return nil
	}

	for k, vs := range b.headers {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}

	if host := b.headers.Get("Host"); host != "" {
		req.Host = host
	}

	oldCheckRedirect := b.client.CheckRedirect

	if b.noFollow {
		b.client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	if b.beforeRequest != nil {
		req = b.beforeRequest(req)
	}

	res, err := b.client.Do(req)

	if b.afterRequest != nil {
		b.afterRequest(req, res, err)
	}

	b.client.CheckRedirect = oldCheckRedirect

	if err != nil {
		b.onError(err)
		return nil
	}

	return NewResponse(res, req, b.onError)
}
