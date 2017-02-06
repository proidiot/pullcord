package main

import (
	"fmt"
	"github.com/fitstar/falcore"
	"github.com/stuphlabs/pullcord/authentication"
	"github.com/stuphlabs/pullcord/config"
	"log/syslog"
	"os"
	"sync"
)

const syslogFacility = syslog.LOG_DAEMON
const syslogIdentity = "Pullcord"

func init() {
	authentication.LoadPlugin()
}

var syslogger *syslog.Writer

var once sync.Once

func log() *syslog.Writer {
	once.Do(func() {
		var err error
		syslogger, err = syslog.New(syslogFacility, syslogIdentity)
		if err != nil {
			panic(err)
		}
	})
	return syslogger
}

/*
func serveLandingPage(landingServer *falcore.Server) {
	if err := landingServer.ListenAndServe(); err != nil {
		panic(err)
	}
}
*/

func reportReady(server *falcore.Server, done <-chan struct{}) {
	select {
	case <-server.AcceptReady:
		log().Notice(
			fmt.Sprintf(
				"Now serving at http://localhost:%d",
				server.Port(),
			),
		)
	case <-done:
		//bye
		log().Notice("Server stopped")
	}
}

func main() {
	r, e := os.Open("config.json")
	if e != nil {
		panic(e)
	}

	s, e := config.ServerFromReader(r)
	if e != nil {
		panic(e)
	}

	done := make(chan struct{})
	go reportReady(s, done)
	defer close(done)

	if e := s.ListenAndServe(); e != nil {
		panic(e)
	}

	/*
	go serveLandingPage(landingServer)
	defer landingServer.StopAccepting()

	<-landingServer.AcceptReady

	log().Notice(
		fmt.Sprintf(
			"Now serving landing page on port %d",
			landingServer.Port(),
		),
	)
	*/
}

