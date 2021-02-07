package labdh

import (
	"strings"
	"time"

	"github.com/vbauerster/mpb/v5"
	"github.com/vbauerster/mpb/v5/decor"
)

// maxNameSize while displaying name on bar
var maxNameSize = 30

var p *mpb.Progress
var refreshChan chan time.Time

func init() {
	p = mpb.New(
		mpb.WithRefreshRate(time.Millisecond * 400),
	)
}

// newBar returns new Bar
func newBar(name string, removeBar bool, length int64) *mpb.Bar {
	var barEnd mpb.BarOption

	if removeBar {
		barEnd = mpb.BarRemoveOnComplete()
	} else {
		barEnd = mpb.BarFillerClearOnComplete()
	}

	tempName := shortName(name, maxNameSize)

	return p.Add(length, mpb.NewBarFiller(" ██  █▒", true),
		mpb.PrependDecorators(
			decor.OnComplete(
				decor.Name(tempName, decor.WC{W: maxNameSize + 2, C: decor.DidentRight}),
				name,
			),
			decor.OnComplete(
				decor.AverageSpeed(decor.UnitKB, "%.2f", decor.WC{W: 11, C: decor.DidentRight}),
				"",
			),
			decor.OnComplete(
				decor.CountersKiloByte("%.2f/%.2f", decor.WC{W: 18, C: decor.DidentRight}),
				"",
			),
		),
		mpb.AppendDecorators(
			decor.OnComplete(
				decor.Percentage(decor.WC{W: 5}), ""),
		),
		barEnd,
	)
}

func cbFuncsBar(previousBar *mpb.Bar, name, operation string, total int) *mpb.Bar {

	if name == "" {
		return nil
	}

	tempName := finalName(name, operation)

	return p.Add(int64(total), mpb.NewSpinnerFiller([]string{"∙∙∙", "●∙∙", "∙●∙", "∙∙●", "∙∙∙"}, mpb.SpinnerOnMiddle),
		mpb.BarQueueAfter(previousBar),
		mpb.PrependDecorators(
			decor.OnComplete(
				decor.Name(tempName, decor.WC{W: maxNameSize + 2, C: decor.DidentRight}),
				name,
			),
			decor.OnComplete(
				decor.CountersNoUnit("%d/%d", decor.WC{W: 5, C: decor.DidentRight}),
				"",
			),
		),
		mpb.BarFillerClearOnComplete(),
	)
}

func groupCBFuncsBar(name, operation string, total int) *mpb.Bar {

	if name == "" {
		return nil
	}

	tempName := finalName(name, operation)

	return p.Add(int64(total), mpb.NewSpinnerFiller([]string{"∙∙∙", "●∙∙", "∙●∙", "∙∙●", "∙∙∙"}, mpb.SpinnerOnMiddle),
		mpb.PrependDecorators(
			decor.OnComplete(
				decor.Name(tempName, decor.WC{W: maxNameSize + 2, C: decor.DidentRight}),
				name,
			),
			decor.OnComplete(
				decor.CountersNoUnit("%d%d", decor.WC{W: 5, C: decor.DidentRight}),
				"",
			),
		),
		mpb.BarFillerClearOnComplete(),
	)
}

var continuation = "..."

// shortName returns a shortened string of size maxNameSize
func shortName(s string, max int) string {

	i := strings.LastIndex(s, ".")
	if len(s) > max {
		if i > 0 {
			return s[:max-3] + continuation + s[i:]
		}
		return s[:max-3] + continuation
	}

	return s
}

func finalName(name, operation string) string {
	tempName := shortName(name, maxNameSize)

	if operation != "" {
		tempName = shortName(name+" - "+operation, maxNameSize)
	}
	return tempName
}
