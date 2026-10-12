#!/bin/sh
# Runs the showcase in a tmux pane, presses keys, and prints the pane.
#
#   scripts/tmux-shot.sh [-s WxH] [-i icon] [-p powerline] [-c] [-- keys...]
#
#   -s 120x40   pane size (default 120x40)
#   -i 2        cells tmux gives a Nerd Font icon (default 1)
#   -p 1        cells tmux gives a powerline symbol (default 1)
#   -c          keep the colours (capture with escape sequences)
#
# Keys are tmux key names (j, Space, Escape, Tab, C-c, Enter) and are sent one by
# one, 0.4s apart, after a second for startup. Every line is printed as
# "<cells>|<line>|", each icon replaced by one "@" per cell tmux gave it. Run on
# tmux 3.7c (codepoint-widths).
set -e
SIZE=120x40 ICON=1 PL=1 COLOR=""
while [ $# -gt 0 ]; do
	case "$1" in
	-s) SIZE=$2; shift 2 ;;
	-i) ICON=$2; shift 2 ;;
	-p) PL=$2; shift 2 ;;
	-c) COLOR=-e; shift ;;
	--) shift; break ;;
	*) break ;;
	esac
done
W=${SIZE%x*} H=${SIZE#*x}
DIR=$(mktemp -d)
trap 'rm -rf "$DIR"' EXIT
(cd "$(dirname "$0")/.." && go build -o "$DIR/showcase" ./examples/showcase)
T="tmux -L tshot -f /dev/null"
$T kill-server 2>/dev/null || true
$T new-session -d -s s -x "$W" -y "$H" "sleep 600"
$T set -g default-terminal "tmux-256color"
WIDTHS=""
[ "$ICON" = 2 ] && WIDTHS="U+E000-U+E09F=2,U+E0D8-U+F8FF=2,U+F0000-U+FFFFD=2"
[ "$PL" = 2 ] && WIDTHS="${WIDTHS:+$WIDTHS,}U+E0A0-U+E0D7=2"
[ -n "$WIDTHS" ] && $T set -s codepoint-widths "$WIDTHS"
$T respawn-pane -k -t s "COLORTERM=truecolor $DIR/showcase; sleep 30"
sleep 1.2
for k in "$@"; do
	$T send-keys -t s "$k"
	sleep 0.4
done
sleep 0.6
$T capture-pane -p $COLOR -N -t s >"$DIR/cap.txt"
$T kill-server
if [ -n "$COLOR" ]; then
	cat "$DIR/cap.txt"
	exit 0
fi
LC_ALL=en_US.UTF-8 perl -CSD -ne '
	chomp;
	s/([\x{E0A0}-\x{E0D7}])/"@" x '"$PL"'/ge;
	s/([\x{E000}-\x{F8FF}\x{F0000}-\x{FFFFD}])/"@" x '"$ICON"'/ge;
	printf "%3d|%s|\n", length, $_' "$DIR/cap.txt"
