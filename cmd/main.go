package main

import (
	"fmt"

	"github.com/dedovvlad/go-ms-template/internal/config"
	"github.com/dedovvlad/go-ms-template/internal/generated/restapi"
	"github.com/dedovvlad/go-ms-template/internal/generated/restapi/operations"
	"github.com/dedovvlad/go-ms-template/internal/version"

	"github.com/go-openapi/loads"
)

func main() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("catch err: %s", err)
		}
	}()

	cfg, err := config.InitConfig(version.SERVICE_NAME)
	if err != nil {
		panic(err)
	}

	swaggerSpec, err := loads.Analyzed(restapi.SwaggerJSON, "")
	if err != nil {
		panic(err)
	}

	api := operations.NewGoMsTemplateAPI(swaggerSpec)
	server := restapi.NewServer(api)

	//nolint: errcheck
	defer server.Shutdown()

	server.ConfigureAPI()

	server.Port = cfg.HTTPBindPort
	server.GracefulTimeout = cfg.GracefulTimeout
	if err := server.Serve(); err != nil {
		panic(err)
	}
}
