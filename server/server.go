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
	WeatherReport   string
	City            string
	Messages        []template.HTML
	Version         string
	CreationTime    string
	WeatherIconURL  string
	FontAwesomeIcon string
}

type Server struct {
	config            ServerConfig
	currentContent    *Content
	staticFileHandler http.Handler
	currentImageData  []byte
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
		log.Warn("Error when parsing template: ", err)
		return
	}
	err = t.Execute(w, server.currentContent)
	if err != nil {
		log.Warn("Error when executing template: ", err)
		return
	}
}

func (server *Server) UpdateImage(data []byte) {
	server.currentImageData = data
}

func (server *Server) imageHandler(w http.ResponseWriter, r *http.Request) {
	if server.currentImageData != nil && len(server.currentImageData) > 0 {
		w.Write(server.currentImageData)
	}
}

func (server *Server) genericHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/eInkImage" {
		server.imageHandler(w, r)
	} else if r.URL.Path == "/" {
		server.indexHandler(w, r)
	} else {
		server.staticFileHandler.ServeHTTP(w, r)
	}
}

func (server *Server) UpdateData(data *Content) {
	server.currentContent = data
}

func (server *Server) Serve() {
	http.HandleFunc("/", server.genericHandler)

	log.Infof("Listening at %s ...", server.config.Listen)
	err := http.ListenAndServe(server.config.Listen, nil)
	if err != nil {
		log.Error(err)
	}
}
