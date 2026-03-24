package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"geodatfilter/internal/filterdat"
)

func main() {
	cfg, err := parseFlags()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	if err := filterdat.Run(context.Background(), cfg, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func parseFlags() (filterdat.Config, error) {
	var cfg filterdat.Config

	flag.StringVar(&cfg.CategoryFile, "category-file", "category-for-save.txt", "Path to category-for-save.txt")
	flag.StringVar(&cfg.GeoSiteInput, "geosite-input", "", "Path or http/https URL for geosite .dat")
	flag.StringVar(&cfg.GeoSiteOutput, "geosite-output", "", "Output path for filtered geosite .dat")
	flag.StringVar(&cfg.GeoIPInput, "geoip-input", "", "Path or http/https URL for geoip .dat")
	flag.StringVar(&cfg.GeoIPOutput, "geoip-output", "", "Output path for filtered geoip .dat")
	flag.Parse()

	if cfg.GeoSiteInput == "" && cfg.GeoIPInput == "" {
		return filterdat.Config{}, fmt.Errorf("at least one of --geosite-input or --geoip-input must be provided")
	}
	if cfg.GeoSiteInput != "" && cfg.GeoSiteOutput == "" {
		return filterdat.Config{}, fmt.Errorf("--geosite-output is required when --geosite-input is set")
	}
	if cfg.GeoIPInput != "" && cfg.GeoIPOutput == "" {
		return filterdat.Config{}, fmt.Errorf("--geoip-output is required when --geoip-input is set")
	}

	return cfg, nil
}
