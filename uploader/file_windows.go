package main

import (
	"os"
	"syscall"
	"time"
)

func addOSFileInfo(item *fileInfo, info os.FileInfo) {
	winInfo := info.Sys().(*syscall.Win32FileAttributeData)
	accessTime := time.Unix(0, winInfo.LastAccessTime.Nanoseconds())
	createTime := time.Unix(0, winInfo.CreationTime.Nanoseconds())
	item.Accessed, item.AccessedStr = accessTime.Unix(), accessTime.String()
	item.Created, item.CreatedStr = createTime.Unix(), createTime.String()
}
