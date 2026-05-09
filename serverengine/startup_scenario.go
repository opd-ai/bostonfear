package serverengine

import "sync"

var startupScenarioMu sync.Mutex
var startupScenarioDefaultID string

// SetStartupScenarioDefaultID configures the next GameServer constructed in this
// process to use the selected Nightglass scenario ID as its startup default.
// The value is consumed on the next NewGameServer call.
func SetStartupScenarioDefaultID(id string) {
	startupScenarioMu.Lock()
	defer startupScenarioMu.Unlock()
	startupScenarioDefaultID = id
}

func consumeStartupScenarioDefaultID() string {
	startupScenarioMu.Lock()
	defer startupScenarioMu.Unlock()
	id := startupScenarioDefaultID
	startupScenarioDefaultID = ""
	return id
}
