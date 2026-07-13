package repository

import (
	"context"
	"strings"

	"github.com/kageos/kageos/core/hr-server/model"
	"gorm.io/gorm"
)

type CompanyRepository struct {
	db *gorm.DB
}

func NewCompanyRepository(db *gorm.DB) *CompanyRepository {
	return &CompanyRepository{db: db}
}

func (r *CompanyRepository) GetCompanyByCode(ctx context.Context, code string) (*model.Company, error) {
	var company model.Company
	if err := r.db.WithContext(ctx).Where("code = ?", code).First(&company).Error; err != nil {
		return nil, err
	}
	return &company, nil
}

func (r *CompanyRepository) GetCompanyByName(ctx context.Context, name string) (*model.Company, error) {
	var company model.Company
	if err := r.db.WithContext(ctx).Where("name = ?", name).First(&company).Error; err != nil {
		return nil, err
	}
	return &company, nil
}

func (r *CompanyRepository) CreateCompany(ctx context.Context, company *model.Company) error {
	return r.db.WithContext(ctx).Create(company).Error
}

func (r *CompanyRepository) GetCompaniesByCodes(ctx context.Context, codes []string) ([]*model.Company, error) {
	if len(codes) == 0 {
		return []*model.Company{}, nil
	}
	var companies []*model.Company
	if err := r.db.WithContext(ctx).Where("code IN ?", codes).Find(&companies).Error; err != nil {
		return nil, err
	}
	return companies, nil
}

func (r *CompanyRepository) SearchCompaniesFuzzy(ctx context.Context, keyword string, limit int) ([]*model.Company, error) {
	var companies []*model.Company
	keyword = strings.TrimSpace(keyword)
	query := r.db.WithContext(ctx).Model(&model.Company{})
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("code LIKE ? OR name LIKE ?", like, like)
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	err := query.Order("updated_at DESC").Limit(limit).Find(&companies).Error
	return companies, err
}
