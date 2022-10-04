package app

import (
	"github.com/dedovvlad/go-ms-template/internal/config"
)

// ConfigureService конфигурирует сервис
func (srv *Service) ConfigureService() error {
	_, err := config.InitConfig(version.SERVICE_NAME)
	if err != nil {
		return err
	}

	return nil
}
