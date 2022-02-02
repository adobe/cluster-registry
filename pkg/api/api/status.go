package api

import (
	"log"
	"net/http"

	"github.com/adobe/cluster-registry/pkg/api/database"
	"github.com/adobe/cluster-registry/pkg/api/utils"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"

	"github.com/labstack/echo/v4"
)

type serviceStatus struct {
	Status bool   `json:"status"`
	Error  string `json:"error"`
}

type status struct {
	Database serviceStatus `json:"database"`
	Sqs      serviceStatus `json:"sqs"`
}

// StatusSessions is used to keep the same objects and state for the database
// and sqs that are used for the rest of the calls inside the project
type StatusSessions struct {
	Sqs       sqsiface.SQSAPI
	Db        database.Db
	AppConfig *utils.AppConfig
}

func (s *StatusSessions) checkDBStatus() (databaseStatus serviceStatus) {
	if err := s.Db.CheckConnectivity(); err != nil {
		databaseStatus = serviceStatus{
			Status: false,
			Error:  err.Error(),
		}
	} else {
		databaseStatus = serviceStatus{
			Status: true,
			Error:  "",
		}
	}

	return databaseStatus
}

func (s *StatusSessions) checkSqsStatus() (sqsStatus serviceStatus) {
	_, err := s.Sqs.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: &s.AppConfig.SqsQueueName,
	})

	if err != nil {
		log.Fatal(err.Error())
		sqsStatus = serviceStatus{
			Status: false,
			Error:  err.Error(),
		}
	} else {
		sqsStatus = serviceStatus{
			Status: true,
			Error:  "",
		}
	}

	return sqsStatus
}

// ServiceStatus checks if the services that the api uses are healthy
func (s *StatusSessions) ServiceStatus(c echo.Context) error {

	statusResponse := status{
		Database: s.checkDBStatus(),
		Sqs:      s.checkSqsStatus(),
	}

	return c.JSON(http.StatusOK, statusResponse)
}
