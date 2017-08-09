package timeutil

import (
	"time"

	"github.com/appscode/client/cli"
)

var (
	DateFormat = map[string]string{
		"Y-m-d": "2006-01-02",
		"n/j/Y": "2-1-2006",
		"d-m-Y": "02-01-2006",
	}

	TimeFormat = map[string]string{
		"g:i A": "3:04PM",
		"H:i":   "15:04",
	}
)

func Format(epoc int64) string {
	user := cli.GetAuthOrDie()
	local, err := time.LoadLocation(user.Settings.TimeZone)
	if err != nil || user.Settings.TimeZone == "" {
		local, _ = time.LoadLocation("Local")
	}
	var formatter string
	if tf, ok := TimeFormat[user.Settings.TimeFormat]; ok {
		formatter = tf
	} else {
		formatter = "3:04PM"
	}

	if df, ok := DateFormat[user.Settings.DateFormat]; ok {
		formatter = formatter + " " + df
	} else {
		formatter = formatter + " " + "02-01-2006"
	}
	return time.Unix(epoc, 0).In(local).Format(formatter)
}
