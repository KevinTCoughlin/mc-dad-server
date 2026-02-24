package management_test

import (
	"github.com/KevinTCoughlin/mc-dad-server/internal/management"
	"github.com/KevinTCoughlin/mc-dad-server/internal/platform"
)

// Compile-time interface compliance checks.
var _ management.ServerManager = (*management.ScreenManager)(nil)

// Verify NewScreenManager returns a type satisfying the interface.
var _ management.ServerManager = management.NewScreenManager(platform.NewMockRunner(), "test", "start.sh")
