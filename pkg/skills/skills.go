package skills

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type Skill struct {
	Name          string
	Description   string
	SkillFilePath string
}

// skillFrontMatter mirrors the YAML front matter in SKILL.md.
type skillFrontMatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

func loadSkillsFromDir(dir string) ([]*Skill, error) {
	var skills []*Skill
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.EqualFold(d.Name(), "SKILL.md") {
			skill, err := parseSkillFile(path)
			if err != nil {
				return fmt.Errorf("parse %s: %w", path, err)
			}
			skills = append(skills, skill)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(skills, func(i, j int) bool {
		return strings.ToLower(skills[i].Name) < strings.ToLower(skills[j].Name)
	})

	return skills, nil
}

// LoadFromDirs loads and parses all SKILL.md files under the provided directories.
func LoadFromDirs(dirs []string) ([]*Skill, error) {
	var skills []*Skill
	for _, dir := range dirs {
		if strings.TrimSpace(dir) == "" {
			continue
		}
		found, err := loadSkillsFromDir(dir)
		if err != nil {
			return nil, err
		}
		skills = append(skills, found...)
	}

	sort.Slice(skills, func(i, j int) bool {
		left := strings.ToLower(skills[i].Name)
		right := strings.ToLower(skills[j].Name)
		if left == right {
			return strings.ToLower(skills[i].SkillFilePath) < strings.ToLower(skills[j].SkillFilePath)
		}
		return left < right
	})

	return skills, nil
}

func parseSkillFile(path string) (*Skill, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	fm, err := parseFrontMatter(content)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(fm.Name) == "" {
		return nil, fmt.Errorf("missing front matter name")
	}

	return &Skill{
		Name:          strings.TrimSpace(fm.Name),
		Description:   strings.TrimSpace(fm.Description),
		SkillFilePath: path,
	}, nil
}

// parseFrontMatter extracts YAML front matter from the file content.
func parseFrontMatter(content []byte) (skillFrontMatter, error) {
	lines := strings.Split(string(content), "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return skillFrontMatter{}, fmt.Errorf("missing YAML front matter")
	}

	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end == -1 {
		return skillFrontMatter{}, fmt.Errorf("unterminated YAML front matter")
	}

	fmText := strings.Join(lines[1:end], "\n")
	var fm skillFrontMatter
	if err := yaml.Unmarshal([]byte(fmText), &fm); err != nil {
		return skillFrontMatter{}, err
	}
	return fm, nil
}
