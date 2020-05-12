package main

import (
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var waitgroup sync.WaitGroup
var httpRunning bool = false

func getNasaData(nasaUrl string) []byte {
	response, err := http.Get(nasaUrl)
	if err != nil {
		panic("Keep calm  and take a deep breath there is an issue with your request ")
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		panic("Opps... What?Again ?! " + err.Error())
	}
	return body
}

/*
func buildNasaURL() (string, error) {
	key := "6jJOD1PJqau8lVZ8UXXf04dkOLerjvDACkChQ2NU"
	feedtype := "json"
	version := "1.0"
	base, err := url.Parse("https://api.nasa.gov/insight_weather/")
	if err != nil {
		return "", err
	}

	params := url.Values{}
	params.Add("api_key", key)
	params.Add("feedtype", feedtype)
	params.Add("ver", version)
	base.RawQuery = params.Encode()
	return base.String(), nil
}*/

type myTcpListener struct {
	*net.TCPListener
}

// Implementing Accept , I'm sick of wating when port is busy
// lets close it correctly
func (lst myTcpListener) Accept() (c net.Conn, err error) {
	tc, err := lst.AcceptTCP()
	if err != nil {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(1 * time.Minute)
	return tc, nil
}

func myHTTPServer(addr string, handler http.Handler) (sc io.Closer, err error) {

	var listener net.Listener
	srv := &http.Server{Addr: addr, Handler: handler}

	if addr == "" {
		addr = ":https"
	}

	listener, err = net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	/*Let's start our server */
	go func() {

		waitgroup.Add(1)
		httpRunning = true

		err := srv.Serve(myTcpListener{listener.(*net.TCPListener)})

		if err != nil && httpRunning {
			panic(err)
		}
		if httpRunning {
			waitgroup.Done()
		}
	}()

	return listener, nil
}

//Handler that will help use to handle each path separatly
func myHandler(w http.ResponseWriter, req *http.Request) {

	//fmt.Println(req.URL.Path)
	if req.URL.Path == "/insight_weather/" {
		out := string(getNasaData("https://api.nasa.gov" + req.URL.String()))
		w.Write([]byte(out))
	} else if len(req.URL.Path) >= 5 && req.URL.Path[0:5] == "/quit" {
		w.Write([]byte("Buy ! I'm too tired"))
		httpRunning = false
		waitgroup.Done()
	} else {
		w.Write([]byte("Hey! Seems like you asked about something I know nonthing about"))

	}

}

func main() {

	lc, err := myHTTPServer(":8888", http.HandlerFunc(myHandler))
	defer lc.Close()
	/*Lets handle Ctrl C corectly */
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		httpRunning = false
		waitgroup.Done()
	}()

	if err != nil {
		panic(" Can't start local server " + err.Error())

	}

	waitgroup.Wait()

	//Gracefully close, meaning our message before close will be delivered

}
