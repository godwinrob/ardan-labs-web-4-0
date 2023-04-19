package handlers

import (
	"go.uber.org/zap"
	"net/http"
	"os"
)

type APIMuxConfig struct {
	Shutdown chan os.Signal
	Log      *zap.SugaredLogger
}

func APIMux(cfg APIMuxConfig) http.Handler {

	return nil
}
