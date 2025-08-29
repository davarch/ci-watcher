package logging

import "go.uber.org/zap"

func New() *zap.Logger {
	l, _ := zap.NewProduction()

	return l
}
