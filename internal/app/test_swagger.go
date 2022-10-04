package app

import (
	apiTest "test_tmp/internal/generated/restapi/operations/test"

	"github.com/go-openapi/runtime/middleware"
)

func (srv *Service) TestHandler(params apiTest.TestParams) middleware.Responder {
	return middleware.NotImplemented("operation test Test has not yet been implemented")
}
