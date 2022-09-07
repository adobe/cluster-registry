package request

import (
	"github.com/labstack/echo/v4"

	h "github.com/adobe/cluster-registry/test/slt/helpers"
)

var logger echo.Logger

// GetClusterConfig sets the parameters for the GET cluster
type GetClusterConfig struct {
	url         string
	clusterName string
}

// GetAllClusterConfig sets the parameters for the GET cluster
type GetAllClusterConfig struct {
	url          string
	pageNr       string
	perPageLimit string
}

// SetLogger sets the global logger for the slt package
func SetLogger(lgr echo.Logger) {
	logger = lgr
}

// GetClusterConfigFromEnv gets from the env the needed global env
func GetClusterConfigFromEnv() GetClusterConfig {
	return GetClusterConfig{
		url:         h.GetEnv("URL", "http://localhost:8080", logger),
		clusterName: h.GetEnv("CLUSTER_NAME", "", logger),
	}
}

// GetAllClusterConfigFromEnv gets from the env the needed global env
func GetAllClusterConfigFromEnv() GetAllClusterConfig {
	return GetAllClusterConfig{
		url:          h.GetEnv("URL", "http://localhost:8080", logger),
		pageNr:       h.GetEnv("PAGE_NR", "0", logger),
		perPageLimit: h.GetEnv("PER_PAGE_LIMIT", "200", logger),
	}
}

// RunGetCluster as
func RunGetCluster(config GetClusterConfig, jwtToken string) error {
	_, err := h.GetCluster(config.url, config.clusterName, jwtToken)
	if err != nil {
		return err
	}
	return nil
}

// RunGetAllClusters as
func RunGetAllClusters(config GetAllClusterConfig, jwtToken string) error {
	_, err := h.GetClusters(config.url, config.perPageLimit, config.pageNr, jwtToken)
	if err != nil {
		return err
	}
	return nil
}
