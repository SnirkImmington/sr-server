package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sr"
	"sr/config"
	redisUtil "sr/redis"
	"sr/routes"
	"sr/setup"
	"sr/task"
	"time"
)

// SHADOWROLLER ascii art from  http://www.patorjk.com/software/taag/ "Small Slant"
const SHADOWROLLER = `
   ____ __            __                         __ __
  / __// /  ___ _ ___/ /___  _    __ ____ ___   / // /___  ____
 _\ \ / _ \/ _ '// _  // _ \| |/|/ // __// _ \ / // // -_)/ __/
/___//_//_/\_._/ \___/ \___/|__.__//_/   \___//_//_/ \__//_/
`

var taskFlag = flag.String("task", "", "Select a task to run interactively")

func runServer(name string, server *http.Server, tls bool) {
	log.Printf("Running %v server at %v...", name, server.Addr)

	if false { // TODO the ctrl-C should either send to all server threads or just to API for cleanup.
		go func() {
			sigint := make(chan os.Signal, 1)
			signal.Notify(sigint, os.Interrupt)
			<-sigint

			// Interrupt received
			log.Print("Interrupt received, closing in 10s. Ctrl-C to force close")
			ctx, cancel := context.WithCancel(context.Background())
			go func() {
				select {
				case <-sigint:
					log.Print("Aborting...")
				case <-time.After(time.Duration(10) * time.Second):
					log.Print("Timed out, aborting...")
				}
				cancel()
				panic("Server closed")
			}()

			err := server.Shutdown(ctx)
			log.Printf("Serve3r shutdown: %v", err)
		}()
	}

	for {
		var err error
		if tls {
			var pemFile, keyFile string
			if len(config.TLSCertFiles) != 2 {
				pemFile = ""
				keyFile = ""
				log.Print("TLS server with autocert started.")
			} else {
				pemFile = config.TLSCertFiles[0]
				keyFile = config.TLSCertFiles[1]
				log.Print(
					"TLS server with cert files ", pemFile, ", ",
					keyFile, " started.",
				)
			}
			err = server.ListenAndServeTLS(pemFile, keyFile)
		} else {
			log.Print("HTTP (unencrypted) server started.")
			err = server.ListenAndServe()
		}

		if errors.Is(err, http.ErrServerClosed) {
			log.Print("Server shutdown request received.")
		}

		if err != nil {
			log.Print(name, " server failed! Restarting in 10s.", err)
			time.Sleep(time.Duration(10) * time.Second)
			log.Print(name, " server restarting.")
		}
	}
}

func main() {
	log.SetOutput(os.Stdout)
	if config.IsProduction {
		log.SetFlags(
			log.Ldate | log.Ltime | log.LUTC | log.Lmicroseconds,
		)
	} else {
		log.SetFlags(log.Ltime | log.Lshortfile)
	}
	flag.Parse()
	config.VerifyConfig()
	if taskFlag != nil && *taskFlag != "" {
		task.RunSelectedTask(*taskFlag, flag.Args())
	}

	log.Print("Starting up...")
	redisUtil.SetupWithConfig()
	rand.Seed(time.Now().UnixNano())
	sr.BeginGeneratingRolls()
	setup.CheckGamesAndPlayers()
	routes.RegisterTasksViaConfig()

	log.Print("Shadowroller:", SHADOWROLLER, "\n")
	err := routes.DisplaySiteRoutes()
	if err != nil {
		panic(fmt.Sprintf("Unable to walk routes: %v", err))
	}

	if config.PublishRedirect != "" {
		redirectServer := routes.MakeHTTPRedirectServer()
		go runServer("redirect", redirectServer, false)
	}

	if config.PublishHTTP != "" {
		siteServer := routes.MakeHTTPSiteServer()
		runServer("API", siteServer, false)
	} else {
		siteServer := routes.MakeHTTPSSiteServer()
		runServer("API", siteServer, true)
	}
}
