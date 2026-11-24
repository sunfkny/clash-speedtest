package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/v4/process"
	"gopkg.in/yaml.v2"
)

type configSource struct {
	client  string
	path    string
	workDir string
}

func regularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func partySource(root, dir string) configSource {
	if !regularFile(filepath.Join(root, "profile.yaml")) {
		return configSource{}
	}
	return configSource{"Clash Party", filepath.Join(dir, "config.yaml"), dir}
}

func partySnapshot(root string) (configSource, error) {
	if !regularFile(filepath.Join(root, "profile.yaml")) {
		return configSource{}, nil
	}
	var settings struct {
		Separate bool `yaml:"diffWorkDir"`
	}
	if err := readMetadata(filepath.Join(root, "config.yaml"), &settings); err != nil {
		return configSource{}, err
	}
	dir := filepath.Join(root, "work")
	if settings.Separate {
		var profile struct {
			Current string `yaml:"current"`
		}
		if err := readMetadata(filepath.Join(root, "profile.yaml"), &profile); err != nil {
			return configSource{}, err
		}
		if profile.Current == "" {
			profile.Current = "default"
		}
		if !filepath.IsLocal(profile.Current) || filepath.Base(profile.Current) != profile.Current || profile.Current == "." {
			return configSource{}, fmt.Errorf("invalid current profile in %s", root)
		}
		dir = filepath.Join(dir, profile.Current)
	}
	return partySource(root, dir), nil
}

func readMetadata(path string, target any) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err = yaml.Unmarshal(body, target); err != nil {
		return fmt.Errorf("read metadata %s: %w", path, err)
	}
	return nil
}

func clashctlSource(root, runtimePath string) configSource {
	if !regularFile(filepath.Join(root, "scripts", "cmd", "clashctl.sh")) {
		return configSource{}
	}
	return configSource{"clashctl", runtimePath, filepath.Join(root, "resources")}
}

func runningSource(args []string) configSource {
	var dir, config string
	for i := 1; i < len(args); i++ {
		key, value, hasValue := strings.Cut(args[i], "=")
		if key == "-t" {
			return configSource{}
		}
		if key != "-d" && key != "-f" {
			continue
		}
		if !hasValue {
			i++
			if i == len(args) {
				return configSource{}
			}
			value = args[i]
		}
		if key == "-d" {
			dir = filepath.Clean(value)
		} else {
			config = filepath.Clean(value)
		}
	}
	if !filepath.IsAbs(dir) {
		return configSource{}
	}
	parent := filepath.Dir(dir)
	if filepath.Base(dir) == "resources" && filepath.IsAbs(config) && filepath.Base(config) == "runtime.yaml" {
		return clashctlSource(parent, config)
	}
	if config != "" && config != filepath.Join(dir, "config.yaml") {
		return configSource{}
	}
	if filepath.Base(dir) == "work" {
		return partySource(parent, dir)
	}
	if filepath.Base(parent) == "work" {
		return partySource(filepath.Dir(parent), dir)
	}
	return configSource{}
}

type sources []configSource

func (list *sources) add(s configSource) {
	if s.path == "" || !regularFile(s.path) {
		return
	}
	for _, existing := range *list {
		if existing.path == s.path || (runtime.GOOS == "windows" && strings.EqualFold(existing.path, s.path)) {
			return
		}
	}
	*list = append(*list, s)
}

func discoverConfigs() (sources, error) {
	var found sources
	processes, _ := process.Processes()
	for _, p := range processes {
		name, _ := p.Name()
		switch strings.TrimSuffix(strings.ToLower(name), ".exe") {
		case "mihomo", "mihomo-smart", "mihomo-alpha", "clash", "clash-meta":
			args, err := p.CmdlineSlice()
			if err == nil {
				found.add(runningSource(args))
			}
		}
	}
	if len(found) > 0 {
		return found, nil
	}

	return installedConfigs()
}

func installedConfigs() (sources, error) {
	var found sources
	var partyRoots []string
	if configDir, err := os.UserConfigDir(); err == nil {
		partyRoots = append(partyRoots, filepath.Join(configDir, "mihomo-party"))
	}
	executable, _ := os.Executable()
	cwd, _ := os.Getwd()
	for _, dir := range []string{filepath.Dir(executable), cwd} {
		if regularFile(filepath.Join(dir, "PORTABLE")) {
			partyRoots = append(partyRoots, filepath.Join(dir, "data"))
		}
	}
	var metadataErr error
	for _, root := range partyRoots {
		source, err := partySnapshot(root)
		if err != nil {
			metadataErr = err
			continue
		}
		found.add(source)
	}
	if runtime.GOOS == "linux" {
		var roots []string
		if root := os.Getenv("CLASHCTL_HOME"); root != "" {
			roots = append(roots, root)
		} else if home, err := os.UserHomeDir(); err == nil {
			roots = append(roots, filepath.Join(home, ".clashctl"), filepath.Join(home, "clashctl"))
		}
		for _, root := range roots {
			source := clashctlSource(root, filepath.Join(root, "data", "runtime.yaml"))
			if !regularFile(source.path) {
				source = clashctlSource(root, filepath.Join(root, "resources", "runtime.yaml"))
			}
			found.add(source)
		}
	}
	if len(found) == 0 && metadataErr != nil {
		return nil, metadataErr
	}
	return found, nil
}

func chooseConfig(found sources, input io.Reader, output io.Writer, interactive bool) (configSource, error) {
	switch len(found) {
	case 0:
		return configSource{}, fmt.Errorf("未找到 Clash Party 或 clashctl 的运行配置，请使用 -c 指定配置文件")
	case 1:
		return found[0], nil
	}
	sort.Slice(found, func(i, j int) bool { return found[i].path < found[j].path })
	for i, s := range found {
		fmt.Fprintf(output, "%d. %s  %s\n", i+1, s.client, s.path)
	}
	if !interactive {
		return configSource{}, fmt.Errorf("找到多个配置，请使用 -c 指定配置文件")
	}
	fmt.Fprint(output, "选择要测速的配置编号：")
	line, err := bufio.NewReader(input).ReadString('\n')
	if err != nil && err != io.EOF {
		return configSource{}, err
	}
	choice, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil || choice < 1 || choice > len(found) {
		return configSource{}, fmt.Errorf("无效的配置编号")
	}
	return found[choice-1], nil
}
