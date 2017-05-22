package http

import (
	"fmt"
	"net/http"
	"log"
	"html"
	"github.com/slaskawi/external-ip-proxy/logging"

	"github.com/slaskawi/external-ip-proxy/configuration"
)

type HttpServer struct {
	IpAddress string
	Port uint32
	Configuration *configuration.Configuration
}

var Logger *logging.Logger = logging.NewLogger("http")

func NewHttpServer(ip string, port uint32, content *configuration.Configuration) *HttpServer {
	return &HttpServer{
		IpAddress: ip,
		Port: port,
		Configuration: content,
	}
}

func (server *HttpServer) Start() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		content, err := configuration.Marshal(server.Configuration);
		if err != nil {
			fmt.Fprintf(w, "%v", err)
		} else {
			fmt.Fprintf(w, "%v", content)
		}
	})
	go http.ListenAndServe(fmt.Sprintf("%v:%v", server.IpAddress, server.Port), nil)
	Logger.Info("Started HTTP Server at %v:%v", server.IpAddress, server.Port)
}

func main() {

	http.HandleFunc("/test/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello2, %q", html.EscapeString(r.URL.Path))
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
	})

	log.Fatal()
}
