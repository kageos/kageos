package app

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// CompileAndValidate compiles the registered app definition into the platform
// schema contract. It does not execute business handlers or callbacks.
func (a *App) CompileAndValidate() error {
	if a == nil {
		return fmt.Errorf("app is nil")
	}

	var errs []error
	for _, info := range sortedRouterInfos(a.routerInfo) {
		if info == nil || info.IsDefaultRouter() {
			continue
		}
		if err := validateRouterDefinition(info); err != nil {
			errs = append(errs, err)
		}
	}

	if _, _, err := a.getApis(); err != nil {
		errs = append(errs, err)
	}
	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("SDK schema compile failed: %w", err)
	}
	return nil
}

func sortedRouterInfos(routes map[string]*routerInfo) []*routerInfo {
	result := make([]*routerInfo, 0, len(routes))
	for _, info := range routes {
		result = append(result, info)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i] == nil {
			return false
		}
		if result[j] == nil {
			return true
		}
		left := result[i].Method + " " + result[i].Router
		right := result[j].Method + " " + result[j].Router
		return left < right
	})
	return result
}

func validateRouterDefinition(info *routerInfo) error {
	route := strings.TrimSpace(info.Router)
	var errs []error
	if route == "" {
		errs = append(errs, fmt.Errorf("router is empty"))
	}
	if info.HandleFunc == nil {
		errs = append(errs, fmt.Errorf("%s handler is nil", route))
	}
	if info.Template == nil {
		errs = append(errs, fmt.Errorf("%s template is nil", route))
		return errors.Join(errs...)
	}

	templateType := info.Template.TemplateType()
	switch templateType {
	case TemplateTypeTable:
		if _, ok := info.Template.(*TableTemplate); !ok {
			errs = append(errs, fmt.Errorf("%s declares template type %q but template is not *TableTemplate", route, templateType))
			break
		}
		if err := validateRouteSuffix(route, ".table", templateType); err != nil {
			errs = append(errs, err)
		}
	case TemplateTypeForm:
		if _, ok := info.Template.(*FormTemplate); !ok {
			errs = append(errs, fmt.Errorf("%s declares template type %q but template is not *FormTemplate", route, templateType))
			break
		}
		if err := validateRouteSuffix(route, ".form", templateType); err != nil {
			errs = append(errs, err)
		}
	case TemplateTypeChart:
		if _, ok := info.Template.(*ChartTemplate); !ok {
			errs = append(errs, fmt.Errorf("%s declares template type %q but template is not *ChartTemplate", route, templateType))
			break
		}
		if err := validateRouteSuffix(route, ".chart", templateType); err != nil {
			errs = append(errs, err)
		}
	default:
		errs = append(errs, fmt.Errorf("%s has unsupported template type: %s", route, templateType))
	}
	return errors.Join(errs...)
}

func validateRouteSuffix(route, suffix string, templateType TemplateType) error {
	if !strings.HasSuffix(strings.TrimSpace(route), suffix) {
		return fmt.Errorf("%s uses template type %q but route must end with %s", route, templateType, suffix)
	}
	return nil
}
