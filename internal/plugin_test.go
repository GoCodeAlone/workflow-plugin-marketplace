package internal

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/GoCodeAlone/workflow-plugin-marketplace/internal/contracts"
	pb "github.com/GoCodeAlone/workflow/plugin/external/proto"
	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestMarketplacePluginTypedContracts(t *testing.T) {
	provider := NewMarketplacePlugin()
	typedProvider, ok := provider.(sdk.TypedStepProvider)
	if !ok {
		t.Fatal("expected typed step provider")
	}
	stepProvider, ok := provider.(sdk.StepProvider)
	if !ok {
		t.Fatal("expected step provider")
	}
	contractProvider, ok := provider.(sdk.ContractProvider)
	if !ok {
		t.Fatal("expected contract provider")
	}

	wantTypes := []string{
		"step.marketplace_search",
		"step.marketplace_detail",
		"step.marketplace_install",
		"step.marketplace_installed",
		"step.marketplace_uninstall",
		"step.marketplace_update",
	}
	assertStringSet(t, typedProvider.TypedStepTypes(), wantTypes)
	assertStepTypeSlicesAreDefensive(t, stepProvider, typedProvider)

	registry := contractProvider.ContractRegistry()
	if registry == nil {
		t.Fatal("expected contract registry")
	}
	if registry.FileDescriptorSet == nil || len(registry.FileDescriptorSet.File) == 0 {
		t.Fatal("expected file descriptor set")
	}
	if len(registry.Contracts) != len(wantTypes) {
		t.Fatalf("contract count = %d, want %d", len(registry.Contracts), len(wantTypes))
	}

	files, err := protodesc.NewFiles(registry.FileDescriptorSet)
	if err != nil {
		t.Fatalf("descriptor set: %v", err)
	}
	manifestContracts := loadManifestContracts(t)
	got := make([]string, 0, len(registry.Contracts))
	for _, descriptor := range registry.Contracts {
		if descriptor.Kind != pb.ContractKind_CONTRACT_KIND_STEP {
			t.Fatalf("%s kind = %s, want step", descriptor.StepType, descriptor.Kind)
		}
		if descriptor.Mode != pb.ContractMode_CONTRACT_MODE_STRICT_PROTO {
			t.Fatalf("%s mode = %s, want strict proto", descriptor.StepType, descriptor.Mode)
		}
		if descriptor.ConfigMessage == "" || descriptor.InputMessage == "" || descriptor.OutputMessage == "" {
			t.Fatalf("%s has incomplete message contract: %#v", descriptor.StepType, descriptor)
		}
		for _, name := range []string{descriptor.ConfigMessage, descriptor.InputMessage, descriptor.OutputMessage} {
			if _, err := files.FindDescriptorByName(protoreflect.FullName(name)); err != nil {
				t.Fatalf("%s references unknown message %s: %v", descriptor.StepType, name, err)
			}
		}
		if want, ok := manifestContracts[descriptor.StepType]; !ok {
			t.Fatalf("%s missing from plugin.contracts.json", descriptor.StepType)
		} else if want.ConfigMessage != descriptor.ConfigMessage || want.InputMessage != descriptor.InputMessage || want.OutputMessage != descriptor.OutputMessage {
			t.Fatalf("%s manifest contract = %#v, runtime = %#v", descriptor.StepType, want, descriptor)
		}
		got = append(got, descriptor.StepType)
	}
	assertStringSet(t, got, wantTypes)
	if len(manifestContracts) != len(registry.Contracts) {
		t.Fatalf("plugin.contracts.json contract count = %d, runtime = %d", len(manifestContracts), len(registry.Contracts))
	}
}

func TestTypedMarketplaceSearchProviderValidatesTypedConfig(t *testing.T) {
	provider := NewMarketplacePlugin().(sdk.TypedStepProvider)
	config, err := anypb.New(&contracts.MarketplaceSearchConfig{Category: "storage"})
	if err != nil {
		t.Fatalf("pack config: %v", err)
	}
	step, err := provider.CreateTypedStep("step.marketplace_search", "search", config)
	if err != nil {
		t.Fatalf("CreateTypedStep: %v", err)
	}
	if _, err := step.Execute(context.Background(), nil, nil, nil, nil, nil); err == nil {
		t.Fatal("legacy Execute succeeded for typed-only step")
	}

	wrongConfig, err := anypb.New(&contracts.MarketplaceInstallConfig{Name: "storage-s3"})
	if err != nil {
		t.Fatalf("pack wrong config: %v", err)
	}
	if _, err := provider.CreateTypedStep("step.marketplace_search", "search", wrongConfig); err == nil {
		t.Fatal("CreateTypedStep accepted wrong typed config")
	}
}

func TestTypedMarketplaceSearchMergesConfigAndInput(t *testing.T) {
	registry := newLocalRegistry(t.TempDir())
	result, err := typedMarketplaceSearch(registry)(context.Background(), sdk.TypedStepRequest[*contracts.MarketplaceSearchConfig, *contracts.MarketplaceSearchInput]{
		Config: &contracts.MarketplaceSearchConfig{Category: "storage"},
		Input:  &contracts.MarketplaceSearchInput{Query: "s3"},
	})
	if err != nil {
		t.Fatalf("typedMarketplaceSearch: %v", err)
	}
	if result == nil || result.Output == nil {
		t.Fatal("expected search output")
	}
	if result.Output.Count != 1 {
		t.Fatalf("count = %d, want 1", result.Output.Count)
	}
	if got := result.Output.Results[0].Name; got != "storage-s3" {
		t.Fatalf("first result = %q, want storage-s3", got)
	}
}

type manifestContract struct {
	Mode          string `json:"mode"`
	ConfigMessage string `json:"config"`
	InputMessage  string `json:"input"`
	OutputMessage string `json:"output"`
}

func loadManifestContracts(t *testing.T) map[string]manifestContract {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(file), "..", "plugin.contracts.json"))
	if err != nil {
		t.Fatalf("read plugin.contracts.json: %v", err)
	}
	var manifest struct {
		Version   string `json:"version"`
		Contracts []struct {
			Kind string `json:"kind"`
			Type string `json:"type"`
			manifestContract
		} `json:"contracts"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse plugin.contracts.json: %v", err)
	}
	if manifest.Version != "v1" {
		t.Fatalf("plugin.contracts.json version = %q, want v1", manifest.Version)
	}
	contracts := make(map[string]manifestContract, len(manifest.Contracts))
	for _, contract := range manifest.Contracts {
		if contract.Kind == "step" {
			if contract.Mode != "strict" {
				t.Fatalf("%s mode = %q, want strict", contract.Type, contract.Mode)
			}
			if _, exists := contracts[contract.Type]; exists {
				t.Fatalf("duplicate step contract %q in plugin.contracts.json", contract.Type)
			}
			contracts[contract.Type] = contract.manifestContract
			continue
		}
		t.Fatalf("unexpected contract kind %q in plugin.contracts.json", contract.Kind)
	}
	return contracts
}

func assertStepTypeSlicesAreDefensive(t *testing.T, provider sdk.StepProvider, typedProvider sdk.TypedStepProvider) {
	t.Helper()
	stepTypes := provider.StepTypes()
	stepTypes[0] = "mutated"
	if got := provider.StepTypes()[0]; got == "mutated" {
		t.Fatal("StepTypes exposed mutable package-level slice")
	}

	typedStepTypes := typedProvider.TypedStepTypes()
	typedStepTypes[0] = "mutated"
	if got := typedProvider.TypedStepTypes()[0]; got == "mutated" {
		t.Fatal("TypedStepTypes exposed mutable package-level slice")
	}
}

func assertStringSet(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d: got %v", len(got), len(want), got)
	}
	seen := make(map[string]bool, len(got))
	for _, item := range got {
		seen[item] = true
	}
	for _, item := range want {
		if !seen[item] {
			t.Fatalf("missing %q in %v", item, got)
		}
	}
}
