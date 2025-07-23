#!/bin/bash
# container_playground.sh - A simple script to demonstrate container concepts
# This script shows basic containerization using Linux namespaces and chroot

set -e

# Check if running as root
if [ "$(id -u)" -ne 0 ]; then
    echo "This script must be run as root"
    exit 1
fi

echo "======= CONTAINER PLAYGROUND ======="
echo "This script demonstrates basic container concepts using Linux namespaces and chroot"

# Create a directory for our container filesystem
CONTAINER_ROOT="/tmp/container_playground"
CONTAINER_HOSTNAME="my-container"

# Clean up any previous runs
if [ -d "$CONTAINER_ROOT" ]; then
    echo "Cleaning up previous container playground..."
    umount -l $CONTAINER_ROOT/proc 2>/dev/null || true
    umount -l $CONTAINER_ROOT/sys 2>/dev/null || true
    umount -l $CONTAINER_ROOT/dev/pts 2>/dev/null || true
    umount -l $CONTAINER_ROOT/dev 2>/dev/null || true
    rm -rf "$CONTAINER_ROOT"
fi

echo "1. Creating a minimal container rootfs..."
mkdir -p "$CONTAINER_ROOT"/{bin,lib,lib64,proc,sys,dev,dev/pts,etc,tmp}

# Copy essential binaries and their dependencies
echo "2. Copying essential binaries..."
cp /bin/bash $CONTAINER_ROOT/bin/
cp /bin/ls $CONTAINER_ROOT/bin/
cp /bin/cat $CONTAINER_ROOT/bin/
cp /bin/ps $CONTAINER_ROOT/bin/
cp /bin/mount $CONTAINER_ROOT/bin/
cp /bin/echo $CONTAINER_ROOT/bin/

echo "3. Resolving and copying library dependencies..."
# Find dependencies and copy them
for binary in /bin/bash /bin/ls /bin/cat /bin/ps /bin/mount /bin/echo; do
    deps=$(ldd $binary | grep -o '/lib.*\.so[^ ]*' | sort -u)
    for dep in $deps; do
        mkdir -p "$CONTAINER_ROOT$(dirname $dep)"
        cp $dep "$CONTAINER_ROOT$dep"
    done
done

# Create /etc/passwd with root user
echo "4. Creating minimal /etc/passwd..."
cat > $CONTAINER_ROOT/etc/passwd << EOF
root:x:0:0:root:/root:/bin/bash
EOF

# Create a simple hostname file
echo "$CONTAINER_HOSTNAME" > $CONTAINER_ROOT/etc/hostname

echo "5. Setting up mount points..."
# Mount proc, sys, and dev
mount -t proc proc $CONTAINER_ROOT/proc
mount -t sysfs sysfs $CONTAINER_ROOT/sys
mount -t devtmpfs devtmpfs $CONTAINER_ROOT/dev
mount -t devpts devpts $CONTAINER_ROOT/dev/pts

echo "6. Creating demo script inside container..."
cat > $CONTAINER_ROOT/demo.sh << 'EOF'
#!/bin/bash
echo "--- INSIDE CONTAINER ---"
echo "Hostname: $(cat /etc/hostname)"
echo "Process tree:"
ps aux
echo
echo "Network interfaces:"
cat /proc/net/dev
echo
echo "Mount points:"
mount
echo
echo "PID 1 in container is actually PID $(cat /proc/1/status | grep "^NSpid:" | awk '{print $3}') on host"
echo
echo "Try commands like: ls, cat /etc/passwd"
EOF
chmod +x $CONTAINER_ROOT/demo.sh

echo "============================================="
echo "Entering container with isolated namespaces"
echo "============================================="
echo
echo "The container will have its own PID, UTS (hostname), and mount namespaces"
echo "Type 'exit' to leave the container environment"
echo

# Enter the container with new namespaces and chroot
unshare --mount --uts --pid --fork --mount-proc=$CONTAINER_ROOT/proc chroot $CONTAINER_ROOT /bin/bash -c "hostname $CONTAINER_HOSTNAME && /demo.sh && exec /bin/bash"

echo "Exited container environment"
echo "7. Cleaning up..."

# Unmount filesystems
umount -l $CONTAINER_ROOT/proc 2>/dev/null || true
umount -l $CONTAINER_ROOT/sys 2>/dev/null || true
umount -l $CONTAINER_ROOT/dev/pts 2>/dev/null || true
umount -l $CONTAINER_ROOT/dev 2>/dev/null || true

# Remove container directory
rm -rf "$CONTAINER_ROOT"

echo "8. Cleanup complete!"
echo
echo "This demo showed a simplified version of how containers work:"
echo "1. Isolated filesystem (chroot)"
echo "2. Process isolation (PID namespace)"
echo "3. Hostname isolation (UTS namespace)"
echo "4. Mount isolation (mount namespace)"
echo
echo "Real containers also use:"
echo "- Network namespaces for network isolation"
echo "- User namespaces for better security"
echo "- Cgroups for resource limiting"
echo "- Seccomp and capabilities for security"
echo "- Overlay filesystems for efficient storage" 