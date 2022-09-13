package checks

import (
	"sync"
	"time"

	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/labstack/echo/v4"

	"github.com/adobe/cluster-registry/pkg/config"
	"github.com/adobe/cluster-registry/test/jwt"
	"github.com/adobe/cluster-registry/test/slt/checks/request"
	"github.com/adobe/cluster-registry/test/slt/checks/update"
	h "github.com/adobe/cluster-registry/test/slt/helpers"
	"github.com/adobe/cluster-registry/test/slt/metrics"
)

var (
	token       string
	tokenRWLock sync.RWMutex
)

const MetricLabelToken = "token_refresh"

var logger echo.Logger

func init() {
	metrics.RegisterMetrics()

	update.InitMetrics()
	request.InitMetrics()

	metrics.ErrCnt.WithLabelValues(MetricLabelToken).Add(0)
}

// SetLogger sets the global logger for the all the checks
func SetLogger(lgr echo.Logger) {
	logger = lgr
	update.SetLogger(lgr)
	request.SetLogger(lgr)
}

// readToken is an atomic getter for the token
func readToken() string {
	tokenRWLock.RLock()
	defer tokenRWLock.RUnlock()

	for token == "" {
		// Wait for the token to get initialized
		tokenRWLock.RUnlock()
		time.Sleep(1 * time.Second)
		tokenRWLock.RLock()
	}

	return token
}

// GetAuthDetails gets auth details from the env
func GetAuthDetails() (resourceID, tenantID, clientID, clientSecret string) {
	resourceID = h.GetEnv("RESOURCE_ID", "", logger)  // Cluster Registry App ID
	tenantID = h.GetEnv("TENANT_ID", "", logger)      // Adobe.com tenant ID
	clientID = h.GetEnv("CLIENT_ID", "", logger)      // Your App ID
	clientSecret = h.GetEnv("APP_SECRET", "", logger) // Your App Secret

	return resourceID, tenantID, clientID, clientSecret
}

// requestToken gets an jwt for authenticating to CR
func requestToken(resourceID, tenantID, clientID, clientSecret string) (string, error) {
	clientCredentials := auth.NewClientCredentialsConfig(clientID, clientSecret, tenantID)

	token, err := clientCredentials.ServicePrincipalToken()
	if err != nil {
		return "", err
	}

	err = token.RefreshExchange(resourceID)
	if err != nil {
		return "", err
	}

	return token.Token().AccessToken, nil
}

// Use this function when testing generating a test token in the local env
func debugGenerateToken(resourceID, tenantID, clientID, clientSecret string) (string, error) {
	appConfig, _ := config.LoadApiConfig()
	return jwt.GenerateDefaultSignedToken(appConfig), nil

}

// RefreshToken refreshes the token global variable in this package
func RefreshToken(_ interface{}) {
	// The '_' parameter is only so this function can be passed to RunFunctionInLoop

	resourceID, tenantID, clientID, clientSecret := GetAuthDetails()

	localToken, err := requestToken(resourceID, tenantID, clientID, clientSecret)
	if err != nil {
		metrics.ErrCnt.WithLabelValues(MetricLabelToken).Inc()
		logger.Fatalf("error getting jwt token: %s", err)
	}

	tokenRWLock.Lock()
	token = localToken
	tokenRWLock.Unlock()
	logger.Info("the Cluster Registry token just got refreshed.")
}

// RunE2eTest starts e2e test
func RunE2eTest(config interface{}) {

	conf, ok := config.(update.TestConfig)
	if !ok {
		logger.Fatal("failed to type assert config for the e2e test")
	}

	jwtToken := readToken()

	start := time.Now()

	_, nrOfTries, err := update.Run(conf, jwtToken)
	if err != nil {
		metrics.ErrCnt.WithLabelValues(update.MetricLabel).Inc()
		metrics.TestStatus.WithLabelValues(update.MetricLabel).Set(0)
		logger.Error(err)
		return
	}

	timeTook := float64(time.Since(start).Seconds())

	// 11 is the sleep between the tries
	metrics.TestDuration.WithLabelValues(update.MetricLabel).Observe(timeTook - float64(nrOfTries*11))
	metrics.TestStatus.WithLabelValues(update.MetricLabel).Set(1)
}

// RunClusterRequest run a GET request to CR on the /clusters/[clustername] endpoint
func RunClusterRequest(config interface{}) {
	logger.Info("timing the request that gets a cluster...")

	conf, ok := config.(request.GetClusterConfig)
	if !ok {
		logger.Fatal("failed to type assert config for the get a cluster test")
	}

	jwtToken := readToken()

	start := time.Now()
	err := request.RunGetCluster(conf, jwtToken)
	timeTook := float64(time.Since(start).Seconds())

	if err != nil {
		logger.Errorf("failed timing the request that gets a cluster: %s", err.Error())
		metrics.TestStatus.WithLabelValues(request.MetricLabelGetCluster).Set(0)
		metrics.ErrCnt.WithLabelValues(request.MetricLabelGetCluster).Inc()
		return
	}
	logger.Infof("timing completed for the request that gets a cluster: took %fs", timeTook)

	metrics.TestStatus.WithLabelValues(request.MetricLabelGetCluster).Set(1)
	metrics.TestDuration.WithLabelValues(request.MetricLabelGetCluster).Observe(timeTook)
}

// RunAllClustersRequests run a GET request to CR on the /clusters endpoint
func RunAllClustersRequests(config interface{}) {
	logger.Info("timing the request that gets multiple clusters...")

	conf, ok := config.(request.GetAllClusterConfig)
	if !ok {
		logger.Fatal("failed to type assert config for the get multiple clusters test")
	}

	jwtToken := readToken()

	start := time.Now()
	err := request.RunGetAllClusters(conf, jwtToken)
	timeTook := float64(time.Since(start).Seconds())

	if err != nil {
		logger.Errorf("failed timing the request that gets multiple clusters: %s", err.Error())
		metrics.TestStatus.WithLabelValues(request.MetricLabelGetAllClusters).Set(0)
		metrics.ErrCnt.WithLabelValues(request.MetricLabelGetAllClusters).Inc()
		return
	}
	logger.Infof("timing completed for the request that gets multiple clusters: took %fs", timeTook)

	metrics.TestStatus.WithLabelValues(request.MetricLabelGetAllClusters).Set(1)
	metrics.TestDuration.WithLabelValues(request.MetricLabelGetAllClusters).Observe(timeTook)
}
