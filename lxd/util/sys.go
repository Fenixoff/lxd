//go:build linux && cgo && !agent
// +build linux,cgo,!agent

package util

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/unix"
	log "gopkg.in/inconshreveable/log15.v2"

	"github.com/lxc/lxd/shared/idmap"
	"github.com/lxc/lxd/shared/logger"
	"github.com/lxc/lxd/shared/osarch"
)

// GetArchitectures returns the list of supported architectures.
func GetArchitectures() ([]int, error) {
	architectures := []int{}

	architectureName, err := osarch.ArchitectureGetLocal()
	if err != nil {
		return nil, err
	}

	architecture, err := osarch.ArchitectureId(architectureName)
	if err != nil {
		return nil, err
	}
	architectures = append(architectures, architecture)

	personalities, err := osarch.ArchitecturePersonalities(architecture)
	if err != nil {
		return nil, err
	}
	for _, personality := range personalities {
		architectures = append(architectures, personality)
	}
	return architectures, nil
}

// GetIdmapSet reads the uid/gid allocation.
func GetIdmapSet() *idmap.IdmapSet {
	idmapSet, err := idmap.DefaultIdmapSet("", "")
	if err != nil {
		logger.Warn("Error reading default uid/gid map", log.Ctx{"err": err.Error()})
		logger.Warnf("Only privileged containers will be able to run")
		idmapSet = nil
	} else {
		kernelIdmapSet, err := idmap.CurrentIdmapSet()
		if err == nil {
			logger.Infof("Kernel uid/gid map:")
			for _, lxcmap := range kernelIdmapSet.ToLxcString() {
				logger.Infof(fmt.Sprintf(" - %s", lxcmap))
			}
		}

		if len(idmapSet.Idmap) == 0 {
			logger.Warnf("No available uid/gid map could be found")
			logger.Warnf("Only privileged containers will be able to run")
			idmapSet = nil
		} else {
			logger.Infof("Configured LXD uid/gid map:")
			for _, lxcmap := range idmapSet.Idmap {
				suffix := ""

				if lxcmap.Usable() != nil {
					suffix = " (unusable)"
				}

				for _, lxcEntry := range lxcmap.ToLxcString() {
					logger.Infof(" - %s%s", lxcEntry, suffix)
				}
			}

			err = idmapSet.Usable()
			if err != nil {
				logger.Warnf("One or more uid/gid map entry isn't usable (typically due to nesting)")
				logger.Warnf("Only privileged containers will be able to run")
				idmapSet = nil
			}
		}
	}
	return idmapSet
}

// GetExecPath returns the path to the current binary
func GetExecPath() string {
	execPath := os.Getenv("LXD_EXEC_PATH")
	if execPath != "" {
		return execPath
	}

	execPath, err := os.Readlink("/proc/self/exe")
	if err != nil {
		execPath = "bad-exec-path"
	}

	// The execPath from /proc/self/exe can end with " (deleted)" if the lxd binary has been removed/changed
	// since the lxd process was started, strip this so that we only return a valid path.
	return strings.TrimSuffix(execPath, " (deleted)")
}

// ReplaceDaemon replaces the LXD process.
func ReplaceDaemon() error {
	err := unix.Exec(GetExecPath(), os.Args, os.Environ())
	if err != nil {
		return err
	}

	return nil
}
