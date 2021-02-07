package labdh

import (
	"sync"

	aria "github.com/zyxar/argo/rpc"
)

type ariaNotifier struct {
	gidMap *sync.Map
}

func (an ariaNotifier) OnDownloadStart(events []aria.Event)      {}
func (an ariaNotifier) OnDownloadPause(events []aria.Event)      {}
func (an ariaNotifier) OnDownloadStop(events []aria.Event)       {}
func (an ariaNotifier) OnDownloadError(events []aria.Event)      {}
func (an ariaNotifier) OnBtDownloadComplete(events []aria.Event) {}

func (an ariaNotifier) OnDownloadComplete(events []aria.Event) {

	for _, event := range events {
		if sigInterface, ok := an.gidMap.Load(event.Gid); ok {
			bd := sigInterface.(*barData)
			// inform download is done
			bd.done <- struct{}{}
		}
	}

}
