// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER

//go:build !unix

package model

import "os"

// ownerCheckSupported is false: this platform has no file owner IDs.
const ownerCheckSupported = false

// currentUID is unused without owner IDs.
var currentUID = func() int { return -1 }

// fileOwner reports no owner on this platform.
// TRLC-LINKS: REQ-EMG-108
func fileOwner(os.FileInfo) (int, bool) { return 0, false }
