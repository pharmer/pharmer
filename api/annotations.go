package api

import (
	"errors"
)

const (
	RoleMaster = "master"
	RoleNode   = "node"

	RoleKeyPrefix = "node-role.kubernetes.io/"
	RoleMasterKey = RoleKeyPrefix + RoleMaster
	RoleNodeKey = RoleKeyPrefix + RoleNode
)
