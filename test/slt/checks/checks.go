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

var logger echo.Logger

var (
	token       string
	tokenRWLock sync.RWMutex
)

// SetLogger sets the global logger for the all the checks
func SetLogger(lgr echo.Logger) {
	update.SetLogger(lgr)
	request.SetLogger(lgr)
	logger = lgr
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
		logger.Fatalf("error getting jwt token: %s", err)
	}

	tokenRWLock.Lock()
	token = localToken
	tokenRWLock.Unlock()
	logger.Info("the Cluster Registry token just got refreshed.")
}

func init() {
	metrics.RegisterMetrics()
}

// RunE2eTest starts e2e test
func RunE2eTest(config interface{}) {

	conf, ok := config.(update.TestConfig)
	if !ok {
		logger.Fatal("failed to type assert config for the e2e test")
	}

	jwtToken := readToken()

	start := time.Now()

	status, nrOfTries, err := update.Run(conf, jwtToken)
	if err != nil {
		logger.Fatal(err)
	}

	timeTook := float64(time.Since(start).Seconds())

	metrics.E2eStatus.Set(float64(status))
	metrics.E2eDuration.Set(timeTook)
	metrics.E2eProcessingDuration.Set(timeTook - float64(nrOfTries*11)) // 11 is the sleep between the tries
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
		metrics.AllClustersReqDuration.Set(0)
		return
	}
	logger.Infof("timing completed for the request that gets a cluster: took %fs", timeTook)

	metrics.ClusterReqDuration.Set(timeTook)
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
		metrics.AllClustersReqDuration.Set(0)
		return
	}
	logger.Infof("timing completed for the request that gets multiple clusters: took %fs", timeTook)

	metrics.AllClustersReqDuration.Set(timeTook)
}
