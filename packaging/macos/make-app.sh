#!/bin/sh
# Build "Gascoigne Workshop.app" on macOS from a built binary.
#   packaging/macos/make-app.sh VERSION BINARY OUTDIR
# Needs sips and iconutil, which ship with macOS.
set -eu
version=$1 binary=$2 out=$3
here=$(cd "$(dirname "$0")" && pwd)
app="$out/Gascoigne Workshop.app"
rm -rf "$app"
mkdir -p "$app/Contents/MacOS" "$app/Contents/Resources"
cp "$binary" "$app/Contents/MacOS/gascoigne-workshop"
chmod 755 "$app/Contents/MacOS/gascoigne-workshop"
sed "s/@VERSION@/$version/g" "$here/Info.plist" > "$app/Contents/Info.plist"

# The icon is drawn by the program itself: write it large, scale it down.
iconset=$(mktemp -d)/icon.iconset
mkdir -p "$iconset"
"$binary" -icon-png "$iconset/icon_256x256.png"
for size in 16 32 64 128; do
	sips -z $size $size "$iconset/icon_256x256.png" --out "$iconset/icon_${size}x${size}.png" >/dev/null
done
cp "$iconset/icon_32x32.png" "$iconset/icon_16x16@2x.png"
cp "$iconset/icon_64x64.png" "$iconset/icon_32x32@2x.png"
cp "$iconset/icon_256x256.png" "$iconset/icon_128x128@2x.png"
iconutil -c icns "$iconset" -o "$app/Contents/Resources/icon.icns"
echo "$app"
