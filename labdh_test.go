package labdh_test

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/golang/glog"
	"github.com/nilsocket/labdh"
)

var testDir = "test"
var NumOfFiles = 10

func TestDownload(t *testing.T) {
	if !fileExists(testDir) {
		os.MkdirAll(testDir, os.ModePerm)
		for i := 0; i < NumOfFiles; i++ {
			generateData(testDir, strconv.Itoa(i), 1e8)
		}
	}

	// start a webserver
	port := closedPortNum()
	server := server("localhost:"+strconv.Itoa(port), testDir)

	files := make([]string, 0, NumOfFiles)
	for i := 0; i < NumOfFiles; i++ {
		files = append(files, fmt.Sprintf("http://localhost:%d/%d", port, i))
	}

	fs := labdh.URL(files...)

	d := labdh.WithOpts(&labdh.Opts{BaseDir: "test/output", AriaArgs: []string{"--max-overall-download-limit=4M"}})
	d.Download(fs)
	d.Close()

	err := server.Shutdown(context.Background())
	glog.Println("Close", err)

}

func generateData(dir string, name string, size int64) {
	f, err := os.Create(filepath.Join(dir, name))
	defer f.Close()
	if err != nil {
		log.Println(err)
	}

	data := make([]byte, size)
	rand.Read(data)
	f.Write(data)
}

func server(addr, dir string) *http.Server {
	server := &http.Server{Addr: addr, Handler: http.FileServer(http.Dir(dir))}

	listenChan := make(chan struct{})

	go func() {
		listenChan <- struct{}{}
		server.ListenAndServe()
	}()

	<-listenChan
	return server
}

// copied from file.go
func fileExists(name string) bool {
	_, err := os.Stat(name)
	if os.IsNotExist(err) {
		// if file doesn't exist return false
		return false
	}
	return true
}

// copied from pkg/iaria/iaria.go
func closedPort(port int) bool {
	conn, err := net.Dial("tcp", net.JoinHostPort("localhost", strconv.Itoa(port)))
	if err == nil && conn != nil {
		conn.Close()
		return false
	}
	return true
}

// copied from pkg/iaria/iaria.go
func closedPortNum() int {
	for i := 1024; i < 65535; i++ {
		if closedPort(i) {
			return i
		}
	}
	return 0
}
