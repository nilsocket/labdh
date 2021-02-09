package labdh

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/vbauerster/mpb/v5"
	aria "github.com/zyxar/argo/rpc"
)

type barData struct {
	totalLength int64 // Used for dynamic length
	isResumed   bool
	refilled    bool
	done        chan struct{} // To inform worker, when done
	bar         *mpb.Bar
}

// File to download
type File struct {
	// URLs, multiple sources of single file
	URLs               []string
	Name, Dir, Referer string
	Cookies            []*http.Cookie

	// ConcReqs, no.of concurrent requests,
	// default 4
	ConcReqs int

	// OnComplete send signal
	OnComplete chan struct{}

	// CBs ,callback functions to call once download is completed
	CBs                 []func(f *File)
	CBName, CBOperation string

	// cbChan ,is sent a signal
	// initalized by `GroupCB()`
	cbChan    chan struct{}
	removeBar bool

	// progress bar for this file
	b *mpb.Bar

	workerID int
}

// Handling Options

// toAriaOpts, converts `File` to `aria.Option`
func toAriaOpts(f *File) aria.Option {
	ariaOpt := make(aria.Option)

	if f.Name != "" {
		ariaOpt["out"] = f.Name
	}

	if f.Dir != "" {
		ariaOpt["dir"] = f.Dir
	}

	if f.Referer != "" {
		ariaOpt["referer"] = f.Referer
	}

	if f.Cookies != nil {

	}

	if f.ConcReqs != 0 {
		ariaOpt["split"] = f.ConcReqs
	}

	return ariaOpt
}

// Files ,type alias to `[]*File`
type Files []*File

func (f *File) download(c aria.Client, gidMap *sync.Map) {

	defer f.onFinish()

	var fileDone, resumed bool
	if fileDone, resumed = dlDone(f); fileDone {
		log.Println(f.workerID, f, "already downloaded")
		return
	}

	gid, err := c.AddURI(f.URLs, toAriaOpts(f))
	if err != nil {
		log.Println(err)
	}

	done := make(chan struct{})

	f.b = newBar(f.Name, f.removeBar, 10000)

	gidMap.Store(gid,
		&barData{
			isResumed: resumed,
			done:      done,
			bar:       f.b,
		},
	)

	<-done

	// download is done, fill bar
	f.b.SetTotal(0, true)

	// remove gid from fileMap
	gidMap.Delete(gid)

	close(done)
}

func (f *File) onFinish() {

	if f.CBs != nil {
		bb := cbFuncsBar(f.b, f.CBName, f.CBOperation, len(f.CBs))
		for _, cb := range f.CBs {
			if cb != nil {
				cb(f)
				if bb != nil {
					bb.Increment()
				}
			}
		}
		log.Println("called callback functions", f.workerID)
	}

	if f.OnComplete != nil {
		log.Println("sent OnComplete signal")
		f.OnComplete <- struct{}{}
	}

	if f.cbChan != nil {
		log.Println("sent cbChan signal")
		f.cbChan <- struct{}{}
	}
}

// dlDone returns true, if file is already downloaded
func dlDone(f *File) (done bool, resumed bool) {

	name := filepath.Join(f.Dir, f.Name)

	// if file already exists, see if name.aria2 exists,
	// else return false
	if !fileExists(name) {
		return false, false
	}

	// if name.aria2 file exists, download isn't finished
	// return false
	if fileExists(name + ".aria2") {
		return false, true
	}

	// return tue, so we don't download this file
	return true, false
}

func fileExists(name string) bool {
	_, err := os.Stat(name)
	if os.IsNotExist(err) {
		// if file doesn't exist return false
		return false
	}
	return true
}

func (f *File) String() string {
	return fmt.Sprintf("%#v\n", *f)
}
