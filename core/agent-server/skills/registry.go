package skills

import (
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"sync"
)

type Registry struct {
	skills     []*Skill
	byID       map[string]*Skill
	byName     map[string]*Skill
	loadErrors []error
}

var (
	defaultOnce sync.Once
	defaultReg  *Registry
)

func DefaultRegistry() *Registry {
	defaultOnce.Do(func() {
		defaultReg = LoadEmbedded()
	})
	return defaultReg
}

func LoadEmbedded() *Registry {
	reg := &Registry{
		byID:   make(map[string]*Skill),
		byName: make(map[string]*Skill),
	}

	err := fs.WalkDir(skillFS, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			reg.loadErrors = append(reg.loadErrors, fmt.Errorf("%s: %w", path, err))
			return nil
		}
		if entry.IsDir() || entry.Name() != "SKILL.md" {
			return nil
		}
		data, err := skillFS.ReadFile(path)
		if err != nil {
			reg.loadErrors = append(reg.loadErrors, fmt.Errorf("%s: %w", path, err))
			return nil
		}
		skill, err := ParseSkill(string(data), path)
		if err != nil {
			reg.loadErrors = append(reg.loadErrors, err)
			return nil
		}
		reg.add(skill)
		return nil
	})
	if err != nil {
		reg.loadErrors = append(reg.loadErrors, err)
	}

	sort.Slice(reg.skills, func(i, j int) bool {
		return reg.skills[i].Meta.ID < reg.skills[j].Meta.ID
	})
	return reg
}

func (r *Registry) add(skill *Skill) {
	if r == nil || skill == nil {
		return
	}
	idKey := strings.ToLower(skill.Meta.ID)
	if existing := r.byID[idKey]; existing != nil {
		r.loadErrors = append(r.loadErrors, fmt.Errorf("duplicate skill id %q in %s and %s", skill.Meta.ID, existing.Meta.Path, skill.Meta.Path))
		return
	}
	nameKey := strings.ToLower(skill.Meta.Name)
	if existing := r.byName[nameKey]; existing != nil {
		r.loadErrors = append(r.loadErrors, fmt.Errorf("duplicate skill name %q in %s and %s", skill.Meta.Name, existing.Meta.Path, skill.Meta.Path))
		return
	}
	r.skills = append(r.skills, skill)
	r.byID[idKey] = skill
	r.byName[nameKey] = skill
}

func (r *Registry) LoadError() error {
	if r == nil || len(r.loadErrors) == 0 {
		return nil
	}
	return errors.Join(r.loadErrors...)
}

func (r *Registry) LoadErrors() []error {
	if r == nil || len(r.loadErrors) == 0 {
		return nil
	}
	out := make([]error, len(r.loadErrors))
	copy(out, r.loadErrors)
	return out
}

func (r *Registry) All() []*Skill {
	if r == nil {
		return nil
	}
	out := make([]*Skill, len(r.skills))
	copy(out, r.skills)
	return out
}

func (r *Registry) Get(idOrName string) (*Skill, bool) {
	if r == nil {
		return nil, false
	}
	key := strings.ToLower(strings.TrimSpace(idOrName))
	if key == "" {
		return nil, false
	}
	if skill := r.byID[key]; skill != nil {
		return skill, true
	}
	if skill := r.byName[key]; skill != nil {
		return skill, true
	}
	for _, skill := range r.skills {
		if strings.EqualFold(strings.TrimSuffix(skill.Meta.Path, "/SKILL.md"), key) {
			return skill, true
		}
	}
	return nil, false
}
