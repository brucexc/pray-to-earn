package hub

import (
	"context"
	"fmt"
	"net"

	"github.com/redis/go-redis/v9"

	"github.com/brucexc/pray-to-earn/internal/config"
	"github.com/brucexc/pray-to-earn/internal/service"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const Name = "hub"

const (
	DefaultHost = "0.0.0.0"
	DefaultPort = "80"
)

type Server struct {
	httpServer *echo.Echo
	hub        *Hub
}

func (s *Server) Name() string {
	return Name
}

func (s *Server) Run(_ context.Context) error {
	address := net.JoinHostPort(DefaultHost, DefaultPort)

	return s.httpServer.Start(address)
}

func NewServer(conf *config.File, ethereumClient *ethclient.Client, redisClient *redis.Client) (service.Server, error) {
	hub, err := NewHub(context.Background(), *conf, ethereumClient, redisClient)
	if err != nil {
		return nil, fmt.Errorf("new hub: %w", err)
	}

	instance := Server{
		httpServer: echo.New(),
		hub:        hub,
	}

	instance.httpServer.HideBanner = true
	instance.httpServer.HidePort = true
	instance.httpServer.Validator = defaultValidator
	instance.httpServer.Use(middleware.CORSWithConfig(middleware.DefaultCORSConfig))

	nodes := instance.httpServer.Group("/pray")
	{
		nodes.POST("/knock", instance.hub.Knock)
		nodes.POST("/reply", instance.hub.Reply)
		nodes.POST("/peekNote", instance.hub.PeekNote)
		nodes.POST("/faucet", instance.hub.Faucet)
	}

	return &instance, nil
}
