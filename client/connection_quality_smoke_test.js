#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

const gameFile = path.join(__dirname, 'game.js');
const source = fs.readFileSync(gameFile, 'utf8');

const methodMatch = source.match(/handleConnectionQuality\(qualityMessage\)\s*\{([\s\S]*?)\n\s*\}\n\s*\/\/ Update connection status based on quality/);
if (!methodMatch) {
  console.error('Could not locate handleConnectionQuality method in client/game.js');
  process.exit(1);
}

const methodBody = methodMatch[1];
const handleConnectionQuality = new Function('qualityMessage', methodBody);

const ownUpdates = [];
const mockClient = {
  playerId: 'player1',
  connectionQualities: {},
  updatePlayersListCalls: 0,
  updateOwnConnectionStatus(quality) {
    ownUpdates.push(quality);
  },
  updatePlayersList() {
    this.updatePlayersListCalls += 1;
  }
};

handleConnectionQuality.call(mockClient, {
  playerId: 'player1',
  allPlayerQualities: {
    player1: {
      latencyMs: 42,
      quality: 'good'
    }
  },
  quality: {
    latencyMs: 42,
    quality: 'good'
  }
});

if (!mockClient.connectionQualities.player1) {
  console.error('Expected connection quality map to include player1 fixture');
  process.exit(1);
}

if (ownUpdates.length !== 1) {
  console.error('Expected exactly one own-status update for matching playerId fixture');
  process.exit(1);
}

handleConnectionQuality.call(mockClient, {
  playerId: 'player2',
  allPlayerQualities: {
    player1: {
      latencyMs: 42,
      quality: 'good'
    },
    player2: {
      latencyMs: 80,
      quality: 'fair'
    }
  },
  quality: {
    latencyMs: 80,
    quality: 'fair'
  }
});

if (ownUpdates.length !== 1) {
  console.error('Expected no additional own-status update for non-matching playerId fixture');
  process.exit(1);
}

if (mockClient.updatePlayersListCalls !== 2) {
  console.error('Expected updatePlayersList to run for each fixture message');
  process.exit(1);
}

console.log('connection quality smoke test passed');
