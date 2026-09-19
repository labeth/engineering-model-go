#!/usr/bin/env bash
# Install the pinned SysML validator and Sysand into the ignored .engmod tree.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"
# shellcheck source=../tools/sysml-toolchain.env
source tools/sysml-toolchain.env

tool_root="$repo_root/.engmod/tooling"
download_dir="$tool_root/downloads"
source_dir="$tool_root/src"
bin_dir="$tool_root/bin"
maven_cache="$repo_root/.engmod/cache/m2"
mkdir -p "$download_dir" "$source_dir" "$bin_dir" "$maven_cache"

verify_sha256() {
  local checksum="$1" path="$2"
  if command -v sha256sum >/dev/null 2>&1; then
    printf '%s  %s\n' "$checksum" "$path" | sha256sum -c -
  elif command -v shasum >/dev/null 2>&1; then
    test "$(shasum -a 256 "$path" | awk '{print $1}')" = "$checksum"
  else
    echo "sha256sum or shasum is required" >&2
    return 1
  fi
}

download_verified() {
  local url="$1" destination="$2" checksum="$3"
  if [ -f "$destination" ] && verify_sha256 "$checksum" "$destination" >/dev/null 2>&1; then
    return
  fi
  rm -f "$destination"
  curl --fail --location --retry 3 --output "$destination" "$url"
  verify_sha256 "$checksum" "$destination"
}

install_sysand() {
  local os arch asset checksum archive
  os="$(uname -s)"
  arch="$(uname -m)"
  case "$os/$arch" in
    Linux/x86_64)
      asset="sysand-linux-x86_64-gnu.tar.gz"
      checksum="$SYSAND_LINUX_X86_64_GNU_SHA256"
      ;;
    Linux/aarch64|Linux/arm64)
      asset="sysand-linux-arm64-gnu.tar.gz"
      checksum="$SYSAND_LINUX_ARM64_GNU_SHA256"
      ;;
    Darwin/x86_64)
      asset="sysand-macos-x86_64.tar.gz"
      checksum="$SYSAND_MACOS_X86_64_SHA256"
      ;;
    Darwin/arm64)
      asset="sysand-macos-arm64.tar.gz"
      checksum="$SYSAND_MACOS_ARM64_SHA256"
      ;;
    *)
      echo "unsupported Sysand platform: $os/$arch" >&2
      return 1
      ;;
  esac
  archive="$download_dir/$asset"
  download_verified \
    "https://github.com/sensmetry/sysand/releases/download/v${SYSAND_VERSION}/${asset}" \
    "$archive" "$checksum"
  tar -xzf "$archive" -C "$bin_dir" sysand
  chmod +x "$bin_dir/sysand"
  "$bin_dir/sysand" --version | grep -F "sysand ${SYSAND_VERSION}" >/dev/null
}

install_validator() {
  command -v git >/dev/null || { echo "git is required" >&2; return 1; }
  command -v mvn >/dev/null || { echo "Maven is required" >&2; return 1; }
  command -v java >/dev/null || { echo "Java 21+ is required" >&2; return 1; }
  command -v unzip >/dev/null || { echo "unzip is required" >&2; return 1; }

  local validator_dir kernel_zip kernel_url
  validator_dir="$source_dir/sysmlv2-validator"
  kernel_zip="$download_dir/jupyter-sysml-kernel-${SYSML_ARTIFACT_VERSION}.zip"
  kernel_url="https://github.com/Systems-Modeling/SysML-v2-Pilot-Implementation/releases/download/${SYSML_RELEASE_TAG}/jupyter-sysml-kernel-${SYSML_ARTIFACT_VERSION}.zip"
  download_verified "$kernel_url" "$kernel_zip" "$SYSML_KERNEL_SHA256"

  if [ ! -d "$validator_dir/.git" ]; then
    rm -rf "$validator_dir"
    git clone --filter=blob:none --no-checkout https://github.com/DeciSym/sysmlv2-validator.git "$validator_dir"
  fi
  git -C "$validator_dir" fetch --quiet origin "$SYSML_VALIDATOR_COMMIT"
  git -C "$validator_dir" checkout --quiet --detach "$SYSML_VALIDATOR_COMMIT"
  test "$(git -C "$validator_dir" rev-parse HEAD)" = "$SYSML_VALIDATOR_COMMIT"

  rm -rf "$validator_dir/target/sysml-download"
  mkdir -p "$validator_dir/target/sysml-download"
  unzip -q "$kernel_zip" -d "$validator_dir/target/sysml-download"
  local kernel_jar="$validator_dir/target/sysml-download/sysml/jupyter-sysml-kernel-${SYSML_ARTIFACT_VERSION}-all.jar"
  test -f "$kernel_jar"

  mvn --quiet -f "$validator_dir/pom.xml" -Dmaven.repo.local="$maven_cache" \
    install:install-file -Dfile="$kernel_jar" -DgroupId=org.omg.sysml \
    -DartifactId=jupyter-sysml-kernel -Dversion="$SYSML_ARTIFACT_VERSION" \
    -Dpackaging=jar -DgeneratePom=true
  mvn --quiet -f "$validator_dir/pom.xml" -Dmaven.repo.local="$maven_cache" \
    -Dsysml.release.tag="$SYSML_RELEASE_TAG" \
    -Dsysml.artifact.version="$SYSML_ARTIFACT_VERSION" package

  cat >"$bin_dir/validate-sysml" <<EOF
#!/usr/bin/env sh
exec "$validator_dir/validate-sysml" "\$@"
EOF
  chmod +x "$bin_dir/validate-sysml" "$validator_dir/validate-sysml"
}

case "${1:-all}" in
  all)
    install_sysand
    install_validator
    ;;
  sysand)
    install_sysand
    ;;
  validator)
    install_validator
    ;;
  *)
    echo "usage: $0 [all|sysand|validator]" >&2
    exit 2
    ;;
esac

printf 'installed pinned SysML toolchain in %s\n' "$tool_root"
