# Shipyard
Containerization software inspired by Liz Rice's [Containers From Scratch](https://www.youtube.com/watch?v=8fi7uSYlOdc) talk.

# How To Use It
```
cd Shipyard/
sudo go build -o ship main.go
sudo ./ship run /bin/bash
```

# How Does It Work
- ./ship **run** `<cmd>`   
**run** is a special keyword that essentially forks or *clones* the running process by
executing `/proc/self/exe` and assign it new namespaces. Then, in the cloned process,
an [OverlayFS](https://docs.kernel.org/filesystems/overlayfs.html) is created to allow
`Shipyard/rootfs` to be read-only and the container interacts with a copy of it.


The root of the container is changed to `Shipyard/rootfs/`, a control group is set with a maximum
number of processes, and `/proc` (host) is mounted to `/proc` (container).

> [!NOTE]
> Mounting /proc and setting the `syscall.CLONE_NEWPID` flag in the child (container) process
> is key to have `ps` functionality and PIDs starting from 1.
> ```
> root@DESKTOP-JFEHBLD:/# ps
>    PID TTY          TIME CMD
>      1 ?        00:00:00 exe
>      6 ?        00:00:00 bash
>      9 ?        00:00:00 ps

This is really all you need to have a (relatively) isolated environment with some resource limits, control groups, and namespaces. 

# Building This Up Yourself

## Permissions
You will need root permissions using `sudo` in order to run this container.

## To Download and Extract an Ubuntu FS (if you don't want to use `Shipyard/rootfs`):

```
wget http://cdimage.ubuntu.com/ubuntu-base/releases/22.04/release/ubuntu-base-22.04-base-amd64.tar.gz

tar -xzf ubuntu-base-22.04-base-amd64.tar.gz -C rootfs
```

## Fix DNS Resolution
The file system used is (extremely) lightweight, and doesn't have basic things like curl, wget, nano, sudo, etc. I did this because **a)** I didn't know this ahead of time... but, **b)** I used a heavyweight Ubuntu file system and it was excessive in a few ways. 

Thankfully, `apt` is included, but executing `apt-get update` will fail with a message like `Failed...Temporary failure resolving 'security.ubuntu.com'` since the default image above does not come with DNS resolution configurations. Essentially, you just need to make a `resolv.conf` in `/etc/` with `nameserver <DNS server IP address>`. 
```
cd Shipyard/
cp /etc/resolv.conf ./rootfs/etc/resolv.conf
```
> [!NOTE]
> If you are using WSL, your default nameserver in `/etc/resolv.conf` may be `10.255.255.254` which is essentially the 'default gateway' within WSL to bridge to your host machine. This address will not work outside of WSL, so you may want to replace it with your LAN default gateway, e.g. `192.168.1.1` if a DNS service is provided, 
or use a popular DNS server like `1.1.1.1` or `8.8.8.8`.
