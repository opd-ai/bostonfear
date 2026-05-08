package serverengine

// actionProcessor encapsulates action execution concerns.
type actionProcessor struct {
	gs *GameServer
}

func newActionProcessor(gs *GameServer) *actionProcessor {
	return &actionProcessor{gs: gs}
}

func (p *actionProcessor) Process(action PlayerActionMessage) error {
	return p.gs.processActionCore(action)
}

// sessionManager encapsulates player connection/session lifecycle concerns.
type sessionManager struct {
	gs *GameServer
}

func newSessionManager(gs *GameServer) *sessionManager {
	return &sessionManager{gs: gs}
}

func (s *sessionManager) HandleDisconnect(playerID, addr string) {
	s.gs.handlePlayerDisconnectCore(playerID, addr)
}

// metricsCollector encapsulates derived metrics and game-stat aggregation.
type metricsCollector struct {
	gs *GameServer
}

func newMetricsCollector(gs *GameServer) *metricsCollector {
	return &metricsCollector{gs: gs}
}

func (m *metricsCollector) GameStatistics() map[string]interface{} {
	return m.gs.getGameStatisticsCore()
}
