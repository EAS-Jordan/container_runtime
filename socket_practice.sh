#!/bin/bash
# socket_practice.sh - Demonstrates Linux file descriptors with sockets
# This script shows how Unix domain sockets work with file descriptors

set -e

echo "======= UNIX SOCKET DEMONSTRATION ======="
echo "This script demonstrates how Linux file descriptors work with sockets"

# Create a directory for our socket
SOCKET_DIR="/tmp/socket_demo"
SOCKET_PATH="$SOCKET_DIR/demo.sock"

# Clean up any previous runs
if [ -d "$SOCKET_DIR" ]; then
    rm -rf "$SOCKET_DIR"
fi
mkdir -p "$SOCKET_DIR"

echo "1. Creating a Unix domain socket server..."
# Create a simple socat server that echoes back data
socat UNIX-LISTEN:$SOCKET_PATH,fork EXEC:'echo "Server received: $(cat); date"' &
SERVER_PID=$!

# Give the server a moment to start
sleep 1

echo "2. Socket file created, observe how it appears in the filesystem:"
ls -la $SOCKET_PATH

echo "3. Check open file descriptors for the server process:"
echo "   Process ID: $SERVER_PID"
echo "   Open file descriptors:"
ls -la /proc/$SERVER_PID/fd | grep -v "total"

echo "4. Sending a message to the socket..."
echo "Hello, this is a test message" | socat - UNIX-CONNECT:$SOCKET_PATH

echo "5. Let's create a socket pair in a Python script:"
cat > $SOCKET_DIR/socket_pair.py << 'EOF'
#!/usr/bin/env python3
import socket
import os

# Create a pair of connected sockets
sock1, sock2 = socket.socketpair()
print(f"Created socket pair with file descriptors: {sock1.fileno()} and {sock2.fileno()}")

# Get our process ID
pid = os.getpid()
print(f"Process ID: {pid}")

# Write to sock1
message = b"Hello from sock1"
print(f"Writing to socket {sock1.fileno()}: {message.decode()}")
sock1.send(message)

# Read from sock2
data = sock2.recv(1024)
print(f"Read from socket {sock2.fileno()}: {data.decode()}")

# Write to sock2
message = b"Hello from sock2"
print(f"Writing to socket {sock2.fileno()}: {message.decode()}")
sock2.send(message)

# Read from sock1
data = sock1.recv(1024)
print(f"Read from socket {sock1.fileno()}: {data.decode()}")

print("\nPress Enter to continue and close the sockets...")
input()

# Close the sockets
sock1.close()
sock2.close()
EOF

chmod +x $SOCKET_DIR/socket_pair.py
echo "Running socket pair demo (Python)..."
$SOCKET_DIR/socket_pair.py

echo "6. Demonstrating socket-based process communication..."
# Create a script that uses a socket for IPC
cat > $SOCKET_DIR/ipc_server.py << 'EOF'
#!/usr/bin/env python3
import socket
import os
import sys

# Create a Unix domain socket
sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)

socket_path = sys.argv[1]
# Make sure the socket does not already exist
try:
    os.unlink(socket_path)
except OSError:
    if os.path.exists(socket_path):
        raise

print(f"Server starting on {socket_path}, PID: {os.getpid()}")
sock.bind(socket_path)
sock.listen(1)

print(f"Waiting for connections... (check fd in /proc/{os.getpid()}/fd/)")
conn, client_address = sock.accept()
print(f"Connection from client established")

try:
    # Receive the data
    while True:
        data = conn.recv(1024)
        if data:
            print(f"Received: {data.decode()}")
            conn.sendall(f"Echo: {data.decode()}".encode())
        else:
            print("No more data from client")
            break
finally:
    # Clean up the connection
    conn.close()
    sock.close()
    os.unlink(socket_path)
EOF

cat > $SOCKET_DIR/ipc_client.py << 'EOF'
#!/usr/bin/env python3
import socket
import sys
import time

socket_path = sys.argv[1]
print(f"Connecting to {socket_path}")

# Create a UDS socket
sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
sock.connect(socket_path)

try:
    # Send data
    for i in range(3):
        message = f"Message #{i+1} from client"
        print(f"Sending: {message}")
        sock.sendall(message.encode())
        
        data = sock.recv(1024)
        print(f"Received: {data.decode()}")
        time.sleep(1)
finally:
    print("Closing socket")
    sock.close()
EOF

chmod +x $SOCKET_DIR/ipc_server.py
chmod +x $SOCKET_DIR/ipc_client.py

echo "Starting IPC server..."
$SOCKET_DIR/ipc_server.py $SOCKET_PATH &
SERVER_PID2=$!
sleep 1

echo "7. Before client connects, let's examine the server's file descriptors:"
ls -la /proc/$SERVER_PID2/fd | grep -v "total"

echo "8. Now running the client to communicate with the server..."
$SOCKET_DIR/ipc_client.py $SOCKET_PATH

echo "9. Cleaning up processes..."
kill $SERVER_PID
# Server 2 should have terminated after client disconnected
# Just to be sure:
if ps -p $SERVER_PID2 > /dev/null; then
    kill $SERVER_PID2
fi

echo "10. Cleanup complete!"
echo ""
echo "This demo shows how sockets in Linux are represented as file descriptors."
echo "Each socket is assigned a file descriptor number that processes use for I/O."
echo "This is part of the 'everything is a file' philosophy in Unix/Linux systems."
echo ""
