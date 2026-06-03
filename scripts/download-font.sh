#!/usr/bin/env bash
# Generate font.scss from DINPro woff2 font file
# Usage: ./scripts/download-font.sh [path-to-dinpro.woff2]
#
# If no path provided, looks for DINPro.woff2 in current directory

set -euo pipefail

FONT_FILE="${1:-DINPro.woff2}"
OUTPUT_FILE="frontend/src/assets/styles/font.scss"

if [[ ! -f "$FONT_FILE" ]]; then
    echo "Error: Font file not found: $FONT_FILE"
    echo ""
    echo "Usage: $0 [path-to-dinpro.woff2]"
    echo ""
    echo "DINPro is a commercial font. Please provide a valid .woff2 file."
    exit 1
fi

echo "Generating $OUTPUT_FILE from $FONT_FILE..."

# Convert to base64 (macOS and Linux compatible)
if command -v base64 &> /dev/null; then
    # macOS base64 (no line wrapping by default)
    BASE64_DATA=$(base64 -i "$FONT_FILE")
elif command -v openssl &> /dev/null; then
    # Fallback to openssl
    BASE64_DATA=$(openssl base64 -in "$FONT_FILE" | tr -d '\n')
else
    echo "Error: Neither base64 nor openssl found"
    exit 1
fi

# Generate SCSS file
cat > "$OUTPUT_FILE" << EOF
@font-face {
  font-family: 'DINPro';
  src: url('data:font/woff2;charset=utf-8;base64,${BASE64_DATA}') format('woff2');
  font-weight: normal;
  font-style: normal;
  font-display: swap;
}
EOF

echo "✓ Generated $OUTPUT_FILE ($(wc -c < "$OUTPUT_FILE" | tr -d ' ') bytes)"
echo "✓ Font file is excluded from git (.gitignore)"
echo ""
echo "Next steps:"
echo "  1. Verify frontend builds: cd frontend && npm run build"
echo "  2. Commit the generated file if needed (or keep it local)"
