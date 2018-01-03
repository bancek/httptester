package httptester_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bancek/httptester"
)

var base string
var fail func(error)

func newRequest() *httptester.ReqBuilder {
	return httptester.NewReqBuilder(base, http.DefaultClient, fail).Header("X-Http-Tester", "true")
}

func GET(url string) *httptester.ReqBuilder {
	return newRequest().GET(url)
}

func TestReqBuilder(t *testing.T) {
	fail = func(err error) {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(409)
		w.Write([]byte("Already exists\n"))
	}))
	defer server.Close()
	base = server.URL

	GET("/").Do().Status(409).Contains("Already exists")
}
