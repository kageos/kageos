package model

import "github.com/kageos/kageos/pkg/gormx/models"

type Package struct {
	models.Base
	TreeID int64 `json:"tree_id"`
	AppID  int64 `json:"app_id"`
}
