package config

import (
	"log"
	"os/user"
	"runtime"
	"strconv"
	"syscall"

	"github.com/lomik/go-daemon"
)

// Daemonize fork process, change Credential to run-as-user, write pidfile
// Call once from main
func Daemonize(runAsUser *user.User, pidfile string) {
	runtime.LockOSThread()

	context := new(daemon.Context)
	if pidfile != "" {
		context.PidFileName = pidfile
		context.PidFilePerm = 0644
	}

	if runAsUser != nil {
		uid, err := strconv.ParseInt(runAsUser.Uid, 10, 0)
		if err != nil {
			log.Fatal(err)
		}

		gid, err := strconv.ParseInt(runAsUser.Gid, 10, 0)
		if err != nil {
			log.Fatal(err)
		}

		context.Credential = &syscall.Credential{
			Uid: uint32(uid),
			Gid: uint32(gid),
		}
	}

	child, _ := context.Reborn()

	if child != nil {
		return
	}
	defer context.Release()

	runtime.UnlockOSThread()
}
