package application

import (
	"context"

	"github.com/gin-gonic/gin"
)

type Module interface {
	Name() string
}

type HttpModule interface {
	Module
	RegisterRoutes(rg *gin.RouterGroup) error
}

type StartableModule interface {
	Module
	Start(ctx context.Context) error
}

type StoppableModule interface {
	Module
	Stop(ctx context.Context) error
}
