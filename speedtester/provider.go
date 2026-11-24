package speedtester

import (
	"context"
	"fmt"
	"maps"
	"os"
	"path/filepath"

	"github.com/metacubex/mihomo/adapter/provider"
	"github.com/metacubex/mihomo/common/convert"
	"github.com/metacubex/mihomo/common/structure"
	"github.com/metacubex/mihomo/common/utils"
	"github.com/metacubex/mihomo/component/resource"
	"gopkg.in/yaml.v3"
)

// providerNodes takes one snapshot so measurement and export use the same subscription.
func (st *SpeedTester) providerNodes(name string, config map[string]any, workDir string) (map[string]*CProxy, error) {
	if name == provider.ReservedName {
		return nil, fmt.Errorf("reserved provider name %q", name)
	}
	var spec struct {
		Type      string                  `provider:"type"`
		Path      string                  `provider:"path,omitempty"`
		URL       string                  `provider:"url,omitempty"`
		Proxy     string                  `provider:"proxy,omitempty"`
		Header    map[string][]string     `provider:"header,omitempty"`
		SizeLimit int64                   `provider:"size-limit,omitempty"`
		Payload   []map[string]any        `provider:"payload,omitempty"`
		Override  provider.OverrideSchema `provider:"override,omitempty"`
	}
	decoder := structure.NewDecoder(structure.Option{TagName: "provider", WeaklyTypedInput: true})
	if err := decoder.Decode(config, &spec); err != nil {
		return nil, err
	}
	nodes := spec.Payload
	if spec.Type != "inline" {
		var body []byte
		var err error
		switch spec.Type {
		case "file":
			if workDir == "" || spec.Path == "" {
				return nil, fmt.Errorf("file provider requires a local configuration and a path")
			}
			body, err = os.ReadFile(resolveProviderPath(workDir, spec.Path))
		case "http":
			if spec.URL == "" {
				return nil, fmt.Errorf("missing provider URL")
			}
			if workDir != "" {
				cache := spec.Path
				if cache == "" {
					cache = filepath.Join("proxies", utils.MakeHash([]byte(spec.URL)).String())
				}
				body, err = os.ReadFile(resolveProviderPath(workDir, cache))
			}
			if body == nil && (err == nil || os.IsNotExist(err)) {
				if spec.Header == nil {
					spec.Header = map[string][]string{}
				}
				if _, ok := spec.Header["User-Agent"]; !ok && st.config.UserAgent != "" {
					spec.Header["User-Agent"] = []string{st.config.UserAgent}
				}
				vehicle := resource.NewHTTPVehicle(spec.URL, "", spec.Proxy, spec.Header, resource.DefaultHttpTimeout, spec.SizeLimit)
				body, _, err = vehicle.Read(context.Background(), utils.HashType{})
			}
		default:
			return nil, fmt.Errorf("unknown provider type %q", spec.Type)
		}
		if err != nil && len(nodes) == 0 {
			return nil, err
		}
		if err == nil {
			var raw provider.ProxySchema
			if err := yaml.Unmarshal(body, &raw); err != nil {
				raw.Proxies, err = convert.ConvertsV2Ray(body)
				if err != nil {
					return nil, fmt.Errorf("invalid provider payload")
				}
			}
			nodes = raw.Proxies
		}
	}
	snapshot := maps.Clone(config)
	snapshot["type"] = "inline"
	snapshot["payload"] = nodes
	for _, rule := range spec.Override.ProxyName {
		if rule.Pattern == nil {
			return nil, fmt.Errorf("missing provider rename pattern")
		}
	}
	pd, err := provider.ParseProxyProvider(name, snapshot)
	if err != nil {
		return nil, err
	}
	byName := make(map[string]map[string]any, len(nodes))
	overrides, _ := config["override"].(map[string]any)
	for _, node := range nodes {
		original, ok := node["name"].(string)
		if !ok {
			continue
		}
		effective := original
		if spec.Override.AdditionalPrefix != nil {
			effective = *spec.Override.AdditionalPrefix + effective
		}
		if spec.Override.AdditionalSuffix != nil {
			effective += *spec.Override.AdditionalSuffix
		}
		for _, rule := range spec.Override.ProxyName {
			effective, err = rule.Pattern.Replace(effective, rule.Target, 0, -1)
			if err != nil {
				return nil, err
			}
		}
		node = maps.Clone(node)
		node["name"] = effective
		if dialer, ok := config["dialer-proxy"].(string); ok && dialer != "" {
			node["dialer-proxy"] = dialer
		}
		for key, value := range overrides {
			switch key {
			case "additional-prefix", "additional-suffix", "proxy-name":
			default:
				node[key] = value
			}
		}
		if _, exists := byName[effective]; !exists {
			byName[effective] = node
		}
	}
	proxies := make(map[string]*CProxy)
	for _, proxy := range pd.Proxies() {
		proxies[fmt.Sprintf("[%s] %s", name, proxy.Name())] = &CProxy{Proxy: proxy, Config: byName[proxy.Name()]}
	}
	return proxies, nil
}

func resolveProviderPath(dir, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(dir, path)
}
