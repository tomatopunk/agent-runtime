package runc

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/tomatopunk/agent-runtime/internal/backend"
)

//go:embed runc.config.yaml.tpl
var configTplFS embed.FS

type configData struct {
	BuildArgs      []string
	HostType       string
	HostName       string
	RootFs         string
	PluginId       string
	PluginVersion  string
	DeviceId       string
	CPU            int   // shares
	Memory         int64 // bytes
	Env            []string
}

func parseCPU(s string) int {
	if s == "" {
		return 1024
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 1024
	}
	if f <= 0 {
		return 1024
	}
	return int(f * 1024)
}

func parseMemory(s string) int64 {
	if s == "" {
		return 512 * 1024 * 1024 // 512Mi default
	}
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	var mult int64 = 1
	if strings.HasSuffix(s, "k") {
		mult = 1024
		s = strings.TrimSuffix(s, "k")
	} else if strings.HasSuffix(s, "m") {
		mult = 1024 * 1024
		s = strings.TrimSuffix(s, "m")
	} else if strings.HasSuffix(s, "g") {
		mult = 1024 * 1024 * 1024
		s = strings.TrimSuffix(s, "g")
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 512 * 1024 * 1024
	}
	if n <= 0 {
		return 512 * 1024 * 1024
	}
	return n * mult
}

func writeConfigJSON(workDir string, opts backend.RunOptions) error {
	tplBytes, err := configTplFS.ReadFile("runc.config.yaml.tpl")
	if err != nil {
		return fmt.Errorf("read runc config template: %w", err)
	}
	tpl, err := template.New("config").Funcs(template.FuncMap{
		"jsonQuote": func(s string) (string, error) {
			b, err := jsonMarshalString(s)
			return string(b), err
		},
	}).Parse(string(tplBytes))
	if err != nil {
		return fmt.Errorf("parse runc config template: %w", err)
	}
	buildArgs := []string{opts.Config}
	if opts.Config == "" {
		buildArgs = []string{"/bin/sh"}
	}
	data := configData{
		BuildArgs:     buildArgs,
		HostType:      opts.HostType,
		HostName:      opts.HostName,
		RootFs:        "rootfs",
		PluginId:      opts.PluginID,
		PluginVersion: opts.PluginVersion,
		DeviceId:      opts.DeviceId,
		CPU:           parseCPU(opts.CPU),
		Memory:        parseMemory(opts.Mem),
		Env:           opts.Env,
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute runc config template: %w", err)
	}
	configPath := filepath.Join(workDir, "config.json")
	return os.WriteFile(configPath, buf.Bytes(), 0644)
}

func jsonMarshalString(s string) ([]byte, error) {
	// Standard JSON encoding for a string (escapes quotes etc.)
	return []byte(strconv.Quote(s)), nil
}
