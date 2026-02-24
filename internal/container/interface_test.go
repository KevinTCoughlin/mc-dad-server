package container_test

import (
	"github.com/KevinTCoughlin/mc-dad-server/internal/container"
	"github.com/KevinTCoughlin/mc-dad-server/internal/management"
	"github.com/KevinTCoughlin/mc-dad-server/internal/platform"
)

// Compile-time interface compliance check.
var _ management.ServerManager = (*container.Manager)(nil)
var _ management.ServerManager = container.NewManager(platform.NewMockRunner(), "test", "", "")
