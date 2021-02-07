package labdh

import (
	"log"
	"strconv"
	"sync"
	"time"

	aria "github.com/zyxar/argo/rpc"
)

var refreshPeriod = time.Millisecond * 300
var refreshTicker = time.NewTicker(refreshPeriod)

type fileStatus struct {
	gid             string
	completedLength int64
	totalLength     int64
	downloadSpeed   string
}

func updateFileStatuses(client aria.Client, gidMap *sync.Map) {

	for range refreshTicker.C {

		fileStatuses := getFileStatuses(client, gidMap)

		setFileStatuses(gidMap, fileStatuses)
	}
}

func getFileStatuses(client aria.Client, gidMap *sync.Map) []fileStatus {
	methods := fileStatusMethods(gidMap)

	if len(methods) != 0 {

		res, err := client.Multicall(methods)

		if err != nil {
			log.Println("Multicall failed:", err)
		}

		return interfaceToFileStatuses(res)
	}

	return nil
}

func setFileStatuses(gidMap *sync.Map, fileStatuses []fileStatus) {

	for _, fs := range fileStatuses {
		if ki, ok := gidMap.Load(fs.gid); ok {
			bd := ki.(*barData)

			// first setTotal
			if fs.totalLength >= bd.totalLength {
				bd.bar.SetTotal(fs.totalLength, false)
			}

			if !bd.refilled && bd.isResumed {
				bd.bar.SetRefill(fs.completedLength)
				bd.refilled = true
			}

			bd.bar.SetCurrent(fs.completedLength)
			bd.bar.DecoratorEwmaUpdate(refreshPeriod)
		}
	}
}

func fileStatusMethods(gidMap *sync.Map) []aria.Method {

	methods := make([]aria.Method, 0)
	gidMap.Range(func(k interface{}, v interface{}) bool {

		gid := k.(string)

		methods = append(methods, aria.Method{
			Name: "aria2.tellStatus",
			Params: []interface{}{
				"token:" + "nilsocket/dl", gid, []string{"gid", "completedLength", "totalLength", "downloadSpeed"},
			},
		})

		return true
	})

	return methods
}

func interfaceToFileStatuses(res []interface{}) []fileStatus {

	s := make([]fileStatus, 0, len(res))

	for _, r := range res {
		rSlice := r.([]interface{})

		// fmt.Println(rSlice)
		valueMap := rSlice[0].(map[string]interface{})

		cl, _ := strconv.ParseInt(valueMap["completedLength"].(string), 10, 64)
		tl, _ := strconv.ParseInt(valueMap["totalLength"].(string), 10, 64)

		s = append(s, fileStatus{
			gid:             valueMap["gid"].(string),
			completedLength: cl,
			totalLength:     tl,
			downloadSpeed:   valueMap["downloadSpeed"].(string),
		})
	}

	return s
}
