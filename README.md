# F2FS Raw Image Extractor
### A tool to browse and extract Android F2FS images

A fast, standalone command-line tool to browse and extract files from unsparsed Android F2FS images (`.img`) directly - without needing root privileges, loop devices, or kernel mounting. 

Written in pure Go, `f2fs-extractor` reads raw filesystem blocks in user-space, making it a perfect rootless cross-platform tool for extracting/dumping an F2FS image.

## Features

* **Rootless & No Mounting:** Parses F2FS superblocks, NATs, and inodes directly. You do not need `sudo`, `mount`, or a compatible Linux kernel to extract files.
* **Cross-Platform:** Compiles to a single static binary for Linux, macOS, and Windows.
* **Fast & Standalone:** Zero external C dependencies. Drop the executable anywhere and it just works.
* **Smart Extraction:** Automatically handles recursive directory extraction and gracefully manages F2FS symlink creation across different host operating systems.

---

## Download
Grab the latest executable for your OS from the **[Releases](https://github.com/Sushrut1101/f2fs-extractor/releases)** page.

## Building from Source
### If you want to compile it youself:

You will need [Go](https://go.dev/) installed. The repository includes a `Makefile` that handles cross-compilation and binary stripping automatically.

```bash
# Clone the repository
git clone https://github.com/Sushrut1101/f2fs-extractor.git
cd f2fs-extractor

# Build for your current OS
go build -o f2fs-extractor .

# OR: Build for all platforms (Linux, macOS, Windows) simultaneously
make all
```
*Compiled binaries will be neatly organized in the `out/` directory.*

---

## 🛠️ Usage

**Syntax:** `f2fs-extractor <command> <image.img> [options] [args...]`

### Quick Examples

**1. View Filesystem Info**
```bash
f2fs-extractor info system.img
```

**2. List Directory Contents**
```bash
# List root directory
f2fs-extractor list system.img 

# List specific path
f2fs-extractor list system.img system/bin
```

**3. Extract a Directory (or File)**
```bash
# Extract the system image to the system directory
f2fs-extractor extract system.img system

# Extract a specific file or directory to a target destination
f2fs-extractor extract system.img --file /system/etc system/
```

### Available Commands
| Command | Description |
| :--- | :--- |
| `info` | Show raw filesystem information (Superblock details) |
| `list` | List directory contents |
| `extract` | Extract a file, symlink, or directory recursively |

Run `f2fs-extractor <command> -h` for specific options available for each command.

---

## 📝 License

This project is licensed under the **GNU General Public License v3.0 (GPLv3)**. See the `LICENSE` file for details.
