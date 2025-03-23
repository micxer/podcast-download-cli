# Justfile for cross-compiling for Raspberry Pi Zero

# Set the target architecture and OS for Raspberry Pi Zero
set shell := ["bash"]

# Command to cross-compile for Raspberry Pi Zero
cross-compile:
    @echo "Cross-compiling for Raspberry Pi 3 Model A Plus..."
    GOOS=linux GOARCH=arm64 go build -o download_rss_episodes main.go
    @echo "Build complete: download_rss_episodes"