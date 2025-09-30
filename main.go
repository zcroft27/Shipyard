package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"log"
)

const (
CGROUP_VERSION_1 = 1
CGROUP_VERSION_2 = 2
)

// go run main.go run <cmd> <args>
func main() {
	switch os.Args[1] {
	case "run":
		run()
	case "child":
		cwd, _ := os.Getwd()
		rootfs := filepath.Join(cwd, "rootfs")
		overlayfs := filepath.Join(cwd, "overlay")
		child(rootfs, overlayfs)
	default:
		panic("help")
	}
}

func run() {

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

func child(rootfs string, overlayRoot string) {
    cgroupPath, cgroupVersion, err := cg()
    if err != nil {
        log.Fatalf("Couldn't create cgroups:, %w\n", err)
    }
    if cgroupVersion == CGROUP_VERSION_2 {
        defer cgroupCleanup(cgroupPath)
    }
    
    upperdir := filepath.Join(overlayRoot, "upper")
    workdir := filepath.Join(overlayRoot, "work")
    merged := filepath.Join(overlayRoot, "merged")
    
    must(os.MkdirAll(upperdir, 0755))
    must(os.MkdirAll(workdir, 0755))
    must(os.MkdirAll(merged, 0755))
    
    // CREATE PROC IN UPPERDIR (writable layer)
    must(os.MkdirAll(filepath.Join(upperdir, "proc"), 0755))
    
    // Mount OverlayFS
    must(syscall.Mount("overlay", merged, "overlay", 0,
        fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", rootfs, upperdir, workdir)))
    
    // Switch root to the merged directory
    must(syscall.Chroot(merged))
    must(syscall.Chdir("/"))
    must(syscall.Mount("proc", "proc", "proc", 0, ""))
    
    cmd := exec.Command(os.Args[2], os.Args[3:]...)
    cmd.Stdin = os.Stdin
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    
    must(cmd.Run())
    
    must(syscall.Unmount("/proc", 0))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func cg() (string, int, error) {
	cgroups := "/sys/fs/cgroup/"

	var cgroupPath string
	// Check if cgroup v2 (unified hierarchy)
	if _, err := os.Stat(filepath.Join(cgroups, "cgroup.controllers")); err == nil {
		// cgroup v2
		cgroupName := "shipyard"
		cgroupPath = filepath.Join(cgroups, cgroupName)

		if _, err := os.Stat(cgroupPath); os.IsNotExist(err) {
			if err := os.Mkdir(cgroupPath, 0755); err != nil {
				fmt.Printf("Warning: Could not create cgroup (skipping): %v\n", err)
				return cgroupPath, CGROUP_VERSION_2, err
			}
		}

		// cgroup v2 uses pids.max directly
		os.WriteFile(filepath.Join(cgroupPath, "pids.max"), []byte("20"), 0644)
		os.WriteFile(filepath.Join(cgroupPath, "cgroup.procs"), []byte(strconv.Itoa(os.Getpid())), 0644)
		return cgroupPath, CGROUP_VERSION_2, nil
	}

	// cgroup v1
	pids := filepath.Join(cgroups, "pids")

	// Check if pids controller exists
	if _, err := os.Stat(pids); os.IsNotExist(err) {
		fmt.Printf("Warning: cgroups not available (skipping resource limits)\n")
		return cgroupPath, CGROUP_VERSION_1, err
	}

	cgroupName := "shipyard"
	cgroupPath = filepath.Join(pids, cgroupName)

	if _, err := os.Stat(cgroupPath); os.IsNotExist(err) {
		if err := os.Mkdir(cgroupPath, 0755); err != nil {
			fmt.Printf("Warning: Could not create cgroup (skipping): %v\n", err)
			return cgroupPath, CGROUP_VERSION_1, err
		}
	}

	os.WriteFile(filepath.Join(cgroupPath, "pids.max"), []byte("20"), 0644)
	os.WriteFile(filepath.Join(cgroupPath, "notify_on_release"), []byte("1"), 0644)
	os.WriteFile(filepath.Join(cgroupPath, "cgroup.procs"), []byte(strconv.Itoa(os.Getpid())), 0644)
	return cgroupPath, CGROUP_VERSION_1, nil
}

func cgroupCleanup(cgroupPath string) {
    if cgroupPath != "" {
     	   os.RemoveAll(cgroupPath)
    }
}
