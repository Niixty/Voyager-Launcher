//go:build windows

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"voyager/native/internal/jsonstore"
)

type gameConfig struct {
	SchemaVersion    int    `json:"schemaVersion"`
	Title            string `json:"title"`
	Platform         string `json:"platform"`
	CoreName         string `json:"coreName"`
	Description      string `json:"description"`
	Mode             string `json:"mode"`
	RetroarchPath    string `json:"retroarchPath,omitempty"`
	CorePath         string `json:"corePath,omitempty"`
	RomPath          string `json:"romPath,omitempty"`
	ExecutablePath   string `json:"executablePath,omitempty"`
	ShortcutPath     string `json:"shortcutPath,omitempty"`
	Arguments        string `json:"arguments,omitempty"`
	WorkingDirectory string `json:"workingDirectory,omitempty"`
	WaitForProcess   string `json:"waitForProcess,omitempty"`
	Hero             string `json:"hero"`
	Grid             string `json:"grid"`
	Cartridge        string `json:"cartridge,omitempty"`
}

var validID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,70}$`)
var rootDir string

func gameDir(id string) string    { return filepath.Join(rootDir, "Games", id) }
func configFile(id string) string { return filepath.Join(gameDir(id), "config.json") }
func listGames() []string {
	entries, err := os.ReadDir(filepath.Join(rootDir, "Games"))
	if err != nil {
		return nil
	}
	var ids []string
	for _, e := range entries {
		if e.IsDir() && validID.MatchString(e.Name()) {
			if _, err := os.Stat(configFile(e.Name())); err == nil {
				ids = append(ids, e.Name())
			}
		}
	}
	sort.Strings(ids)
	return ids
}
func readConfig(id string) (gameConfig, error) {
	if !validID.MatchString(id) {
		return gameConfig{}, errors.New("ID de jogo inválido")
	}
	data, err := os.ReadFile(configFile(id))
	if err != nil {
		return gameConfig{}, err
	}
	var c gameConfig
	if err = json.Unmarshal(data, &c); err != nil {
		return c, err
	}
	if c.Title == "" {
		c.Title = id
	}
	if c.Platform == "" {
		c.Platform = "JOGO"
	}
	if c.Mode == "" {
		c.Mode = "retroarch"
	}
	if c.Grid == "" {
		c.Grid = "grid.png"
	}
	if c.Hero == "" {
		c.Hero = "hero.png"
	}
	if c.Cartridge == "" {
		c.Cartridge = "cartridge.png"
	}
	return c, nil
}
func writeConfig(id string, c gameConfig) error {
	if !validID.MatchString(id) {
		return fmt.Errorf("ID inválido. Use letras, números, _ e -")
	}
	if strings.TrimSpace(c.Title) == "" {
		return errors.New("Informe o título do jogo")
	}
	if c.Mode == "retroarch" {
		if c.RetroarchPath == "" || c.CorePath == "" || c.RomPath == "" {
			return errors.New("Preencha RetroArch, núcleo e ROM")
		}
	} else {
		if c.ExecutablePath == "" {
			return errors.New("Selecione o .exe ou .lnk do jogo")
		}
		ext := strings.ToLower(filepath.Ext(c.ExecutablePath))
		if ext != ".exe" && ext != ".lnk" {
			return errors.New("O jogo Windows deve usar um executável .exe ou atalho .lnk")
		}
	}
	if err := os.MkdirAll(gameDir(id), 0755); err != nil {
		return err
	}
	if c.SchemaVersion < 6 {
		c.SchemaVersion = 6
	}
	if c.Hero == "" {
		c.Hero = "hero.png"
	}
	if c.Grid == "" {
		c.Grid = "grid.png"
	}
	if c.Cartridge == "" {
		c.Cartridge = "cartridge.png"
	}
	// Preserve optional future metadata but remove all obsolete time fields.
	doc := map[string]json.RawMessage{}
	if data, err := os.ReadFile(configFile(id)); err == nil {
		if err = json.Unmarshal(data, &doc); err != nil || doc == nil {
			return errors.New("config.json inválido. O arquivo original foi preservado")
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	delete(doc, "loadingDurationSeconds")
	delete(doc, "fadeOutSeconds")
	delete(doc, "startupTimeoutSeconds")
	raw, _ := json.Marshal(c)
	var typed map[string]json.RawMessage
	_ = json.Unmarshal(raw, &typed)
	for _, k := range []string{"title", "platform", "coreName", "description", "mode", "retroarchPath", "corePath", "romPath", "executablePath", "shortcutPath", "arguments", "workingDirectory", "waitForProcess", "hero", "grid", "cartridge", "schemaVersion"} {
		delete(doc, k)
	}
	for k, v := range typed {
		doc[k] = v
	}
	saved, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return jsonstore.Write(configFile(id), append(saved, '\n'))
}
func resolveGameFile(name, folder string) string {
	if name == "" {
		return ""
	}
	name = expandWindows(name)
	if filepath.IsAbs(name) {
		return name
	}
	return filepath.Join(folder, name)
}
func artworkPath(c gameConfig, id, which string) string {
	folder := gameDir(id)
	name := c.Hero
	fallback := "background.png"
	if which == "grid" {
		name = c.Grid
		fallback = "logo.png"
	} else if which == "cartridge" {
		name = c.Cartridge
		fallback = ""
	}
	preferred := which + ".png"
	if _, err := os.Stat(filepath.Join(folder, preferred)); err == nil {
		return filepath.Join(folder, preferred)
	}
	if name != "" {
		p := resolveGameFile(name, folder)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if fallback != "" {
		if _, err := os.Stat(filepath.Join(folder, fallback)); err == nil {
			return filepath.Join(folder, fallback)
		}
	}
	return ""
}
func duplicateDir(src, dst string) error {
	return filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("link não importado: %s", p)
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		in, err := os.Open(p)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.Create(target)
		if err != nil {
			return err
		}
		_, err = io.Copy(out, in)
		e := out.Close()
		if err != nil {
			return err
		}
		return e
	})
}
func screenshotName(id string) string {
	if id == "" {
		id = "Jogo"
	}
	return fmt.Sprintf("Voyager_%s_50%%_%s.png", id, time.Now().Format("20060102_150405.000"))
}
