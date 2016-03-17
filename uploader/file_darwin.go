package main

import (
	"os"
	"syscall"
	"time"
)

func timespecToTime(ts syscall.Timespec) time.Time {
	return time.Unix(int64(ts.Sec), int64(ts.Nsec))
}

func addOSFileInfo(item *fileInfo, info os.FileInfo) {
	st := info.Sys().(*syscall.Stat_t)
	accessTime := timespecToTime(st.Atimespec)
	createTime := timespecToTime(st.Ctimespec)
	item.Accessed, item.AccessedStr = accessTime.Unix(), accessTime.String()
	item.Created, item.CreatedStr = createTime.Unix(), createTime.String()
}
