package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	hrmodel "github.com/kageos/kageos/core/hr-server/model"
	"github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/logger"
	"gorm.io/gorm"
)

var defaultCompanyCodePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func InitDefaultCompany(ctx context.Context, db *gorm.DB) error {
	cfg := config.GetHRServerConfig()
	code := strings.ToLower(strings.TrimSpace(cfg.GetCompanyCode()))
	name := strings.TrimSpace(cfg.GetCompanyName())
	logoURL := strings.TrimSpace(cfg.GetCompanyLogoURL())
	if code == "" {
		code = hrmodel.DefaultCompanyCode
	}
	if name == "" {
		name = "Default"
	}
	if !defaultCompanyCodePattern.MatchString(code) {
		return fmt.Errorf("default company code %q is invalid", code)
	}

	var company hrmodel.Company
	err := db.Where("code = ?", code).First(&company).Error
	switch {
	case err == nil:
		company.Name = name
		company.LogoURL = logoURL
		if err := db.Save(&company).Error; err != nil {
			return fmt.Errorf("update default company failed: %w", err)
		}
	case errors.Is(err, gorm.ErrRecordNotFound):
		var legacy hrmodel.Company
		if code != hrmodel.DefaultCompanyCode && db.Where("code = ?", hrmodel.DefaultCompanyCode).First(&legacy).Error == nil {
			legacy.Code = code
			legacy.Name = name
			legacy.LogoURL = logoURL
			if err := db.Save(&legacy).Error; err != nil {
				return fmt.Errorf("rename legacy default company failed: %w", err)
			}
		} else {
			company = hrmodel.Company{
				Code:      code,
				Name:      name,
				CreatedBy: SystemUsername,
				LogoURL:   logoURL,
			}
			if err := db.Create(&company).Error; err != nil {
				return fmt.Errorf("create default company failed: %w", err)
			}
		}
	default:
		return err
	}

	if err := db.Model(&hrmodel.User{}).
		Where("company_code = '' OR company_code IS NULL OR company_code = ?", hrmodel.DefaultCompanyCode).
		Update("company_code", code).Error; err != nil {
		return fmt.Errorf("sync default user company failed: %w", err)
	}
	logger.Infof(ctx, "[Company] default company ensured: code=%s name=%s", code, name)
	return nil
}

func defaultCompanyCode() string {
	code := strings.ToLower(strings.TrimSpace(config.GetHRServerConfig().GetCompanyCode()))
	if code == "" {
		return hrmodel.DefaultCompanyCode
	}
	return code
}
