#!/usr/bin/env bash

set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: SMALL_BIN=/path/to/small $0 /path/to/small-enabled-project" >&2
  exit 2
fi

for tool in git jq vhs ffmpeg ffprobe; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    echo "missing required tool: $tool" >&2
    exit 1
  fi
done

source_project=$(cd "$1" && pwd)
script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
repo_root=$(cd "$script_dir/../.." && pwd)
tape="$script_dir/small-v1-1-0-session-demo.tape"
output_dir="$repo_root/docs/media"
small_bin=${SMALL_BIN:-$(command -v small)}

if [[ ! -d "$source_project/.small" ]]; then
  echo "source project is not SMALL-enabled: $source_project" >&2
  exit 1
fi

if [[ -n $(git -C "$source_project" status --porcelain) ]]; then
  echo "source project must be clean so the recording is reproducible" >&2
  exit 1
fi

version=$($small_bin version | sed -n '1p')
if [[ "$version" != "small v1.1.0" ]]; then
  echo "expected small v1.1.0, got: $version" >&2
  exit 1
fi

$small_bin check --strict --dir "$source_project"
vhs validate "$tape"

render_dir=$(mktemp -d "${TMPDIR:-/tmp}/small-vhs-render.XXXXXX")
cleanup() {
  rm -rf "$render_dir"
}
trap cleanup EXIT

git clone --quiet --no-hardlinks "$source_project" "$render_dir/source"

(
  cd "$render_dir"
  SMALL_DEMO_BIN_DIR=$(dirname "$small_bin") \
  SMALL_DEMO_SOURCE="$render_dir/source" \
  SMALL_DEMO_WORK="$render_dir/project" \
    vhs "$tape"
)

frame_dir="$render_dir/small-session-frames"
if [[ ! -f "$frame_dir/frame-text-00001.png" || ! -f "$frame_dir/frame-cursor-00001.png" ]]; then
  echo "VHS did not produce the expected frame sequence" >&2
  exit 1
fi

mkdir -p "$output_dir"

ffmpeg -hide_banner -loglevel error -y \
  -framerate 30 -start_number 1 -i "$frame_dir/frame-text-%05d.png" \
  -framerate 30 -start_number 1 -i "$frame_dir/frame-cursor-%05d.png" \
  -filter_complex '[0:v][1:v]overlay=shortest=1,format=yuv420p' \
  -c:v libx264 -preset medium -crf 20 -movflags +faststart \
  "$output_dir/small-v1-1-0-session-demo.mp4"

ffmpeg -hide_banner -loglevel error -y \
  -framerate 30 -start_number 1 -i "$frame_dir/frame-text-%05d.png" \
  -framerate 30 -start_number 1 -i "$frame_dir/frame-cursor-%05d.png" \
  -filter_complex '[0:v][1:v]overlay=shortest=1,format=yuv420p' \
  -c:v libvpx-vp9 -crf 30 -b:v 0 -row-mt 1 \
  "$output_dir/small-v1-1-0-session-demo.webm"

ffmpeg -hide_banner -loglevel error -y \
  -i "$output_dir/small-v1-1-0-session-demo.mp4" -ss 0.8 -frames:v 1 \
  "$output_dir/small-v1-1-0-session-demo-poster.png"

ffmpeg -hide_banner -loglevel error -y \
  -i "$output_dir/small-v1-1-0-session-demo.mp4" \
  -filter_complex "fps=12,scale=800:-1:flags=lanczos,split[gif][palette];[palette]palettegen=max_colors=128:stats_mode=diff[p];[gif][p]paletteuse=dither=bayer:bayer_scale=5:diff_mode=rectangle" \
  -loop 0 "$output_dir/small-v1-1-0-session-demo.gif"

mp4_codec=$(ffprobe -v error -select_streams v:0 -show_entries stream=codec_name -of default=noprint_wrappers=1:nokey=1 "$output_dir/small-v1-1-0-session-demo.mp4")
webm_codec=$(ffprobe -v error -select_streams v:0 -show_entries stream=codec_name -of default=noprint_wrappers=1:nokey=1 "$output_dir/small-v1-1-0-session-demo.webm")
gif_codec=$(ffprobe -v error -select_streams v:0 -show_entries stream=codec_name -of default=noprint_wrappers=1:nokey=1 "$output_dir/small-v1-1-0-session-demo.gif")

if [[ "$mp4_codec" != "h264" || "$webm_codec" != "vp9" || "$gif_codec" != "gif" ]]; then
  echo "unexpected media codecs: mp4=$mp4_codec webm=$webm_codec gif=$gif_codec" >&2
  exit 1
fi

gif_size=$(wc -c < "$output_dir/small-v1-1-0-session-demo.gif")
if (( gif_size > 10000000 )); then
  echo "animated README preview exceeds GitHub's 10 MB image limit: $gif_size bytes" >&2
  exit 1
fi

printf 'created %s\n' \
  "$output_dir/small-v1-1-0-session-demo.mp4" \
  "$output_dir/small-v1-1-0-session-demo.webm" \
  "$output_dir/small-v1-1-0-session-demo-poster.png" \
  "$output_dir/small-v1-1-0-session-demo.gif"
