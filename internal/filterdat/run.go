package filterdat

import (
	"context"
	"fmt"
	"io"
	"os"
)

type Config struct {
	CategoryFile  string
	GeoSiteInput  string
	GeoSiteOutput string
	GeoIPInput    string
	GeoIPOutput   string
}

func Run(ctx context.Context, cfg Config, stdout, stderr io.Writer) error {
	cats, err := LoadCategories(cfg.CategoryFile)
	if err != nil {
		return err
	}

	if cfg.GeoSiteInput != "" {
		stats, err := processGeoSite(ctx, cfg.GeoSiteInput, cfg.GeoSiteOutput, cats.GeoSite)
		if err != nil {
			return err
		}
		reportStats(stdout, stderr, stats)
	}

	if cfg.GeoIPInput != "" {
		stats, err := processGeoIP(ctx, cfg.GeoIPInput, cfg.GeoIPOutput, cats.GeoIP)
		if err != nil {
			return err
		}
		reportStats(stdout, stderr, stats)
	}

	return nil
}

func processGeoSite(ctx context.Context, input, output string, wanted []string) (FilterStats, error) {
	data, err := LoadSource(ctx, input)
	if err != nil {
		return FilterStats{}, err
	}

	filtered, stats, err := FilterGeoSite(data, wanted)
	if err != nil {
		return FilterStats{}, err
	}

	if err := os.WriteFile(output, filtered, 0o644); err != nil {
		return FilterStats{}, fmt.Errorf("write geosite output %q: %w", output, err)
	}

	return stats, nil
}

func processGeoIP(ctx context.Context, input, output string, wanted []string) (FilterStats, error) {
	data, err := LoadSource(ctx, input)
	if err != nil {
		return FilterStats{}, err
	}

	filtered, stats, err := FilterGeoIP(data, wanted)
	if err != nil {
		return FilterStats{}, err
	}

	if err := os.WriteFile(output, filtered, 0o644); err != nil {
		return FilterStats{}, fmt.Errorf("write geoip output %q: %w", output, err)
	}

	return stats, nil
}

func reportStats(stdout, stderr io.Writer, stats FilterStats) {
	for _, missing := range stats.Missing {
		fmt.Fprintf(stderr, "warning: %s category %q not found in input\n", stats.Type, missing)
	}

	fmt.Fprintf(stdout, "%s: kept %d of %d entries\n", stats.Type, stats.Kept, stats.Total)
}
