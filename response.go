package httptester

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

type Response struct {
	*http.Response
	req     *http.Request
	onError func(error)
	Body    []byte
	URL     *url.URL
}

func NewResponse(res *http.Response, req *http.Request, onError func(error)) *Response {
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		onError(err)
		return nil
	}

	return &Response{
		Response: res,
		req:      req,
		onError:  onError,
		Body:     body,
		URL:      res.Request.URL,
	}
}

func (r *Response) err(err error) {
	r.onError(fmt.Errorf("%s %s: %s", r.req.Method, r.req.URL.String(), err))
}

func (r *Response) bodyExcerpt() string {
	if len(r.Body) > 100 {
		return string(r.Body[:100]) + "..."
	}

	return string(r.Body)
}

func (r *Response) Status(statuses ...int) *Response {
	if len(statuses) > 0 {
		ok := false

		for _, status := range statuses {
			if r.StatusCode == status {
				ok = true
			}
		}

		if !ok {
			r.err(fmt.Errorf("expected status %v got %d: %s", statuses, r.StatusCode, r.bodyExcerpt()))
		}
	}

	return r
}

func (r *Response) JSON(j interface{}) interface{} {
	contentType := r.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		r.err(fmt.Errorf("Content-Type is not application/json, got %s: %s", contentType, r.bodyExcerpt()))
	}
	err := json.Unmarshal([]byte(r.Body), j)
	if err != nil {
		r.err(err)
		return nil
	}
	return j
}

func (r *Response) XML(j interface{}) interface{} {
	contentType := r.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/xml") && !strings.HasPrefix(contentType, "text/xml") {
		r.err(fmt.Errorf("Content-Type is not application/xml or text/xml, got %s: %s", contentType, r.bodyExcerpt()))
	}
	err := xml.Unmarshal([]byte(r.Body), j)
	if err != nil {
		r.err(err)
		return nil
	}
	return j
}

func (r *Response) BodyStr() string {
	return string(r.Body)
}

func (r *Response) Contains(substr string) *Response {
	if !strings.Contains(r.BodyStr(), substr) {
		r.err(fmt.Errorf("body does not contain %s: %s", substr, r.bodyExcerpt()))
	}

	return r
}

func (r *Response) Eq(substr string) *Response {
	if r.BodyStr() != substr {
		r.err(fmt.Errorf("body does not equal %s: %s", substr, r.bodyExcerpt()))
	}

	return r
}

func (r *Response) HeaderEq(key string, value string) *Response {
	if resVal := r.Header.Get(key); resVal != value {
		r.err(fmt.Errorf("header %s: expected %s to equal %s", key, resVal, value))
	}

	return r
}
