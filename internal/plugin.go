// Package internal implements the workflow-plugin-marketplace external plugin,
// providing marketplace step types for plugin discovery and management.
package internal

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/GoCodeAlone/workflow-plugin-marketplace/internal/contracts"
	pb "github.com/GoCodeAlone/workflow/plugin/external/proto"
	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/anypb"
)

// Version is set at build time via -ldflags
// "-X github.com/GoCodeAlone/workflow-plugin-marketplace/internal.Version=X.Y.Z".
// Default is a bare semver so plugin loaders that validate semver accept
// unreleased dev builds; goreleaser overrides with the real release tag.
var Version = "0.0.0"

const marketplaceRegistryDir = "data/plugins"

// marketplacePlugin implements sdk.PluginProvider, sdk.TypedStepProvider, and sdk.ContractProvider.
type marketplacePlugin struct{}

// NewMarketplacePlugin returns a new marketplacePlugin instance.
func NewMarketplacePlugin() sdk.PluginProvider {
	return &marketplacePlugin{}
}

// Manifest returns plugin metadata.
func (p *marketplacePlugin) Manifest() sdk.PluginManifest {
	return sdk.PluginManifest{
		Name:        "workflow-plugin-marketplace",
		Version:     Version,
		Author:      "GoCodeAlone",
		Description: "Marketplace steps for searching, installing, and managing workflow plugins",
	}
}

// ModuleTypes returns the module type names this plugin provides.
func (p *marketplacePlugin) ModuleTypes() []string {
	return []string{}
}

// CreateModule creates a module instance of the given type.
func (p *marketplacePlugin) CreateModule(typeName, name string, _ map[string]any) (sdk.ModuleInstance, error) {
	return nil, fmt.Errorf("marketplace plugin: unknown module type %q", typeName)
}

// StepTypes returns the step type names this plugin provides.
func (p *marketplacePlugin) StepTypes() []string {
	return append([]string(nil), marketplaceStepTypes...)
}

var marketplaceStepTypes = []string{
	"step.marketplace_search",
	"step.marketplace_detail",
	"step.marketplace_install",
	"step.marketplace_installed",
	"step.marketplace_uninstall",
	"step.marketplace_update",
}

// CreateStep creates a step instance of the given type.
func (p *marketplacePlugin) CreateStep(typeName, name string, config map[string]any) (sdk.StepInstance, error) {
	registry := newLocalRegistry(marketplaceRegistryDir)
	switch typeName {
	case "step.marketplace_search":
		return &marketplaceSearchStep{name: name, config: config, registry: registry}, nil
	case "step.marketplace_detail":
		return &marketplaceDetailStep{name: name, config: config, registry: registry}, nil
	case "step.marketplace_install":
		return &marketplaceInstallStep{name: name, config: config, registry: registry}, nil
	case "step.marketplace_installed":
		return &marketplaceInstalledStep{name: name, config: config, registry: registry}, nil
	case "step.marketplace_uninstall":
		return &marketplaceUninstallStep{name: name, config: config, registry: registry}, nil
	case "step.marketplace_update":
		return &marketplaceUpdateStep{name: name, config: config, registry: registry}, nil
	default:
		return nil, fmt.Errorf("marketplace plugin: unknown step type %q", typeName)
	}
}

// TypedStepTypes returns the typed step type names this plugin provides.
func (p *marketplacePlugin) TypedStepTypes() []string {
	return append([]string(nil), marketplaceStepTypes...)
}

// CreateTypedStep creates a typed step instance of the given type.
func (p *marketplacePlugin) CreateTypedStep(typeName, name string, config *anypb.Any) (sdk.StepInstance, error) {
	registry := newLocalRegistry(marketplaceRegistryDir)
	switch typeName {
	case "step.marketplace_search":
		factory := sdk.NewTypedStepFactory(typeName, &contracts.MarketplaceSearchConfig{}, &contracts.MarketplaceSearchInput{}, typedMarketplaceSearch(registry))
		return factory.CreateTypedStep(typeName, name, config)
	case "step.marketplace_detail":
		factory := sdk.NewTypedStepFactory(typeName, &contracts.MarketplaceDetailConfig{}, &contracts.MarketplaceDetailInput{}, typedMarketplaceDetail(registry))
		return factory.CreateTypedStep(typeName, name, config)
	case "step.marketplace_install":
		factory := sdk.NewTypedStepFactory(typeName, &contracts.MarketplaceInstallConfig{}, &contracts.MarketplaceInstallInput{}, typedMarketplaceInstall(registry))
		return factory.CreateTypedStep(typeName, name, config)
	case "step.marketplace_installed":
		factory := sdk.NewTypedStepFactory(typeName, &contracts.MarketplaceInstalledConfig{}, &contracts.MarketplaceInstalledInput{}, typedMarketplaceInstalled(registry))
		return factory.CreateTypedStep(typeName, name, config)
	case "step.marketplace_uninstall":
		factory := sdk.NewTypedStepFactory(typeName, &contracts.MarketplaceUninstallConfig{}, &contracts.MarketplaceUninstallInput{}, typedMarketplaceUninstall(registry))
		return factory.CreateTypedStep(typeName, name, config)
	case "step.marketplace_update":
		factory := sdk.NewTypedStepFactory(typeName, &contracts.MarketplaceUpdateConfig{}, &contracts.MarketplaceUpdateInput{}, typedMarketplaceUpdate(registry))
		return factory.CreateTypedStep(typeName, name, config)
	default:
		return nil, fmt.Errorf("marketplace plugin: unknown step type %q", typeName)
	}
}

// ContractRegistry returns strict protobuf descriptors for plugin step boundaries.
func (p *marketplacePlugin) ContractRegistry() *pb.ContractRegistry {
	const pkg = "workflow.plugins.marketplace.v1."
	return &pb.ContractRegistry{
		FileDescriptorSet: &descriptorpb.FileDescriptorSet{
			File: []*descriptorpb.FileDescriptorProto{
				protodesc.ToFileDescriptorProto(contracts.File_internal_contracts_marketplace_proto),
			},
		},
		Contracts: []*pb.ContractDescriptor{
			stepContract("step.marketplace_search", pkg+"MarketplaceSearchConfig", pkg+"MarketplaceSearchInput", pkg+"MarketplaceSearchOutput"),
			stepContract("step.marketplace_detail", pkg+"MarketplaceDetailConfig", pkg+"MarketplaceDetailInput", pkg+"MarketplaceDetailOutput"),
			stepContract("step.marketplace_install", pkg+"MarketplaceInstallConfig", pkg+"MarketplaceInstallInput", pkg+"MarketplaceInstallOutput"),
			stepContract("step.marketplace_installed", pkg+"MarketplaceInstalledConfig", pkg+"MarketplaceInstalledInput", pkg+"MarketplaceInstalledOutput"),
			stepContract("step.marketplace_uninstall", pkg+"MarketplaceUninstallConfig", pkg+"MarketplaceUninstallInput", pkg+"MarketplaceUninstallOutput"),
			stepContract("step.marketplace_update", pkg+"MarketplaceUpdateConfig", pkg+"MarketplaceUpdateInput", pkg+"MarketplaceUpdateOutput"),
		},
	}
}

func stepContract(stepType, configMessage, inputMessage, outputMessage string) *pb.ContractDescriptor {
	return &pb.ContractDescriptor{
		Kind:          pb.ContractKind_CONTRACT_KIND_STEP,
		StepType:      stepType,
		ConfigMessage: configMessage,
		InputMessage:  inputMessage,
		OutputMessage: outputMessage,
		Mode:          pb.ContractMode_CONTRACT_MODE_STRICT_PROTO,
	}
}

// ─── Registry ─────────────────────────────────────────────────────────────────

type marketplaceEntry struct {
	Name        string
	Version     string
	Description string
	Author      string
	Category    string
	Tags        []string
	Downloads   int
	Rating      float64
	Installed   bool
	InstalledAt string
}

type localRegistry struct {
	baseDir string
	catalog []marketplaceEntry
}

func newLocalRegistry(baseDir string) *localRegistry {
	return &localRegistry{baseDir: baseDir, catalog: defaultCatalog()}
}

func (r *localRegistry) search(query, category string, tags []string) []marketplaceEntry {
	var results []marketplaceEntry
	installed := r.installedSet()
	for i := range r.catalog {
		if query != "" &&
			!strings.Contains(strings.ToLower(r.catalog[i].Name), strings.ToLower(query)) &&
			!strings.Contains(strings.ToLower(r.catalog[i].Description), strings.ToLower(query)) {
			continue
		}
		if category != "" && r.catalog[i].Category != category {
			continue
		}
		if len(tags) > 0 && !entryHasTags(r.catalog[i], tags) {
			continue
		}
		entry := r.catalog[i]
		entry.Installed = installed[entry.Name]
		results = append(results, entry)
	}
	return results
}

func entryHasTags(entry marketplaceEntry, tags []string) bool {
	if len(tags) == 0 {
		return true
	}
	entryTags := make(map[string]bool, len(entry.Tags))
	for _, tag := range entry.Tags {
		entryTags[strings.ToLower(tag)] = true
	}
	for _, tag := range tags {
		if !entryTags[strings.ToLower(tag)] {
			return false
		}
	}
	return true
}

func (r *localRegistry) detail(name string) (*marketplaceEntry, error) {
	installed := r.installedSet()
	for i := range r.catalog {
		if r.catalog[i].Name == name {
			entry := r.catalog[i]
			entry.Installed = installed[name]
			return &entry, nil
		}
	}
	return nil, fmt.Errorf("plugin %q not found", name)
}

func (r *localRegistry) install(name string) error {
	if _, err := r.detail(name); err != nil {
		return err
	}
	dir := filepath.Join(r.baseDir, name)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, ".installed"), []byte(time.Now().UTC().Format(time.RFC3339)), 0o600)
}

func (r *localRegistry) uninstall(name string) error {
	if !r.installedSet()[name] {
		return fmt.Errorf("plugin %q is not installed", name)
	}
	return os.RemoveAll(filepath.Join(r.baseDir, name))
}

func (r *localRegistry) listInstalled() []marketplaceEntry {
	installed := r.installedSet()
	var result []marketplaceEntry
	for i := range r.catalog {
		if installed[r.catalog[i].Name] {
			entry := r.catalog[i]
			entry.Installed = true
			result = append(result, entry)
		}
	}
	return result
}

func (r *localRegistry) installedSet() map[string]bool {
	set := make(map[string]bool)
	entries, err := os.ReadDir(r.baseDir)
	if err != nil {
		return set
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(r.baseDir, e.Name(), ".installed")); err == nil {
			set[e.Name()] = true
		}
	}
	return set
}

func defaultCatalog() []marketplaceEntry {
	return []marketplaceEntry{
		{Name: "auth-oidc", Version: "1.2.0", Description: "OpenID Connect authentication provider", Author: "GoCodeAlone", Category: "auth", Tags: []string{"auth", "oidc", "sso"}, Downloads: 4200, Rating: 4.8},
		{Name: "storage-s3", Version: "2.0.1", Description: "AWS S3 blob storage backend", Author: "GoCodeAlone", Category: "storage", Tags: []string{"storage", "aws", "s3"}, Downloads: 8900, Rating: 4.9},
		{Name: "messaging-kafka", Version: "1.0.3", Description: "Apache Kafka messaging integration", Author: "GoCodeAlone", Category: "messaging", Tags: []string{"messaging", "kafka"}, Downloads: 3100, Rating: 4.6},
		{Name: "observability-otel", Version: "0.9.0", Description: "OpenTelemetry tracing and metrics", Author: "GoCodeAlone", Category: "observability", Tags: []string{"otel", "tracing"}, Downloads: 2700, Rating: 4.5},
		{Name: "cicd-github-actions", Version: "1.1.0", Description: "GitHub Actions CI/CD integration", Author: "GoCodeAlone", Category: "cicd", Tags: []string{"cicd", "github"}, Downloads: 1850, Rating: 4.4},
		{Name: "ai-openai", Version: "0.5.0", Description: "OpenAI GPT integration", Author: "GoCodeAlone", Category: "ai", Tags: []string{"ai", "openai", "llm"}, Downloads: 5600, Rating: 4.7},
	}
}

// ─── Steps ───────────────────────────────────────────────────────────────────

type marketplaceSearchStep struct {
	name     string
	config   map[string]any
	registry *localRegistry
}

func (s *marketplaceSearchStep) Execute(_ context.Context, _ map[string]any, _ map[string]map[string]any, _ map[string]any, _ map[string]any, _ map[string]any) (*sdk.StepResult, error) {
	query, _ := s.config["query"].(string)
	category, _ := s.config["category"].(string)
	results := s.registry.search(query, category, nil)
	items := make([]map[string]any, len(results))
	for i, r := range results {
		items[i] = map[string]any{"name": r.Name, "version": r.Version, "description": r.Description, "category": r.Category, "installed": r.Installed}
	}
	return &sdk.StepResult{Output: map[string]any{"results": items, "count": len(items)}}, nil
}

type marketplaceDetailStep struct {
	name     string
	config   map[string]any
	registry *localRegistry
}

func (s *marketplaceDetailStep) Execute(_ context.Context, _ map[string]any, _ map[string]map[string]any, _ map[string]any, _ map[string]any, _ map[string]any) (*sdk.StepResult, error) {
	pluginName, _ := s.config["name"].(string)
	if pluginName == "" {
		return nil, fmt.Errorf("step.marketplace_detail %q: 'name' is required", s.name)
	}
	entry, err := s.registry.detail(pluginName)
	if err != nil {
		return nil, err
	}
	return &sdk.StepResult{Output: map[string]any{"name": entry.Name, "version": entry.Version, "description": entry.Description, "author": entry.Author, "installed": entry.Installed}}, nil
}

type marketplaceInstallStep struct {
	name     string
	config   map[string]any
	registry *localRegistry
}

func (s *marketplaceInstallStep) Execute(_ context.Context, _ map[string]any, _ map[string]map[string]any, _ map[string]any, _ map[string]any, _ map[string]any) (*sdk.StepResult, error) {
	pluginName, _ := s.config["name"].(string)
	if pluginName == "" {
		return nil, fmt.Errorf("step.marketplace_install %q: 'name' is required", s.name)
	}
	if err := s.registry.install(pluginName); err != nil {
		return nil, err
	}
	return &sdk.StepResult{Output: map[string]any{"installed": true, "name": pluginName}}, nil
}

type marketplaceInstalledStep struct {
	name     string
	config   map[string]any
	registry *localRegistry
}

func (s *marketplaceInstalledStep) Execute(_ context.Context, _ map[string]any, _ map[string]map[string]any, _ map[string]any, _ map[string]any, _ map[string]any) (*sdk.StepResult, error) {
	items := s.registry.listInstalled()
	result := make([]map[string]any, len(items))
	for i, item := range items {
		result[i] = map[string]any{"name": item.Name, "version": item.Version, "installed_at": item.InstalledAt}
	}
	return &sdk.StepResult{Output: map[string]any{"plugins": result, "count": len(result)}}, nil
}

type marketplaceUninstallStep struct {
	name     string
	config   map[string]any
	registry *localRegistry
}

func (s *marketplaceUninstallStep) Execute(_ context.Context, _ map[string]any, _ map[string]map[string]any, _ map[string]any, _ map[string]any, _ map[string]any) (*sdk.StepResult, error) {
	pluginName, _ := s.config["name"].(string)
	if pluginName == "" {
		return nil, fmt.Errorf("step.marketplace_uninstall %q: 'name' is required", s.name)
	}
	if err := s.registry.uninstall(pluginName); err != nil {
		return nil, err
	}
	return &sdk.StepResult{Output: map[string]any{"uninstalled": true, "name": pluginName}}, nil
}

type marketplaceUpdateStep struct {
	name     string
	config   map[string]any
	registry *localRegistry
}

func (s *marketplaceUpdateStep) Execute(_ context.Context, _ map[string]any, _ map[string]map[string]any, _ map[string]any, _ map[string]any, _ map[string]any) (*sdk.StepResult, error) {
	pluginName, _ := s.config["name"].(string)
	if pluginName == "" {
		return nil, fmt.Errorf("step.marketplace_update %q: 'name' is required", s.name)
	}
	if err := s.registry.install(pluginName); err != nil {
		return nil, fmt.Errorf("update failed: %w", err)
	}
	return &sdk.StepResult{Output: map[string]any{"updated": true, "name": pluginName}}, nil
}
