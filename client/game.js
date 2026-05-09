// Arkham Horror Multiplayer Web Game - JavaScript Client
// Handles WebSocket communication, game state rendering, and user input

class ArkhamHorrorClient {
    constructor() {
        // WebSocket connection with automatic reconnection
        this.ws = null;
        this.playerId = null;
        this.reconnectToken = localStorage.getItem('arkham_reconnect_token') || null; // restored from storage on page load
        this.gameState = null;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = Infinity;
        this.baseReconnectDelay = 5000;
        this.reconnectDelay = this.baseReconnectDelay; // 5 seconds
        this.pendingAction = false;
        this.lastKnownDoom = 0;
        
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
        this.uiMessage = document.getElementById('uiMessage');
        this.turnOrder = document.getElementById('turnOrder');
        this.doomExplanation = document.getElementById('doomExplanation');
        this.clueProgressValue = document.getElementById('clueProgressValue');
        this.clueProgressHint = document.getElementById('clueProgressHint');

        this.logicalCanvasWidth = 800;
        this.logicalCanvasHeight = 600;
        
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
        
        // Connection quality monitoring
        this.connectionQualities = {};
        this.latencyHistory = [];
        this.maxLatencyHistory = 20; // Keep last 20 latency measurements
        
        // Initialize connection
        this.connect();

        this.setupResponsiveCanvas();
        this.resizeCanvas();
        
        // Initial paint; subsequent draws occur on state/resize updates.
        this.render();
    }
    
    // WebSocket connection management with automatic reconnection
    connect() {
        try {
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            // Include reconnection token in URL if we have one from a prior session.
            const tokenParam = this.reconnectToken ? `?token=${encodeURIComponent(this.reconnectToken)}` : '';
            const wsUrl = `${protocol}//${window.location.host}/ws${tokenParam}`;
            
            this.ws = new WebSocket(wsUrl);
            this.updateConnectionStatus('connecting');
            
            this.ws.onopen = () => {
                console.log('Connected to Arkham Horror server');
                this.updateConnectionStatus('connected');
                this.reconnectAttempts = 0;
                this.reconnectDelay = this.baseReconnectDelay;
                this.showMessage('success', 'Connected. Waiting for game state sync...');
            };
            
            this.ws.onmessage = (event) => {
                try {
                    this.handleMessage(JSON.parse(event.data));
                } catch (error) {
                    console.error('Invalid message payload:', error);
                    this.showMessage('error', 'Received an invalid server message.');
                }
            };
            
            this.ws.onclose = () => {
                console.log('Disconnected from server');
                this.updateConnectionStatus('disconnected');
                this.showMessage('warning', 'Disconnected from server. Reconnecting...');
                this.attemptReconnect();
            };
            
            this.ws.onerror = (error) => {
                console.error('WebSocket error:', error);
                this.updateConnectionStatus('error');
                this.showMessage('error', 'Connection error. Please wait while reconnecting.');
            };
            
        } catch (error) {
            console.error('Connection error:', error);
            this.attemptReconnect();
        }
    }
    
    // Automatic reconnection with exponential backoff (doubles each attempt, capped at 30s).
    // Retries indefinitely — no hard attempt limit.
    attemptReconnect() {
        this.reconnectAttempts++;
        this.updateConnectionStatus('reconnecting');

        setTimeout(() => {
            console.log(`Reconnection attempt ${this.reconnectAttempts} (delay ${this.reconnectDelay}ms)`);
            this.connect();
        }, this.reconnectDelay);
        // Double the delay for the next attempt, capped at 30 seconds
        this.reconnectDelay = Math.min(this.reconnectDelay * 2, 30000);
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
                statusElement.textContent = `Reconnecting... (attempt ${this.reconnectAttempts})`;
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
                if (message.token) {
                    this.reconnectToken = message.token;
                    localStorage.setItem('arkham_reconnect_token', message.token); // persist across page refreshes
                }
                console.log('Player ID assigned:', this.playerId);
                this.showMessage('info', `Connected as ${this.playerId}.`);
                break;
                
            case 'gameState':
                this.gameState = message.data;
                this.pendingAction = false;
                this.updateGameDisplay();
                break;
                
            case 'gameUpdate':
                // Lightweight action-event notification preceding the full gameState broadcast.
                this.pendingAction = false;
                this.displayGameUpdate(message);
                break;
                
            case 'diceResult':
                this.displayDiceResult(message);
                break;
                
            case 'ping':
                this.handlePingMessage(message);
                break;
                
            case 'connectionQuality':
                this.handleConnectionQuality(message);
                break;

            case 'error':
            case 'invalidAction':
            case 'actionError':
                this.pendingAction = false;
                this.handleActionErrorMessage(message);
                break;
                
            default:
                console.log('Unknown message type:', message.type);
        }
    }
    
    // Send action to server with player ID validation
    sendAction(action, target = null) {
        if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
            console.error('WebSocket not connected');
            this.showMessage('error', 'Cannot act while disconnected.');
            return;
        }
        
        if (!this.playerId) {
            console.error('Player ID not assigned');
            this.showMessage('error', 'Player identity not assigned yet.');
            return;
        }

        if (!this.gameState) {
            this.showMessage('warning', 'Game state not synced yet.');
            return;
        }

        const myPlayer = this.gameState.players[this.playerId];
        if (!myPlayer) {
            this.showMessage('error', 'You are not in the current game state.');
            return;
        }

        if (this.gameState.currentPlayer !== this.playerId) {
            this.showMessage('warning', 'It is not your turn.');
            return;
        }

        if (myPlayer.actionsRemaining <= 0) {
            this.showMessage('warning', 'No actions remaining this turn.');
            return;
        }

        if (this.pendingAction) {
            this.showMessage('warning', 'Action already submitted. Waiting for server...');
            return;
        }

        if (action === 'ward' && myPlayer.resources.sanity <= 0) {
            this.showMessage('warning', 'Cast Ward requires at least 1 Sanity.');
            return;
        }
        
        const actionMessage = {
            type: 'playerAction',
            playerId: this.playerId,
            action: action,
            target: target
        };
        
        this.ws.send(JSON.stringify(actionMessage));
        this.pendingAction = true;
        this.updateActionButtons();
        this.showMessage('info', `Submitted action: ${action}${target ? ` -> ${target}` : ''}.`);
        console.log('Action sent:', actionMessage);
    }
    
    // Update all game display elements
    updateGameDisplay() {
        if (!this.gameState) return;
        
        this.updateGamePhase();
        this.updateDoomCounter();
        this.updatePlayersList();
        this.updateTurnOrder();
        this.updateActionButtons();
        this.updateObjectiveProgress();
        this.checkGameEndConditions();
        this.render();

        if (this.gameState.gamePhase === 'playing') {
            const isMyTurn = this.gameState.currentPlayer === this.playerId;
            this.showMessage('info', isMyTurn
                ? 'Your turn. Choose 1 of the available actions.'
                : `Waiting for ${this.gameState.currentPlayer} to act.`);
        }
    }
    
    // Update game phase display
    updateGamePhase() {
        const phase = this.gameState.gamePhase;
        const currentPlayer = this.gameState.currentPlayer;
        
        let statusText = '';
        let statusClass = '';
        
        switch (phase) {
            case 'waiting':
                statusText = `Waiting for players... (${Object.keys(this.gameState.players).length}/6)`;
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

        if (this.doomExplanation) {
            if (doom >= 12) {
                this.doomExplanation.textContent = 'Doom reached 12. Investigators lose.';
            } else if (doom >= 10) {
                this.doomExplanation.textContent = 'Critical Doom level. Avoid tentacles and timeouts.';
            } else {
                this.doomExplanation.textContent = 'Tentacles and timeouts raise Doom.';
            }
        }

        this.lastKnownDoom = doom;
    }
    
    // Update players list with resources and status
    updatePlayersList() {
        if (!this.playersList || !this.gameState || !this.gameState.players) {
            return;
        }

        this.playersList.innerHTML = '';

        this.getOrderedPlayerIDs().forEach(playerID => {
            const player = this.gameState.players[playerID];
            if (!player) {
                return;
            }

            const playerDiv = document.createElement('div');
            playerDiv.className = 'player-info';
            
            // Highlight current player
            if (player.id === this.gameState.currentPlayer) {
                playerDiv.classList.add('current-player');
            }
            
            // Connection status indicator with quality
            const connectionIcon = player.connected ? this.getConnectionQualityIndicator(player.id) : '🔴';
            const quality = this.connectionQualities[player.id];
            const latencyText = quality && quality.latencyMs > 0 ? ` (${Math.round(quality.latencyMs)}ms)` : '';
            
            playerDiv.innerHTML = `
                <div style="display: flex; justify-content: space-between; align-items: center;">
                    <strong>${player.id}</strong>
                    <span title="Connection Quality${latencyText}">${connectionIcon}</span>
                </div>
                <div style="font-size: 0.9em; margin: 5px 0;">📍 ${player.location}</div>
                <div class="resource-bar">
                    <div class="resource-item">
                        <span>❤️ Health</span>
                        <span class="resource-value">${player.resources.health}</span>
                    </div>
                    <div class="resource-item">
                        <span>🧠 Sanity</span>
                        <span class="resource-value">${player.resources.sanity}</span>
                    </div>
                    <div class="resource-item">
                        <span>🔍 Clues</span>
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
        if (!this.gameState || !this.gameState.players) {
            return;
        }

        const isMyTurn = this.gameState.currentPlayer === this.playerId;
        const myPlayer = this.gameState.players[this.playerId];
        const hasActions = myPlayer && myPlayer.actionsRemaining > 0;
        const gameActive = this.gameState.gamePhase === 'playing';
        
        // Enable buttons only if it's player's turn, they have actions, and game is active
        const buttonsEnabled = isMyTurn && hasActions && gameActive && !this.pendingAction;
        
        Object.values(this.actionButtons).forEach(button => {
            button.disabled = !buttonsEnabled;
        });
        
        // Special validation for Cast Ward (requires sanity > 1)
        if (myPlayer && myPlayer.resources.sanity <= 0) {
            this.actionButtons.ward.disabled = true;
        }

        const moveUiAllowed = buttonsEnabled && !this.pendingAction;
        this.confirmMoveBtn.disabled = !moveUiAllowed;
        if (!moveUiAllowed) {
            this.locationSelect.style.display = 'none';
            this.confirmMoveBtn.style.display = 'none';
            this.locationSelect.value = '';
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
        const requiredSuccesses = this.getRequiredSuccessesForAction(diceMessage);
        const requiredText = requiredSuccesses > 0 ? ` | Needed: ${requiredSuccesses}` : '';
        
        resultDiv.innerHTML = `
            <div><strong>${diceMessage.playerId}</strong> - ${diceMessage.action}</div>
            <div class="dice-roll">${diceHtml}</div>
            <div>Successes: ${diceMessage.successes}${requiredText} | Tentacles: ${diceMessage.tentacles}</div>
            <div style="font-weight: bold; color: ${diceMessage.success ? '#90EE90' : '#FF6347'}">
                ${successText}
            </div>
            ${doomText ? `<div style="color: #FF0000">${doomText}</div>` : ''}
        `;

        const thresholdSummary = requiredSuccesses > 0
            ? ` Needed ${requiredSuccesses}.`
            : '';
        const summary = `${diceMessage.action}: ${successText}. ${diceMessage.successes} successes, ${diceMessage.tentacles} tentacles.${thresholdSummary}`;
        this.showMessage(diceMessage.success ? 'success' : 'warning', summary);
    }
    
    // Display a transient gameUpdate notification showing what changed in the last action.
    // The gameUpdate message arrives before the full gameState snapshot, allowing the UI
    // to show an action summary immediately.
    displayGameUpdate(update) {
        const resultColor = update.result === 'success' ? '#90EE90' : '#FF6347';
        const doomLine = update.doomDelta > 0
            ? `<div style="color:#FF0000">Doom +${update.doomDelta}</div>` : '';
        const deltaLines = [];
        if (update.resourceDelta) {
            if (update.resourceDelta.health !== 0)
                deltaLines.push(`Health ${update.resourceDelta.health > 0 ? '+' : ''}${update.resourceDelta.health}`);
            if (update.resourceDelta.sanity !== 0)
                deltaLines.push(`Sanity ${update.resourceDelta.sanity > 0 ? '+' : ''}${update.resourceDelta.sanity}`);
            if (update.resourceDelta.clues !== 0)
                deltaLines.push(`Clues ${update.resourceDelta.clues > 0 ? '+' : ''}${update.resourceDelta.clues}`);
        }
        const deltaHtml = deltaLines.length > 0
            ? `<div>${deltaLines.join(' | ')}</div>` : '';
        const notification = document.createElement('div');
        notification.style.cssText = 'position:fixed;top:10px;right:10px;background:#1a1a2e;' +
            'border:1px solid #4a4a8a;padding:10px;border-radius:4px;z-index:1000;max-width:250px;';
        notification.innerHTML = `
            <div><strong>${update.playerId}</strong>: ${update.event}</div>
            <div style="color:${resultColor};font-weight:bold">${update.result.toUpperCase()}</div>
            ${deltaHtml}${doomLine}
        `;
        document.body.appendChild(notification);
        // Auto-dismiss after 3 seconds
        setTimeout(() => { if (notification.parentNode) notification.parentNode.removeChild(notification); }, 3000);

        const messageBits = [`${update.playerId} used ${update.event}: ${update.result}.`];
        if (deltaLines.length > 0) {
            messageBits.push(deltaLines.join(', '));
        }
        if (update.doomDelta > 0) {
            messageBits.push(`Doom +${update.doomDelta}`);
            if (this.doomExplanation) {
                this.doomExplanation.textContent = `Doom increased by ${update.doomDelta} from ${update.playerId}'s action.`;
            }
        }
        this.showMessage(update.result === 'success' ? 'success' : 'warning', messageBits.join(' '));
    }

    handleActionErrorMessage(message) {
        const reason = message.reason || message.message || 'Action rejected by server.';
        const action = message.action ? ` (${message.action})` : '';
        this.showMessage('error', `Action rejected${action}: ${reason}`);
    }

    updateTurnOrder() {
        if (!this.turnOrder || !this.gameState || !this.gameState.players) {
            return;
        }

        this.turnOrder.innerHTML = '';

        this.getOrderedPlayerIDs().forEach(playerID => {
            const player = this.gameState.players[playerID];
            if (!player) {
                return;
            }
            const chip = document.createElement('span');
            chip.className = 'turn-chip';
            if (player.id === this.gameState.currentPlayer) {
                chip.classList.add('current');
            }
            chip.textContent = `${player.id}${player.id === this.gameState.currentPlayer ? ' (Now)' : ''}`;
            this.turnOrder.appendChild(chip);
        });
    }

    setupResponsiveCanvas() {
        window.addEventListener('resize', () => this.resizeCanvas());
    }

    resizeCanvas() {
        const dpr = window.devicePixelRatio || 1;
        this.canvas.width = Math.floor(this.logicalCanvasWidth * dpr);
        this.canvas.height = Math.floor(this.logicalCanvasHeight * dpr);
        this.ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
        this.ctx.imageSmoothingEnabled = true;
        this.render();
    }

    getOrderedPlayerIDs() {
        if (!this.gameState || !this.gameState.players) {
            return [];
        }

        const players = this.gameState.players;
        const turnOrder = Array.isArray(this.gameState.turnOrder)
            ? this.gameState.turnOrder.filter(playerID => players[playerID])
            : [];

        if (turnOrder.length > 0) {
            return turnOrder;
        }

        return Object.keys(players).sort((a, b) => a.localeCompare(b, undefined, {
            numeric: true,
            sensitivity: 'base'
        }));
    }

    getRequiredClues() {
        if (!this.gameState || !this.gameState.players) {
            return 0;
        }

        const explicit = Number(this.gameState.requiredClues);
        if (Number.isFinite(explicit) && explicit > 0) {
            return explicit;
        }

        return Object.keys(this.gameState.players).length * 4;
    }

    getTeamClues() {
        if (!this.gameState || !this.gameState.players) {
            return 0;
        }

        return Object.values(this.gameState.players).reduce((total, player) => {
            if (!player || !player.resources) {
                return total;
            }
            return total + (Number(player.resources.clues) || 0);
        }, 0);
    }

    updateObjectiveProgress() {
        if (!this.clueProgressValue || !this.clueProgressHint || !this.gameState) {
            return;
        }

        const currentClues = this.getTeamClues();
        const requiredClues = this.getRequiredClues();
        const playerCount = this.getOrderedPlayerIDs().length;
        this.clueProgressValue.textContent = `Team Clues: ${currentClues} / ${requiredClues}`;
        this.clueProgressHint.textContent = `Goal: 4 clues each (${playerCount} investigators).`;
    }

    getRequiredSuccessesForAction(diceMessage) {
        const explicit = Number(diceMessage.requiredSuccesses);
        if (Number.isFinite(explicit) && explicit > 0) {
            return explicit;
        }

        switch (diceMessage.action) {
            case 'investigate':
                return 2;
            case 'ward':
                return 3;
            case 'gather':
                return 1;
            default:
                return 0;
        }
    }

    render() {
        this.drawBoard();
        this.drawPlayersOnBoard();
    }

    drawBoard() {
        this.ctx.clearRect(0, 0, this.logicalCanvasWidth, this.logicalCanvasHeight);

        this.ctx.fillStyle = '#1f2d2d';
        this.ctx.fillRect(0, 0, this.logicalCanvasWidth, this.logicalCanvasHeight);

        this.ctx.strokeStyle = '#8B4513';
        this.ctx.lineWidth = 3;

        const links = [
            ['Downtown', 'University'],
            ['Downtown', 'Rivertown'],
            ['University', 'Northside'],
            ['Rivertown', 'Northside']
        ];

        links.forEach(([from, to]) => {
            const a = this.locations[from];
            const b = this.locations[to];
            this.ctx.beginPath();
            this.ctx.moveTo(a.x, a.y);
            this.ctx.lineTo(b.x, b.y);
            this.ctx.stroke();
        });

        Object.entries(this.locations).forEach(([name, location]) => {
            this.ctx.beginPath();
            this.ctx.fillStyle = location.color;
            this.ctx.arc(location.x, location.y, 42, 0, Math.PI * 2);
            this.ctx.fill();

            this.ctx.strokeStyle = '#f0e6d2';
            this.ctx.lineWidth = 2;
            this.ctx.stroke();

            this.ctx.fillStyle = '#ffffff';
            this.ctx.font = 'bold 14px Georgia';
            this.ctx.textAlign = 'center';
            this.ctx.fillText(name, location.x, location.y + 5);
        });
    }

    drawPlayersOnBoard() {
        if (!this.gameState || !this.gameState.players) {
            return;
        }

        const playersByLocation = {};
        Object.values(this.gameState.players).forEach(player => {
            if (!playersByLocation[player.location]) {
                playersByLocation[player.location] = [];
            }
            playersByLocation[player.location].push(player);
        });

        Object.entries(playersByLocation).forEach(([locationName, players]) => {
            const location = this.locations[locationName];
            if (!location) {
                return;
            }

            players.forEach((player, index) => {
                const offsetX = (index % 3) * 16 - 16;
                const offsetY = Math.floor(index / 3) * 16 - 8;
                const x = location.x + offsetX;
                const y = location.y - 58 + offsetY;

                this.ctx.beginPath();
                this.ctx.arc(x, y, 7, 0, Math.PI * 2);
                this.ctx.fillStyle = player.id === this.gameState.currentPlayer ? '#ffd700' : '#90ee90';
                this.ctx.fill();

                this.ctx.strokeStyle = '#111';
                this.ctx.lineWidth = 1;
                this.ctx.stroke();

                this.ctx.fillStyle = '#ffffff';
                this.ctx.font = '11px Georgia';
                this.ctx.textAlign = 'left';
                this.ctx.fillText(player.id, x + 10, y + 3);
            });
        });
    }

    checkGameEndConditions() {
        if (!this.gameState || this.gameState.gamePhase !== 'ended') {
            return;
        }

        if (this.gameState.winCondition) {
            this.showMessage('success', 'Victory! Your team met the clue goal before doom consumed Arkham.');
        } else {
            this.showMessage('error', 'Defeat. Doom reached its limit before the team could contain the threat.');
        }
    }

    showMessage(type, text) {
        if (!this.uiMessage) {
            return;
        }

        this.uiMessage.className = `ui-message ${type}`;
        this.uiMessage.textContent = text;
    }
    
    // Handle ping messages and respond with pong
    handlePingMessage(pingMessage) {
        if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
            return;
        }
        
        // Respond with pong message
        const pongMessage = {
            type: 'pong',
            playerId: pingMessage.playerId,
            timestamp: pingMessage.timestamp,
            pingId: pingMessage.pingId
        };
        
        this.ws.send(JSON.stringify(pongMessage));
    }
    
    // Handle connection quality updates
    handleConnectionQuality(qualityMessage) {
        // Store connection qualities for all players
        this.connectionQualities = qualityMessage.allPlayerQualities;
        
        // Update player list to show connection quality indicators
        this.updatePlayersList();
        
        // Update own connection status if applicable
        if (qualityMessage.playerId === this.playerId) {
            this.updateOwnConnectionStatus(qualityMessage.quality);
        }
    }
    
    // Update connection status based on quality
    updateOwnConnectionStatus(quality) {
        const statusElement = this.connectionStatus;
        
        // Update text and styling based on connection quality
        switch (quality.quality) {
            case 'excellent':
                statusElement.textContent = `Connected (${Math.round(quality.latencyMs)}ms)`;
                statusElement.className = 'connection-status connection-excellent';
                break;
            case 'good':
                statusElement.textContent = `Connected (${Math.round(quality.latencyMs)}ms)`;
                statusElement.className = 'connection-status connection-good';
                break;
            case 'fair':
                statusElement.textContent = `Slow Connection (${Math.round(quality.latencyMs)}ms)`;
                statusElement.className = 'connection-status connection-fair';
                break;
            case 'poor':
                statusElement.textContent = `Poor Connection (${Math.round(quality.latencyMs)}ms)`;
                statusElement.className = 'connection-status connection-poor';
                break;
            default:
                statusElement.textContent = 'Connected';
                statusElement.className = 'connection-status connection-unknown';
        }
    }
    
    // Get connection quality indicator for a player
    getConnectionQualityIndicator(playerId) {
        const quality = this.connectionQualities[playerId];
        if (!quality) {
            return '⚪'; // Unknown
        }
        
        switch (quality.quality) {
            case 'excellent':
                return '🟢'; // Green
            case 'good':
                return '🟡'; // Yellow
            case 'fair':
                return '🟠'; // Orange
            case 'poor':
                return '🔴'; // Red
            default:
                return '⚪'; // Unknown
        }
    }
}

// Action handlers for UI interactions
function selectMoveAction() {
    const client = window.gameClient;
    const locationSelect = client.locationSelect;
    const confirmBtn = client.confirmMoveBtn;

    if (!client.gameState || client.gameState.currentPlayer !== client.playerId) {
        client.showMessage('warning', 'You can move only during your own turn.');
        return;
    }
    
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

    if (!client.gameState || client.gameState.currentPlayer !== client.playerId) {
        client.showMessage('warning', 'Move cancelled: it is no longer your turn.');
        locationSelect.style.display = 'none';
        client.confirmMoveBtn.style.display = 'none';
        return;
    }
    
    if (selectedLocation) {
        client.sendAction('move', selectedLocation);
        
        // Hide move UI
        locationSelect.style.display = 'none';
        client.confirmMoveBtn.style.display = 'none';
        locationSelect.value = '';
    } else {
        client.showMessage('warning', 'Please select an adjacent location first.');
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
