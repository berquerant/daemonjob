// Copyright 2026 berquerant.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package daemonjobctl

import (
	"fmt"
	"runtime"
)

// Version is injected at build time via -ldflags="-X 'github.com/berquerant/daemonjob/internal/daemonjobctl.Version=...'".
var Version = "dev"

// VersionInfo contains the application version and Go runtime version.
type VersionInfo struct {
	Version   string
	GoVersion string
}

// GetVersion returns VersionInfo populated with the build version and Go runtime version.
func GetVersion() VersionInfo {
	return VersionInfo{
		Version:   Version,
		GoVersion: runtime.Version(),
	}
}

// String returns a formatted version string.
func (v VersionInfo) String() string {
	return fmt.Sprintf("daemonjobctl version %s (%s)", v.Version, v.GoVersion)
}
