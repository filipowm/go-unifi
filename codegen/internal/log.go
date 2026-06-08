package internal

import (
	"github.com/filipowm/go-unifi/v2/codegen/shared"
	"github.com/sirupsen/logrus"
)

// log is the package-global logger used as the default sink when individual
// pipeline components are constructed without an explicit logger. Functions that
// accept a shared.Logger use orDefaultLogger so tests can inject an isolated
// instance and assert output without touching this global.
var log shared.Logger = logrus.New()

// orDefaultLogger returns logger, or the package-global fallback when it is nil.
func orDefaultLogger(logger shared.Logger) shared.Logger {
	return shared.OrDefaultLogger(logger, func() shared.Logger { return log })
}
