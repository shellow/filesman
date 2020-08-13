package main

import (
	"flag"
	"github.com/gin-gonic/gin"
	"github.com/shellow/filesman"
	"go.uber.org/zap"
	"net/http"
	"os"
	"time"
)

var Logger *zap.SugaredLogger
var LISTENADDR string
var Filesm *filesman.Filesman

func main() {
	initApp()
	server()
}

func initarg() {
	flag.StringVar(&LISTENADDR, "addr", ":8080", "listen address")
	flag.Parse()
}

func initApp() {
	gin.SetMode(gin.ReleaseMode)
	initarg()

	logger, _ := zap.NewProduction()
	defer logger.Sync()
	Logger = logger.Sugar()

	Filesm = filesman.NewFilesman()

	Logger.Info("init finish")
}

func server() {
	router := gin.Default()

	router.GET("/test", func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.String(http.StatusOK, "Hello World")
	})
	router.POST("/files/upload", upload)
	router.GET("/files/download/:filename", Filesm.Download)
	router.POST("/files/imgsignpdf", Filesm.ImgAddPdfOnce)

	s := &http.Server{
		Addr:           LISTENADDR,
		Handler:        router,
		ReadTimeout:    5 * time.Second,
		WriteTimeout:   5 * time.Second,
		MaxHeaderBytes: 1 << 10,
	}

	Logger.Info("server run")
	err := s.ListenAndServe()
	if err != nil {
		Logger.Error(err)
		os.Exit(-1)
	}
}

func upload(c *gin.Context) {
	Filesm.Upload(c)
}
