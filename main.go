package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
)

// go run main.go run <cmd> <args>
func main() {
	switch os.Args[1] {
	case "run":
		run()
	case "child":
		child()
	default:
		panic("help")
	}
}

func run() {
	fmt.Printf("Running %v \n", os.Args[2:])

	cmd := exec.Command("/proc/self/exe", append([]string{"child"}, os.Args[2:]...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags:   syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS,
		Unshareflags: syscall.CLONE_NEWNS,
	}

	must(cmd.Run())
}

func child() {
	fmt.Printf("Running %v \n", os.Args[2:])

	cg()

	// Define paths for OverlayFS.
	rootfs := "/home/zcroft27/shipyard/Shipyard/rootfs"
	overlayRoot := "/home/zcroft27/shipyard/Shipyard/overlay"
	upperdir := filepath.Join(overlayRoot, "upper")
	workdir := filepath.Join(overlayRoot, "work")
	merged := filepath.Join(overlayRoot, "merged")

	// Ensure the overlay directories exist.
	must(os.MkdirAll(upperdir, 0755))
	must(os.MkdirAll(workdir, 0755))
	must(os.MkdirAll(merged, 0755))

	// Mount OverlayFS
	must(syscall.Mount("overlay", merged, "overlay", 0,
		fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", rootfs, upperdir, workdir)))

	// Switch root to the merged directory.
	must(syscall.Chroot(merged))
	must(syscall.Chdir("/"))

	must(syscall.Mount("proc", "proc", "proc", 0, ""))

	cmd := exec.Command(os.Args[2], os.Args[3:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	must(cmd.Run())

	must(syscall.Unmount("/proc", 0))
	must(syscall.Unmount(merged, 0))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func cg() {
	cgroups := "/sys/fs/cgroup/"
	pids := filepath.Join(cgroups, "pids")
	cgroupName := "shipyard"
	cgroupPath := filepath.Join(pids, cgroupName)

	// Ensure cgroup directory exists.
	if _, err := os.Stat(cgroupPath); os.IsNotExist(err) {
		must(os.Mkdir(cgroupPath, 0755))
	}

	// Set max process limit to 20.
	must(ioutil.WriteFile(filepath.Join(cgroupPath, "pids.max"), []byte("20"), 0700))

	// Remove cgroup automatically after process exit.
	must(ioutil.WriteFile(filepath.Join(cgroupPath, "notify_on_release"), []byte("1"), 0700))

	// Add current process to cgroup.
	must(ioutil.WriteFile(filepath.Join(cgroupPath, "cgroup.procs"), []byte(strconv.Itoa(os.Getpid())), 0700))
}
