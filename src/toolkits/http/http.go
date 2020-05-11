package http

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/didi/nightingale/src/toolkits/address"
	"github.com/didi/nightingale/src/toolkits/http/middleware"
)

var srv = &http.Server{
	ReadTimeout:    10 * time.Second,
	WriteTimeout:   10 * time.Second,
	MaxHeaderBytes: 1 << 20,
}

// Start http server
func Start(r *gin.Engine, mod string, level string) {
	loggerMid := middleware.LoggerWithConfig(middleware.LoggerConfig{})
	recoveryMid := middleware.Recovery()

	if level != "DEBUG" {
		gin.SetMode(gin.ReleaseMode)
		middleware.DisableConsoleColor()
	} else {
		srv.WriteTimeout = 120 * time.Second
	}

	r.Use(loggerMid, recoveryMid)

	srv.Addr = address.GetHTTPListen(mod)
	srv.Handler = r

	go func() {
		log.Println("starting http server, listening on:", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listening %s occur error: %s\n", srv.Addr, err)
		}
	}()
}

// Shutdown http server
func Shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalln("cannot shutdown http server:", err)
	}

	// catching ctx.Done(). timeout of 5 seconds.
	select {
	case <-ctx.Done():
		log.Println("shutdown http server timeout of 5 seconds.")
	default:
		log.Println("http server stopped")
	}
}

// AutoMountEndpoint report to server
func AutoMountEndpoint(url string, ip string, user string, passwd string) {

	client := &http.Client{}

	edp := &struct {
		Endpoints []string `json:"endpoints"`
	}{
		Endpoints: []string{ip},
	}
	endpoint, err := json.Marshal(edp)
	if err != nil {
		log.Println(err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(endpoint))
	if err != nil {
		log.Println(err)
	}

	req.SetBasicAuth(user, passwd)
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
	}

	byt, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
	}
	defer resp.Body.Close()

	content := &struct {
		Err string `json:"err"`
	}{}

	err = json.Unmarshal(byt, content)
	if err != nil {
		log.Println(err)
	}

	if len(content.Err) != 0 {
		log.Println("mount endpoint fail", err)
	}

}
