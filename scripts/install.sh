#!/bin/sh
# Install the chaosproxy binary from a GitHub release.
#
#   curl -sSL https://raw.githubusercontent.com/Raxuis/ChaosProxy/main/scripts/install.sh | sh
#
# Environment:
#   CHAOSPROXY_VERSION            version to install, e.g. v1.2.3 (default: latest)
#   CHAOSPROXY_BIN_DIR            install directory (default: /usr/local/bin, else ~/.local/bin)
#   CHAOSPROXY_DOWNLOAD_BASE_URL  release base URL (default: derived from the version)

set -eu

repository="Raxuis/ChaosProxy"

main() {
  need_cmd uname
  need_cmd tar
  downloader_probe

  os="$(detect_os)"
  arch="$(detect_arch)"
  version="${CHAOSPROXY_VERSION:-$(latest_version)}"
  version="${version#v}"
  base_url="${CHAOSPROXY_DOWNLOAD_BASE_URL:-https://github.com/${repository}/releases/download/v${version}}"

  archive="chaosproxy_${version}_${os}_${arch}.tar.gz"
  bin_dir="$(install_dir)"

  tmp="$(mktemp -d)"
  trap 'rm -rf "$tmp"' EXIT

  info "downloading ${archive}"
  download "${base_url}/${archive}" "${tmp}/${archive}"
  download "${base_url}/checksums.txt" "${tmp}/checksums.txt"

  verify_checksum "$tmp" "$archive"
  tar -xf "${tmp}/${archive}" -C "$tmp" chaosproxy

  mkdir -p "$bin_dir"
  install -m 0755 "${tmp}/chaosproxy" "${bin_dir}/chaosproxy" 2>/dev/null ||
    { cp "${tmp}/chaosproxy" "${bin_dir}/chaosproxy" && chmod 0755 "${bin_dir}/chaosproxy"; }

  info "installed chaosproxy ${version} to ${bin_dir}/chaosproxy"
  case ":${PATH}:" in
    *":${bin_dir}:"*) ;;
    *) info "add ${bin_dir} to your PATH to run chaosproxy" ;;
  esac
}

detect_os() {
  case "$(uname -s)" in
    Linux) echo linux ;;
    Darwin) echo darwin ;;
    *) err "unsupported OS $(uname -s); see https://github.com/${repository}/releases" ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64 | amd64) echo amd64 ;;
    arm64 | aarch64) echo arm64 ;;
    *) err "unsupported architecture $(uname -m); see https://github.com/${repository}/releases" ;;
  esac
}

install_dir() {
  if [ -n "${CHAOSPROXY_BIN_DIR:-}" ]; then
    echo "$CHAOSPROXY_BIN_DIR"
  elif [ -w /usr/local/bin ] 2>/dev/null; then
    echo /usr/local/bin
  else
    echo "${HOME}/.local/bin"
  fi
}

latest_version() {
  api="https://api.github.com/repos/${repository}/releases/latest"
  tag="$(download_stdout "$api" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)"
  [ -n "$tag" ] || err "could not resolve the latest version; set CHAOSPROXY_VERSION"
  echo "$tag"
}

verify_checksum() {
  dir="$1"
  archive="$2"
  expected="$(grep " \*\{0,1\}${archive}\$" "${dir}/checksums.txt" | awk '{print $1}')"
  [ -n "$expected" ] || err "checksums.txt has no entry for ${archive}"
  if command -v sha256sum >/dev/null 2>&1; then
    actual="$(sha256sum "${dir}/${archive}" | awk '{print $1}')"
  elif command -v shasum >/dev/null 2>&1; then
    actual="$(shasum -a 256 "${dir}/${archive}" | awk '{print $1}')"
  else
    err "need sha256sum or shasum to verify the download"
  fi
  [ "$actual" = "$expected" ] ||
    err "checksum mismatch for ${archive}: expected ${expected}, got ${actual}"
}

downloader_probe() {
  if command -v curl >/dev/null 2>&1; then
    downloader=curl
  elif command -v wget >/dev/null 2>&1; then
    downloader=wget
  else
    err "need curl or wget to download releases"
  fi
}

download() {
  if [ "$downloader" = curl ]; then
    curl -fsSL "$1" -o "$2"
  else
    wget -qO "$2" "$1"
  fi
}

download_stdout() {
  if [ "$downloader" = curl ]; then
    curl -fsSL "$1"
  else
    wget -qO - "$1"
  fi
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || err "need $1 but it is not installed"
}

info() { echo "install.sh: $1" >&2; }
err() {
  echo "install.sh: error: $1" >&2
  exit 1
}

main "$@"
