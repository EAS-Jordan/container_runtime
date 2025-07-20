# UnionFS and OverlayFS

Union filesystems are a critical technology for container runtimes, enabling efficient image layering and copy-on-write functionality. They allow multiple filesystems (or directories) to be mounted simultaneously, with their contents appearing to be merged.

## Core Concept

A union filesystem takes multiple directories (called branches or layers) and presents them as a single unified directory structure. This allows for:

1. **Layering** - Stacking multiple filesystem trees
2. **Copy-on-Write (CoW)** - Modifications to read-only layers are stored in a writable layer
3. **Efficient storage** - Common files are stored only once across multiple containers

## OverlayFS

OverlayFS is the most widely used union filesystem for modern container engines. It's integrated into the mainline Linux kernel since version 3.18 and is the default storage driver for Docker and many other container runtimes.

### Basic Structure

OverlayFS combines an upper directory and a lower directory into a merged view:

```
          Merged View (Container's /)
               /    \
              /      \
   Upper Dir (RW)    Lower Dir(s) (RO)
   (container layer)  (image layers)
```

When searching for a file, OverlayFS first looks in the upper directory, and if not found, it looks in the lower directory.

### Key Components

- **lowerdir**: Read-only layers (can be multiple, colon-separated directories)
- **upperdir**: Read-write layer where changes are stored
- **workdir**: Used internally by OverlayFS for CoW operations
- **merged**: The mount point where the unified view is exposed

### File Operations in OverlayFS

1. **Reading a file**:
   - If the file exists in the upper layer, return it
   - Otherwise, return the file from the lower layer

2. **Writing to a file**:
   - If the file exists only in the lower layer, copy it to the upper layer (copy-up)
   - Make modifications to the copy in the upper layer
   - The original file in the lower layer remains unchanged

3. **Creating a file**:
   - New files are created directly in the upper layer

4. **Deleting a file**:
   - If the file exists in the upper layer, delete it
   - If the file exists only in the lower layer, create a "whiteout" file in the upper layer to mark it as deleted

## Container Image Layers

Container images typically consist of multiple layers stacked on top of each other:

```
Read-Write Layer (Container)
└── Image Layer N (e.g., app code)
    └── Image Layer N-1 (e.g., dependencies)
        └── Image Layer N-2 (e.g., OS packages)
            └── Base Image Layer (e.g., minimal OS)
```

### Layer Operations in Container Runtimes

1. **Image Pull**: Downloading individual layers (often as compressed tarballs)
2. **Layer Extraction**: Each layer is extracted to its own directory
3. **Mounting**: All layers are mounted together using OverlayFS
4. **Container Creation**: A new writable layer is added on top for the container

## Advantages of UnionFS/OverlayFS in Containers

1. **Storage Efficiency**: 
   - Multiple containers using the same base image share the image layers
   - Only store differences (deltas) between containers

2. **Quick Container Creation**:
   - Creating a new container only requires setting up the overlay mount
   - No need to copy the entire filesystem

3. **Immutable Infrastructure**:
   - Base layers remain read-only
   - Makes reproducible builds and deployments possible

4. **Build Caching**:
   - Each build step can be a separate layer
   - Rebuild only from the changed layer

## Limitations and Considerations

1. **Performance**:
   - Additional overhead for file lookups across multiple layers
   - Copy-up operations can be expensive for large files

2. **File inode changes**:
   - When a file is copied up, its inode changes
   - Can affect applications that rely on inode numbers

3. **Hard links**:
   - Hard links in the lower layer that point to files not visible in the merged view can cause issues

4. **Lower layer limits**:
   - Older kernel versions had limits on the number of lower layers (e.g., 128 layers)

## Comparison to Other Storage Drivers

| Driver     | Advantages                          | Disadvantages                      |
|------------|-------------------------------------|-----------------------------------|
| OverlayFS  | Fast, native in kernel, efficient   | Requires kernel 3.18+             |
| AUFS       | Mature, used by early Docker        | Not in mainline kernel            |
| DeviceMapper | Block-level CoW, quotas           | Complex setup, slower performance |
| Btrfs      | Native CoW, snapshots               | Requires specific filesystem      |
| ZFS        | Advanced features, data integrity   | Higher memory usage               |

## Integration with Container Runtimes

### Docker Implementation

Docker uses OverlayFS as follows:
1. Images are stored as layers in `/var/lib/docker/overlay2/`
2. Each layer gets its own directory containing files and directories
3. Container layers are created as upper directories with their own ID
4. The "diff" directory in each layer holds the actual contents

### containerd/OCI Implementation

Lower-level runtimes follow a similar pattern but with different paths and structures based on the OCI specification.

## Advanced Topics

### Layer Optimization

1. **Layer Squashing**:
   - Combining multiple layers into one to reduce depth
   - Trades off build cache efficiency for runtime performance

2. **Multi-stage Builds**:
   - Keep build tools in one stage
   - Copy only necessary artifacts to the final stage

### Security Implications

1. **Layer Inspection**:
   - Each layer can be examined separately
   - Secrets in earlier layers can persist even if "deleted" in later layers

2. **Vulnerabilities**:
   - Kernel vulnerabilities in the OverlayFS implementation could potentially affect container isolation 