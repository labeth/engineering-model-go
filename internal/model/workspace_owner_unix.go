// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER

//go:build unix

package model

import (
	"os"
	"syscall"
)

// ownerCheckSupported reports whether fileOwner can name a file's owner.
const ownerCheckSupported = true

// currentUID is the user the search trusts besides root; tests replace it.
var currentUID = os.Getuid

// fileOwner returns the user ID that owns the file described by info.
// TRLC-LINKS: REQ-EMG-108
func fileOwner(info os.FileInfo) (int, bool) {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, false
	}
	return int(st.Uid), true
}
