//go:build !unix

package style

import "os"

func termWidth(*os.File) int { return 0 }
