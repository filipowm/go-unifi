package unifi //nolint: testpackage

import (
	"errors"
	"net/http"
)

type TestInterceptor struct {
	request       *http.Request
	response      *http.Response
	failOnRequest bool
}

func NewTestInterceptor() *TestInterceptor {
	return &TestInterceptor{}
}

func (i *TestInterceptor) IsRequestIntercepted() bool {
	return i.request != nil
}

func (i *TestInterceptor) IsResponseIntercepted() bool {
	return i.response != nil
}

func (i *TestInterceptor) InterceptRequest(req *http.Request) error {
	i.request = req
	if i.failOnRequest {
		return errors.New("request interceptor failed")
	}
	return nil
}

func (i *TestInterceptor) InterceptResponse(resp *http.Response) error {
	i.response = resp
	return nil
}

func (i *TestInterceptor) RequestHeader(key string) string {
	return i.request.Header.Get(key)
}

func (i *TestInterceptor) ResponseHeader(key string) string {
	return i.response.Header.Get(key)
}

func (i *TestInterceptor) Method() string {
	return i.request.Method
}

func (i *TestInterceptor) AsList() []ClientInterceptor {
	return []ClientInterceptor{i}
}
