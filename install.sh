#!/bin/sh
# This script is for installing gherkio from a release archive.
#
# It is intended to be run like this:
#
#   curl -sSL https://raw.githubusercontent.com/muhfaris/gherkio/main/install.sh | sh
#
# This script will:
#
#   1. Detect the user's OS and architecture.
#   2. Download the latest release of gherkio for that platform.
#   3. Unpack the binary into /usr/local/bin.
#
# This script is inspired by the install script for fly.io's flyctl.

set -e

main() {
  if [ -z "$OS" ]; then
    OS=$(uname -s)
  fi

  if [ -z "$ARCH" ]; then
    ARCH=$(uname -m)
  fi

  case $OS in
    Linux)
      OS=linux
      ;;
    Darwin)
      OS=darwin
      ;;
    *)
      echo "OS $OS is not supported."
      exit 1
      ;;
  esac

  case $ARCH in
    x86_64)
      ARCH=x86_64
      ;;
    amd64)
      ARCH=x86_64
      ;;
    arm64)
      ARCH=arm64
      ;;
    aarch64)
      ARCH=arm64
      ;;
    *)
      echo "Architecture $ARCH is not supported."
      exit 1
      ;;
  esac

  # The `tr` command is used to convert the output of `uname` to lowercase.
  os_arch_suffix=$(echo "${OS}_${ARCH}" | tr '[:upper:]' '[:lower:]')

  # The github release URL.
  github_url="https://github.com/muhfaris/gherkio"

  # The URL to download the latest release from.
  download_url=$(curl -sL "${github_url}/releases" | \
                  grep -o "/muhfaris/gherkio/releases/download/.*/gherkio_.*_${os_arch_suffix}.tar.gz" | \
                  head -n 1)

  if [ -z "$download_url" ]; then
    echo "Could not find a release for your OS/Arch"
    exit 1
  fi

  # The full URL to download the latest release from.
  download_url="${github_url}${download_url}"

  # The destination to download the release to.
  download_dest=$(mktemp -t gherkio.XXXXXXXXXX)

  echo "Downloading gherkio from $download_url"

  # Download the latest release.
  curl -fL "$download_url" -o "$download_dest"

  # The directory to unpack the release to.
  unpack_dir=$(mktemp -d -t gherkio.XXXXXXXXXX)

  # Unpack the release.
  tar -C "$unpack_dir" -xzf "$download_dest"

  # The path to the binary.
  binary_path="$unpack_dir/gherkio"

  # The destination to install the binary to.
  install_path="/usr/local/bin/gherkio"

  echo "Installing gherkio to $install_path"

  # Install the binary.
  install -d "$(dirname "$install_path")"
  install "$binary_path" "$install_path"

  echo "gherkio installed successfully."
}

main "$@"