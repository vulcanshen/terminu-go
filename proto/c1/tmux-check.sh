#!/bin/sh
# Runs the C1 prototype in an 80x24 tmux pane where Nerd Font icons can be made
# to take two cells (tmux codepoint-widths; run on 3.7c), then prints the pane with
# every icon replaced by one "@" per cell it takes, so the columns read true.
#
#   proto/c1/tmux-check.sh <terminal icon width> <app icon width> <c1 args> [-- keys]
#
#   proto/c1/tmux-check.sh 2 2 dump e5 80 24 menu fits
#   proto/c1/tmux-check.sh 2 2 tui -- j j p
#
# Each line is printed as "<cells>|<line>|". Lines bubbletea drew with hard tabs
# can read 81: expand(1) does not know a tab stops at the last column.
set -e
TW=$1 AW=$2
shift 2
ARGS="" KEYS=""
while [ $# -gt 0 ]; do
	if [ "$1" = "--" ]; then
		shift
		KEYS="$*"
		break
	fi
	ARGS="$ARGS $1"
	shift
done
DIR=$(mktemp -d)
trap 'rm -rf "$DIR"' EXIT
(cd "$(dirname "$0")" && go build -o "$DIR/c1" .)
T="tmux -L c1check -f /dev/null"
$T kill-server 2>/dev/null || true
$T new-session -d -s c1 -x 80 -y 24 "sleep 600"
if [ "$TW" = 2 ]; then
	$T set -s codepoint-widths "U+E000-U+F8FF=2,U+F0000-U+FFFFD=2"
fi
$T respawn-pane -k -t c1 "SHOWCASE__ICON_WIDTH=$AW $DIR/c1 $ARGS; sleep 30"
sleep 1
for k in $KEYS; do
	$T send-keys -t c1 "$k"
	sleep 0.3
done
sleep 0.5
R=$(printf '@%.0s' $(seq 1 "$TW"))
$T capture-pane -p -N -t c1 |
	LC_ALL=en_US.UTF-8 perl -CSD -pe "s/[\x{E000}-\x{F8FF}\x{F0000}-\x{FFFFD}]/$R/g" |
	expand -t 8 >"$DIR/cap.txt"
$T kill-server
LC_ALL=en_US.UTF-8 perl -CSD -ne 'chomp; printf "%3d|%s|\n", length, $_' "$DIR/cap.txt"
