package labdh

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/nilsocket/labdh/pkg/iaria"
	aria "github.com/zyxar/argo/rpc"
)

var baseDir, _ = os.Getwd()
var homeDir, _ = os.UserHomeDir()
var logDir = filepath.Join(homeDir, ".dl")

// Opts for downloader.
type Opts struct {
	// BaseDir for Downloader
	BaseDir string

	LogPrefix string

	// ConcDls ,max no.of downloads at any point of time
	ConcDls int

	// OnComplete channel is used to inform user once a download is done
	OnComplete chan struct{}

	// AriaArgs are passed on to aria
	AriaArgs []string

	// fic file input channel, downloads passed to workers
	fic chan *File

	wg sync.WaitGroup

	cmd    *exec.Cmd
	client aria.Client
	gidMap *sync.Map
	notify ariaNotifier
}

// Labdh Downloader
type Labdh Opts

// DefaultLabdh to download
var DefaultLabdh = setDefaultOpts(&Opts{})

// WithOpts ,returns *Labdh object with given opts,
// unset fields are set internally.
func WithOpts(opts *Opts) *Labdh {
	if opts != nil {
		return setDefaultOpts(opts)
	}

	return setDefaultOpts(&Opts{})
}

// URL ,Download files with single url
func URL(url ...string) Files {
	files := make(Files, 0, len(url))
	for _, u := range url {
		files = append(files, &File{URLs: []string{u}})
	}
	return files
}

// URLs ,Download files with multiple urls
func URLs(urls ...[]string) Files {
	files := make(Files, 0, len(urls))
	for _, us := range urls {
		files = append(files, &File{URLs: us})
	}
	return files
}

// Download given set of files.
// Download is a blocking call, if more than `ConcDls` are in progress.
func Download(files Files) {
	DefaultLabdh.Download(files)
}

// Download given set of files.
// Download is a blocking call, if more than `ConcDls` are in progress.
func (dl *Labdh) Download(files Files) {
	for _, f := range files {
		dl.fic <- f
	}
}

// GroupCB calls provided `fns` once all `fs` are downloaded.
// Can be used for merging multiple files,
// or for any operation on multiple set of files
// `operation` is replaced with ``, onComplete.
// if `replaceBars` true, removes `fs` bars and replaces with spinnerBar.
func GroupCB(fs Files, name, operation string, replaceBars bool, fns ...func(fs Files)) Files {
	return DefaultLabdh.GroupCB(fs, name, operation, replaceBars, fns...)
}

// GroupCB calls provided `fns` once all `fs` are downloaded.
// Can be used for merging multiple files,
// or for any operation on multiple set of files
// `operation` is replaced with ``, onComplete.
// if `replaceBars` true, removes `fs` bars and replaces with spinnerBar.
func (dl *Labdh) GroupCB(fs Files, name, operation string, replaceBars bool, fns ...func(fs Files)) Files {
	rcv := make(chan struct{})

	for _, f := range fs {
		f.cbChan = rcv
		f.removeBar = replaceBars
	}

	// once a file is downloaded,
	// we get informed through `rcv`,
	// once all given set of files are downloaded,
	// we call provided callback functions
	n := len(fs)

	dl.wg.Add(1)

	go func() {
		defer dl.wg.Done()
		for range rcv {
			n--
			if n == 0 {
				close(rcv)
				break
			}
		}

		bb := groupCBFuncsBar(name, operation, len(fns))
		for _, fn := range fns {
			if fn != nil {
				fn(fs)
				if bb != nil {
					bb.Increment()
				}
			}
		}
	}()

	return fs
}

// Close this downloader
// waits till all downloads are completed
func Close() {
	DefaultLabdh.Close()
}

// Close this downloader
// waits till all downloads are completed
func (dl *Labdh) Close() {
	close(dl.fic)
	dl.wg.Wait()

	p.Wait()

	refreshTicker.Stop()

	if ok, err := dl.client.ForceShutdown(); err != nil {
		log.Println(ok, err)
	}
}

func (dl *Labdh) download(f *File) {
	// set default options
	if f.Name == "" {
		f.Name = extractName(f.URLs[0])
	}

	if f.ConcReqs == 0 {
		f.ConcReqs = 3
	}

	if f.OnComplete == nil {
		f.OnComplete = dl.OnComplete
	}

	if dl.BaseDir != "" {
		f.Dir = filepath.Join(dl.BaseDir, f.Dir)
	}

	f.download(dl.client, dl.gidMap)
}

// workers are dispatched
func (dl *Labdh) workers() {

	dl.wg.Add(dl.ConcDls)

	for i := 0; i < dl.ConcDls; i++ {

		go func(i int) {
			log.Println("Worker", i, "started")
			defer dl.wg.Done()
			for f := range dl.fic {
				f.workerID = i
				dl.download(f)
			}
		}(i)
	}

}

/*************************************
  Different kind Operations on Files
*************************************/

// Dir sets `dir` for all given files
func (fs Files) Dir(dir string) Files {
	for _, f := range fs {
		f.Dir = dir
	}
	return fs
}

// OnComplete sets `onComplete` for all given files
func (fs Files) OnComplete(onComplete chan struct{}) Files {
	for _, f := range fs {
		f.OnComplete = onComplete
	}
	return fs
}

// Cookies sets `Cookies` for all given files
func (fs Files) Cookies(cookies []*http.Cookie) Files {
	for _, f := range fs {
		f.Cookies = cookies
	}
	return fs
}

// ConcReqs sets `concReqs` for all given files
func (fs Files) ConcReqs(concReqs int) Files {
	for _, f := range fs {
		f.ConcReqs = concReqs
	}
	return fs
}

// Referer sets `referer` for all given files
func (fs Files) Referer(referer string) Files {
	for _, f := range fs {
		f.Referer = referer
	}
	return fs
}

// CB sets `fns` for all given files
// also see `GroupCB()`.
func (fs Files) CB(name, operation string, fns ...func(f *File)) Files {
	for _, f := range fs {
		f.CBs = fns
		f.CBName = name
		f.CBOperation = operation
	}
	return fs
}

func setDefaultOpts(opts *Opts) *Labdh {

	logWriter := getLogWriter(opts.LogPrefix)

	log.SetOutput(logWriter)

	if opts.BaseDir == "" {
		opts.BaseDir = baseDir
	}

	if opts.ConcDls == 0 {
		opts.ConcDls = 3
	}

	opts.gidMap = &sync.Map{}
	opts.notify.gidMap = opts.gidMap

	var err error
	var port int

	opts.cmd, port, err = iaria.StartAria(opts.AriaArgs, opts.ConcDls)
	if err != nil {
		log.Fatalln("unable to start aria", err)
	}

	// let aria2c start, so we can connect
	time.Sleep(time.Millisecond * 100)

	rpcURI := "ws://localhost:" + strconv.Itoa(port) + "/jsonrpc"

	opts.client, err = aria.New(
		context.Background(),
		rpcURI,
		iaria.Secret,
		time.Second,
		opts.notify,
	)

	if err != nil {
		log.Fatalln("unable to create client:", err)
	}

	if _, err := opts.client.GetVersion(); err != nil {
		log.Fatalln("unable to get version info:", err)
	}

	// unexported fields
	opts.fic = make(chan *File)

	d := (*Labdh)(opts)

	// init workers
	d.workers()

	go updateFileStatuses(d.client, d.gidMap)

	return d
}

func getLogWriter(logPrefix string) io.Writer {

	err := os.MkdirAll(logDir, os.ModePerm)
	if err != nil {
		log.Fatalln(err)
	}

	now := time.Now().Format("Jan-2-2006-15:04:05-000")
	if logPrefix != "" {
		now = logPrefix + "-" + now
	}

	logFile := filepath.Join(logDir, now)

	f, err := os.Open(logFile)
	if err != nil {
		log.Println(err)
	}

	return f
}
