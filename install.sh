#!/bin/sh
# install.sh — Gherkio installer
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/muhfaris/gherkio/main/install.sh | sh
#
# Detects OS and architecture, downloads the latest release from GitHub,
# and installs the binary to /usr/local/bin.

set -e

main() {
  # ── Detect OS ──
  raw_os=$(uname -s)
  case "$raw_os" in
    Linux)  os="Linux" ;;
    Darwin) os="Darwin" ;;
    MINGW*|MSYS*|CYGWIN*) os="Windows" ;;
    *)
      echo "Unsupported OS: $raw_os"
      echo "Supported: Linux, macOS, Windows"
      exit 1
      ;;
  esac

  # ── Detect architecture ──
  raw_arch=$(uname -m)
  case "$raw_arch" in
    x86_64|amd64) arch="x86_64" ;;
    arm64|aarch64) arch="arm64" ;;
    *)
      echo "Unsupported architecture: $raw_arch"
      echo "Supported: x86_64 (amd64), arm64 (aarch64)"
      exit 1
      ;;
  esac

  echo "Detected: ${os} / ${arch}"

  # ── GitHub API — find latest release ──
  github_repo="muhfaris/gherkio"
  api_url="https://api.github.com/repos/${github_repo}/releases"

  # Fetch releases, find the first non-draft, non-prerelease? Allow prerelease (alpha).
  # We use the latest (first) release regardless of draft/prerelease status.
  release_data=$(curl -sfL "${api_url}" 2>/dev/null || echo "")

  if [ -z "$release_data" ] || [ "$release_data" = "[]" ]; then
    echo "No releases found for ${github_repo}."
    echo "The project may not have published a release yet."
    echo "See: https://github.com/${github_repo}/releases"
    exit 1
  fi

  # Pick the first release (latest), get its tag name
  tag_name=$(echo "$release_data" | sed -n 's/.*"tag_name": "\([^"]*\)".*/\1/p' | head -1)
  if [ -z "$tag_name" ]; then
    echo "Could not determine latest release tag."
    exit 1
  fi

  echo "Latest release: $tag_name"
  
  # Strip leading 'v' from tag name for GoReleaser asset matching
  version_number=$(echo "$tag_name" | sed 's/^v//')

  # ── Find the right asset ──
  # GoReleaser naming: gherkio_{version}_{Os}_{Arch}.tar.gz (or .zip for Windows)
  if [ "$os" = "Windows" ]; then
    extension=".zip"
    binary_name="gherkio.exe"
  else
    extension=".tar.gz"
    binary_name="gherkio"
  fi

  # Match the asset name — GoReleaser uses title-case OS (Linux, Darwin, Windows)
  asset_pattern="gherkio_${version_number}_${os}_${arch}${extension}"

  download_url=$(echo "$release_data" | \
    grep -o "https://github.com/${github_repo}/releases/download/${tag_name}/${asset_pattern}" | \
    head -1)

  if [ -z "$download_url" ]; then
    # Fallback: try to find any matching asset by name
    echo "Could not find asset: ${asset_pattern}"
    echo "Available assets for this release:"
    echo "$release_data" | grep -o '"name": "[^"]*' | sed 's/"name": "//' | while read -r name; do
      echo "  - $name"
    done
    exit 1
  fi

  # ── Download ──
  tmp_dir=$(mktemp -d -t gherkio.XXXXXXXXXX)
  archive_file="${tmp_dir}/gherkio${extension}"

  echo "Downloading ${asset_pattern}..."
  curl -fL "$download_url" -o "$archive_file"

  # ── Extract ──
  if [ "$os" = "Windows" ]; then
    unzip -qo "$archive_file" -d "$tmp_dir" 2>/dev/null || {
      echo "Extraction failed. Requires 'unzip'."
      exit 1
    }
  else
    tar -C "$tmp_dir" -xzf "$archive_file" 2>/dev/null || {
      echo "Extraction failed. Requires 'tar'."
      exit 1
    }
  fi

  binary_path="${tmp_dir}/${binary_name}"
  if [ ! -f "$binary_path" ]; then
    echo "Binary not found in archive: ${binary_name}"
    ls -la "$tmp_dir"
    exit 1
  fi

  # ── Install ──
  install_path="/usr/local/bin/${binary_name}"
  echo "Installing to ${install_path}..."

  install -d "$(dirname "$install_path")"
  install "$binary_path" "$install_path"

  # Cleanup
  rm -rf "$tmp_dir"

  echo "Gherkio ${tag_name} installed successfully."
  echo "Run 'gherkio --help' to get started."
}

main "$@"
