package internal

import (
	"context"
	"fmt"

	"github.com/GoCodeAlone/workflow-plugin-marketplace/internal/contracts"
	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

func typedMarketplaceSearch(registry *localRegistry) sdk.TypedStepHandler[*contracts.MarketplaceSearchConfig, *contracts.MarketplaceSearchInput, *contracts.MarketplaceSearchOutput] {
	return func(_ context.Context, req sdk.TypedStepRequest[*contracts.MarketplaceSearchConfig, *contracts.MarketplaceSearchInput]) (*sdk.TypedStepResult[*contracts.MarketplaceSearchOutput], error) {
		query := firstNonEmpty(req.Input.GetQuery(), req.Config.GetQuery())
		category := firstNonEmpty(req.Input.GetCategory(), req.Config.GetCategory())
		tags := req.Config.GetTags()
		if len(req.Input.GetTags()) > 0 {
			tags = req.Input.GetTags()
		}
		results := registry.search(query, category, tags)
		output := &contracts.MarketplaceSearchOutput{Count: int32(len(results))}
		for i := range results {
			output.Results = append(output.Results, marketplaceEntryToProto(results[i]))
		}
		return &sdk.TypedStepResult[*contracts.MarketplaceSearchOutput]{Output: output}, nil
	}
}

func typedMarketplaceDetail(registry *localRegistry) sdk.TypedStepHandler[*contracts.MarketplaceDetailConfig, *contracts.MarketplaceDetailInput, *contracts.MarketplaceDetailOutput] {
	return func(_ context.Context, req sdk.TypedStepRequest[*contracts.MarketplaceDetailConfig, *contracts.MarketplaceDetailInput]) (*sdk.TypedStepResult[*contracts.MarketplaceDetailOutput], error) {
		name := firstNonEmpty(req.Input.GetName(), req.Config.GetName())
		if name == "" {
			return nil, fmt.Errorf("step.marketplace_detail: 'name' is required")
		}
		entry, err := registry.detail(name)
		if err != nil {
			return nil, err
		}
		return &sdk.TypedStepResult[*contracts.MarketplaceDetailOutput]{
			Output: &contracts.MarketplaceDetailOutput{Plugin: marketplaceEntryToProto(*entry)},
		}, nil
	}
}

func typedMarketplaceInstall(registry *localRegistry) sdk.TypedStepHandler[*contracts.MarketplaceInstallConfig, *contracts.MarketplaceInstallInput, *contracts.MarketplaceInstallOutput] {
	return func(_ context.Context, req sdk.TypedStepRequest[*contracts.MarketplaceInstallConfig, *contracts.MarketplaceInstallInput]) (*sdk.TypedStepResult[*contracts.MarketplaceInstallOutput], error) {
		name := firstNonEmpty(req.Input.GetName(), req.Config.GetName())
		if name == "" {
			return nil, fmt.Errorf("step.marketplace_install: 'name' is required")
		}
		if err := registry.install(name); err != nil {
			return nil, err
		}
		return &sdk.TypedStepResult[*contracts.MarketplaceInstallOutput]{
			Output: &contracts.MarketplaceInstallOutput{Installed: true, Name: name},
		}, nil
	}
}

func typedMarketplaceInstalled(registry *localRegistry) sdk.TypedStepHandler[*contracts.MarketplaceInstalledConfig, *contracts.MarketplaceInstalledInput, *contracts.MarketplaceInstalledOutput] {
	return func(context.Context, sdk.TypedStepRequest[*contracts.MarketplaceInstalledConfig, *contracts.MarketplaceInstalledInput]) (*sdk.TypedStepResult[*contracts.MarketplaceInstalledOutput], error) {
		items := registry.listInstalled()
		output := &contracts.MarketplaceInstalledOutput{Count: int32(len(items))}
		for i := range items {
			output.Plugins = append(output.Plugins, marketplaceEntryToProto(items[i]))
		}
		return &sdk.TypedStepResult[*contracts.MarketplaceInstalledOutput]{Output: output}, nil
	}
}

func typedMarketplaceUninstall(registry *localRegistry) sdk.TypedStepHandler[*contracts.MarketplaceUninstallConfig, *contracts.MarketplaceUninstallInput, *contracts.MarketplaceUninstallOutput] {
	return func(_ context.Context, req sdk.TypedStepRequest[*contracts.MarketplaceUninstallConfig, *contracts.MarketplaceUninstallInput]) (*sdk.TypedStepResult[*contracts.MarketplaceUninstallOutput], error) {
		name := firstNonEmpty(req.Input.GetName(), req.Config.GetName())
		if name == "" {
			return nil, fmt.Errorf("step.marketplace_uninstall: 'name' is required")
		}
		if err := registry.uninstall(name); err != nil {
			return nil, err
		}
		return &sdk.TypedStepResult[*contracts.MarketplaceUninstallOutput]{
			Output: &contracts.MarketplaceUninstallOutput{Uninstalled: true, Name: name},
		}, nil
	}
}

func typedMarketplaceUpdate(registry *localRegistry) sdk.TypedStepHandler[*contracts.MarketplaceUpdateConfig, *contracts.MarketplaceUpdateInput, *contracts.MarketplaceUpdateOutput] {
	return func(_ context.Context, req sdk.TypedStepRequest[*contracts.MarketplaceUpdateConfig, *contracts.MarketplaceUpdateInput]) (*sdk.TypedStepResult[*contracts.MarketplaceUpdateOutput], error) {
		name := firstNonEmpty(req.Input.GetName(), req.Config.GetName())
		if name == "" {
			return nil, fmt.Errorf("step.marketplace_update: 'name' is required")
		}
		if err := registry.install(name); err != nil {
			return nil, fmt.Errorf("update failed: %w", err)
		}
		return &sdk.TypedStepResult[*contracts.MarketplaceUpdateOutput]{
			Output: &contracts.MarketplaceUpdateOutput{Updated: true, Name: name},
		}, nil
	}
}

func marketplaceEntryToProto(entry marketplaceEntry) *contracts.MarketplacePlugin {
	return &contracts.MarketplacePlugin{
		Name:        entry.Name,
		Version:     entry.Version,
		Description: entry.Description,
		Author:      entry.Author,
		Category:    entry.Category,
		Tags:        entry.Tags,
		Downloads:   int32(entry.Downloads),
		Rating:      entry.Rating,
		Installed:   entry.Installed,
		InstalledAt: entry.InstalledAt,
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
