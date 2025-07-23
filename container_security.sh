#!/bin/bash
# container_security.sh - Demonstrates container security concepts
# WARNING: This script is for EDUCATIONAL purposes only

set -e

# Check if running as root
if [ "$(id -u)" -ne 0 ]; then
    echo "This script must be run as root"
    exit 1
fi

echo "======= CONTAINER SECURITY DEMONSTRATION ======="
echo "This script demonstrates container security concepts and potential risks"
echo "WARNING: For educational purposes only. Do not run in production environments."
echo

CONTAINER_ROOT="/tmp/container_security"
SECURITY_DIR="$CONTAINER_ROOT/security"

# Clean up any previous runs
if [ -d "$CONTAINER_ROOT" ]; then
    echo "Cleaning up previous container security playground..."
    umount -l $CONTAINER_ROOT/proc 2>/dev/null || true
    umount -l $CONTAINER_ROOT/host 2>/dev/null || true
    rm -rf "$CONTAINER_ROOT"
fi

echo "1. Creating a basic container environment..."
mkdir -p "$CONTAINER_ROOT"/{bin,lib,lib64,proc,security,host}

# Copy essential binaries and their dependencies
cp /bin/bash $CONTAINER_ROOT/bin/
cp /bin/ls $CONTAINER_ROOT/bin/
cp /bin/cat $CONTAINER_ROOT/bin/

# Find dependencies and copy them
for binary in /bin/bash /bin/ls /bin/cat; do
    deps=$(ldd $binary | grep -o '/lib.*\.so[^ ]*' | sort -u)
    for dep in $deps; do
        mkdir -p "$CONTAINER_ROOT$(dirname $dep)"
        cp $dep "$CONTAINER_ROOT$dep"
    done
done

# Mount proc inside the container
mount -t proc proc $CONTAINER_ROOT/proc

echo "2. Creating security demonstration scripts..."
mkdir -p $SECURITY_DIR

# Script 1: Mount namespace escape
cat > $SECURITY_DIR/mount_escape.sh << 'EOF'
#!/bin/bash
echo "[SECURITY RISK] Mount namespace escape demonstration"
echo "This shows how a container with lax mount restrictions can access host files"

# Create a mount point to the host filesystem
mkdir -p /host
mount -t proc proc /proc

# Try to mount the host root filesystem
if mount -o bind /proc/1/root /host; then
    echo "[SECURITY BREACH] Successfully mounted host filesystem at /host"
    echo "Contents of host's /etc/passwd:"
    cat /host/etc/passwd
    echo
    echo "This is possible when containers have CAP_SYS_ADMIN or --privileged"
else
    echo "Mount failed - container properly secured"
fi
EOF
chmod +x $SECURITY_DIR/mount_escape.sh

# Script 2: Kernel module loading (if exposed)
cat > $SECURITY_DIR/module_loading.sh << 'EOF'
#!/bin/bash
echo "[SECURITY RISK] Kernel module loading demonstration"
echo "This shows how a container with access to kernel modules can affect the host"

if [ -w /proc/sys/kernel/modules_disabled ]; then
    echo "[SECURITY BREACH] Container can load kernel modules"
    echo "0" > /proc/sys/kernel/modules_disabled
    echo "With access to the host's kernel modules, an attacker could load malicious modules"
else
    echo "Module loading properly restricted - container secured"
fi
EOF
chmod +x $SECURITY_DIR/module_loading.sh

# Script 3: Capability demonstration
cat > $SECURITY_DIR/capabilities_demo.sh << 'EOF'
#!/bin/bash
echo "[INFO] Container capabilities demonstration"
echo "This shows what Linux capabilities the container has"

echo "Process capabilities:"
cat /proc/self/status | grep ^Cap

echo
echo "Explanation:"
echo "CAP_SYS_ADMIN - Allows mount operations and other privileged actions"
echo "CAP_NET_ADMIN - Allows network configuration changes"
echo "CAP_SYS_PTRACE - Allows process tracing and memory access"
echo "CAP_SYS_BOOT - Allows reboot and kexec operations"
echo
echo "In secure containers, capabilities should be dropped to the minimum required"
EOF
chmod +x $SECURITY_DIR/capabilities_demo.sh

# Script 4: Resource exhaustion
cat > $SECURITY_DIR/resource_abuse.sh << 'EOF'
#!/bin/bash
echo "[SECURITY RISK] Resource exhaustion demonstration"
echo "This shows how a container without resource limits can exhaust host resources"

echo "Starting a process that consumes CPU resources..."
echo "Press Ctrl+C to stop"

# Start CPU-intensive process
cat /dev/zero > /dev/null &
PID=$!

echo "Process $PID is consuming CPU"
echo "Without cgroups limits, this could affect other containers and the host"
echo
echo "Waiting 5 seconds..."
sleep 5
kill $PID
echo "Process stopped"

echo
echo "In properly configured containers, cgroups would limit CPU, memory, and I/O usage"
EOF
chmod +x $SECURITY_DIR/resource_abuse.sh

# Script 5: Shared kernel detection
cat > $SECURITY_DIR/shared_kernel.sh << 'EOF'
#!/bin/bash
echo "[INFO] Shared kernel demonstration"
echo "This shows how containers share the kernel with the host"

echo "Container's kernel version:"
uname -a

echo
echo "Kernel modules:"
ls -la /proc/modules 2>/dev/null || echo "Access to kernel modules restricted (good)"

echo
echo "This demonstrates that containers use the host's kernel"
echo "A vulnerability in the kernel affects ALL containers on the host"
echo "This is a key difference from VMs, which have their own kernels"
EOF
chmod +x $SECURITY_DIR/shared_kernel.sh

# Script 6: Device access
cat > $SECURITY_DIR/device_access.sh << 'EOF'
#!/bin/bash
echo "[SECURITY RISK] Device access demonstration"
echo "This shows the risks of exposing device files to containers"

echo "Checking access to sensitive devices:"
devices=("/dev/mem" "/dev/kmem" "/dev/sda" "/dev/nvme0")

for dev in "${devices[@]}"; do
    if [ -r "$dev" ] 2>/dev/null; then
        echo "[SECURITY BREACH] Container has read access to $dev"
        if [ -w "$dev" ] 2>/dev/null; then
            echo "[SECURITY BREACH] Container has write access to $dev"
        fi
    else
        echo "No access to $dev (good)"
    fi
done

echo
echo "Direct access to device files can lead to security breaches"
echo "Containers should only have access to necessary devices"
EOF
chmod +x $SECURITY_DIR/device_access.sh

# Main demo script
cat > $CONTAINER_ROOT/security_demo.sh << 'EOF'
#!/bin/bash
echo "====== CONTAINER SECURITY DEMONSTRATION ======"
echo "This environment demonstrates container security concepts"
echo

PS3="Select a security demonstration: "
options=(
    "Mount Namespace Escape" 
    "Kernel Module Loading" 
    "Container Capabilities" 
    "Resource Exhaustion"
    "Shared Kernel Detection"
    "Device Access Risks"
    "Exit"
)

select opt in "${options[@]}"
do
    case $opt in
        "Mount Namespace Escape")
            /security/mount_escape.sh
            echo
            ;;
        "Kernel Module Loading")
            /security/module_loading.sh
            echo
            ;;
        "Container Capabilities")
            /security/capabilities_demo.sh
            echo
            ;;
        "Resource Exhaustion")
            /security/resource_abuse.sh
            echo
            ;;
        "Shared Kernel Detection")
            /security/shared_kernel.sh
            echo
            ;;
        "Device Access Risks")
            /security/device_access.sh
            echo
            ;;
        "Exit")
            echo "Exiting security demonstration"
            exit 0
            ;;
        *) 
            echo "Invalid option"
            ;;
    esac
done
EOF
chmod +x $CONTAINER_ROOT/security_demo.sh

echo "3. Starting security demonstration container..."
echo "This will start a basic container with various security demonstrations"
echo 
echo "[IMPORTANT] This is for educational purposes only to understand security risks"
echo "Press Ctrl+D or type 'exit' to leave the container"
echo

# Enter the container with different security settings
# Note: Using unshare for simplicity, real demos would use Docker/Podman/etc.
unshare --mount --uts --pid --fork --mount-proc=$CONTAINER_ROOT/proc chroot $CONTAINER_ROOT /security_demo.sh

echo "Exited container environment"
echo "Cleaning up..."

# Unmount and clean up
umount -l $CONTAINER_ROOT/proc 2>/dev/null || true
umount -l $CONTAINER_ROOT/host 2>/dev/null || true
rm -rf "$CONTAINER_ROOT"

echo "Clean up complete!"
echo
echo "Key security lessons:"
echo "1. Containers share the host kernel - kernel exploits affect all containers"
echo "2. Privileged containers or those with excessive capabilities are risky"
echo "3. Resource limits (cgroups) are essential to prevent DoS attacks"
echo "4. Mount namespace isolation is critical for security"
echo "5. Device access should be restricted to the minimum necessary"
echo
echo "Best practices:"
echo "- Run containers as non-root users"
echo "- Use read-only root filesystems where possible"
echo "- Drop all capabilities except those specifically required"
echo "- Apply seccomp and AppArmor/SELinux profiles"
echo "- Set resource limits for all containers" 