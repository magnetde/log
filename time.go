package log

import (
	"strconv"
	"strings"
	"time"
)

func now() int64 {
	return time.Now().UnixNano()
}

var layout = strings.Split(strings.Replace(time.RFC3339, "T", " ", 1), "Z")[0]

func formatDate(t time.Time) string {
	return t.Local().Format(layout)
}

func formatDiff(time int64) string {
	// nanoseconds
	if time < 1000 {
		return "0"
	}

	// microseconds
	time = (time / 1000)

	if time < 1000 {
		if time < 10 {
			return "0"
		}

		if time >= 951 { // print 1 ms instead of 1.0 ms
			return "1 ms"
		}

		var prec int
		if time < 100 {
			prec = 2
		} else {
			prec = 1
		}
		return strconv.FormatFloat(float64(time)/1000, 'f', prec, 64) + " ms"
	}

	// milliseconds
	time = (time / 1000)

	if time < 1000 {
		return strconv.FormatInt(time, 10) + " ms"
	}

	// seconds
	time = (time / 1000)

	if time < 60 {
		return strconv.FormatInt(time, 10) + " s"
	}

	// minutes
	time = (time / 60)

	if time < 60 {
		return strconv.FormatInt(time, 10) + " m"
	}

	time = (time / 60)

	return strconv.FormatInt(time, 10) + " h"
}
