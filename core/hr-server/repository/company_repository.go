package repository

import (
	"strings"

	"github.com/ai-agent-os/ai-agent-os/core/hr-server/model"
	"gorm.io/gorm"
)

type CompanyRepository struct {
	db *gorm.DB
}

func NewCompanyRepository(db *gorm.DB) *CompanyRepository {
	return &CompanyRepository{db: db}
}

func (r *CompanyRepository) GetCompanyByCode(code string) (*model.Company, error) {
	var company model.Company
	if err := r.db.Where("code = ?", code).First(&company).Error; err != nil {
		return nil, err
	}
	return &company, nil
}

func (r *CompanyRepository) GetCompanyByName(name string) (*model.Company, error) {
	var company model.Company
	if err := r.db.Where("name = ?", name).First(&company).Error; err != nil {
		return nil, err
	}
	return &company, nil
}

func (r *CompanyRepository) CreateCompany(company *model.Company) error {
	return r.db.Create(company).Error
}

func (r *CompanyRepository) GetCompaniesByCodes(codes []string) ([]*model.Company, error) {
	if len(codes) == 0 {
		return []*model.Company{}, nil
	}
	var companies []*model.Company
	if err := r.db.Where("code IN ?", codes).Find(&companies).Error; err != nil {
		return nil, err
	}
	return companies, nil
}

func (r *CompanyRepository) SearchCompaniesFuzzy(keyword string, limit int) ([]*model.Company, error) {
	var companies []*model.Company
	keyword = strings.TrimSpace(keyword)
	query := r.db.Model(&model.Company{})
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
