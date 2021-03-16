package main

import (
	"context"
	"flag"
	"fmt"
	"html"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	logg "github.com/menucha-de/App.Log/log"

	"github.com/menucha-de/utils"
	"github.com/sirupsen/logrus"
)

var client mqtt.Client
var brokerURL = "ws://169.254.0.1:9001"

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	opts := mqtt.NewClientOptions()
	opts.AddBroker(brokerURL)
	var timeout int = 10000
	opts.SetConnectTimeout(time.Duration(timeout/1000) * time.Second)
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(30 * time.Second)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(2 * time.Second)
	opts.SetCleanSession(false)
	//opts.SetUsername(uri.User.Username())
	//password, _ := uri.User.Password()
	//opts.SetPassword(password)
	opts.SetOnConnectHandler(onConnect)
	opts.SetConnectionLostHandler(onDisconnect)
	opts.SetClientID(uuid.New().String())
	cl := mqtt.NewClient(opts)

	go func() {
		if token := cl.Connect(); token.WaitTimeout(time.Duration(timeout/1000)*time.Second) && token.Error() == nil {

		} else {
			if token.Error() != nil {
				logrus.Info(token.Error())
			} else {
				logrus.Info("Connection from App.Log timed out.")
			}
		}
	}()
	var port = flag.Int("p", 8087, "port")
	flag.Parse()

	router := logg.NewRouter()

	router.NotFoundHandler = http.HandlerFunc(notFound)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", *port),
		Handler: router,
	}
	done := make(chan os.Signal, 1)
	errs := make(chan error)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errs <- err
		}
	}()
	logrus.Info("Server Started")
	select {
	case err := <-errs:
		logrus.Error(err)
	case <-done:
		logrus.Info("Server Stopped")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logrus.Fatal("Server Shutdown failed:", err.Error())
	}
	logrus.Info("Server Exited Properly")
}
func onDisconnect(c mqtt.Client, err error) {
	logrus.WithError(err).Info("ConnectionLost")
}
func onConnect(c mqtt.Client) {
	if c.IsConnectionOpen() && c.IsConnected() {
		logrus.Info("Connection established")

		if token := c.Subscribe("log/#", 2, logg.MsgRcvdLog); token.Wait() && token.Error() != nil {
			logrus.Info(token.Error())

		}
		if token := c.Subscribe("topic", 2, logg.MsgRcvdTarget); token.Wait() && token.Error() != nil {
			logrus.Info(token.Error())
		}
	}
}
func notFound(w http.ResponseWriter, r *http.Request) {
	if !(r.Method == "GET") {
		w.WriteHeader(404)
	}
	file := "./www" + html.EscapeString(r.URL.Path)
	if file == "./www/" {
		file = "./www/index.html"
	}
	if utils.FileExists(file) {
		http.ServeFile(w, r, file)
	} else {
		w.WriteHeader(404)
	}
}
