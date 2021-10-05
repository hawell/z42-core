package recaptcha

import "net/http"

type MockServer struct {
	server *http.Server
	HandlerFunc func(writer http.ResponseWriter, request *http.Request)
}

func NewMockServer(address string) *MockServer {
	m := MockServer{}
	server := &http.Server{
		Addr: address,
		Handler: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			m.HandlerFunc(writer, request)
		}),
	}
	m.server = server
	return &m
}

func (m *MockServer) Start() {
	m.server.ListenAndServe()
}