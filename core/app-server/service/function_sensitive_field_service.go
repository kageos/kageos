package service

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/kageos/kageos-sdk/agent-app/widget"
	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/core/app-server/repository"
	"github.com/kageos/kageos/pkg/functionschema"
)

const (
	sensitiveFieldSectionRequest  = "request"
	sensitiveFieldSectionResponse = "response"
	sensitiveFieldSectionFields   = "fields"

	sensitiveFieldSourceSchema = "schema"
)

type FunctionSensitiveFieldService struct {
	repo  *repository.FunctionSensitiveFieldRepository
	mu    sync.RWMutex
	cache sensitiveFieldIndex
}

type sensitiveFieldIndex map[string]sensitiveFieldSections
type sensitiveFieldSections map[string]sensitiveFieldPathSet
type sensitiveFieldPathSet map[string]struct{}

func newSensitiveFieldIndex() sensitiveFieldIndex {
	return make(sensitiveFieldIndex)
}

func (idx sensitiveFieldIndex) add(fullCodePath, section, fieldPath string) {
	fullCodePath = strings.TrimSpace(fullCodePath)
	section = strings.TrimSpace(section)
	fieldPath = strings.TrimSpace(fieldPath)
	if fullCodePath == "" || section == "" || fieldPath == "" {
		return
	}
	if idx[fullCodePath] == nil {
		idx[fullCodePath] = make(sensitiveFieldSections)
	}
	idx[fullCodePath].add(section, fieldPath)
}

func (idx sensitiveFieldIndex) replace(fullCodePath string, sections sensitiveFieldSections) {
	fullCodePath = strings.TrimSpace(fullCodePath)
	if fullCodePath == "" {
		return
	}
	if len(sections) == 0 {
		delete(idx, fullCodePath)
		return
	}
	idx[fullCodePath] = sections
}

func (sections sensitiveFieldSections) add(section, fieldPath string) {
	if sections[section] == nil {
		sections[section] = make(sensitiveFieldPathSet)
	}
	sections[section][fieldPath] = struct{}{}
}

func (sections sensitiveFieldSections) only(names ...string) sensitiveFieldPathSet {
	result := make(sensitiveFieldPathSet)
	for _, name := range names {
		for path := range sections[name] {
			result[path] = struct{}{}
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func NewFunctionSensitiveFieldService(repo *repository.FunctionSensitiveFieldRepository) *FunctionSensitiveFieldService {
	return &FunctionSensitiveFieldService{
		repo:  repo,
		cache: newSensitiveFieldIndex(),
	}
}

func (s *FunctionSensitiveFieldService) LoadAll(ctx context.Context) error {
	fields, err := s.repo.ListAll(ctx)
	if err != nil {
		return err
	}
	cache := newSensitiveFieldIndex()
	for _, field := range fields {
		cache.add(field.FullCodePath, field.Section, field.FieldPath)
	}
	s.mu.Lock()
	s.cache = cache
	s.mu.Unlock()
	return nil
}

func (s *FunctionSensitiveFieldService) SyncFunctionSchema(ctx context.Context, tenantUser, app, fullCodePath string, functionID int64, schema *functionschema.FunctionSchema) error {
	fields := collectFunctionSensitiveFields(tenantUser, app, fullCodePath, functionID, schema)
	if err := s.repo.ReplaceForFunction(ctx, tenantUser, app, fullCodePath, fields); err != nil {
		return err
	}

	sections := make(sensitiveFieldSections)
	for _, field := range fields {
		sections.add(field.Section, field.FieldPath)
	}

	s.mu.Lock()
	s.cache.replace(fullCodePath, sections)
	s.mu.Unlock()
	return nil
}

func (s *FunctionSensitiveFieldService) DeleteFunction(ctx context.Context, tenantUser, app, fullCodePath string) error {
	if err := s.repo.DeleteForFunction(ctx, tenantUser, app, fullCodePath); err != nil {
		return err
	}
	s.mu.Lock()
	delete(s.cache, fullCodePath)
	s.mu.Unlock()
	return nil
}

func (s *FunctionSensitiveFieldService) SensitiveFieldSet(fullCodePath string, sections ...string) map[string]struct{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	bySection := s.cache[fullCodePath]
	if len(bySection) == 0 {
		return nil
	}
	return bySection.only(sections...)
}

func collectFunctionSensitiveFields(tenantUser, app, fullCodePath string, functionID int64, schema *functionschema.FunctionSchema) []*model.FunctionSensitiveField {
	if schema == nil {
		return nil
	}
	var fields []*model.FunctionSensitiveField
	appendFields := func(section string, sectionFields []*widget.Field) {
		fields = append(fields, collectSensitiveFieldsFromSection(tenantUser, app, fullCodePath, functionID, schema.Type, section, sectionFields)...)
	}
	switch schema.Type {
	case functionschema.TypeForm:
		if schema.Form != nil {
			appendFields(sensitiveFieldSectionRequest, schema.Form.Request)
			appendFields(sensitiveFieldSectionResponse, schema.Form.Response)
		}
	case functionschema.TypeTable:
		if schema.Table != nil {
			appendFields(sensitiveFieldSectionRequest, schema.Table.Request)
			appendFields(sensitiveFieldSectionFields, schema.Table.Fields)
		}
	case functionschema.TypeChart:
		if schema.Chart != nil {
			appendFields(sensitiveFieldSectionRequest, schema.Chart.Request)
			appendFields(sensitiveFieldSectionResponse, schema.Chart.Response)
		}
	}
	return fields
}

func collectSensitiveFieldsFromSection(tenantUser, app, fullCodePath string, functionID int64, schemaType, section string, fields []*widget.Field) []*model.FunctionSensitiveField {
	var collected []*model.FunctionSensitiveField
	var walk func(prefix string, fields []*widget.Field)
	walk = func(prefix string, fields []*widget.Field) {
		for _, field := range fields {
			if field == nil {
				continue
			}
			code := strings.TrimSpace(field.Code)
			if code == "" {
				continue
			}
			fieldPath := joinFieldPath(prefix, code)
			if isSchemaSensitiveField(field) {
				collected = append(collected, &model.FunctionSensitiveField{
					TenantUser:   tenantUser,
					App:          app,
					FullCodePath: fullCodePath,
					FunctionID:   functionID,
					SchemaType:   schemaType,
					Section:      section,
					FieldPath:    fieldPath,
					FieldCode:    code,
					FieldName:    field.Name,
					Source:       sensitiveFieldSourceSchema,
				})
			}
			if len(field.Children) > 0 {
				walk(fieldPath, field.Children)
			}
		}
	}
	walk("", fields)
	return collected
}

func isSchemaSensitiveField(field *widget.Field) bool {
	return field != nil && field.Sensitive
}

func joinFieldPath(prefix, code string) string {
	if strings.TrimSpace(prefix) == "" {
		return code
	}
	return fmt.Sprintf("%s.%s", strings.TrimSpace(prefix), code)
}
