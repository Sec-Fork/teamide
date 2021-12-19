package powerService

import (
	"server/base"
)

func (this_ *PowerService) BindApi(appendApi func(apis ...*base.ApiWorker)) {
	bindManagePowerRoleApi(appendApi)
	bindManagePowerUserApi(appendApi)
	bindManagePowerActionApi(appendApi)
	bindManagePowerDataApi(appendApi)
}
