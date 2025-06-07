// Arkham Horror Multiplayer Web Game - JavaScript Client
// Handles WebSocket communication, game state rendering, and user input

class ArkhamHorrorClient {
    constructor() {
        // WebSocket connection with automatic reconnection
        this.ws = null;
        this.playerId = null;
        this.gameState = null;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 10;
        this.reconnectDelay = 5000; // 5 seconds
        
        // Canvas and rendering
        this.canvas = document.getElementById('gameCanvas');
        this.ctx = this.canvas.getContext('2d');
        
        // UI elements
        this.connectionStatus = document.getElementById('connectionStatus');
        this.gamePhase = document.getElementById('gamePhase');
        this.doomCounter = document.getElementById('doomCounter');
        this.doomValue = document.getElementById('doomValue');
        this.playersList = document.getElementById('playersList');
        this.diceResult = document.getElementById('diceResult');
        this.locationSelect = document.getElementById('locationSelect');
        this.confirmMoveBtn = document.getElementById('confirmMoveBtn');
        
        // Action buttons
        this.actionButtons = {
            move: document.getElementById('moveBtn'),
            gather: document.getElementById('gatherBtn'),
            investigate: document.getElementById('investigateBtn'),
            ward: document.getElementById('wardBtn')
        };
        
        // Location definitions with positions for rendering
        this.locations = {
            'Downtown': { x: 200, y: 450, color: '#8B4513' },
            'University': { x: 600, y: 150, color: '#4682B4' },
            'Rivertown': { x: 600, y: 450, color: '#228B22' },
            'Northside': { x: 200, y: 150, color: '#8B0000' }
        };
        
        // Location adjacency for movement validation
        this.locationAdjacency = {
            'Downtown': ['University', 'Rivertown'],
            'University': ['Downtown', 'Northside'],
            'Rivertown': ['Downtown', 'Northside'],
            'Northside': ['University', 'Rivertown']
        };
        
        // Initialize connection
        this.connect();
        
        // Set up canvas rendering loop
        this.render();
        setInterval(() => this.render(), 100); // Render at 10 FPS
    }
    
    // WebSocket connection management with automatic reconnection
    connect() {
        try {
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const wsUrl = `${protocol}//${window.location.host}/ws`;
            
            this.ws = new WebSocket(wsUrl);
            this.updateConnectionStatus('connecting');
            
            this.ws.onopen = () => {
                console.log('Connected to Arkham Horror server');
                this.updateConnectionStatus('connected');
                this.reconnectAttempts = 0;
            };
            
            this.ws.onmessage = (event) => {
                this.handleMessage(JSON.parse(event.data));
            };
            
            this.ws.onclose = () => {
                console.log('Disconnected from server');
                this.updateConnectionStatus('disconnected');
                this.attemptReconnect();
            };
            
            this.ws.onerror = (error) => {
                console.error('WebSocket error:', error);
                this.updateConnectionStatus('error');
            };
            
        } catch (error) {
            console.error('Connection error:', error);
            this.attemptReconnect();
        }
    }
    
    // Automatic reconnection with exponential backoff
    attemptReconnect() {
        if (this.reconnectAttempts < this.maxReconnectAttempts) {
            this.reconnectAttempts++;
            this.updateConnectionStatus('reconnecting');
            
            setTimeout(() => {
                console.log(`Reconnection attempt ${this.reconnectAttempts}`);
                this.connect();
            }, this.reconnectDelay);
        } else {
            this.updateConnectionStatus('failed');
            console.error('Max reconnection attempts reached');
        }
    }
    
    // Update connection status display
    updateConnectionStatus(status) {
        const statusElement = this.connectionStatus;
        statusElement.className = `connection-status connection-${status}`;
        
        switch (status) {
            case 'connecting':
                statusElement.textContent = 'Connecting...';
                break;
            case 'connected':
                statusElement.textContent = 'Connected';
                break;
            case 'disconnected':
                statusElement.textContent = 'Disconnected';
                break;
            case 'reconnecting':
                statusElement.textContent = `Reconnecting... (${this.reconnectAttempts}/${this.maxReconnectAttempts})`;
                break;
            case 'error':
                statusElement.textContent = 'Connection Error';
                break;
            case 'failed':
                statusElement.textContent = 'Connection Failed';
                break;
        }
    }
    
    // Handle incoming WebSocket messages
    handleMessage(message) {
        switch (message.type) {
            case 'connectionStatus':
                this.playerId = message.playerId;
                console.log('Player ID assigned:', this.playerId);
                break;
                
            case 'gameState':
                this.gameState = message.data;
                this.updateGameDisplay();
                break;
                
            case 'diceResult':
                this.displayDiceResult(message);
                break;
                
            default:
                console.log('Unknown message type:', message.type);
        }
    }
    
    // Send action to server with player ID validation
    sendAction(action, target = null) {
        if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
            console.error('WebSocket not connected');
            return;
        }
        
        if (!this.playerId) {
            console.error('Player ID not assigned');
            return;
        }
        
        const actionMessage = {
            type: 'playerAction',
            playerId: this.playerId,
            action: action,
            target: target
        };
        
        this.ws.send(JSON.stringify(actionMessage));
        console.log('Action sent:', actionMessage);
    }
    
    // Update all game display elements
    updateGameDisplay() {
        if (!this.gameState) return;
        
        this.updateGamePhase();
        this.updateDoomCounter();
        this.updatePlayersList();
        this.updateActionButtons();
        this.checkGameEndConditions();
    }
    
    // Update game phase display
    updateGamePhase() {
        const phase = this.gameState.gamePhase;
        const currentPlayer = this.gameState.currentPlayer;
        
        let statusText = '';
        let statusClass = '';
        
        switch (phase) {
            case 'waiting':
                statusText = `Waiting for players... (${Object.keys(this.gameState.players).length}/4)`;
                statusClass = 'status-waiting';
                break;
            case 'playing':
                const isMyTurn = currentPlayer === this.playerId;
                statusText = isMyTurn ? 'Your Turn!' : `${currentPlayer}'s Turn`;
                statusClass = 'status-playing';
                break;
            case 'ended':
                statusText = this.gameState.winCondition ? 'Victory!' : 'Defeat!';
                statusClass = 'status-ended';
                break;
        }
        
        this.gamePhase.textContent = statusText;
        this.gamePhase.className = statusClass;
    }
    
    // Update doom counter with visual feedback
    updateDoomCounter() {
        const doom = this.gameState.doom;
        this.doomValue.textContent = doom;
        
        // Add critical warning for high doom levels
        if (doom >= 10) {
            this.doomCounter.classList.add('critical');
        } else {
            this.doomCounter.classList.remove('critical');
        }
    }
    
    // Update players list with resources and status
    updatePlayersList() {
        this.playersList.innerHTML = '';
        
        Object.values(this.gameState.players).forEach(player => {
            const playerDiv = document.createElement('div');
            playerDiv.className = 'player-info';
            
            // Highlight current player
            if (player.id === this.gameState.currentPlayer) {
                playerDiv.classList.add('current-player');
            }
            
            // Connection status indicator
            const connectionIcon = player.connected ? '🟢' : '🔴';
            
            playerDiv.innerHTML = `
                <div style="display: flex; justify-content: space-between; align-items: center;">
                    <strong>${player.id}</strong>
                    <span>${connectionIcon}</span>
                </div>
                <div style="font-size: 0.9em; margin: 5px 0;">📍 ${player.location}</div>
                <div class="resource-bar">
                    <div class="resource-item">
                        <span>❤️</span>
                        <span class="resource-value">${player.resources.health}</span>
                    </div>
                    <div class="resource-item">
                        <span>🧠</span>
                        <span class="resource-value">${player.resources.sanity}</span>
                    </div>
                    <div class="resource-item">
                        <span>🔍</span>
                        <span class="resource-value">${player.resources.clues}</span>
                    </div>
                </div>
                <div style="font-size: 0.8em; color: #DAA520;">
                    Actions: ${player.actionsRemaining}
                </div>
            `;
            
            this.playersList.appendChild(playerDiv);
        });
    }
    
    // Update action button availability
    updateActionButtons() {
        const isMyTurn = this.gameState.currentPlayer === this.playerId;
        const myPlayer = this.gameState.players[this.playerId];
        const hasActions = myPlayer && myPlayer.actionsRemaining > 0;
        const gameActive = this.gameState.gamePhase === 'playing';
        
        // Enable buttons only if it's player's turn, they have actions, and game is active
        const buttonsEnabled = isMyTurn && hasActions && gameActive;
        
        Object.values(this.actionButtons).forEach(button => {
            button.disabled = !buttonsEnabled;
        });
        
        // Special validation for Cast Ward (requires sanity > 1)
        if (myPlayer && myPlayer.resources.sanity <= 1) {
            this.actionButtons.ward.disabled = true;
        }
    }
    
    // Display dice roll results with visual feedback
    displayDiceResult(diceMessage) {
        const resultDiv = this.diceResult;
        
        // Create dice visual representation
        const diceHtml = diceMessage.results.map(result => {
            let className = '';
            let symbol = '';
            
            switch (result) {
                case 'success':
                    className = 'dice-success';
                    symbol = '✓';
                    break;
                case 'blank':
                    className = 'dice-blank';
                    symbol = '○';
                    break;
                case 'tentacle':
                    className = 'dice-tentacle';
                    symbol = '🐙';
                    break;
            }
            
            return `<div class="dice-face ${className}">${symbol}</div>`;
        }).join('');
        
        const successText = diceMessage.success ? 'Success!' : 'Failed';
        const doomText = diceMessage.doomIncrease > 0 ? `Doom +${diceMessage.doomIncrease}` : '';
        
        resultDiv.innerHTML = `
            <div><strong>${diceMessage.playerId}</strong> - ${diceMessage.action}</div>
            <div class="dice-roll">${diceHtml}</div>
            <div>Successes: ${diceMessage.successes} | Tentacles: ${diceMessage.tentacles}</div>
            <div style="font-weight: bold; color: ${diceMessage.success ? '#90EE90' : '#FF6347'}">
                ${successText}
            </div>
            ${doomText ? `<div style="color: #FF0000">${doomText}</div>` : ''}
        `;
    }
    
    // Check and display game end conditions
    checkGameEndConditions() {
        if (this.gameState.winCondition) {
            this.showGameEndMessage('Victory! The investigators have succeeded!', 'win-condition');
        } else if (this.gameState.loseCondition) {
            this.showGameEndMessage('Defeat! The doom has consumed Arkham!', 'lose-condition');
        }
    }
    
    // Display game end message
    showGameEndMessage(message, className) {
        // Remove existing end game messages
        const existingMessages = document.querySelectorAll('.win-condition, .lose-condition');
        existingMessages.forEach(msg => msg.remove());
        
        // Create new message
        const messageDiv = document.createElement('div');
        messageDiv.className = className;
        messageDiv.textContent = message;
        
        // Insert at the top of the sidebar
        const sidebar = document.querySelector('.sidebar');
        sidebar.insertBefore(messageDiv, sidebar.firstChild);
    }
    
    // Canvas rendering for game board visualization
    render() {
        const ctx = this.ctx;
        const canvas = this.canvas;
        
        // Clear canvas with atmospheric background
        const gradient = ctx.createLinearGradient(0, 0, canvas.width, canvas.height);
        gradient.addColorStop(0, '#1a1a2e');
        gradient.addColorStop(1, '#16213e');
        ctx.fillStyle = gradient;
        ctx.fillRect(0, 0, canvas.width, canvas.height);
        
        // Draw location connections
        this.drawLocationConnections();
        
        // Draw locations
        this.drawLocations();
        
        // Draw players if game state is available
        if (this.gameState && this.gameState.players) {
            this.drawPlayers();
        }
        
        // Draw game information overlay
        this.drawGameInfo();
    }
    
    // Draw connections between adjacent locations
    drawLocationConnections() {
        const ctx = this.ctx;
        
        ctx.strokeStyle = '#666';
        ctx.lineWidth = 3;
        ctx.setLineDash([10, 5]);
        
        // Draw connections based on adjacency rules
        Object.entries(this.locationAdjacency).forEach(([from, adjacentLocations]) => {
            const fromLoc = this.locations[from];
            
            adjacentLocations.forEach(to => {
                const toLoc = this.locations[to];
                
                ctx.beginPath();
                ctx.moveTo(fromLoc.x, fromLoc.y);
                ctx.lineTo(toLoc.x, toLoc.y);
                ctx.stroke();
            });
        });
        
        ctx.setLineDash([]); // Reset line dash
    }
    
    // Draw location circles with names
    drawLocations() {
        const ctx = this.ctx;
        
        Object.entries(this.locations).forEach(([name, location]) => {
            // Draw location circle
            ctx.fillStyle = location.color;
            ctx.strokeStyle = '#DAA520';
            ctx.lineWidth = 3;
            
            ctx.beginPath();
            ctx.arc(location.x, location.y, 60, 0, 2 * Math.PI);
            ctx.fill();
            ctx.stroke();
            
            // Draw location name
            ctx.fillStyle = 'white';
            ctx.font = 'bold 14px Georgia';
            ctx.textAlign = 'center';
            ctx.textBaseline = 'middle';
            ctx.fillText(name, location.x, location.y);
        });
    }
    
    // Draw player tokens at their locations
    drawPlayers() {
        const ctx = this.ctx;
        
        // Group players by location for proper positioning
        const playersByLocation = {};
        Object.values(this.gameState.players).forEach(player => {
            if (!playersByLocation[player.location]) {
                playersByLocation[player.location] = [];
            }
            playersByLocation[player.location].push(player);
        });
        
        // Draw players with offset positioning for multiple players at same location
        Object.entries(playersByLocation).forEach(([locationName, players]) => {
            const location = this.locations[locationName];
            
            players.forEach((player, index) => {
                const angle = (index * 2 * Math.PI) / players.length;
                const offsetX = Math.cos(angle) * 35;
                const offsetY = Math.sin(angle) * 35;
                
                const playerX = location.x + offsetX;
                const playerY = location.y + offsetY;
                
                // Player token circle
                ctx.fillStyle = player.id === this.playerId ? '#FFD700' : '#FFF';
                ctx.strokeStyle = player.id === this.gameState.currentPlayer ? '#00FF00' : '#000';
                ctx.lineWidth = player.id === this.gameState.currentPlayer ? 4 : 2;
                
                ctx.beginPath();
                ctx.arc(playerX, playerY, 15, 0, 2 * Math.PI);
                ctx.fill();
                ctx.stroke();
                
                // Player ID text
                ctx.fillStyle = '#000';
                ctx.font = 'bold 10px Arial';
                ctx.textAlign = 'center';
                ctx.textBaseline = 'middle';
                ctx.fillText(player.id.replace('player_', 'P'), playerX, playerY);
            });
        });
    }
    
    // Draw game information overlay
    drawGameInfo() {
        const ctx = this.ctx;
        
        // Draw legend
        ctx.fillStyle = 'rgba(0, 0, 0, 0.8)';
        ctx.fillRect(10, 10, 200, 120);
        
        ctx.fillStyle = '#DAA520';
        ctx.font = 'bold 16px Georgia';
        ctx.textAlign = 'left';
        ctx.fillText('Legend:', 20, 30);
        
        ctx.font = '12px Arial';
        ctx.fillStyle = '#FFF';
        ctx.fillText('• Yellow circle: Your investigator', 20, 50);
        ctx.fillText('• Green border: Current player', 20, 70);
        ctx.fillText('• Dotted lines: Connections', 20, 90);
        ctx.fillText('• Click actions in sidebar →', 20, 110);
    }
}

// Action handlers for UI interactions
function selectMoveAction() {
    const client = window.gameClient;
    const locationSelect = client.locationSelect;
    const confirmBtn = client.confirmMoveBtn;
    
    // Show location selection UI
    locationSelect.style.display = 'block';
    confirmBtn.style.display = 'block';
    
    // Populate valid adjacent locations
    if (client.gameState && client.playerId) {
        const myPlayer = client.gameState.players[client.playerId];
        const currentLocation = myPlayer.location;
        const adjacentLocations = client.locationAdjacency[currentLocation] || [];
        
        // Clear and populate select options
        locationSelect.innerHTML = '<option value="">Select location...</option>';
        adjacentLocations.forEach(location => {
            const option = document.createElement('option');
            option.value = location;
            option.textContent = location;
            locationSelect.appendChild(option);
        });
    }
}

function confirmMove() {
    const client = window.gameClient;
    const locationSelect = client.locationSelect;
    const selectedLocation = locationSelect.value;
    
    if (selectedLocation) {
        client.sendAction('move', selectedLocation);
        
        // Hide move UI
        locationSelect.style.display = 'none';
        client.confirmMoveBtn.style.display = 'none';
        locationSelect.value = '';
    } else {
        alert('Please select a location to move to.');
    }
}

function performAction(actionType) {
    const client = window.gameClient;
    client.sendAction(actionType);
}

// Initialize game client when page loads
window.addEventListener('load', () => {
    window.gameClient = new ArkhamHorrorClient();
    console.log('Arkham Horror client initialized');
});
