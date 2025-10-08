package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"log"
	"strings"
	"net/http"
	"encoding/json"
	"io"
	"compress/gzip"
	"archive/tar"
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
	case "pull":
		if len(os.Args) < 3 {
			panic("usage: pull <image>:<tag>")
		}
		pull(os.Args[2])
	case "child":
		cwd, _ := os.Getwd()
		rootfs := filepath.Join(cwd, "rootfs")
		if _, err := os.Stat(filepath.Join(cwd, "pull")); err == nil {
			rootfs = filepath.Join(cwd, "pull")
		}
		overlayfs := filepath.Join(cwd, "overlay")
		child(rootfs, overlayfs)
	default:
		panic("help")
	}
}

type ManifestV2 struct {
	SchemaVersion int    `json:"schemaVersion"`
	MediaType     string `json:"mediaType"`
	Config        struct {
		Digest string `json:"digest"`
	} `json:"config"`
	Layers []struct {
		MediaType string `json:"mediaType"`
		Digest    string `json:"digest"`
		Size      int64  `json:"size"`
	} `json:"layers"`
}

type AuthToken struct {
	Token       string `json:"token"`
	AccessToken string `json:"access_token"`
}

func pull(imageTag string) {
	fmt.Printf("Pulling image: %s\n", imageTag)

	// Parse image name and tag
	parts := strings.Split(imageTag, ":")
	imageName := parts[0]
	tag := "latest"
	if len(parts) > 1 {
		tag = parts[1]
	}

	// Handle library images (e.g., "alpine" -> "library/alpine")
	if !strings.Contains(imageName, "/") {
		imageName = "library/" + imageName
	}

	registry := "registry-1.docker.io"
	
	// Get auth token
	token := getAuthToken(registry, imageName)

	// Fetch manifest
	manifest := getManifest(registry, imageName, tag, token)

	// Create rootfs directory
	path, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}
	rootfs := path + "/custom-root-fs"
	must(os.MkdirAll(rootfs, 0755))

	// Download and extract each layer
	for i, layer := range manifest.Layers {
		fmt.Printf("Pulling layer %d/%d: %s\n", i+1, len(manifest.Layers), layer.Digest[:12])
		downloadAndExtractLayer(registry, imageName, layer.Digest, rootfs, token)
	}

	fmt.Println("Image pulled successfully!")
}

func getAuthToken(registry, imageName string) string {
	
	// Request authentication token from Docker Hub
	authURL := fmt.Sprintf("https://auth.docker.io/token?service=registry.docker.io&scope=repository:%s:pull", imageName)
	
	resp, err := http.Get(authURL)
	must(err)
	defer resp.Body.Close()

	var authResp AuthToken
	must(json.NewDecoder(resp.Body).Decode(&authResp))

	if authResp.Token != "" {
		return authResp.Token
	}
	return authResp.Token
}

func getManifest(registry, imageName, tag, token string) ManifestV2 {
	manifestURL := fmt.Sprintf("https://%s/v2/%s/manifests/%s", registry, imageName, tag)
	
	req, err := http.NewRequest("GET", manifestURL, nil)
	must(err)
	
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")

	fmt.Printf("manifest url: %s\n", manifestURL)
	// fmt.Println("Token: " + token)


	client := &http.Client{}
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		panic(fmt.Sprintf("Failed to get manifest: %d %s", resp.StatusCode, string(body)))
	}

	var manifest ManifestV2
	must(json.NewDecoder(resp.Body).Decode(&manifest))
	
	return manifest
}

func downloadAndExtractLayer(registry, imageName, digest, rootfs, token string) {
	blobURL := fmt.Sprintf("https://%s/v2/%s/blobs/%s", registry, imageName, digest)
	
	req, err := http.NewRequest("GET", blobURL, nil)
	must(err)
	
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		panic(fmt.Sprintf("Failed to download layer: %d", resp.StatusCode))
	}

	// Decompress gzip
	gzReader, err := gzip.NewReader(resp.Body)
	must(err)
	defer gzReader.Close()

	// Extract tar
	tarReader := tar.NewReader(gzReader)
	
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		must(err)

		target := filepath.Join(rootfs, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			must(os.MkdirAll(target, os.FileMode(header.Mode)))
		case tar.TypeReg:
			must(os.MkdirAll(filepath.Dir(target), 0755))
			outFile, err := os.Create(target)
			must(err)
			_, err = io.Copy(outFile, tarReader)
			outFile.Close()
			must(err)
			must(os.Chmod(target, os.FileMode(header.Mode)))
		case tar.TypeSymlink:
			must(os.MkdirAll(filepath.Dir(target), 0755))
			// Remove if exists
			os.Remove(target)
			must(os.Symlink(header.Linkname, target))
		}
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
