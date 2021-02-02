package server

import (
	"fmt"
	"github.com/ide70/ide70/util/file"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
	"text/template"
)

var AppVersion string


var htmlTemplates *template.Template = nil
var Cwd string

func handlerIde(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/ide/" {
		http.ServeFile(w, r, "ide70/ide.html")
		return
	}
	
	tokens := strings.Split(r.URL.Path, ".")
	ext := tokens[len(tokens)-1]
	switch ext {
		case "woff":
		w.Header().Set("Content-Type", "font/woff")
		case "woff2":
		w.Header().Set("Content-Type", "font/woff2")
	}
	http.ServeFile(w, r, "ide70/"+strings.TrimPrefix(r.URL.Path, "/ide"))
}

func handlerAlive(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("yes"))
}

func handlerDesignList(w http.ResponseWriter, r *http.Request) {
	list := file.CompactFileList(Cwd)
	//fmt.Println(list)
	w.Write([]byte(list))
}

func handlerDesignSave(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	logger.Debug("save url:", r.URL)
	designTxt := r.FormValue("content")
	logger.Debug("Incoming report template:")
	logger.Debug(designTxt)
	fileName := strings.TrimSpace(r.FormValue("designFileName"))
	logger.Debug("Incoming report file name:", fileName)


	if fileName != "" {
		if strings.Contains(fileName, "/") {
			filePath := path.Dir(fileName)
			os.Mkdir("design/"+filePath, 0755)
		}
	

		err := ioutil.WriteFile("design/"+fileName, []byte(designTxt), 0644)
		if err != nil {
			logger.Error("cannot save file", err)
		}

	}

}

func handlerUnit(w http.ResponseWriter, r *http.Request) {
	
}


func handlerDesignCodeComplete(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
}

func handlerDesignLoad(w http.ResponseWriter, r *http.Request) {
	path := Cwd + strings.TrimPrefix(r.URL.Path, "/load")
	fmt.Println("load:", path);
	content, _ := ioutil.ReadFile(path)
	serveContent(w, string(content))
}

func serveContent(w http.ResponseWriter, content string) {
	w.Header().Set("Content-Type", "text/plain")
	io.WriteString(w, content)
}

func StartServer() {
	
	srv := &http.Server{Addr: ":7070"}

	http.HandleFunc("/ide/", handlerIde)
	http.HandleFunc("/favicon.ico", handlerIde)
	http.HandleFunc("/load/", handlerDesignLoad)
	http.HandleFunc("/save", handlerDesignSave)
	http.HandleFunc("/codeComplete", handlerDesignCodeComplete)
	http.HandleFunc("/list", handlerDesignList)
	http.HandleFunc("/alive", handlerAlive)
	http.HandleFunc("/unit/", handlerUnit)

	//go func() {
		if err := srv.ListenAndServe(); err != nil {
			// cannot panic, because this probably is an intentional close
			logger.Error("Httpserver: ListenAndServe() error: %s", err)
		}
	//}()

	// returning reference so caller can call Shutdown()
	// return srv
}
