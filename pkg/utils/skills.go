package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// SkillsTemplateDir is the directory containing built-in skill templates
	SkillsTemplateDir = "skills"
)

// CopyBuiltinSkills copies built-in skills from project template to user workspace (skip existing)
func CopyBuiltinSkills(workspace string) error {
	if _, err := os.Stat(SkillsTemplateDir); err != nil {
		return fmt.Errorf("skills template directory not found: %s", SkillsTemplateDir)
	}
	
	targetDir := filepath.Join(workspace, ".claude", "skills")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return err
	}
	
	entries, err := os.ReadDir(SkillsTemplateDir)
	if err != nil {
		return err
	}
	
	skipped := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillName := entry.Name()
		srcPath := filepath.Join(SkillsTemplateDir, skillName)
		dstPath := filepath.Join(targetDir, skillName)
		
		// Skip if already exists
		if _, err := os.Stat(dstPath); err == nil {
			fmt.Printf("Skipped: %s already exists\n", skillName)
			skipped++
			continue
		}
		
		// Copy new skill
		if err := CopyDir(srcPath, dstPath); err != nil {
			fmt.Printf("Warning: failed to copy skill %s: %v\n", skillName, err)
			continue
		}
		fmt.Printf("Installed: %s\n", skillName)
	}
	
	if skipped > 0 {
		fmt.Printf("\nTip: Use 'myclaw skills update' to reinstall existing skills.\n")
	}
	
	return nil
}

// UpdateBuiltinSkills updates all built-in skills (overwrite existing)
func UpdateBuiltinSkills(workspace string) error {
	if _, err := os.Stat(SkillsTemplateDir); err != nil {
		return fmt.Errorf("skills template directory not found: %s", SkillsTemplateDir)
	}
	
	targetDir := filepath.Join(workspace, ".claude", "skills")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return err
	}
	
	entries, err := os.ReadDir(SkillsTemplateDir)
	if err != nil {
		return err
	}
	
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillName := entry.Name()
		srcPath := filepath.Join(SkillsTemplateDir, skillName)
		dstPath := filepath.Join(targetDir, skillName)
		
		// Remove existing if present
		if _, err := os.Stat(dstPath); err == nil {
			if err := os.RemoveAll(dstPath); err != nil {
				fmt.Printf("Warning: failed to remove existing skill %s: %v\n", skillName, err)
				continue
			}
		}
		
		// Copy new version
		if err := CopyDir(srcPath, dstPath); err != nil {
			fmt.Printf("Warning: failed to copy skill %s: %v\n", skillName, err)
			continue
		}
		fmt.Printf("Updated: %s\n", skillName)
	}
	
	return nil
}

// CopyDir recursively copies a directory
func CopyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)
		
		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}
		
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dstPath, data, info.Mode())
	})
}

// ListInstalledSkills returns a list of installed skill names
func ListInstalledSkills(workspace string) ([]string, error) {
	skillsDir := filepath.Join(workspace, ".claude", "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	
	var skills []string
	for _, entry := range entries {
		if entry.IsDir() {
			skillMD := filepath.Join(skillsDir, entry.Name(), "SKILL.md")
			if _, err := os.Stat(skillMD); err == nil {
				skills = append(skills, entry.Name())
			}
		}
	}
	
	return skills, nil
}

// InstallSkill installs a specific skill (skip if exists)
func InstallSkill(workspace, skillName string) error {
	srcPath := filepath.Join(SkillsTemplateDir, skillName)
	
	if _, err := os.Stat(srcPath); err != nil {
		return fmt.Errorf("skill %s not found in templates", skillName)
	}
	
	dstPath := filepath.Join(workspace, ".claude", "skills", skillName)
	
	// Skip if already exists
	if _, err := os.Stat(dstPath); err == nil {
		fmt.Printf("Skipped: %s already exists (use 'myclaw skills update %s' to reinstall)\n", skillName, skillName)
		return nil
	}
	
	// Copy new skill
	if err := CopyDir(srcPath, dstPath); err != nil {
		return err
	}
	
	fmt.Printf("Installed skill: %s\n", skillName)
	return nil
}

// UpdateSkill updates a specific skill (overwrite if exists)
func UpdateSkill(workspace, skillName string) error {
	srcPath := filepath.Join(SkillsTemplateDir, skillName)
	
	if _, err := os.Stat(srcPath); err != nil {
		return fmt.Errorf("skill %s not found in templates", skillName)
	}
	
	dstPath := filepath.Join(workspace, ".claude", "skills", skillName)
	
	// Remove existing if present
	if _, err := os.Stat(dstPath); err == nil {
		if err := os.RemoveAll(dstPath); err != nil {
			return fmt.Errorf("failed to remove existing skill: %w", err)
		}
	}
	
	// Copy new version
	if err := CopyDir(srcPath, dstPath); err != nil {
		return err
	}
	
	fmt.Printf("Updated skill: %s\n", skillName)
	return nil
}

// UninstallSkill removes a skill from the workspace
func UninstallSkill(workspace, skillName string) error {
	skillPath := filepath.Join(workspace, ".claude", "skills", skillName)
	
	if _, err := os.Stat(skillPath); os.IsNotExist(err) {
		return fmt.Errorf("skill %s not installed", skillName)
	}
	
	if err := os.RemoveAll(skillPath); err != nil {
		return fmt.Errorf("failed to uninstall skill: %w", err)
	}
	
	fmt.Printf("Uninstalled skill: %s\n", skillName)
	return nil
}

// VerifySkill checks if a skill is valid
func VerifySkill(workspace, skillName string) error {
	skillDir := filepath.Join(workspace, ".claude", "skills", skillName)
	skillMD := filepath.Join(skillDir, "SKILL.md")
	
	if _, err := os.Stat(skillMD); err != nil {
		return fmt.Errorf("missing SKILL.md")
	}
	
	data, err := os.ReadFile(skillMD)
	if err != nil {
		return fmt.Errorf("cannot read SKILL.md: %w", err)
	}
	
	content := string(data)
	if !strings.Contains(content, "---") || !strings.Contains(content, "name:") {
		return fmt.Errorf("invalid SKILL.md format")
	}
	
	return nil
}

// VerifyAllSkills verifies all installed skills
func VerifyAllSkills(workspace string) (map[string]error, error) {
	skills, err := ListInstalledSkills(workspace)
	if err != nil {
		return nil, err
	}
	
	results := make(map[string]error)
	for _, skillName := range skills {
		results[skillName] = VerifySkill(workspace, skillName)
	}
	
	return results, nil
}

// FormatRelativeTime formats a Unix timestamp as a relative time string
func FormatRelativeTime(unixTime int64) string {
	t := time.Unix(unixTime, 0)
	now := time.Now()
	diff := now.Sub(t)
	
	if diff < time.Minute {
		return "just now"
	} else if diff < time.Hour {
		mins := int(diff.Minutes())
		return fmt.Sprintf("%d minute(s) ago", mins)
	} else if diff < 24*time.Hour {
		hours := int(diff.Hours())
		return fmt.Sprintf("%d hour(s) ago", hours)
	} else if diff < 7*24*time.Hour {
		days := int(diff.Hours() / 24)
		return fmt.Sprintf("%d day(s) ago", days)
	} else {
		return t.Format("2006-01-02 15:04")
	}
}

