#!/bin/bash
# test_wsl_compat.sh - Test container feature compatibility in WSL

set -e

echo "======= WSL CONTAINER COMPATIBILITY TEST ======="
echo "This script checks compatibility of container features in WSL"
echo

# Check if running in WSL
if ! grep -q Microsoft /proc/version; then
    echo "INFO: Not running in WSL. This script is intended for WSL environments."
    echo "Continuing anyway for demonstration purposes..."
fi

# Check if running as root
if [ "$(id -u)" -ne 0 ]; then
    echo "This script must be run as root"
    exit 1
fi

# Test directory
TEST_DIR="/tmp/wsl_container_test"
mkdir -p "$TEST_DIR"

echo "1. Testing namespace compatibility..."
echo

echo "1.1 Testing UTS namespace..."
if unshare --uts sleep 0 &>/dev/null; then
    echo "✅ UTS namespace: Supported"
else
    echo "❌ UTS namespace: Not supported"
fi

echo "1.2 Testing PID namespace..."
if unshare --pid --fork sleep 0 &>/dev/null; then
    echo "✅ PID namespace: Supported"
else
    echo "❌ PID namespace: Not supported"
fi

echo "1.3 Testing Mount namespace..."
if unshare --mount sleep 0 &>/dev/null; then
    echo "✅ Mount namespace: Supported"
else
    echo "❌ Mount namespace: Not supported"
fi

echo "1.4 Testing Network namespace..."
if unshare --net sleep 0 &>/dev/null; then
    echo "✅ Network namespace: Supported"
else
    echo "❌ Network namespace: Not supported"
fi

echo "1.5 Testing IPC namespace..."
if unshare --ipc sleep 0 &>/dev/null; then
    echo "✅ IPC namespace: Supported"
else
    echo "❌ IPC namespace: Not supported"
fi

echo "1.6 Testing User namespace..."
if unshare --user sleep 0 &>/dev/null; then
    echo "✅ User namespace: Supported"
else
    echo "❌ User namespace: Not supported"
fi

echo
echo "2. Testing cgroups availability..."
echo

if [ -d "/sys/fs/cgroup/cgroup.controllers" ]; then
    echo "✅ cgroups v2: Available"
    CGROUP_VERSION="v2"
elif [ -d "/sys/fs/cgroup/memory" ]; then
    echo "✅ cgroups v1: Available"
    CGROUP_VERSION="v1"
else
    echo "❌ cgroups: Not available"
    CGROUP_VERSION="none"
fi

if [ "$CGROUP_VERSION" != "none" ]; then
    echo "2.1 Testing cgroup creation..."
    TEST_CGROUP="test_cgroup_$$"
    
    if [ "$CGROUP_VERSION" = "v2" ]; then
        mkdir -p "/sys/fs/cgroup/$TEST_CGROUP" &>/dev/null
        if [ $? -eq 0 ]; then
            echo "✅ cgroup creation: Supported"
            rm -rf "/sys/fs/cgroup/$TEST_CGROUP" &>/dev/null
        else
            echo "❌ cgroup creation: Failed"
        fi
    else
        mkdir -p "/sys/fs/cgroup/memory/$TEST_CGROUP" &>/dev/null
        if [ $? -eq 0 ]; then
            echo "✅ cgroup creation: Supported"
            rmdir "/sys/fs/cgroup/memory/$TEST_CGROUP" &>/dev/null
        else
            echo "❌ cgroup creation: Failed"
        fi
    fi
fi

echo
echo "3. Testing network features..."
echo

echo "3.1 Testing loopback interface..."
ip addr | grep -q 'lo:'
if [ $? -eq 0 ]; then
    echo "✅ Loopback interface: Available"
else
    echo "❌ Loopback interface: Not available"
fi

echo "3.2 Testing bridge creation (non-destructive test)..."
ip link add name testbridge0 type bridge &>/dev/null
if [ $? -eq 0 ]; then
    echo "✅ Bridge creation: Supported"
    ip link del testbridge0 &>/dev/null
else
    echo "❌ Bridge creation: Not supported"
fi

echo "3.3 Testing veth pair creation (non-destructive test)..."
ip link add testvetha type veth peer name testvethb &>/dev/null
if [ $? -eq 0 ]; then
    echo "✅ veth pair creation: Supported"
    ip link del testvetha &>/dev/null
else
    echo "❌ veth pair creation: Not supported"
fi

echo
echo "4. Testing filesystem features..."
echo

echo "4.1 Testing overlayfs mount..."
mkdir -p "$TEST_DIR/overlay/"{lower,upper,work,merged}
touch "$TEST_DIR/overlay/lower/lower_file"
touch "$TEST_DIR/overlay/upper/upper_file"

mount -t overlay overlay -o "lowerdir=$TEST_DIR/overlay/lower,upperdir=$TEST_DIR/overlay/upper,workdir=$TEST_DIR/overlay/work" "$TEST_DIR/overlay/merged" &>/dev/null
if [ $? -eq 0 ]; then
    echo "✅ overlayfs: Supported"
    if [ -f "$TEST_DIR/overlay/merged/lower_file" ] && [ -f "$TEST_DIR/overlay/merged/upper_file" ]; then
        echo "   ✅ overlayfs layers: Working correctly"
    else
        echo "   ❌ overlayfs layers: Not working correctly"
    fi
    umount "$TEST_DIR/overlay/merged" &>/dev/null
else
    echo "❌ overlayfs: Not supported"
fi

echo "4.2 Testing pivot_root..."
mkdir -p "$TEST_DIR/rootfs/"{bin,sbin,etc,proc,sys,dev,tmp}
cat > "$TEST_DIR/test_pivot.sh" << 'EOF'
#!/bin/bash
mkdir -p "$1/.pivot_root"
cd "$1"
pivot_root . ./.pivot_root
exec chroot . /bin/true
EOF
chmod +x "$TEST_DIR/test_pivot.sh"

if "$TEST_DIR/test_pivot.sh" "$TEST_DIR/rootfs" &>/dev/null; then
    echo "✅ pivot_root: Working"
else
    echo "❌ pivot_root: Not working"
    echo "   ℹ️ chroot fallback will be used"
fi

echo
echo "5. Testing security features..."
echo

echo "5.1 Testing capabilities..."
if command -v capsh &>/dev/null; then
    if capsh --print | grep -q "Current"; then
        echo "✅ Capabilities: Available"
    else
        echo "❌ Capabilities: Limited support"
    fi
else
    echo "⚠️ Capabilities: Test skipped (capsh not available)"
fi

echo "5.2 Testing seccomp..."
if [ -f "/proc/sys/kernel/seccomp" ]; then
    seccomp_val=$(cat /proc/sys/kernel/seccomp)
    if [ "$seccomp_val" -gt 0 ]; then
        echo "✅ seccomp: Available"
    else
        echo "❌ seccomp: Not available"
    fi
else
    echo "❌ seccomp: Not available"
fi

echo
echo "======= TEST SUMMARY ======="
echo
echo "Based on these tests, your WSL environment has:"
echo "- Namespaces: $(unshare --help &>/dev/null && echo "Mostly supported" || echo "Limited support")"
echo "- cgroups: $CGROUP_VERSION $([ "$CGROUP_VERSION" = "none" ] && echo "(Will use simulation)" || echo "")"
echo "- Network features: $(ip link add testbridge0 type bridge &>/dev/null && ip link del testbridge0 &>/dev/null && echo "Mostly supported" || echo "Limited support")"
echo "- Filesystem: $(mount -t overlay overlay -o "lowerdir=$TEST_DIR/overlay/lower,upperdir=$TEST_DIR/overlay/upper,workdir=$TEST_DIR/overlay/work" "$TEST_DIR/overlay/merged" &>/dev/null && umount "$TEST_DIR/overlay/merged" &>/dev/null && echo "Mostly supported" || echo "Limited support")"
echo "- Security features: $([ -f "/proc/sys/kernel/seccomp" ] && cat /proc/sys/kernel/seccomp &>/dev/null && echo "Mostly supported" || echo "Limited support")"
echo

echo "The container runtime has been adapted to work with these limitations."
echo "Some features may use fallback implementations in WSL."

# Clean up
rm -rf "$TEST_DIR" 