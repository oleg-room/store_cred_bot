package db

import (
	"github.com/leesper/couchdb-golang"
	"github.com/sirupsen/logrus"
	"strings"
	"tg_bot/models"
)

type Couch struct {
	CouchClient    *couchdb.Server
	WorkingCouchDB *couchdb.Database
}

func (c *Couch) InitConnection(url, login, pass string) error {
	logrus.Trace("Creating elastic client...")

	serv, err := couchdb.NewServer("http://" + login + ":" + pass + "@" + strings.TrimLeft(url, "http://"))
	if err != nil {
		return err
	}
	c.CouchClient = serv
	return nil
}

func (c *Couch) CreateDatabase(dbName string) error {
	db, err := c.CouchClient.Create(dbName)
	if err != nil {
		return err
	}
	c.WorkingCouchDB = db
	return nil
}

func (c *Couch) UpdateService() error {
	return nil
}

func (c *Couch) SaveServiceCreds(service models.Service) error {
	return nil
}

func (c *Couch) GetService(serviceName string) (*models.Service, error) {
	return nil, nil
}

func (c *Couch) DeleteService(serviceName string) (bool, error) {
	return false, nil
}
