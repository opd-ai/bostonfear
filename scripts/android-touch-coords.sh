#!/bin/bash
# Calculate Android emulator touch coordinates for BostonFear actions
# Based on 800x600 logical resolution scaled to actual device screen

set -euo pipefail

# Get actual device screen size
SCREEN_SIZE=$(adb shell wm size | grep -oE '[0-9]+x[0-9]+' | head -1)
DEVICE_WIDTH=$(echo "$SCREEN_SIZE" | cut -d'x' -f1)
DEVICE_HEIGHT=$(echo "$SCREEN_SIZE" | cut -d'x' -f2)

# BostonFear logical resolution
LOGICAL_WIDTH=800
LOGICAL_HEIGHT=600

# Calculate scaling factors
SCALE_X=$(echo "scale=6; $DEVICE_WIDTH / $LOGICAL_WIDTH" | bc)
SCALE_Y=$(echo "scale=6; $DEVICE_HEIGHT / $LOGICAL_HEIGHT" | bc)

# Function to convert logical coordinates to device coordinates
logical_to_device() {
    local logical_x=$1
    local logical_y=$2
    local device_x=$(echo "scale=0; $logical_x * $SCALE_X / 1" | bc)
    local device_y=$(echo "scale=0; $logical_y * $SCALE_Y / 1" | bc)
    echo "$device_x $device_y"
}

# Action grid constants (from client/ebiten/app/input.go)
ACTION_GRID_ORIGIN_X=10
ACTION_GRID_GAP=6
ACTION_GRID_CELL_HEIGHT=44
ACTION_GRID_HEADER=28

# Calculate action grid Y position (bottom of screen)
# For 800x600, with 2 rows of actions:
# Total height = 28 (header) + 2*44 (cells) + 1*6 (gap) + 8 (padding) = 130
ACTION_GRID_TOTAL_HEIGHT=130
ACTION_GRID_Y=$((LOGICAL_HEIGHT - ACTION_GRID_TOTAL_HEIGHT))

# Action button positions (row 0: gather, investigate, ward, focus, research, trade)
# For 800px width with 6 buttons: availableWidth = 800-20-(5*6) = 750
# cellWidth = 750/6 = 125
CELL_WIDTH=125

# Calculate center coordinates for each action button in row 0
# Row 0 has 6 buttons, centered
ROW_WIDTH=$((6 * CELL_WIDTH + 5 * ACTION_GRID_GAP))
ROW_START_X=$(( (LOGICAL_WIDTH - 20 - ROW_WIDTH) / 2 + ACTION_GRID_ORIGIN_X ))

# Action positions (center of each button)
GATHER_X=$((ROW_START_X + CELL_WIDTH / 2))
GATHER_Y=$((ACTION_GRID_Y + ACTION_GRID_HEADER + ACTION_GRID_CELL_HEIGHT / 2))

INVESTIGATE_X=$((ROW_START_X + CELL_WIDTH + ACTION_GRID_GAP + CELL_WIDTH / 2))
INVESTIGATE_Y=$GATHER_Y

WARD_X=$((ROW_START_X + 2 * (CELL_WIDTH + ACTION_GRID_GAP) + CELL_WIDTH / 2))
WARD_Y=$GATHER_Y

FOCUS_X=$((ROW_START_X + 3 * (CELL_WIDTH + ACTION_GRID_GAP) + CELL_WIDTH / 2))
FOCUS_Y=$GATHER_Y

# Location rectangles (from client/ebiten/app/game.go)
# Downtown: {40, 60, 160, 100} -> center at (120, 110)
DOWNTOWN_X=120
DOWNTOWN_Y=110

# University: {220, 60, 160, 100} -> center at (300, 110)
UNIVERSITY_X=300
UNIVERSITY_Y=110

# Rivertown: {40, 220, 160, 100} -> center at (120, 270)
RIVERTOWN_X=120
RIVERTOWN_Y=270

# Northside: {220, 220, 160, 100} -> center at (300, 270)
NORTHSIDE_X=300
NORTHSIDE_Y=270

# Export coordinates for use in tests
case "${1:-}" in
    gather)
        logical_to_device $GATHER_X $GATHER_Y
        ;;
    investigate)
        logical_to_device $INVESTIGATE_X $INVESTIGATE_Y
        ;;
    ward)
        logical_to_device $WARD_X $WARD_Y
        ;;
    focus)
        logical_to_device $FOCUS_X $FOCUS_Y
        ;;
    downtown)
        logical_to_device $DOWNTOWN_X $DOWNTOWN_Y
        ;;
    university)
        logical_to_device $UNIVERSITY_X $UNIVERSITY_Y
        ;;
    rivertown)
        logical_to_device $RIVERTOWN_X $RIVERTOWN_Y
        ;;
    northside)
        logical_to_device $NORTHSIDE_X $NORTHSIDE_Y
        ;;
    *)
        echo "Usage: $0 {gather|investigate|ward|focus|downtown|university|rivertown|northside}"
        echo "Device: ${DEVICE_WIDTH}x${DEVICE_HEIGHT}"
        echo "Scaling: X=${SCALE_X}, Y=${SCALE_Y}"
        exit 1
        ;;
esac
