#!/bin/sh
set -eu

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
project_dir="$(CDPATH= cd -- "$script_dir/../.." && pwd)"
package_dir="$script_dir/package"
output_dir="$project_dir/dist/fpk"
version="${VERSION:-1.1.7}"

case "$version" in
  ''|*[!0-9A-Za-z._-]*) echo "版本号格式无效: $version" >&2; exit 1 ;;
esac

temporary_dir="$(mktemp -d /tmp/cmsingbox-fpk.XXXXXX)"
cleanup() { rm -rf -- "$temporary_dir"; }
trap cleanup EXIT INT TERM

find_fnpack() {
  if command -v fnpack >/dev/null 2>&1; then
    command -v fnpack
    return
  fi
  case "$(uname -m)" in
    x86_64|amd64) fnpack_arch="amd64" ;;
    aarch64|arm64) fnpack_arch="arm64" ;;
    *) echo "当前构建机架构不受 fnpack 支持: $(uname -m)" >&2; exit 1 ;;
  esac
  fnpack_path="$temporary_dir/fnpack"
  echo "正在下载飞牛官方 fnpack 1.2.3..." >&2
  curl -fL --retry 4 -o "$fnpack_path" \
    "https://static2.fnnas.com/fnpack/fnpack-1.2.3-linux-${fnpack_arch}" >&2
  chmod 0755 "$fnpack_path"
  printf '%s\n' "$fnpack_path"
}

fnpack_bin="$(find_fnpack)"
stage="$temporary_dir/cmsingbox"
cp -a "$package_dir/." "$stage/"
sed "s/@VERSION@/$version/g" "$package_dir/manifest.in" > "$stage/manifest"
rm -f "$stage/manifest.in" "$stage/app/ui/images/.gitkeep"
chmod 0755 "$stage/cmd/"* "$stage/app/ui/index.cgi"

mkdir -p "$output_dir" "$temporary_dir/output"
(cd "$temporary_dir/output" && "$fnpack_bin" build --directory "$stage")
generated="$temporary_dir/output/cmsingbox.fpk"
if [ ! -f "$generated" ]; then
  generated="$(find "$temporary_dir/output" -maxdepth 1 -type f -name '*.fpk' | head -n 1)"
fi
if [ -z "$generated" ] || [ ! -f "$generated" ]; then
  echo "fnpack 未生成 FPK 文件" >&2
  exit 1
fi
final="$output_dir/CMSingBox-fnOS-${version}-all.fpk"
cp "$generated" "$final"
(cd "$output_dir" && sha256sum "$(basename "$final")" > "$(basename "$final").sha256")
echo "已生成: $final"
