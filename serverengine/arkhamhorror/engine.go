package arkhamhorror

import "github.com/opd-ai/bostonfear/serverengine"

// Engine is the Arkham Horror module-owned runtime wrapper.
// It keeps the executable ownership boundary in this package while the
// current gameplay implementation still lives in the serverengine facade.
type Engine struct {
	*serverengine.GameServer
}
