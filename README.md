# Arkham Horror - Multiplayer Web Game

A functional multiplayer web implementation of Arkham Horror featuring investigators managing resources while exploring locations and facing supernatural threats. Built with Go WebSocket server and JavaScript client supporting 2-4 concurrent players.

## Features

### Core Game Mechanics
1. **Location System**: 4 interconnected neighborhoods (Downtown, University, Rivertown, Northside) with movement restrictions
2. **Resource Tracking**: Health (1-10), Sanity (1-10), and Clues (0-5) with gain/loss mechanics
3. **Action System**: 2 actions per turn from Move, Gather Resources, Investigate, Cast Ward
4. **Doom Counter**: Global doom tracker (0-12) that increments on failed dice rolls
5. **Dice Resolution**: 3-sided dice (Success/Blank/Tentacle) with configurable difficulty thresholds

### Multiplayer Features
- Support for 2-4 concurrent players
- Real-time game state synchronization
- Turn-based gameplay with action validation
- Automatic reconnection handling
- WebSocket-based communication

## Quick Setup (3 Steps)

### Step 1: Install Dependencies
```bash
cd /workspaces/bostonfear
go mod tidy
```

### Step 2: Start Server
```bash
cd server
go run main.go
```

### Step 3: Access Client
Open your browser and navigate to:
```
http://localhost:8080
```

## Game Rules

### Objective
Investigators must work cooperatively to gather clues and cast protective wards before the doom counter reaches 12.

### Turn Structure
Each player gets 2 actions per turn:
- **Move**: Travel between adjacent locations only
- **Gather**: Roll 2 dice to potentially gain Health and Sanity
- **Investigate**: Roll 3 dice, need 2+ successes to gain a Clue
- **Cast Ward**: Costs 1 Sanity, roll 3 dice, need 3 successes to reduce Doom by 2

### Dice Mechanics
- **Success** (✓): Counts toward action success
- **Blank** (○): No effect
- **Tentacle** (🐙): Increases Doom counter by 1

### Win/Lose Conditions
- **Win**: Achieve sufficient collective clues (cooperative victory)
- **Lose**: Doom counter reaches 12

## Technical Implementation

### Go Server Architecture
- **Interface-based Design**: Uses `net.Conn`, `net.Listener`, and `net.Addr` interfaces
- **Concurrent Connection Handling**: Goroutines with channel-based communication
- **State Management**: Centralized game state with mutex protection
- **Error Handling**: Explicit Go-style error checking and propagation

### JavaScript Client Features
- **WebSocket Connection**: Automatic reconnection with exponential backoff
- **Canvas Rendering**: 800x600px game board with location visualization
- **Real-time Updates**: Live game state synchronization
- **Responsive UI**: Modern web interface with visual feedback

### JSON Message Protocol
```json
// Player Action
{"type": "playerAction", "playerId": "player1", "action": "investigate", "target": "University"}

// Game State Update
{"type": "gameState", "data": {"currentPlayer": "player2", "doom": 5, "players": {...}}}

// Dice Result
{"type": "diceResult", "playerId": "player1", "results": ["success", "blank", "tentacle"]}
```

## Development

### Project Structure
```
/workspaces/bostonfear/
├── server/main.go          # Go WebSocket server
├── client/index.html       # HTML game interface
├── client/game.js          # JavaScript game client
├── go.mod                  # Go module dependencies
└── README.md               # This file
```

### Dependencies
- **Server**: Go 1.24+ with gorilla/websocket
- **Client**: Modern web browser with HTML5 Canvas and WebSocket support

### Testing Multi-player
1. Start the server
2. Open multiple browser tabs/windows to `http://localhost:8080`
3. Each tab represents a different player
4. Game starts automatically with 2+ players

## Game Flow Example

1. **Player 1** moves from Downtown to University (Location System validates adjacency)
2. **Player 1** investigates (Action System calls Dice Resolution)
3. **Dice Results**: Success, Blank, Tentacle (need 2 successes)
4. **Investigation fails** (Resource Tracking - no clue gained)
5. **Tentacle result** increments global doom counter (Doom Counter system)
6. **Turn advances** to Player 2
7. **All clients** receive updated game state within 500ms

## Performance Standards
- Maintains stable operation with 4 concurrent players
- Supports continuous gameplay for 15+ minutes
- Sub-500ms state synchronization across all clients
- Automatic handling of connection drops with 30-second timeout

## Troubleshooting

### Connection Issues
- Ensure server is running on port 8080
- Check firewall settings
- Verify WebSocket support in browser

### Game State Sync Issues
- Refresh browser to re-establish connection
- Check browser console for WebSocket errors
- Verify all players are using same server instance

### Performance Issues
- Close unnecessary browser tabs
- Ensure stable internet connection
- Check server resources if hosting remotely
