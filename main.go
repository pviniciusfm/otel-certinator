package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"go.opentelemetry.io/otel/api/global"
	"go.uber.org/zap"
)

//indexHtml is the htlm rendered in / route
var indexHtml = `
<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8" />
</head>
<body>
<div>
  <form method="POST" action="/create">     
      <label>Domain Name</label><input name="domain" type="text" value="" />
      <input type="submit" value="Request Certificate" />
  </form>
</div>
</body>
</html>
`

var indexResponse = `
<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8" />
</head>
<body>
	<div>
		<h1>Request for domain %s generated successfully<h1>
	</div>
</body>
</html>
`

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Println("could not create zap logger")
		os.Exit(1)
	}
	defer logger.Sync()

	hostPortRaw := os.Getenv("HOST_PORT")
	hostPort, err := strconv.Atoi(hostPortRaw)
	if err != nil {
		logger.Fatal("is not a valid port", zap.Error(err))
	}
	svr := NewServer("certinator", logger, global.Tracer("certinator"), hostPort)
	svr.AddHandlerFunc("/create", issueCertificate)
	svr.AddHandlerFunc("/health", healthHandler)
	svr.AddHandlerFunc("/", handlerHomePage)
	svr.Start()
}

func issueCertificate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method is not supported.", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	r.ParseForm()
	fmt.Fprintf(w, indexResponse, r.Form["domain"])
}

// handlerHomePage is used to render / route
func handlerHomePage(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method is not supported.", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, indexHtml)
}

// healthHandler is used to render up response
func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method is not supported.", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprint(w, `{"status": "UP"}`)
}
