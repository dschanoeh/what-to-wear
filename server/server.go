package server

import (
	"html/template"
	"net/http"

	log "github.com/sirupsen/logrus"
)

type ServerConfig struct {
	Listen string `yaml:"listen"`
}

type Content struct {
	WeatherReport  string
	City           string
	Messages       []template.HTML
	Version        string
	CreationTime   string
	WeatherIconURL string
}

type Server struct {
	config            ServerConfig
	currentContent    *Content
	staticFileHandler http.Handler
}

func New(c ServerConfig) *Server {
	s := Server{
		config:            c,
		staticFileHandler: http.FileServer(http.Dir("./static/")),
	}
	return &s
}

func (server *Server) indexHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/index.gohtml")
	if err != nil {
		return
	}
	t.Execute(w, server.currentContent)
}

func (server *Server) genericHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		server.staticFileHandler.ServeHTTP(w, r)
		return
	}
	server.indexHandler(w, r)
}

func (server *Server) UpdateData(data *Content) {
	server.currentContent = data
}

func (server *Server) Serve() {
	http.HandleFunc("/", server.genericHandler)

	err := http.ListenAndServe(server.config.Listen, nil)
	if err != nil {
		log.Error(err)
	}
}
