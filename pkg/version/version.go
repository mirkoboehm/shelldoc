// This file is part of shelldoc.
// © 2023, Mirko Boehm <mirko@kde.org> and the shelldoc contributors
// SPDX-License-Identifier: GPL-3.0

package version

import "runtime/debug"

var versionString = ""

// Version returns the program version as a string.
// If the version was set via ldflags at build time, that value is returned.
// Otherwise, it uses Go's built-in build info which includes VCS details
// when installed via "go install".
func Version() string {
	if versionString != "" {
		return versionString
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		return info.Main.Version
	}
	return "unknown"
}
