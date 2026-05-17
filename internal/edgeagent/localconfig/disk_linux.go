//go:build linux

package localconfig

import "syscall"

func getDiskUsage(dir string) DiskInfo {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(dir, &stat); err != nil {
		return DiskInfo{}
	}
	total := int64(stat.Blocks) * int64(stat.Bsize)
	free := int64(stat.Bavail) * int64(stat.Bsize)
	return DiskInfo{
		TotalGB: total / (1024 * 1024 * 1024),
		FreeGB:  free / (1024 * 1024 * 1024),
		UsedGB:  (total - free) / (1024 * 1024 * 1024),
	}
}
