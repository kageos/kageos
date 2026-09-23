package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/access"
)

type capabilityAuditContextKey struct{}

func (s *serviceTreeCapabilityBundleService) withCapabilityAudit(ctx context.Context, opts *dto.InstallCapabilityOptions, source string, operation func(context.Context) (*dto.InstallCapabilityBundleResp, error)) (resp *dto.InstallCapabilityBundleResp, resultErr error) {
	if ctx.Value(capabilityAuditContextKey{}) != nil {
		return operation(ctx)
	}
	if opts == nil {
		return nil, fmt.Errorf("install options required")
	}
	path := access.NormalizeResourcePath(opts.TargetDirectoryPath)
	audit, err := beginManagementAudit(ctx, s.appRepo.GetDB(), "directory.installed", "directory", path, path, "", nil, map[string]any{
		"source": source, "overwrite": opts.Overwrite, "force_diff": opts.ForceDiff, "bundle_subpath": opts.BundleSubpath,
	})
	if err != nil {
		return nil, err
	}
	ctx = context.WithValue(ctx, capabilityAuditContextKey{}, audit)
	defer func() {
		var after any
		if resp != nil {
			after = resp
			audit.details["warnings"] = resp.Warnings
		}
		if resultErr == nil {
			audit.details["stage"] = "completed"
		}
		audit.finish(ctx, after, &resultErr)
	}()
	defer audit.capturePanic(&resultErr)
	return operation(ctx)
}

func (s *serviceTreeCapabilityBundleService) InstallCapabilityBundle(ctx context.Context, opts *dto.InstallCapabilityOptions, bundle *dto.CapabilityBundle) (*dto.InstallCapabilityBundleResp, error) {
	return s.withCapabilityAudit(ctx, opts, "inline", func(ctx context.Context) (*dto.InstallCapabilityBundleResp, error) {
		return s.installCapabilityBundle(ctx, opts, bundle)
	})
}

func (s *serviceTreeCapabilityBundleService) InstallCapabilityBundleFromFile(ctx context.Context, opts *dto.InstallCapabilityOptions, filePath string) (*dto.InstallCapabilityBundleResp, error) {
	return s.withCapabilityAudit(ctx, opts, "uploaded_file", func(ctx context.Context) (*dto.InstallCapabilityBundleResp, error) {
		return s.installCapabilityBundleFromFile(ctx, opts, filePath)
	})
}

func (s *serviceTreeCapabilityBundleService) InstallCapabilityBundleFromURL(ctx context.Context, opts *dto.InstallCapabilityOptions, bundleURL, installKey string) (*dto.InstallCapabilityBundleResp, error) {
	source := "remote"
	if u, err := url.Parse(bundleURL); err == nil && u.Hostname() != "" {
		source = u.Scheme + "://" + u.Hostname()
	}
	return s.withCapabilityAudit(ctx, opts, source, func(ctx context.Context) (*dto.InstallCapabilityBundleResp, error) {
		if audit, ok := ctx.Value(capabilityAuditContextKey{}).(*managementAudit); ok {
			audit.secrets = []string{bundleURL, installKey}
		}
		return s.installCapabilityBundleFromURL(ctx, opts, bundleURL, installKey)
	})
}

func recordCapabilityBundleIdentity(ctx context.Context, bundle *dto.CapabilityBundle) {
	if bundle == nil {
		return
	}
	raw, err := json.Marshal(bundle)
	if err != nil {
		return
	}
	capabilityAuditDetail(ctx, "bundle_digest", fmt.Sprintf("%x", sha256.Sum256(raw)))
	capabilityAuditDetail(ctx, "bundle_name", bundle.Name)
	if bundle.Metadata != nil && bundle.Metadata.Directory != nil {
		capabilityAuditDetail(ctx, "release_version", bundle.Metadata.Directory.ReleaseVersion)
		capabilityAuditDetail(ctx, "source_revision", bundle.Metadata.Directory.SourceRevision)
	}
}

func capabilityAuditDetail(ctx context.Context, key string, value any) {
	if audit, ok := ctx.Value(capabilityAuditContextKey{}).(*managementAudit); ok {
		audit.details[key] = value
	}
}

func capabilityAuditStage(ctx context.Context, stage string) error {
	if audit, ok := ctx.Value(capabilityAuditContextKey{}).(*managementAudit); ok {
		return audit.stage(ctx, stage)
	}
	return nil
}
