#!/usr/bin/env bash
set -euo pipefail

frames=${1:?usage: compose.sh FRAMES_DIR [OUTPUT_GIF]}
output=${2:-docs/assets/demo.gif}
font=${FONT:-/System/Library/Fonts/SFNSMono.ttf}

captions=(
  "1  Conduit, a real Vue app, talks to its API through chaosproxy"
  "2  Every request shows up live in the dashboard"
  "3  Turn on one rule: articles[1] becomes null"
  "4  The API still answers 200, but the whole feed spins forever"
  "5  Only the payload changed: one-null-article, mutate"
  "6  Toggle it off: back to normal, no restart"
)
tenths=(22 22 18 32 24 22)

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

index=0
count=0
for frame in "$frames"/*.png; do
  card="$work/card-$index.png"
  magick "$frame" -resize 1280x720 -background '#0d0c0b' -gravity south -splice 0x64 \
    -font "$font" -pointsize 26 -fill '#f1efe9' -annotate +0+18 "${captions[$index]}" "$card"
  for ((repeat = 0; repeat < tenths[index]; repeat++)); do
    cp "$card" "$(printf '%s/f%04d.png' "$work" "$count")"
    count=$((count + 1))
  done
  index=$((index + 1))
done

gifski --fps 10 --width 960 --quality 85 -o "$output" "$work"/f*.png
